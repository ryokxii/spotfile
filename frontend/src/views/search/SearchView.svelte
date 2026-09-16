<script lang="ts">
  import { fade } from 'svelte/transition'
  import { duration } from '../../lib/motion'
  import { engine, search } from '../../stores/app'
  import PdfViewer from '../../viewers/PdfViewer.svelte'
  import TextViewer from '../../viewers/TextViewer.svelte'
  import EmptyState from './EmptyState.svelte'
  import IndexStatus from './IndexStatus.svelte'
  import ResultList from './ResultList.svelte'
  import SearchBar from './SearchBar.svelte'

  let query = $search.query

  $: hasIndex = $engine.documents > 0 || $engine.indexing !== null
  $: canSearch = $engine.ready && $engine.documents > 0
  $: searching = $search.phase !== 'idle'
  $: placeholder = canSearch ? 'Search your files…' : hasIndex ? 'Indexing — search opens when it finishes…' : 'Index a folder to start searching…'
  $: folderTitle = $engine.folder ? 'Index another folder' : 'Choose a folder to index'

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && $search.preview) search.close()
  }
</script>

<svelte:window on:keydown={onKeydown} />

<div class="search-view" class:split={$search.preview !== null} in:fade={{ duration: duration('normal') }}>
  <div class="column" class:searching>
    {#if !searching}
      <h1 class="display">Find anything <em>in your files</em></h1>
    {/if}

    <div class="controls">
      <SearchBar
        bind:value={query}
        disabled={!canSearch}
        loading={$search.phase === 'loading'}
        {placeholder}
        {folderTitle}
        on:submit={(e) => search.submit(e.detail)}
        on:chooseFolder={engine.chooseFolder}
      />
      <IndexStatus state={$engine} />
    </div>

    {#if !searching && $engine.ready && !hasIndex}
      <EmptyState eyebrow="Get started" line="Point Spotfile at a folder." action="Choose folder" variant="primary" on:click={engine.chooseFolder} />
    {/if}

    {#if $search.phase === 'error'}
      <p class="error" role="alert">{$search.error}</p>
    {:else if $search.phase === 'results'}
      <div class="results">
        <p class="count">{$search.results.length} result{$search.results.length === 1 ? '' : 's'} for “{$search.query}”</p>
        {#if $search.results.length > 0}
          <ResultList results={$search.results} selectedKey={$search.preview?.key ?? null} on:open={(e) => search.open(e.detail)} />
        {:else}
          <EmptyState eyebrow="No matches" line="Nothing close to that yet." action="Index another folder" on:click={engine.chooseFolder} />
        {/if}
      </div>
    {/if}
  </div>

  {#if $search.preview}
    {@const preview = $search.preview}
    <aside class="preview" aria-label="Preview" transition:fade={{ duration: duration('quick') }}>
      {#if preview.kind === 'pdf'}
        <PdfViewer docPath={preview.path} initialPage={preview.page} highlightText={preview.highlight} onClose={search.close} />
      {:else}
        <TextViewer docPath={preview.path} mode={preview.kind} highlightText={preview.highlight} onClose={search.close} />
      {/if}
    </aside>
  {/if}
</div>

<style>
  .search-view {
    height: 100%;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
  }

  .search-view.split {
    grid-template-columns: minmax(var(--results-min), 5fr) minmax(0, 7fr);
  }

  .column {
    width: 100%;
    max-width: var(--search-column-max);
    margin: 0 auto;
    padding: var(--space-huge) var(--space-xl) var(--space-xxl);
    display: flex;
    flex-direction: column;
    gap: var(--space-xxl);
    overflow-y: auto;
  }

  .column.searching {
    padding-top: var(--space-xl);
    gap: var(--space-lg);
  }

  .split .column {
    max-width: none;
    margin: 0;
    padding-left: var(--space-lg);
    padding-right: var(--space-lg);
  }

  .display {
    font-family: var(--font-display);
    font-size: var(--text-display-size);
    line-height: var(--text-display-line);
    font-weight: var(--text-display-weight);
    letter-spacing: var(--text-display-tracking);
    color: var(--color-ink);
  }

  .display em {
    font-style: italic;
    color: var(--color-primary);
  }

  .controls {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .results {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
  }

  .count {
    font-size: var(--text-small-size);
    color: var(--color-ink-faint);
    font-variant-numeric: tabular-nums;
  }

  .error {
    color: var(--color-urgent);
  }

  .preview {
    min-width: 0;
    height: 100%;
    border-left: 1px solid var(--color-hairline);
    background-color: var(--color-surface);
  }
</style>
