<script lang="ts">
  // Plain-text and Markdown preview. `mode` selects rendering: 'text' shows the
  // raw file in a <pre>; 'markdown' renders sanitized HTML. Both share one
  // load/decode/highlight lifecycle.
  import { tick, onDestroy } from 'svelte'
  import { marked } from 'marked'
  import DOMPurify from 'dompurify'
  import { ReadFileAsBase64 } from '../../wailsjs/go/main/App.js'
  import { decodeBase64Utf8 } from '../lib/base64'
  import { highlightAndScroll, clearHighlight } from '../lib/textHighlight'
  import ViewerShell from './ViewerShell.svelte'

  export let docPath: string
  export let mode: 'text' | 'markdown' = 'text'
  export let highlightText: string = '' // matched chunk to locate and highlight
  export let onClose: () => void = () => {}

  let text = '' // raw text (mode: 'text')
  let html = '' // sanitized HTML (mode: 'markdown')
  let error = ''
  let loading = true
  let loadedKey = ''
  let bodyEl: HTMLElement
  let scrollEl: HTMLElement

  async function load() {
    loading = true
    error = ''
    try {
      const decoded = decodeBase64Utf8(await ReadFileAsBase64(docPath))
      if (mode === 'markdown') {
        html = DOMPurify.sanitize(await marked.parse(decoded))
      } else {
        text = decoded
      }
    } catch (e: any) {
      error = `Failed to load file: ${e?.message || e}`
    }
    loading = false
    await tick()
    doHighlight()
  }

  // The chunk is raw source text; in markdown the DOM shows *rendered* text, so
  // strip the chunk through the same renderer to align it with what's on screen.
  async function doHighlight() {
    if (loading || error || !bodyEl || !scrollEl || !highlightText) return
    let needle = highlightText
    if (mode === 'markdown') {
      const chunkHtml = await marked.parse(highlightText)
      needle = new DOMParser().parseFromString(chunkHtml, 'text/html').body.textContent || ''
    }
    highlightAndScroll(bodyEl, scrollEl, needle)
  }

  // Reload when the file (or mode) changes; otherwise just re-highlight. Keying
  // on both props keeps this reactive to either.
  $: reactToProps(`${mode}:${docPath}`, highlightText)
  function reactToProps(key: string, _hl: string) {
    if (key !== loadedKey) {
      loadedKey = key
      load()
    } else if (!loading) {
      tick().then(doHighlight)
    }
  }

  onDestroy(clearHighlight)
</script>

<ViewerShell {docPath} {onClose}>
  <div class="content" bind:this={scrollEl}>
    {#if loading}
      <div class="loading">Loading…</div>
    {:else if error}
      <div class="viewer-error">{error}</div>
    {:else if mode === 'markdown'}
      <!-- html is sanitized with DOMPurify before assignment -->
      <div class="markdown-body" bind:this={bodyEl}>{@html html}</div>
    {:else}
      <pre class="text-body" bind:this={bodyEl}>{text}</pre>
    {/if}
  </div>
</ViewerShell>

<style>
  .content {
    flex: 1;
    min-width: 0;
    overflow: auto;
    padding: 1.5rem;
    background: #181818;
  }

  .loading,
  .viewer-error {
    color: #c28a04;
    padding: 2rem;
    text-align: center;
  }

  .text-body {
    color: #eaeaea;
    font-size: 1rem;
    line-height: 1.7;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .markdown-body {
    color: #eaeaea;
    font-size: 1rem;
    line-height: 1.7;
  }

  /* Injected {@html} content isn't visible to Svelte's scoping, so these
     descendant rules must be :global() or they get stripped as unused. */
  .markdown-body :global(h1),
  .markdown-body :global(h2),
  .markdown-body :global(h3),
  .markdown-body :global(h4),
  .markdown-body :global(h5),
  .markdown-body :global(h6) {
    color: #fff;
    font-weight: 600;
    line-height: 1.25;
    margin: 1.6em 0 0.6em;
  }
  .markdown-body :global(h1:first-child),
  .markdown-body :global(h2:first-child),
  .markdown-body :global(h3:first-child) {
    margin-top: 0;
  }
  .markdown-body :global(h1) { font-size: 1.8rem; padding-bottom: 0.3em; border-bottom: 1px solid #2a2a2a; }
  .markdown-body :global(h2) { font-size: 1.45rem; padding-bottom: 0.25em; border-bottom: 1px solid #242424; }
  .markdown-body :global(h3) { font-size: 1.2rem; }
  .markdown-body :global(h4) { font-size: 1.05rem; }
  .markdown-body :global(h5) { font-size: 0.95rem; }
  .markdown-body :global(h6) { font-size: 0.9rem; color: #b8b8b8; }

  .markdown-body :global(p) { margin: 0 0 1em; }

  .markdown-body :global(a) { color: #e0a824; text-decoration: none; }
  .markdown-body :global(a:hover) { text-decoration: underline; }

  .markdown-body :global(ul),
  .markdown-body :global(ol) { margin: 0 0 1em; padding-left: 1.6em; }
  .markdown-body :global(ul) { list-style: disc; }
  .markdown-body :global(ol) { list-style: decimal; }
  .markdown-body :global(li) { margin: 0.25em 0; }
  .markdown-body :global(li > ul),
  .markdown-body :global(li > ol) { margin: 0.25em 0; }

  .markdown-body :global(code) {
    font-family: 'SF Mono', 'JetBrains Mono', Menlo, Consolas, monospace;
    font-size: 0.88em;
    background: #26262b;
    padding: 0.15em 0.4em;
    border-radius: 4px;
  }
  .markdown-body :global(pre) {
    background: #1c1c20;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 1em;
    overflow-x: auto;
    margin: 0 0 1em;
  }
  .markdown-body :global(pre code) {
    background: none;
    padding: 0;
    font-size: 0.85rem;
    line-height: 1.55;
  }

  .markdown-body :global(blockquote) {
    margin: 0 0 1em;
    padding: 0.2em 1em;
    border-left: 3px solid #c28a04;
    color: #b6b6b6;
  }

  .markdown-body :global(hr) {
    border: none;
    border-top: 1px solid #2a2a2a;
    margin: 1.8em 0;
  }

  .markdown-body :global(table) {
    border-collapse: collapse;
    margin: 0 0 1em;
    display: block;
    overflow-x: auto;
    max-width: 100%;
  }
  .markdown-body :global(th),
  .markdown-body :global(td) {
    border: 1px solid #2f2f2f;
    padding: 0.5em 0.8em;
    text-align: left;
  }
  .markdown-body :global(th) { background: #242428; font-weight: 600; }
  .markdown-body :global(tr:nth-child(even) td) { background: #1e1e22; }

  .markdown-body :global(img) { max-width: 100%; height: auto; border-radius: 6px; }

  .markdown-body :global(strong) { color: #fff; font-weight: 600; }
</style>
