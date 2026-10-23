<script lang="ts">
  // The display name at the bottom of the sidebar; opens Profile.
  import { initials } from '../lib/greeting'
  import PressableSurface from './PressableSurface.svelte'

  export let name = ''
  export let compact = false
  export let selected = false

  $: letters = initials(name)
  $: label = name ? `Profile — ${name}` : 'Add your name'
</script>

<PressableSurface
  href="#/profile"
  {selected}
  label={compact ? label : undefined}
  title={compact ? label : undefined}
  class="profile-button {compact ? 'compact' : ''}"
>
  <span class="avatar" aria-hidden={compact ? undefined : 'true'}>{letters || '+'}</span>
  {#if !compact}<span class="name" class:empty={!name}>{name || 'Add your name'}</span>{/if}
</PressableSurface>

<style>
  :global(.profile-button) {
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-xs) var(--space-sm);
    color: var(--color-ink-secondary);
  }

  :global(.profile-button.compact) {
    justify-content: center;
    padding: var(--space-xs) 0;
  }

  :global(.profile-button.selected) {
    color: var(--color-primary);
  }

  .avatar {
    display: grid;
    place-items: center;
    flex-shrink: 0;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background-color: var(--color-secondary-wash-strong);
    color: var(--color-secondary);
    font-size: var(--text-micro-size);
    font-weight: var(--text-micro-weight);
  }

  .name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--text-body-size);
    font-weight: var(--text-body-strong-weight);
  }

  .name.empty {
    color: var(--color-ink-faint);
    font-weight: var(--text-body-weight);
  }
</style>
