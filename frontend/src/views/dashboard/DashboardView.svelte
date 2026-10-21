<script lang="ts">
  // The launch page: greeting, search, and the Bento tiles. Searches open in
  // Chat; recent files open in a panel over the page.
  import { onDestroy, onMount } from 'svelte'
  import { push } from 'svelte-spa-router'
  import { fade } from 'svelte/transition'
  import { greeting } from '../../lib/greeting'
  import { statusLine } from '../../lib/indexStatus'
  import { duration } from '../../lib/motion'
  import { dashboard, engine, profile, search } from '../../stores/app'
  import Button from '../../ui/Button.svelte'
  import DashboardSearch from './DashboardSearch.svelte'
  import DiskTile from './DiskTile.svelte'
  import FreeSpaceTile from './FreeSpaceTile.svelte'
  import RecentFilesTile from './RecentFilesTile.svelte'
  import RecentSearchesTile from './RecentSearchesTile.svelte'
  import StaleFilesDialog from './StaleFilesDialog.svelte'
  import ViewerPanel from './ViewerPanel.svelte'

  const CLOCK_MS = 60_000 // relative times and the greeting move on each minute
  const TRASH_FAILED = "Couldn't move to Trash. Nothing was deleted."

  let now = new Date()
  let viewing: string | null = null
  let showAll = false
  let errors: Record<string, string> = {}
  const clock = setInterval(() => (now = new Date()), CLOCK_MS)

  onMount(() => {
    void dashboard.start()
  })

  onDestroy(() => {
    dashboard.stop()
    clearInterval(clock)
  })

  $: data = $dashboard.data
  $: status = statusLine($engine)
  $: indexing = $engine.indexing !== null
  $: hasLibrary = $engine.folders.length > 0 || $engine.files.length > 0 || $engine.documents > 0
  $: canSearch = $engine.ready && $engine.documents > 0
  $: placeholder = canSearch ? 'Search your files…' : hasLibrary ? 'Indexing — search opens when it finishes…' : 'Add a folder or files to start searching…'

  function runSearch(query: string) {
    void search.submit(query)
    void push('/chat')
  }

  function open(path: string) {
    viewing = path
    dashboard.opened(path)
  }

  async function removeFromIndex(path: string) {
    errors = { ...errors, [path]: '' }
    errors = { ...errors, [path]: await dashboard.removeFromIndex(path) }
  }

  async function moveToTrash(path: string) {
    errors = { ...errors, [path]: '' }
    const error = await dashboard.moveToTrash(path)
    if (error) console.error('dashboard: move to Trash failed', error)
    errors = { ...errors, [path]: error ? TRASH_FAILED : '' }
  }
</script>

<div class="dashboard-view" in:fade={{ duration: duration('normal') }}>
  <div class="content">
    <header class="top">
      <h1 class="display">{greeting(now.getHours(), $profile.name)}</h1>
      <p class="status truncate" title={status.text} class:active={status.tone === 'active'} class:error={status.tone === 'error'}>{status.text}</p>
    </header>

    <DashboardSearch disabled={!canSearch} {placeholder} on:search={(e) => runSearch(e.detail)} />

    {#if !hasLibrary}
      <section class="tile empty">
        <p>Add a folder or individual files to index, and your library's figures appear here.</p>
        <div class="actions">
          <Button variant="primary" on:click={engine.chooseFolder}>Choose a folder</Button>
          <Button on:click={engine.chooseFiles}>Add files</Button>
        </div>
      </section>
    {:else if !data && $dashboard.phase === 'error'}
      <section class="tile empty">
        <p class="error" role="alert">Couldn't load your library figures — {$dashboard.error}</p>
        <Button on:click={dashboard.reload}>Retry</Button>
      </section>
    {:else if data}
      <div class="grid">
        <DiskTile {data} {now} />
        <RecentFilesTile files={data.recentFiles} history={data.history} {now} on:open={(e) => open(e.detail)} on:forget={(e) => dashboard.forget(e.detail)} />
        <RecentSearchesTile searches={data.recentSearches} history={data.history} on:search={(e) => runSearch(e.detail)} />
        <FreeSpaceTile
          stale={data.stale}
          staleBytes={data.staleBytes}
          staleDays={data.staleDays}
          busy={indexing}
          {errors}
          {now}
          on:remove={(e) => removeFromIndex(e.detail)}
          on:trash={(e) => moveToTrash(e.detail)}
          on:showAll={() => (showAll = true)}
        />
      </div>
    {/if}
  </div>

  <StaleFilesDialog
    open={showAll}
    files={data?.stale ?? []}
    busy={indexing}
    {errors}
    {now}
    on:close={() => (showAll = false)}
    on:remove={(e) => removeFromIndex(e.detail)}
    on:trash={(e) => moveToTrash(e.detail)}
  />
  <ViewerPanel path={viewing} on:close={() => (viewing = null)} />
</div>

<style>
  .dashboard-view {
    position: relative;
    height: 100%;
    overflow: hidden;
  }

  .content {
    height: 100%;
    max-width: var(--content-max);
    margin: 0 auto;
    padding: var(--space-xxl) var(--space-xl);
    display: flex;
    flex-direction: column;
    gap: var(--space-lg);
    overflow-y: auto;
  }

  .top {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    gap: var(--space-md);
    flex-wrap: wrap;
  }

  .display {
    font-family: var(--font-display);
    font-size: var(--text-display-size);
    line-height: var(--text-display-line);
    font-weight: var(--text-display-weight);
    letter-spacing: var(--text-display-tracking);
  }

  .status {
    max-width: 100%;
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
  }

  .status.active {
    color: var(--color-primary);
  }

  .status.error,
  .error {
    color: var(--color-urgent);
  }

  .grid {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) minmax(0, 1fr);
    grid-template-areas:
      'disk files'
      'disk searches'
      'free free';
    gap: var(--space-md);
  }

  .grid > :global(.disk) {
    grid-area: disk;
  }

  .grid > :global(.recent-files) {
    grid-area: files;
  }

  .grid > :global(.recent-searches) {
    grid-area: searches;
  }

  .grid > :global(.free-space) {
    grid-area: free;
  }

  /* About 900px of content beside the sidebar. */
  @media (max-width: 1160px) {
    .grid {
      grid-template-columns: minmax(0, 1fr);
      grid-template-areas: 'disk' 'files' 'searches' 'free';
    }
  }

  .dashboard-view :global(.tile) {
    padding: var(--space-md) var(--space-lg);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-hairline);
    background-color: var(--color-surface);
    box-shadow: var(--shadow-raised);
  }

  .dashboard-view :global(.tile .eyebrow) {
    font-size: var(--text-micro-size);
    font-weight: var(--text-micro-weight);
    letter-spacing: var(--text-micro-tracking);
    text-transform: uppercase;
    color: var(--color-ink-faint);
  }

  .dashboard-view :global(.tile .muted) {
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
  }

  .actions {
    display: flex;
    gap: var(--space-xs);
  }

  .empty {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-md);
  }

  @media (prefers-reduced-motion: no-preference) {
    .grid > :global(.tile) {
      animation: rise var(--motion-slow) var(--ease-enter) both;
    }

    .grid > :global(.tile:nth-child(2)) {
      animation-delay: var(--motion-stagger);
    }

    .grid > :global(.tile:nth-child(3)) {
      animation-delay: calc(2 * var(--motion-stagger));
    }

    .grid > :global(.tile:nth-child(4)) {
      animation-delay: calc(3 * var(--motion-stagger));
    }
  }

  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(var(--motion-rise));
    }
  }
</style>
