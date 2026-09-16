<script lang="ts">
  // DESIGN.md §8: eyebrow, one line of Fraunces, one action.
  import Button from '../../ui/Button.svelte'

  export let eyebrow: string
  export let line: string
  export let action: string
  export let variant: 'primary' | 'secondary' = 'secondary'
</script>

<section class="empty">
  <p class="eyebrow">{eyebrow}</p>
  <p class="line">{line}</p>
  <Button {variant} on:click>{action}</Button>
</section>

<style>
  .empty {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-sm);
  }

  .eyebrow {
    font-size: var(--text-micro-size);
    font-weight: var(--text-micro-weight);
    letter-spacing: var(--text-micro-tracking);
    text-transform: uppercase;
    color: var(--color-ink-faint);
  }

  .line {
    font-family: var(--font-display);
    font-weight: var(--text-display-weight);
    font-size: var(--text-title-size);
    line-height: var(--text-title-line);
    color: var(--color-ink);
    margin-bottom: var(--space-xs);
  }
</style>
