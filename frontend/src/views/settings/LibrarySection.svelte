<script lang="ts">
  // The indexed folders and files, what the index holds, and clearing it.
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
  // A folder or listed file awaiting confirmation; `files` counts what leaves the index.
  let removing: { kind: 'folder' | 'file'; path: string; files: number } | null = null
  let removeError = ''

  const fileCount = (n: number): string => `${n.toLocaleString()} ${n === 1 ? 'file' : 'files'}`

  $: indexing = $engine.indexing !== null
  $: hasFolder = $engine.folders.length > 0
  $: hasFiles = $engine.files.length > 0
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

  const askToRemoveFolder = (folder: FolderStat) => (removing = { kind: 'folder', path: folder.path, files: folder.files })
  const askToRemoveFile = (path: string) => (removing = { kind: 'file', path, files: 1 })

  async function remove() {
    if (!removing) return
    const { kind, path } = removing
    removing = null
    removeError = kind === 'folder' ? await engine.removeFolder(path) : await engine.removeFile(path)
  }
</script>

<SettingsSection title="Library">
  <SettingRow label="Indexed folders and files" hint={hasFolder || hasFiles ? 'Search covers everything listed' : 'No folder or file yet'}>
    <div class="add">
      <Button disabled={indexing} on:click={engine.chooseFolder}>Add folder…</Button>
      <Button disabled={indexing} on:click={engine.chooseFiles}>Add files…</Button>
    </div>
  </SettingRow>

  {#if hasFolder}
    <ul class="folders">
      {#each $engine.folders as folder (folder.path)}
        <li class="folder">
          <span class="path truncate" title={folder.path}>{truncateMiddle(folder.path, PATH_MAX)}</span>
          <span class="count">{fileCount(folder.files)}</span>
          <Button disabled={indexing} on:click={() => askToRemoveFolder(folder)}>Remove…</Button>
        </li>
      {/each}
    </ul>
  {/if}

  {#if hasFiles}
    <h3 class="subhead">Individual files</h3>
    <ul class="folders">
      {#each $engine.files as file (file)}
        <li class="folder">
          <span class="path truncate" title={file}>{truncateMiddle(file, PATH_MAX)}</span>
          <Button disabled={indexing} on:click={() => askToRemoveFile(file)}>Remove…</Button>
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
  confirmLabel={removing?.kind === 'file' ? 'Remove file' : 'Remove folder'}
  on:confirm={remove}
  on:cancel={() => (removing = null)}
>
  {#if removing?.kind === 'file'}
    <p>It leaves the index. The file stays on disk, and you can add it again later.</p>
  {:else}
    <p>
      {(removing?.files ?? 0) === 1 ? 'Its 1 file leaves' : `Its ${fileCount(removing?.files ?? 0)} leave`} the index. The files stay on disk,
      and you can add the folder again later.
    </p>
  {/if}
</ConfirmDialog>

<style>
  .add {
    display: flex;
    gap: var(--space-xs);
    flex-shrink: 0;
  }

  .subhead {
    margin: var(--space-xs) 0 var(--space-xxs);
    font-size: var(--text-small-size);
    font-weight: var(--text-body-strong-weight);
    color: var(--color-ink-secondary);
  }

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
