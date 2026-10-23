<script lang="ts">
  // Free up space: files untouched for a while, largest first. Shows three;
  // "Show all" opens the full list.
  import { createEventDispatcher } from 'svelte'
  import { formatBytes } from '../../lib/format'
  import type { StaleFile } from '../../stores/dashboard'
  import StaleFileList from './StaleFileList.svelte'

  export let stale: StaleFile[]
  export let staleBytes: number
  export let staleDays: number
  export let busy = false
  export let errors: Record<string, string> = {}
  export let now = new Date()

  const SHOWN = 3
  const dispatch = createEventDispatcher<{ showAll: null }>()
  $: count = `${stale.length} ${stale.length === 1 ? 'file' : 'files'}`
</script>

<section class="tile free-space" aria-labelledby="free-space-title">
  <header class="head">
    <h2 id="free-space-title" class="eyebrow">Free up space</h2>
    {#if stale.length > 0}
      <span class="muted">{count} untouched for {staleDays}+ days · {formatBytes(staleBytes)} of index</span>
    {/if}
  </header>
  {#if stale.length === 0}
    <p class="muted">Nothing stale. Everything was used in the last {staleDays} days.</p>
  {:else}
    <StaleFileList files={stale.slice(0, SHOWN)} {busy} {errors} {now} on:remove on:trash />
    {#if stale.length > SHOWN}
      <button type="button" class="show-all" on:click={() => dispatch('showAll', null)}>Show all {stale.length} →</button>
    {/if}
  {/if}
</section>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: var(--space-md);
    flex-wrap: wrap;
    margin-bottom: var(--space-xs);
  }

  .show-all {
    margin-top: var(--space-xs);
    color: var(--color-primary);
    font-size: var(--text-small-size);
  }

  .show-all:hover,
  .show-all:focus-visible {
    text-decoration: underline;
  }
</style>
