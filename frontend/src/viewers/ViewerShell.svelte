<script lang="ts">
  // Shared chrome for docked file preview panes: panel frame, header with the
  // filename and a close button, and a body region for viewer-specific content.
  // A `toolbar` slot sits between the filename and close button for per-viewer
  // controls (e.g. the PDF pager/zoom).
  import { filename } from '../lib/path'

  export let docPath: string
  export let onClose: () => void = () => {}
</script>

<div class="viewer-panel">
  <header class="viewer-header">
    <span class="doc-name" title={docPath}>{filename(docPath)}</span>
    <slot name="toolbar" />
    <button class="close-btn" on:click={onClose} aria-label="Close viewer">&times;</button>
  </header>
  <div class="viewer-body">
    <slot />
  </div>
</div>

<style>
  .viewer-panel {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #141414;
    overflow: hidden;
  }

  .viewer-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.65rem 1rem;
    background: #1c1c1e;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    flex-shrink: 0;
  }

  .doc-name {
    flex: 1;
    min-width: 0;
    font-size: 0.85rem;
    color: #aeaeb2;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .close-btn {
    background: rgba(255, 255, 255, 0.08);
    border: none;
    color: #aeaeb2;
    font-size: 1.3rem;
    line-height: 1;
    padding: 0.15rem 0.5rem;
    border-radius: 5px;
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.15s, color 0.15s;
  }
  .close-btn:hover { background: rgba(255, 69, 58, 0.2); color: #ff453a; }

  .viewer-body {
    flex: 1;
    min-height: 0;
    display: flex;
    overflow: hidden;
  }
</style>
