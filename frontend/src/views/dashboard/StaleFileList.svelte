<script lang="ts">
  // Stale files with their actions. Remove from index acts at once; Trash asks
  // first. Each row shows its own error.
  import { createEventDispatcher } from 'svelte'
  import { formatBytes } from '../../lib/format'
  import { filename } from '../../lib/path'
  import { age } from '../../lib/time'
  import type { StaleFile } from '../../stores/dashboard'
  import ConfirmDialog from '../../ui/ConfirmDialog.svelte'

  export let files: StaleFile[]
  export let busy = false // indexing: actions wait until it finishes
  export let errors: Record<string, string> = {}
  export let now = new Date()

  const dispatch = createEventDispatcher<{ remove: string; trash: string }>()
  let confirming: StaleFile | null = null

  function confirmTrash() {
    if (!confirming) return
    const path = confirming.path
    confirming = null
    dispatch('trash', path)
  }
</script>

<ul class="stale-list">
  {#each files as file (file.path)}
    <li class="stale-row">
      <div class="line">
        <span class="name" title={file.path}>{filename(file.path)}</span>
        <span class="meta">~{formatBytes(file.bytes)} · {age(new Date(file.modified), now)}</span>
        <span class="actions">
          <button type="button" class="remove" disabled={busy} on:click={() => dispatch('remove', file.path)}>Remove from index</button>
          <button type="button" class="trash" disabled={busy} on:click={() => (confirming = file)}>Delete</button>
        </span>
      </div>
      {#if errors[file.path]}
        <p class="error" role="alert">{errors[file.path]}</p>
      {/if}
    </li>
  {/each}
</ul>

<ConfirmDialog
  open={confirming !== null}
  title={`Move ${confirming ? filename(confirming.path) : ''} to the Trash?`}
  confirmLabel="Move to Trash"
  on:confirm={confirmTrash}
  on:cancel={() => (confirming = null)}
>
  <p>You can restore it from the Trash.</p>
</ConfirmDialog>

<style>
  .stale-list {
    list-style: none;
  }

  .stale-row {
    padding: var(--space-xs) 0;
    border-top: 1px solid var(--color-hairline);
  }

  .stale-row:first-child {
    border-top: none;
  }

  .line {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .meta {
    flex-shrink: 0;
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
    font-variant-numeric: tabular-nums;
  }

  .actions {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    flex-shrink: 0;
  }

  .remove,
  .trash {
    padding: var(--space-xxs) var(--space-sm);
    border-radius: var(--radius-xs);
    font-size: var(--text-small-size);
    font-weight: var(--text-body-strong-weight);
    transition: background-color var(--motion-quick) var(--ease-standard);
  }

  .remove {
    background-color: var(--color-secondary-wash);
    color: var(--color-secondary);
  }

  .remove:hover:not(:disabled),
  .remove:focus-visible {
    background-color: var(--color-secondary-wash-strong);
  }

  .trash {
    color: var(--color-urgent);
  }

  .trash:hover:not(:disabled),
  .trash:focus-visible {
    background-color: var(--color-urgent-wash);
  }

  .remove:disabled,
  .trash:disabled {
    opacity: 0.5;
  }

  .error {
    margin-top: var(--space-xxs);
    font-size: var(--text-small-size);
    color: var(--color-urgent);
  }
</style>
