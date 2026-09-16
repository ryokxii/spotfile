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
    padding: var(--space-lg) var(--space-xl) var(--space-xxl);
  }

  .loading,
  .viewer-error {
    padding: var(--space-xl);
    text-align: center;
    color: var(--color-ink-faint);
  }

  .viewer-error {
    color: var(--color-urgent);
  }

  .text-body {
    font-family: var(--font-code);
    font-size: var(--text-small-size);
    line-height: var(--text-body-line);
    color: var(--color-ink);
    white-space: pre-wrap;
    word-break: break-word;
  }

  .markdown-body {
    max-width: var(--search-column-max);
    color: var(--color-ink);
    line-height: var(--text-body-line);
  }

  /* Injected {@html} content isn't visible to Svelte's scoping, so these
     descendant rules must be :global() or they get stripped as unused. */
  .markdown-body :global(h1),
  .markdown-body :global(h2),
  .markdown-body :global(h3),
  .markdown-body :global(h4),
  .markdown-body :global(h5),
  .markdown-body :global(h6) {
    font-weight: var(--text-heading-weight);
    line-height: var(--text-title-line);
    margin: var(--space-lg) 0 var(--space-xs);
  }
  .markdown-body :global(h1:first-child),
  .markdown-body :global(h2:first-child),
  .markdown-body :global(h3:first-child) {
    margin-top: 0;
  }
  .markdown-body :global(h1) {
    font-size: var(--text-title-size);
    padding-bottom: var(--space-xs);
    border-bottom: 1px solid var(--color-hairline);
  }
  .markdown-body :global(h2) {
    font-size: var(--text-heading-size);
    padding-bottom: var(--space-xxs);
    border-bottom: 1px solid var(--color-hairline);
  }
  .markdown-body :global(h3),
  .markdown-body :global(h4) {
    font-size: var(--text-body-size);
  }
  .markdown-body :global(h5),
  .markdown-body :global(h6) {
    font-size: var(--text-small-size);
    color: var(--color-ink-secondary);
  }

  .markdown-body :global(p) {
    margin: 0 0 var(--space-md);
  }

  .markdown-body :global(a) {
    color: var(--color-secondary);
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  .markdown-body :global(ul),
  .markdown-body :global(ol) {
    margin: 0 0 var(--space-md);
    padding-left: var(--space-lg);
  }
  .markdown-body :global(ul) { list-style: disc; }
  .markdown-body :global(ol) { list-style: decimal; }
  .markdown-body :global(li) { margin: var(--space-xxs) 0; }

  .markdown-body :global(code) {
    font-family: var(--font-code);
    font-size: 0.9em;
    background-color: var(--color-surface-sunken);
    padding: 1px var(--space-xxs);
    border-radius: var(--radius-xs);
  }
  .markdown-body :global(pre) {
    background-color: var(--color-surface-sunken);
    border: 1px solid var(--color-hairline);
    border-radius: var(--radius-sm);
    padding: var(--space-md);
    overflow-x: auto;
    margin: 0 0 var(--space-md);
  }
  .markdown-body :global(pre code) {
    background: none;
    padding: 0;
    font-size: var(--text-small-size);
  }

  .markdown-body :global(blockquote) {
    margin: 0 0 var(--space-md);
    padding: var(--space-xxs) var(--space-md);
    border-left: 2px solid var(--color-hairline-strong);
    color: var(--color-ink-secondary);
  }

  .markdown-body :global(hr) {
    border: none;
    border-top: 1px solid var(--color-hairline);
    margin: var(--space-xl) 0;
  }

  .markdown-body :global(table) {
    border-collapse: collapse;
    margin: 0 0 var(--space-md);
    display: block;
    overflow-x: auto;
    max-width: 100%;
  }
  .markdown-body :global(th),
  .markdown-body :global(td) {
    border: 1px solid var(--color-hairline);
    padding: var(--space-xs) var(--space-sm);
    text-align: left;
  }
  .markdown-body :global(th) {
    background-color: var(--color-surface-raised);
    font-weight: var(--text-body-strong-weight);
  }

  .markdown-body :global(img) {
    max-width: 100%;
    height: auto;
    border-radius: var(--radius-xs);
  }

  .markdown-body :global(strong) {
    font-weight: var(--text-heading-weight);
  }
</style>
