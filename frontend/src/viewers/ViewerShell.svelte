<script lang="ts">
  // Shared chrome for docked file preview panes: panel frame, header with the
  // filename and a close button, and a body region for viewer-specific content.
  // A `toolbar` slot sits between the filename and close button for per-viewer
  // controls (e.g. the PDF pager/zoom).
  import { filename } from '../lib/path'
  import Icon from '../ui/Icon.svelte'
  import PressableSurface from '../ui/PressableSurface.svelte'

  export let docPath: string
  export let onClose: () => void = () => {}
</script>

<div class="viewer-panel">
  <header class="viewer-header">
    <span class="doc-name" title={docPath}>{filename(docPath)}</span>
    <slot name="toolbar" />
    <PressableSurface label="Close preview" title="Close preview (Esc)" class="viewer-close" on:click={onClose}>
      <Icon name="close" size={16} />
    </PressableSurface>
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
    background-color: var(--color-surface);
    overflow: hidden;
  }

  .viewer-header {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    min-height: calc(var(--space-xxl) + var(--space-xs));
    padding: var(--space-xs) var(--space-md) var(--space-xs) var(--space-lg);
    border-bottom: 1px solid var(--color-hairline);
    flex-shrink: 0;
  }

  .doc-name {
    flex: 1;
    min-width: 0;
    font-weight: var(--text-body-strong-weight);
    color: var(--color-ink);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.viewer-close) {
    padding: var(--space-xs);
    color: var(--color-ink-secondary);
  }

  .viewer-body {
    flex: 1;
    min-height: 0;
    display: flex;
    overflow: hidden;
  }
</style>
