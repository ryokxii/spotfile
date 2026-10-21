<script lang="ts">
  // The Dashboard's search bar. Enter dispatches `search` with the query; the
  // Dashboard then shows the results in Chat. ⌘K / Ctrl+K focuses it.
  import { createEventDispatcher } from 'svelte'
  import Icon from '../../ui/Icon.svelte'

  export let disabled = false
  export let placeholder = 'Search your files…'

  const dispatch = createEventDispatcher<{ search: string }>()
  const shortcut = /Mac|iPhone|iPad/.test(navigator.userAgent) ? '⌘K' : 'Ctrl K'
  let input: HTMLInputElement
  let query = ''

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && query.trim()) dispatch('search', query.trim())
  }

  function onWindowKeydown(e: KeyboardEvent) {
    if (e.key.toLowerCase() === 'k' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      input?.focus()
    }
  }
</script>

<svelte:window on:keydown={onWindowKeydown} />

<label class="dashboard-search" class:disabled>
  <Icon name="search" />
  <input bind:this={input} bind:value={query} type="search" {placeholder} {disabled} aria-label="Search your files" on:keydown={onKeydown} />
  <kbd aria-hidden="true">{shortcut}</kbd>
</label>

<style>
  .dashboard-search {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-hairline-strong);
    background-color: var(--color-surface);
    box-shadow: var(--shadow-raised);
    color: var(--color-ink-faint);
    transition: border-color var(--motion-quick) var(--ease-standard);
  }

  .dashboard-search:focus-within {
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px var(--color-primary-wash);
  }

  .dashboard-search.disabled {
    opacity: 0.6;
  }

  input {
    flex: 1;
    min-width: 0;
    border: none;
    background: none;
    color: var(--color-ink);
    font-size: var(--text-heading-size);
  }

  input:focus {
    outline: none;
  }

  kbd {
    padding: 0 var(--space-xxs);
    border-radius: var(--radius-xs);
    border: 1px solid var(--color-hairline-strong);
    font-family: var(--font-sans);
    font-size: var(--text-micro-size);
  }
</style>
