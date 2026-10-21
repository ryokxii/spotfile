<script lang="ts">
  // The indexed folder, what the index holds, and clearing it.
  import { formatBytes, truncateMiddle } from '../../lib/format'
  import { engine } from '../../stores/app'
  import Button from '../../ui/Button.svelte'
  import ConfirmDialog from '../../ui/ConfirmDialog.svelte'
  import SettingRow from './SettingRow.svelte'
  import SettingsSection from './SettingsSection.svelte'

  const PATH_MAX = 56

  let confirming = false
  let clearing = false
  let clearError = ''

  $: indexing = $engine.indexing !== null
  $: hasFolder = $engine.folder !== ''
  $: files = `${$engine.documents.toLocaleString()} ${$engine.documents === 1 ? 'file' : 'files'}`
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
</script>

<SettingsSection title="Library">
  <SettingRow label="Indexed folder" hint={hasFolder ? '' : 'No folder yet'}>
    <svelte:fragment slot="detail">
      {#if hasFolder}
        <span class="path" title={$engine.folder}>{truncateMiddle($engine.folder, PATH_MAX)}</span>
      {/if}
    </svelte:fragment>
    <Button disabled={indexing} on:click={engine.chooseFolder}>{hasFolder ? 'Change…' : 'Choose…'}</Button>
  </SettingRow>

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

  {#if clearError || $engine.indexError}
    <p class="error" role="alert">{clearError || $engine.indexError}</p>
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

<style>
  .path {
    display: block;
    margin-top: var(--space-xs);
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
