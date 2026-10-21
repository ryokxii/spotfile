<script lang="ts">
  // Recent searches as chips; clicking one runs it again in Chat.
  import { createEventDispatcher } from 'svelte'
  import type { RecentSearch } from '../../stores/dashboard'

  export let searches: RecentSearch[]
  export let history: boolean

  const dispatch = createEventDispatcher<{ search: string }>()
</script>

<section class="tile recent-searches" aria-labelledby="recent-searches-title">
  <h2 id="recent-searches-title" class="eyebrow">Recent searches</h2>
  {#if !history}
    <p class="muted">History is off · <a href="#/settings">Turn on in Settings</a></p>
  {:else if searches.length === 0}
    <p class="muted">Your searches will appear here</p>
  {:else}
    <ul class="chips">
      {#each searches as s (s.query)}
        <li><button type="button" class="chip" on:click={() => dispatch('search', s.query)}>{s.query}</button></li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .chips {
    list-style: none;
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
    margin-top: var(--space-sm);
  }

  .chip {
    padding: var(--space-xxs) var(--space-sm);
    border-radius: 999px;
    border: 1px solid var(--color-hairline);
    background-color: var(--color-surface-raised);
    color: var(--color-ink-secondary);
    font-size: var(--text-small-size);
    transition:
      color var(--motion-quick) var(--ease-standard),
      border-color var(--motion-quick) var(--ease-standard);
  }

  .chip:hover,
  .chip:focus-visible {
    color: var(--color-primary);
    border-color: var(--color-primary);
  }

  a {
    color: var(--color-primary);
  }
</style>
