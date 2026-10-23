<script lang="ts">
  // Profile: the display name, kept on this computer.
  import { fade } from 'svelte/transition'
  import { duration } from '../../lib/motion'
  import { profile } from '../../stores/app'
  import NameField from './NameField.svelte'
</script>

<div class="profile-view" in:fade={{ duration: duration('normal') }}>
  <div class="column">
    <header>
      <h1 class="display">Profile</h1>
      <p class="lede">Kept only on this computer.</p>
    </header>
    <div class="field">
      <label class="eyebrow" for="display-name">Display name</label>
      <NameField value={$profile.name} on:save={(e) => profile.setName(e.detail)} />
      <p class="hint">Used for your initials in the sidebar and the Dashboard greeting.</p>
      {#if $profile.nameError}
        <p class="error" role="alert">{$profile.nameError}</p>
      {/if}
    </div>
  </div>
</div>

<style>
  .profile-view {
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

  .lede,
  .hint {
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
  }

  .lede {
    margin-top: var(--space-xs);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .eyebrow {
    font-size: var(--text-micro-size);
    font-weight: var(--text-micro-weight);
    letter-spacing: var(--text-micro-tracking);
    text-transform: uppercase;
    color: var(--color-ink-faint);
  }

  .error {
    font-size: var(--text-small-size);
    color: var(--color-urgent);
  }
</style>
