<script lang="ts">
  import { onMount } from 'svelte'
  import { fade, fly } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { Search, SelectFolder, IndexFolder } from '../wailsjs/go/main/App.js'
  import { EventsOn, WindowIsFullscreen, WindowFullscreen, WindowUnfullscreen } from '../wailsjs/runtime/runtime.js'
  import StatusBar from './StatusBar.svelte'
  import PdfViewer from './viewers/PdfViewer.svelte'
  import TextViewer from './viewers/TextViewer.svelte'
  import { previewTargetFor, type PreviewTarget } from './lib/fileTypes'
  import { filename, dirpath } from './lib/path'

  interface SearchResult {
    docPath: string
    chunkIdx: number
    pageNum: number
    text: string
    score: number
  }

  type Phase = 'idle' | 'loading' | 'results' | 'error'
  let phase: Phase = 'idle'

  // Engine state — driven by events from startup auto-init
  let engineReady = false
  let engineError = ''
  let hasFiles = false
  let selectedFolder = ''

  // Search
  let query = ''
  let results: SearchResult[] = []
  let searchError = ''

  // Indexing (forwarded to StatusBar)
  let indexing = false
  let indexedCount = 0
  let indexedDocuments = 0
  let totalChunks = 0
  let totalFiles = 0
  let currentFile = ''
  let indexedPaths = new Set<string>()
  let watcherActive = false
  let reindexing = false

  // Docked preview pane — null when closed. `activeKey` marks the open result.
  let preview: (PreviewTarget & { activeKey: string }) | null = null
  $: showPreview = preview !== null

  const topK = 5

  $: hint =
    engineError ? engineError.split('\n')[0]
    : phase === 'idle' && !hasFiles ? 'Choose a folder to get started'
    : phase === 'idle' ? 'Ready — type your question and press Enter'
    : phase === 'loading' ? 'Searching…'
    : phase === 'error' ? searchError
    : ''

  onMount(() => {
    EventsOn('engine:ready', () => { engineReady = true })
    EventsOn('engine:error', (msg: string) => { engineError = msg })
    EventsOn('index:start', (data: any) => {
      indexing = true
      indexedCount = 0
      indexedDocuments = 0
      totalChunks = 0
      totalFiles = data.total
      indexedPaths = new Set<string>()
      selectedFolder = data.dir
    })
    EventsOn('index:prepared', (data: any) => {
      totalChunks = data.totalChunks
    })
    EventsOn('index:chunk', (data: any) => {
      indexedCount = data.total
      currentFile = data.path
      if (!indexedPaths.has(data.path)) {
        indexedPaths.add(data.path)
        indexedDocuments = indexedPaths.size
      }
    })
    EventsOn('index:done', () => {
      indexing = false
      currentFile = ''
      hasFiles = true
      engineReady = true // reaching index:done proves the engine is ready, even if engine:ready was missed
    })
    EventsOn('watcher:started', () => { watcherActive = true; engineReady = true })
    EventsOn('watcher:reindexing', (data: any) => { reindexing = true; currentFile = data.path })
    EventsOn('watcher:done', () => { reindexing = false; currentFile = '' })
  })

  async function pickFolder() {
    try {
      const dir = await SelectFolder()
      if (!dir) return
      await IndexFolder(dir)
    } catch (e: any) {
      engineError = e?.message || String(e)
    }
  }

  async function search() {
    const q = query.trim()
    if (!q || !engineReady || !hasFiles || phase === 'loading') return

    phase = 'loading'
    results = []
    searchError = ''
    preview = null

    try {
      results = (await Search(q, topK)) ?? []
      phase = 'results'
    } catch (e: any) {
      searchError = e?.message || String(e)
      phase = 'error'
    }
  }

  // Unique per result — chunkIdx alone collides across pages (it's the chunk
  // index within a page), so include docPath and pageNum.
  function keyOf(r: SearchResult) {
    return `${r.docPath}|${r.pageNum}|${r.chunkIdx}`
  }

  function openResult(result: SearchResult) {
    preview = { ...previewTargetFor(result), activeKey: keyOf(result) }
  }

  // F11 toggles full screen on every platform (macOS also has ⌃⌘F via the Window menu).
  async function toggleFullscreen() {
    if (await WindowIsFullscreen()) WindowUnfullscreen()
    else WindowFullscreen()
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && showPreview) closePreview()
    else if (e.key === 'F11') {
      e.preventDefault()
      toggleFullscreen()
    }
  }

  function closePreview() {
    preview = null
  }

  function excerpt(text: string) {
    return text.replace(/\s+/g, ' ').trim()
  }
</script>

<!-- ── Root ───────────────────────────────────────────────────── -->
<svelte:window on:keydown={onKeydown} />
<div class="root" class:split={showPreview}>
  <div class="workspace">
    <div class="column" class:in-results={phase === 'results'}>

    {#if phase === 'idle'}
      <div class="logo-area" transition:fade={{ duration: 180 }}>
        <svg class="logo-icon" viewBox="0 0 80 80" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
          <rect x="5"  y="5"  width="70" height="70" rx="18" stroke="white" stroke-width="5.5"/>
          <rect x="17" y="17" width="46" height="46" rx="12" stroke="white" stroke-width="4.5"/>
          <rect x="28" y="28" width="24" height="24" rx="7"  stroke="white" stroke-width="3.5"/>
        </svg>
        <h1 class="logo-title">Spotfile</h1>
        <p class="logo-sub">your personal AI file searcher</p>
      </div>
    {/if}

    <!-- Search bar -->
    <div class="search-wrap">
      <div class="search-bar" class:loading={phase === 'loading'}>
        <svg class="icon-search" viewBox="0 0 20 20" fill="none" aria-hidden="true">
          <circle cx="8.5" cy="8.5" r="5.5" stroke="currentColor" stroke-width="1.8"/>
          <path d="M13 13l3.5 3.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
        </svg>

        <input
          type="text"
          class="search-input"
          bind:value={query}
          placeholder={hasFiles ? 'Search your files…' : 'Index a folder to start searching…'}
          disabled={!engineReady || !hasFiles}
          on:keydown={(e) => e.key === 'Enter' && search()}
          autocomplete="off"
          spellcheck="false"
        />

        {#if phase === 'loading'}
          <span class="spinner" aria-label="Searching" role="status" />
        {:else if engineError}
          <span class="engine-error-dot" title={engineError} aria-label="Engine error" />
        {:else}
          <!-- Folder picker button — always visible -->
          <button
            class="icon-btn"
            title={hasFiles ? `Indexed: ${filename(selectedFolder) || 'folder'} — click to re-index` : 'Choose a folder to index'}
            on:click={pickFolder}
            aria-label="Choose folder to index"
          >
            {#if hasFiles}
              <svg viewBox="0 0 20 20" fill="none">
                <path d="M2 6a2 2 0 012-2h4l2 2h6a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
                <path d="M7 11l2 2 4-4" stroke="rgba(202,138,4,0.9)" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            {:else}
              <svg viewBox="0 0 20 20" fill="none">
                <path d="M2 6a2 2 0 012-2h4l2 2h6a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
                <path d="M2 9h16" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" opacity="0.5"/>
              </svg>
            {/if}
          </button>
        {/if}
      </div>

      <p class="hint" class:error={phase === 'error' || !!engineError}>{hint}</p>
    </div>


    <!-- Results area -->
    {#if phase === 'results'}
      <div class="results-area">
        {#if results.length > 0}
          <div class="result-list">
            {#each results as r, i (keyOf(r))}
              {@const isPDF = r.docPath.toLowerCase().endsWith('.pdf')}
              <article
                class="result"
                class:active={keyOf(r) === preview?.activeKey}
                in:fly={{ y: 12, duration: 300, delay: i * 70, easing: cubicOut }}
              >
                <div class="result-head">
                  <svg class="file-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
                    <path d="M14 2v6h6" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                  <span class="file-name" title={r.docPath}>{filename(r.docPath)}</span>
                  {#if isPDF}
                    <span class="page-badge">p.{r.pageNum || 1}</span>
                  {/if}
                  <button class="open-btn" on:click={() => openResult(r)} title="Open preview">
                    Open ▸
                  </button>
                </div>
                <p class="result-path">{dirpath(r.docPath)}</p>
                <p class="result-text">{excerpt(r.text)}</p>
              </article>
            {/each}
          </div>
        {:else}
          <p class="empty-results">No matching passages found.</p>
        {/if}
      </div>
    {/if}

    </div>

    {#if preview}
      <div class="pdf-pane">
        {#if preview.kind === 'pdf'}
          <PdfViewer
            docPath={preview.path}
            initialPage={preview.page}
            highlightText={preview.highlight}
            onClose={closePreview}
          />
        {:else}
          <TextViewer
            docPath={preview.path}
            mode={preview.kind}
            highlightText={preview.highlight}
            onClose={closePreview}
          />
        {/if}
      </div>
    {/if}
  </div>

  <StatusBar
    {indexing}
    {reindexing}
    {watcherActive}
    {indexedCount}
    {indexedDocuments}
    {totalChunks}
    {totalFiles}
    {currentFile}
  />
</div>

<style>
  :global(*, *::before, *::after) {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  :global(body) {
    background: #0c0c0f;
  }

  :global(:focus-visible) {
    outline: 2px solid rgba(202, 138, 4, 0.7);
    outline-offset: 2px;
    border-radius: 4px;
  }

  @media (prefers-reduced-motion: reduce) {
    :global(*) {
      animation-duration: 0.01ms !important;
      transition-duration: 0.01ms !important;
    }
  }

  .root {
    width: 100%;
    min-height: 100vh;
    background: #0c0c0f;
    color: #f5f5f7;
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Segoe UI', Roboto, sans-serif;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 0 1rem 4rem;
  }

  .workspace {
    width: 100%;
    max-width: 660px;
    display: flex;
    flex-direction: row;
    min-width: 0;
  }

  .column {
    flex: 1 1 auto;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    padding-top: 18vh;
    transition: padding-top 0.3s ease;
  }

  .column.in-results {
    padding-top: 6vh;
  }

  /* ── Split view: results on the left, PDF preview on the right ── */
  .root.split {
    align-items: stretch;
    height: 100vh;
    overflow: hidden;
    padding: 0;
  }

  .root.split .workspace {
    max-width: none;
    height: 100vh;
  }

  .root.split .column {
    flex: 0 0 42%;
    max-width: 620px;
    height: 100vh;
    overflow-y: auto;
    padding: 0 1.25rem 3rem;
  }

  .pdf-pane {
    flex: 1 1 0;
    min-width: 0;
    height: 100vh;
    border-left: 1px solid rgba(255, 255, 255, 0.08);
    background: #141414;
  }

  /* ── Logo ────────────────────────────────────────────────── */
  .logo-area {
    display: flex;
    flex-direction: column;
    align-items: center;
    margin-bottom: 2.5rem;
  }

  .logo-icon {
    width: 72px;
    height: 72px;
    margin-bottom: 1rem;
  }

  .logo-title {
    font-size: 2rem;
    font-weight: 700;
    letter-spacing: -0.02em;
    color: #f5f5f7;
    margin-bottom: 0.35rem;
  }

  .logo-sub {
    font-size: 0.9rem;
    color: #636366;
  }

  /* ── Search bar ─────────────────────────────────────────── */
  .search-wrap {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    position: sticky;      /* stay in view while results scroll below */
    top: 0;
    z-index: 5;
    background: #0c0c0f;
    padding: 0.75rem 0;
  }

  .search-bar {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    background: rgba(255, 255, 255, 0.07);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 14px;
    padding: 0 1rem;
    height: 54px;
    transition: border-color 0.2s, background 0.2s;
  }

  .search-bar:focus-within {
    border-color: rgba(202, 138, 4, 0.55);
    background: rgba(255, 255, 255, 0.09);
  }

  .icon-search {
    width: 18px;
    height: 18px;
    color: #636366;
    flex-shrink: 0;
  }

  .search-input {
    flex: 1;
    background: none;
    border: none;
    outline: none;
    color: #f5f5f7;
    font-size: 1rem;
    caret-color: rgba(202, 138, 4, 0.9);
  }

  .search-input::placeholder { color: #48484a; }
  .search-input:disabled { opacity: 0.45; }

  .icon-btn {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    border: none;
    background: transparent;
    color: #636366;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    flex-shrink: 0;
    transition: color 0.15s, background 0.15s;
  }

  .icon-btn svg { width: 18px; height: 18px; }
  .icon-btn:hover { color: #aeaeb2; background: rgba(255,255,255,0.08); }

  .spinner {
    width: 18px;
    height: 18px;
    border: 2px solid rgba(255,255,255,0.12);
    border-top-color: rgba(202, 138, 4, 0.85);
    border-radius: 50%;
    flex-shrink: 0;
    animation: spin 0.8s linear infinite;
  }

  .engine-error-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #ff453a;
    flex-shrink: 0;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  .hint {
    font-size: 0.82rem;
    color: #48484a;
    padding-left: 0.25rem;
    min-height: 1.2em;
  }

  .hint.error { color: #ff453a; }


  /* ── Results ─────────────────────────────────────────────── */
  .results-area {
    margin-top: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .empty-results {
    color: #636366;
    font-size: 0.9rem;
    padding: 0.5rem 0.25rem;
  }

  .result-list {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }

  .result {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    padding: 0.85rem 0.95rem;
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.06);
    transition: background 0.15s, border-color 0.15s;
  }

  .result.active {
    border-color: rgba(202, 138, 4, 0.55);
    background: rgba(202, 138, 4, 0.08);
  }

  .result-head {
    display: flex;
    align-items: center;
    gap: 0.55rem;
  }

  .file-icon {
    width: 18px;
    height: 18px;
    color: #636366;
    flex-shrink: 0;
  }

  .file-name {
    flex: 1;
    min-width: 0;
    font-size: 0.9rem;
    font-weight: 600;
    color: #f5f5f7;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .page-badge {
    font-size: 0.72rem;
    color: rgba(202, 138, 4, 0.85);
    font-variant-numeric: tabular-nums;
    flex-shrink: 0;
    white-space: nowrap;
  }

  .open-btn {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.28rem 0.7rem;
    border-radius: 7px;
    border: 1px solid rgba(202, 138, 4, 0.4);
    background: rgba(202, 138, 4, 0.12);
    color: rgba(202, 138, 4, 0.95);
    font-size: 0.76rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }

  .open-btn:hover {
    background: rgba(202, 138, 4, 0.22);
    border-color: rgba(202, 138, 4, 0.65);
  }

  .result-path {
    font-size: 0.72rem;
    color: #48484a;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .result-text {
    font-size: 0.85rem;
    line-height: 1.5;
    color: #c7c7cc;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    overflow: hidden;
  }
</style>
