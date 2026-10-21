<script lang="ts">
  // Recently opened files: click to open in the viewer, × to forget.
  import { createEventDispatcher } from 'svelte'
  import { filename } from '../../lib/path'
  import { ago } from '../../lib/time'
  import type { RecentFile } from '../../stores/dashboard'
  import Icon from '../../ui/Icon.svelte'

  export let files: RecentFile[]
  export let history: boolean
  export let now = new Date()

  const dispatch = createEventDispatcher<{ open: string; forget: string }>()
</script>

<section class="tile recent-files" aria-labelledby="recent-files-title">
  <h2 id="recent-files-title" class="eyebrow">Recently opened</h2>
  {#if !history}
    <p class="muted">History is off · <a href="#/settings">Turn on in Settings</a></p>
  {:else if files.length === 0}
    <p class="muted">Files you open will appear here</p>
  {:else}
    <ul class="rows">
      {#each files as file (file.path)}
        <li class="row">
          <button class="open" type="button" title={file.path} on:click={() => dispatch('open', file.path)}>
            <span class="name">{filename(file.path)}</span>
            <span class="when">{ago(new Date(file.at), now)}</span>
          </button>
          <button class="forget" type="button" aria-label="Forget {filename(file.path)}" title="Forget" on:click={() => dispatch('forget', file.path)}>
            <Icon name="close" size={14} />
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .rows {
    list-style: none;
    margin-top: var(--space-sm);
  }

  .row {
    display: flex;
    align-items: center;
    border-top: 1px solid var(--color-hairline);
  }

  .row:first-child {
    border-top: none;
  }

  .open {
    flex: 1;
    min-width: 0;
    display: flex;
    justify-content: space-between;
    gap: var(--space-sm);
    padding: var(--space-xs) var(--space-xs);
    border-radius: var(--radius-xs);
    color: var(--color-ink);
    text-align: left;
    transition: background-color var(--motion-quick) var(--ease-standard);
  }

  .open:hover,
  .open:focus-visible {
    background-color: var(--color-surface-raised);
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .when {
    flex-shrink: 0;
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
    font-variant-numeric: tabular-nums;
  }

  .forget {
    display: grid;
    place-items: center;
    width: 24px;
    height: 24px;
    border-radius: var(--radius-xs);
    color: var(--color-ink-faint);
    opacity: 0;
    transition: opacity var(--motion-quick) var(--ease-standard);
  }

  .row:hover .forget,
  .forget:focus-visible {
    opacity: 1;
  }

  a {
    color: var(--color-primary);
  }
</style>
