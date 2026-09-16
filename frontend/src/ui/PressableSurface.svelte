<script lang="ts">
  // The shared interactive surface (DESIGN.md §8): hover promotes desk → lamp,
  // focus adds the primary ring, press is the sprung scale, disabled is inert.
  // Renders a <button>, or an <a> when href is set.

  export let href: string | undefined = undefined
  export let disabled = false
  export let selected = false
  export let label: string | undefined = undefined // aria-label when content is icon-only
  export let title: string | undefined = undefined
  export let plane: 'none' | 'desk' = 'none' // resting background
  let className = ''
  export { className as class }
</script>

{#if href && !disabled}
  <a
    {href}
    {title}
    class="pressable {className}"
    class:desk={plane === 'desk'}
    class:selected
    aria-label={label}
    aria-current={selected ? 'page' : undefined}
    on:click
  >
    <slot />
  </a>
{:else}
  <button
    type="button"
    {title}
    class="pressable {className}"
    class:desk={plane === 'desk'}
    class:selected
    aria-label={label}
    aria-pressed={selected ? true : undefined}
    {disabled}
    on:click
  >
    <slot />
  </button>
{/if}

<style>
  .pressable {
    display: flex;
    align-items: center;
    text-align: left;
    border-radius: var(--radius-sm);
    border: 1px solid transparent;
    transition:
      transform var(--motion-press) var(--ease-press),
      background-color var(--motion-quick) var(--ease-standard),
      border-color var(--motion-quick) var(--ease-standard),
      color var(--motion-quick) var(--ease-standard);
  }

  .desk {
    background-color: var(--color-surface);
    border-color: var(--color-hairline);
  }

  .pressable:hover:not(:disabled),
  .pressable:focus-visible,
  .selected {
    background-color: var(--color-surface-raised);
    border-color: var(--color-hairline-strong);
    box-shadow: var(--shadow-raised);
  }

  .pressable:active:not(:disabled) {
    transform: scale(var(--press-scale));
    transition-duration: var(--motion-instant);
  }

  .pressable:disabled {
    cursor: default;
    opacity: 0.45;
  }
</style>
