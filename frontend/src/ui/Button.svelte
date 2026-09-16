<script lang="ts">
  // Action button. At most one `primary` per surface (DESIGN.md §2).
  export let variant: 'primary' | 'secondary' = 'secondary'
  export let disabled = false
</script>

<button type="button" class="button {variant}" {disabled} on:click>
  <slot />
</button>

<style>
  .button {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-sm) var(--space-lg);
    border-radius: var(--radius-sm);
    font-size: var(--text-body-size);
    font-weight: var(--text-body-strong-weight);
    line-height: 1;
    transition:
      transform var(--motion-press) var(--ease-press),
      background-color var(--motion-quick) var(--ease-standard);
  }

  .primary {
    background-color: var(--color-primary);
    color: var(--color-on-primary);
  }

  .primary:hover:not(:disabled) {
    background-color: var(--color-primary-hover);
  }

  .primary:active:not(:disabled) {
    background-color: var(--color-primary-deep);
  }

  .secondary {
    background-color: var(--color-secondary-wash);
    color: var(--color-secondary);
  }

  .secondary:hover:not(:disabled) {
    background-color: var(--color-secondary-wash-strong);
  }

  .button:active:not(:disabled) {
    transform: scale(var(--press-scale));
    transition-duration: var(--motion-instant);
  }

  .button:disabled {
    opacity: 0.45;
    cursor: default;
  }
</style>
