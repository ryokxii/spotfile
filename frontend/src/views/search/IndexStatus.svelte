<script lang="ts">
  import { statusLine } from '../../lib/indexStatus'
  import type { EngineState } from '../../stores/engine'

  export let state: EngineState

  $: line = statusLine(state)
</script>

<div class="status {line.tone}" role="status" aria-live="polite">
  <p class="text" title={line.text}>{line.text}</p>
  {#if line.progress !== null}
    <div class="track" aria-hidden="true">
      <div class="fill" style="transform: scaleX({line.progress})" />
    </div>
  {/if}
</div>

<style>
  .status {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
    min-height: calc(var(--text-small-size) * var(--text-small-line));
  }

  .text {
    font-size: var(--text-small-size);
    line-height: var(--text-small-line);
    color: var(--color-ink-faint);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .active .text {
    color: var(--color-ink-secondary);
  }

  .error .text {
    color: var(--color-urgent);
  }

  .track {
    height: 2px;
    border-radius: 2px;
    background-color: var(--color-hairline);
    overflow: hidden;
  }

  .fill {
    height: 100%;
    background-color: var(--color-primary);
    transform-origin: left center;
    transition: transform var(--motion-normal) var(--ease-standard);
  }
</style>
