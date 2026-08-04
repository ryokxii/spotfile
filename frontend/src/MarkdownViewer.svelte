<script lang="ts">
import { tick, onDestroy } from 'svelte';
import { ReadFileAsBase64 } from '../wailsjs/go/main/App.js';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import { decodeBase64Utf8 } from './lib/base64';
import { highlightAndScroll, clearHighlight } from './lib/textHighlight';

export let docPath: string;
export let highlightText: string = '';   // matched chunk (raw markdown) to locate
export let onClose: () => void = () => {};

let html = '';
let error = '';
let loading = true;
let loadedPath = '';
let bodyEl: HTMLElement;
let scrollEl: HTMLElement;

async function loadMarkdown() {
  loading = true;
  error = '';
  try {
    const b64 = await ReadFileAsBase64(docPath);
    const decoded = decodeBase64Utf8(b64);
    html = DOMPurify.sanitize(await marked.parse(decoded));
  } catch (e: any) {
    error = `Failed to load Markdown: ${e?.message || e}`;
  }
  loading = false;
  await tick();
  doHighlight();
}

// The chunk is raw markdown; the DOM shows rendered text. Strip the chunk
// through the same renderer so the plain text lines up with what's on screen.
async function doHighlight() {
  if (loading || error || !bodyEl || !scrollEl || !highlightText) return;
  const chunkHtml = await marked.parse(highlightText);
  const needle = new DOMParser().parseFromString(chunkHtml, 'text/html').body.textContent || '';
  highlightAndScroll(bodyEl, scrollEl, needle);
}

$: reactToProps(docPath, highlightText);
function reactToProps(path: string, _hl: string) {
  if (path !== loadedPath) {
    loadedPath = path;
    loadMarkdown();
  } else if (!loading) {
    tick().then(doHighlight);
  }
}

onDestroy(clearHighlight);
</script>

<div class="md-viewer-panel">
  <div class="md-viewer-container">
    <header class="md-viewer-header">
      <span class="doc-name" title={docPath}>{docPath.split('/').pop()}</span>
      <button class="close-btn" on:click={onClose} aria-label="Close Markdown viewer">&times;</button>
    </header>
    <div class="md-content" bind:this={scrollEl}>
      {#if loading}
        <div class="loading">Loading Markdown…</div>
      {:else if error}
        <div class="viewer-error">{error}</div>
      {:else}
        <!-- html is sanitized with DOMPurify before assignment -->
        <div class="markdown-body" bind:this={bodyEl}>{@html html}</div>
      {/if}
    </div>
  </div>
</div>

<style>
.md-viewer-panel {
  width: 100%;
  height: 100%;
  display: flex;
  background: #141414;
}
.md-viewer-container {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.md-viewer-header {
  display: flex;
  align-items: center;
  padding: 0.5rem 1rem;
  background: #1a1a1a;
  color: #fff;
  border-bottom: 1px solid #222;
}
.doc-name {
  flex: 1;
  font-weight: bold;
  font-size: 1.1rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.close-btn {
  background: none;
  border: none;
  color: #fff;
  font-size: 1.5rem;
  cursor: pointer;
  margin-left: 1rem;
}
.md-content {
  flex: 1;
  overflow: auto;
  padding: 1.5rem;
  background: #181818;
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

.loading, .viewer-error {
  color: #c28a04;
  padding: 2rem;
  text-align: center;
}
</style>
