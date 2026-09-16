<script lang="ts">
  import Icon from './Icon.svelte'
  import type { IconName } from './icons'
  import PressableSurface from './PressableSurface.svelte'

  export let href: string
  export let label: string
  export let icon: IconName
  export let selected = false
  export let compact = false
</script>

<PressableSurface href="#{href}" {selected} label={compact ? label : undefined} title={compact ? label : undefined} class="nav-item {compact ? 'compact' : ''}">
  <span class="marker" class:on={selected} aria-hidden="true" />
  <Icon name={icon} />
  {#if !compact}<span class="label">{label}</span>{/if}
</PressableSurface>

<style>
  :global(.nav-item) {
    position: relative;
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-xs) var(--space-sm);
    color: var(--color-ink-secondary);
  }

  :global(.nav-item.compact) {
    justify-content: center;
    padding: var(--space-sm) 0;
  }

  :global(.nav-item.selected) {
    color: var(--color-primary);
  }

  .marker {
    position: absolute;
    left: calc(-1 * var(--space-sm));
    top: var(--space-xs);
    bottom: var(--space-xs);
    width: 2px;
    border-radius: 2px;
    background-color: var(--color-primary);
    transform: scaleY(0);
    transition: transform var(--motion-quick) var(--ease-standard);
  }

  .marker.on {
    transform: scaleY(1);
  }

  .label {
    font-size: var(--text-body-size);
    font-weight: var(--text-body-strong-weight);
  }
</style>
