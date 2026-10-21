<script lang="ts">
  // The indexed folders, what the index holds, and clearing it.
  import { formatBytes, truncateMiddle } from '../../lib/format'
  import { filename } from '../../lib/path'
  import type { FolderStat } from '../../stores/engine'
  import { engine } from '../../stores/app'
  import Button from '../../ui/Button.svelte'
  import ConfirmDialog from '../../ui/ConfirmDialog.svelte'
  import SettingRow from './SettingRow.svelte'
  import SettingsSection from './SettingsSection.svelte'

  const PATH_MAX = 56

  let confirming = false
  let clearing = false
  let clearError = ''
  let removing: FolderStat | null = null // folder awaiting confirmation
  let removeError = ''

  const fileCount = (n: number): string => `${n.toLocaleString()} ${n === 1 ? 'file' : 'files'}`

  $: indexing = $engine.indexing !== null
  $: hasFolder = $engine.folders.length > 0
  $: files = fileCount($engine.documents)
  $: figures = [
    { label: $engine.documents === 1 ? 'file' : 'files', value: $engine.documents.toLocaleString() },
    { label: $engine.chunks === 1 ? 'chunk' : 'chunks', value: $engine.chunks.toLocaleString() },
    { label: 'on disk', value: formatBytes($engine.indexBytes) },
  ]

  async function clear() {
    confirming = false
    clearing = true
    clearError = await engine.clearIndex()
    clearing = false
  }

  async function remove() {
    if (!removing) return
    const dir = removing.path
    removing = null
    removeError = await engine.removeFolder(dir)
  }
</script>

<SettingsSection title="Library">
  <SettingRow label="Indexed folders" hint={hasFolder ? 'Search covers every folder listed' : 'No folder yet'}>
    <Button disabled={indexing} on:click={engine.chooseFolder}>Add folder…</Button>
  </SettingRow>

  {#if hasFolder}
    <ul class="folders">
      {#each $engine.folders as folder (folder.path)}
        <li class="folder">
          <span class="path truncate" title={folder.path}>{truncateMiddle(folder.path, PATH_MAX)}</span>
          <span class="count">{fileCount(folder.files)}</span>
          <Button disabled={indexing} on:click={() => (removing = folder)}>Remove…</Button>
        </li>
      {/each}
    </ul>
  {/if}

  <dl class="figures">
    {#each figures as figure}
      <div class="figure">
        <dt>{figure.label}</dt>
        <dd>{figure.value}</dd>
      </div>
    {/each}
  </dl>

  <SettingRow
    label="Clear index"
    hint={indexing ? 'Available when indexing finishes' : 'Search is empty until you index a folder again'}
  >
    <Button variant="destructive" disabled={indexing || clearing} on:click={() => (confirming = true)}>Clear index…</Button>
  </SettingRow>

  {#if removeError || clearError || $engine.indexError}
    <p class="error" role="alert">{removeError || clearError || $engine.indexError}</p>
  {/if}
</SettingsSection>

<ConfirmDialog
  open={confirming}
  title="Clear the search index?"
  confirmLabel="Clear index"
  on:confirm={clear}
  on:cancel={() => (confirming = false)}
>
  <p>This removes {files} from search. Your files and models stay on disk. To search again, index a folder.</p>
</ConfirmDialog>

<ConfirmDialog
  open={removing !== null}
  title={`Remove ${removing ? filename(removing.path) : ''} from search?`}
  confirmLabel="Remove folder"
  on:confirm={remove}
  on:cancel={() => (removing = null)}
>
  <p>
    {(removing?.files ?? 0) === 1 ? 'Its 1 file leaves' : `Its ${fileCount(removing?.files ?? 0)} leave`} the index. The files stay on disk,
    and you can add the folder again later.
  </p>
</ConfirmDialog>

<style>
  .folders {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
    padding-bottom: var(--space-sm);
  }

  .folder {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .count {
    flex-shrink: 0;
    font-size: var(--text-small-size);
    color: var(--color-ink-faint);
    font-variant-numeric: tabular-nums;
  }

  .path {
    flex: 1;
    min-width: 0;
    padding: var(--space-xs) var(--space-sm);
    border-radius: var(--radius-xs);
    border: 1px solid var(--color-hairline);
    background-color: var(--color-surface-sunken);
    font-size: var(--text-small-size);
    color: var(--color-ink-secondary);
    white-space: nowrap;
    overflow: hidden;
  }

  .figures {
    display: flex;
    gap: var(--space-xl);
    padding: var(--space-sm) 0 var(--space-md);
  }

  .figure {
    display: flex;
    flex-direction: column-reverse; /* value above its label, label first for screen readers */
  }

  dd {
    font-size: var(--text-title-size);
    line-height: var(--text-title-line);
    font-weight: var(--text-numeric-weight);
    font-variant-numeric: tabular-nums;
  }

  dt {
    font-size: var(--text-small-size);
    color: var(--color-ink-faint);
  }

  .error {
    font-size: var(--text-small-size);
    color: var(--color-urgent);
  }
</style>
