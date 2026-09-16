<script lang="ts">
  import { dirpath, filename } from '../../lib/path'
  import type { SearchResult } from '../../stores/search'
  import Icon from '../../ui/Icon.svelte'
  import PressableSurface from '../../ui/PressableSurface.svelte'

  export let result: SearchResult
  export let selected = false

  $: isPdf = result.docPath.toLowerCase().endsWith('.pdf')
  $: excerpt = result.text.replace(/\s+/g, ' ').trim()
</script>

<PressableSurface plane="desk" {selected} class="result" title={result.docPath} on:click>
  <span class="marker" class:on={selected} aria-hidden="true" />
  <span class="body">
    <span class="head">
      <Icon name="file" size={16} />
      <span class="name">{filename(result.docPath)}</span>
      {#if isPdf}<span class="meta">p.{result.pageNum || 1}</span>{/if}
      <span class="meta score" title="Similarity">{result.score.toFixed(2)}</span>
    </span>
    <span class="path">{dirpath(result.docPath)}</span>
    <span class="excerpt">{excerpt}</span>
  </span>
</PressableSurface>

<style>
  :global(.result) {
    position: relative;
    width: 100%;
    padding: var(--space-md) var(--space-lg);
    border-radius: var(--radius-md);
    overflow: hidden;
  }

  .marker {
    position: absolute;
    left: 0;
    top: var(--space-md);
    bottom: var(--space-md);
    width: 2px;
    background-color: var(--color-primary);
    transform: scaleY(0);
    transition: transform var(--motion-quick) var(--ease-standard);
  }

  .marker.on {
    transform: scaleY(1);
  }

  .body {
    display: flex;
    flex-direction: column;
    gap: var(--space-xxs);
    min-width: 0;
    width: 100%;
  }

  .head {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    color: var(--color-ink-faint);
  }

  .name {
    flex: 1;
    min-width: 0;
    font-weight: var(--text-body-strong-weight);
    color: var(--color-ink);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .meta {
    font-size: var(--text-small-size);
    font-weight: var(--text-numeric-weight);
    font-variant-numeric: tabular-nums;
    color: var(--color-ink-faint);
  }

  .path {
    font-size: var(--text-small-size);
    line-height: var(--text-small-line);
    color: var(--color-ink-faint);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .excerpt {
    margin-top: var(--space-xxs);
    color: var(--color-ink-secondary);
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    overflow: hidden;
  }
</style>
