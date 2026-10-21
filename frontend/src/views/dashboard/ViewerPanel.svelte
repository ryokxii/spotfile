<script lang="ts">
  // A file viewer sliding over the Dashboard. Esc or the close button dismisses it.
  import { createEventDispatcher } from 'svelte'
  import { fly } from 'svelte/transition'
  import { viewerKindFor } from '../../lib/fileTypes'
  import { duration } from '../../lib/motion'
  import PdfViewer from '../../viewers/PdfViewer.svelte'
  import TextViewer from '../../viewers/TextViewer.svelte'

  export let path: string | null = null

  const dispatch = createEventDispatcher<{ close: void }>()
  const close = () => dispatch('close')

  $: kind = path ? viewerKindFor(path) : 'text'

  function onKeydown(e: KeyboardEvent) {
    if (path && e.key === 'Escape') close()
  }
</script>

<svelte:window on:keydown={onKeydown} />

{#if path}
  <aside class="viewer-panel" aria-label="Preview" transition:fly={{ x: 48, duration: duration('normal') }}>
    {#key path}
      {#if kind === 'pdf'}
        <PdfViewer docPath={path} onClose={close} />
      {:else}
        <TextViewer docPath={path} mode={kind === 'markdown' ? 'markdown' : 'text'} onClose={close} />
      {/if}
    {/key}
  </aside>
{/if}

<style>
  .viewer-panel {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(720px, 62%);
    border-left: 1px solid var(--color-hairline-strong);
    box-shadow: var(--shadow-raised);
    z-index: 2;
  }
</style>
