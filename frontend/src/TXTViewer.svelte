<script lang="ts">
import { tick, onDestroy } from 'svelte';
import { ReadFileAsBase64 } from '../wailsjs/go/main/App.js';
import { decodeBase64Utf8 } from './lib/base64';
import { highlightAndScroll, clearHighlight } from './lib/textHighlight';

export let docPath: string;
export let highlightText: string = '';   // matched chunk to locate and highlight
export let onClose: () => void = () => {};

let text = '';
let error = '';
let loading = true;
let loadedPath = '';
let bodyEl: HTMLElement;
let scrollEl: HTMLElement;

async function loadText() {
  loading = true;
  error = '';
  try {
    const b64 = await ReadFileAsBase64(docPath);
    text = decodeBase64Utf8(b64);
  } catch (e: any) {
    error = `Failed to load text: ${e?.message || e}`;
  }
  loading = false;
  await tick();
  doHighlight();
}

function doHighlight() {
  if (loading || error || !bodyEl || !scrollEl || !highlightText) return;
  highlightAndScroll(bodyEl, scrollEl, highlightText);
}

// Load when the file changes; otherwise (same file, different chunk) just
// re-highlight. Referencing both props keeps this reactive to either change.
$: reactToProps(docPath, highlightText);
function reactToProps(path: string, _hl: string) {
  if (path !== loadedPath) {
    loadedPath = path;
    loadText();
  } else if (!loading) {
    tick().then(doHighlight);
  }
}

onDestroy(clearHighlight);
</script>

<div class="txt-viewer-panel">
  <div class="txt-viewer-container">
    <header class="txt-viewer-header">
      <span class="doc-name" title={docPath}>{docPath.split('/').pop()}</span>
      <button class="close-btn" on:click={onClose} aria-label="Close TXT viewer">&times;</button>
    </header>
    <div class="txt-content" bind:this={scrollEl}>
      {#if loading}
        <div class="loading">Loading text…</div>
      {:else if error}
        <div class="viewer-error">{error}</div>
      {:else}
        <pre class="text-body" bind:this={bodyEl}>{text}</pre>
      {/if}
    </div>
  </div>
</div>

<style>
.txt-viewer-panel {
  width: 100%;
  height: 100%;
  display: flex;
  background: #141414;
}
.txt-viewer-container {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.txt-viewer-header {
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
.txt-content {
  flex: 1;
  overflow: auto;
  padding: 1.5rem;
  background: #181818;
}
.text-body {
  color: #eaeaea;
  font-size: 1rem;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}
.loading, .viewer-error {
  color: #c28a04;
  padding: 2rem;
  text-align: center;
}
</style>
