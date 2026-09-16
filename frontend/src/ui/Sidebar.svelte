<script lang="ts">
  import { location } from 'svelte-spa-router'
  import type { Destination } from '../routes'
  import NavItem from './NavItem.svelte'

  export let destinations: Destination[]
  export let compact = false
</script>

<nav class="sidebar" class:compact aria-label="Main">
  <a class="brand" href="#/" aria-label="Spotfile home">
    <svg class="mark" viewBox="0 0 80 80" fill="none" aria-hidden="true">
      <rect x="5" y="5" width="70" height="70" rx="18" stroke="currentColor" stroke-width="6" />
      <rect x="19" y="19" width="42" height="42" rx="11" stroke="currentColor" stroke-width="5" opacity="0.7" />
      <rect x="31" y="31" width="18" height="18" rx="5" fill="currentColor" />
    </svg>
    {#if !compact}<span class="wordmark">Spotfile</span>{/if}
  </a>

  <ul class="nav">
    {#each destinations as d (d.path)}
      <li>
        <NavItem href={d.path} label={d.label} icon={d.icon} selected={$location === d.path} {compact} />
      </li>
    {/each}
  </ul>
</nav>

<style>
  .sidebar {
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--space-xl);
    padding: var(--space-lg) var(--space-md);
    background-color: var(--color-surface-sidebar);
    border-right: 1px solid var(--color-hairline);
    /* Leave room for the macOS window controls above the brand. */
    padding-top: var(--space-xxl);
  }

  .sidebar.compact {
    padding-left: var(--space-xs);
    padding-right: var(--space-xs);
    align-items: center;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: 0 var(--space-sm);
    color: var(--color-ink);
  }

  .mark {
    width: 22px;
    height: 22px;
    color: var(--color-primary);
    flex-shrink: 0;
  }

  .wordmark {
    font-family: var(--font-sans);
    font-size: var(--text-heading-size);
    font-weight: var(--text-heading-weight);
  }

  .nav {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: var(--space-xxs);
    width: 100%;
  }
</style>
