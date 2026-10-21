<script lang="ts">
  import { fade } from 'svelte/transition'
  import { duration } from '../../lib/motion'
  import AppearanceSection from './AppearanceSection.svelte'
</script>

<div class="settings-view" in:fade={{ duration: duration('normal') }}>
  <div class="column">
    <h1 class="display">Settings</h1>
    <AppearanceSection />
  </div>
</div>

<style>
  .settings-view {
    height: 100%;
    overflow-y: auto;
  }

  .column {
    max-width: var(--settings-column-max);
    margin: 0 auto;
    padding: var(--space-huge) var(--space-xl) var(--space-xxl);
    display: flex;
    flex-direction: column;
    gap: var(--space-xxl);
  }

  .display {
    font-family: var(--font-display);
    font-size: var(--text-display-size);
    line-height: var(--text-display-line);
    font-weight: var(--text-display-weight);
    letter-spacing: var(--text-display-tracking);
  }
</style>
