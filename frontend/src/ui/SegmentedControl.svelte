<script lang="ts">
  // One choice from a few options: a radiogroup where the arrow keys move the
  // selection, as native radio buttons do. Controlled: dispatches `change` and
  // waits for the parent to pass the new value back.
  import { createEventDispatcher } from 'svelte'

  export let label: string
  export let options: { value: string; label: string }[]
  export let value: string

  const dispatch = createEventDispatcher<{ change: string }>()
  const NEXT = new Set(['ArrowRight', 'ArrowDown'])
  const PREVIOUS = new Set(['ArrowLeft', 'ArrowUp'])
  let buttons: HTMLButtonElement[] = []

  function select(index: number) {
    buttons[index]?.focus()
    const chosen = options[index].value
    if (chosen !== value) dispatch('change', chosen)
  }

  function onKeydown(e: KeyboardEvent, index: number) {
    const step = NEXT.has(e.key) ? 1 : PREVIOUS.has(e.key) ? -1 : 0
    if (step === 0) return
    e.preventDefault()
    select((index + step + options.length) % options.length)
  }
</script>

<div class="segmented" role="radiogroup" aria-label={label}>
  {#each options as option, i (option.value)}
    <button
      bind:this={buttons[i]}
      type="button"
      role="radio"
      class="segment"
      class:selected={option.value === value}
      aria-checked={option.value === value}
      tabindex={option.value === value ? 0 : -1}
      on:click={() => select(i)}
      on:keydown={(e) => onKeydown(e, i)}
    >
      {option.label}
    </button>
  {/each}
</div>

<style>
  .segmented {
    display: inline-flex;
    gap: var(--space-xxs);
    padding: var(--space-xxs);
    border-radius: var(--radius-sm);
    border: 1px solid var(--color-hairline);
    background-color: var(--color-surface-sunken);
  }

  .segment {
    padding: var(--space-xxs) var(--space-sm);
    border-radius: var(--radius-xs);
    font-size: var(--text-small-size);
    font-weight: var(--text-body-strong-weight);
    color: var(--color-ink-secondary);
    transition:
      transform var(--motion-press) var(--ease-press),
      background-color var(--motion-quick) var(--ease-standard),
      color var(--motion-quick) var(--ease-standard);
  }

  .segment:hover {
    color: var(--color-ink);
  }

  .segment.selected {
    background-color: var(--color-surface-raised);
    color: var(--color-ink);
    box-shadow: var(--shadow-raised);
  }

  .segment:active {
    transform: scale(var(--press-scale));
    transition-duration: var(--motion-instant);
  }
</style>
