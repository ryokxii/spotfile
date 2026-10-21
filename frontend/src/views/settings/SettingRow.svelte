<script lang="ts">
  // One setting: label and hint on the left, its control on the right.
  export let label: string
  export let hint = ''
</script>

<div class="row">
  <div class="text">
    <span class="label">{label}</span>
    {#if hint}<span class="hint">{hint}</span>{/if}
    <slot name="detail" />
  </div>
  <div class="control"><slot /></div>
</div>

<style>
  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-lg);
    padding: var(--space-md) 0;
    border-top: 1px solid var(--color-hairline);
  }

  .row:first-child {
    border-top: none;
  }

  .text {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: var(--space-xxs);
    min-width: 0;
  }

  .label {
    font-weight: var(--text-body-strong-weight);
  }

  .hint {
    font-size: var(--text-small-size);
    line-height: var(--text-small-line);
    color: var(--color-ink-faint);
  }

  .control {
    flex-shrink: 0;
  }
</style>
