<script lang="ts">
  // Results stagger in (DESIGN.md §5): one orchestration for the list, 40ms
  // cadence, 12px rise + fade over `slow`, all from motion tokens.
  import { createEventDispatcher } from 'svelte'
  import { fly } from 'svelte/transition'
  import { duration, easeOutExpo } from '../../lib/motion'
  import { resultKey, type SearchResult } from '../../stores/search'
  import ResultCard from './ResultCard.svelte'

  export let results: SearchResult[]
  export let selectedKey: string | null = null

  const dispatch = createEventDispatcher<{ open: SearchResult }>()
  const slow = duration('slow')
  const stagger = duration('stagger')
  const rise = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--motion-rise')) || 0
</script>

<ol class="results" aria-label="Search results">
  {#each results as result, i (resultKey(result))}
    <li in:fly={{ y: rise, duration: slow, delay: i * stagger, easing: easeOutExpo }}>
      <ResultCard {result} selected={resultKey(result) === selectedKey} on:click={() => dispatch('open', result)} />
    </li>
  {/each}
</ol>

<style>
  .results {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }
</style>
