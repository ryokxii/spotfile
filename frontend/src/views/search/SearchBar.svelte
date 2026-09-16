<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import Icon from '../../ui/Icon.svelte'
  import PressableSurface from '../../ui/PressableSurface.svelte'

  export let value = ''
  export let disabled = false
  export let loading = false
  export let placeholder = 'Search your files…'
  export let folderTitle = 'Choose a folder to index'

  const dispatch = createEventDispatcher<{ submit: string; chooseFolder: void }>()
</script>

<form class="bar" class:disabled role="search" on:submit|preventDefault={() => dispatch('submit', value)}>
  <label class="visually-hidden" for="search-input">Search your files</label>
  <span class="leading" aria-hidden="true">
    {#if loading}<span class="spinner" />{:else}<Icon name="search" />{/if}
  </span>
  <input
    id="search-input"
    type="search"
    bind:value
    {placeholder}
    {disabled}
    autocomplete="off"
    spellcheck="false"
  />
  <PressableSurface label={folderTitle} title={folderTitle} class="folder" on:click={() => dispatch('chooseFolder')}>
    <Icon name="folder" />
  </PressableSurface>
</form>

<style>
  .bar {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    height: calc(var(--space-xxl) + var(--space-xs));
    padding: 0 var(--space-xs) 0 var(--space-md);
    background-color: var(--color-surface-sunken);
    border: 1px solid var(--color-hairline);
    border-radius: var(--radius-md);
    transition: border-color var(--motion-quick) var(--ease-standard);
  }

  .bar:focus-within {
    border-color: var(--color-primary);
  }

  .leading {
    display: flex;
    color: var(--color-ink-faint);
  }

  input {
    flex: 1;
    min-width: 0;
    height: 100%;
    background: none;
    border: none;
    outline: none;
    font-size: var(--text-heading-size);
    caret-color: var(--color-primary);
  }

  input::placeholder {
    color: var(--color-ink-faint);
  }

  input::-webkit-search-cancel-button {
    display: none;
  }

  input:disabled {
    cursor: not-allowed;
  }

  .disabled input::placeholder {
    opacity: 0.7;
  }

  :global(.folder) {
    padding: var(--space-xs);
    color: var(--color-ink-secondary);
  }

  .spinner {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    border: 2px solid var(--color-hairline-strong);
    border-top-color: var(--color-primary);
    animation: spin var(--motion-spin) linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
