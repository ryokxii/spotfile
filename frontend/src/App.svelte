<script lang="ts">
  import { onMount } from 'svelte'
  import { Search, GenerateAnswer, StoreSize } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import PDFViewer from './PDFViewer.svelte'
  import StatusBar from './StatusBar.svelte'
  import SettingsModal from './SettingsModal.svelte'

  interface SearchResult {
    docPath: string
    chunkIdx: number
    pageNum: number
    text: string
    score: number
  }

  // UI phase
  type Phase = 'idle' | 'loading' | 'results' | 'error'
  let phase: Phase = 'idle'

  // Engine & settings
  let engineReady = false
  let showSettings = false
  let storeChunks = 0

  // Search
  let query = ''
  let results: SearchResult[] = []
  let answer = ''
  let searchError = ''

  // Indexing (passed to StatusBar + SettingsModal)
  let indexing = false
  let indexedCount = 0
  let totalFiles = 0
  let currentFile = ''
  let watcherActive = false
  let reindexing = false

  // PDF viewer
  let pdfViewerPath = ''
  let pdfViewerPage = 1
  let showPDFViewer = false

  const topK = 5

  // Unique source PDFs from results
  $: sourcePDFs = [...new Set(
    results
      .filter(r => r.docPath.toLowerCase().endsWith('.pdf'))
      .map(r => r.docPath)
  )]

  // Status hint below search bar
  $: hint = engineReady
    ? phase === 'idle' ? 'Ready — type your question and press Enter'
    : phase === 'loading' ? 'Searching…'
    : phase === 'error' ? searchError
    : ''
    : 'Configure the engine to get started'

  onMount(() => {
    EventsOn('index:chunk', (data: any) => {
      indexedCount = data.total
      currentFile = data.path
    })
    EventsOn('index:done', async () => {
      indexing = false
      currentFile = ''
      storeChunks = await StoreSize()
    })
    EventsOn('watcher:started', () => { watcherActive = true })
    EventsOn('watcher:reindexing', (data: any) => { reindexing = true; currentFile = data.path })
    EventsOn('watcher:done', () => { reindexing = false; currentFile = '' })
  })

  async function search() {
    const q = query.trim()
    if (!q || !engineReady || phase === 'loading') return

    phase = 'loading'
    results = []
    answer = ''
    searchError = ''

    try {
      // Run semantic search and answer generation concurrently
      const [searchRes, answerRes] = await Promise.allSettled([
        Search(q, topK),
        GenerateAnswer(q, topK),
      ])
      results = searchRes.status === 'fulfilled' ? (searchRes.value ?? []) : []
      answer = answerRes.status === 'fulfilled' ? (answerRes.value ?? '') : ''
      phase = 'results'
    } catch (e: any) {
      searchError = e?.message || String(e)
      phase = 'error'
    }
  }

  function openResult(result: SearchResult) {
    if (result.docPath.toLowerCase().endsWith('.pdf')) {
      pdfViewerPath = result.docPath
      pdfViewerPage = result.pageNum > 0 ? result.pageNum : 1
      showPDFViewer = true
    }
  }

  function filename(path: string) {
    return path.split(/[/\\]/).pop() ?? path
  }

  function dirpath(path: string) {
    const parts = path.split(/[/\\]/)
    parts.pop()
    return parts.join('/') || path
  }

  function handleIndexStart(e: CustomEvent<{ paths: string[]; total: number }>) {
    indexing = true
    indexedCount = 0
    totalFiles = e.detail.total
  }
</script>

<!-- ── Root ───────────────────────────────────────────────────── -->
<div class="root">
  <div class="column">

    <!-- Logo (only shown when idle) -->
    {#if phase === 'idle'}
      <div class="logo-area">
        <!-- Nested-square spiral icon -->
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
    <div class="search-wrap" class:results-mode={phase === 'results'}>
      <div class="search-bar" class:loading={phase === 'loading'}>
        <!-- Search icon -->
        <svg class="icon-search" viewBox="0 0 20 20" fill="none" aria-hidden="true">
          <circle cx="8.5" cy="8.5" r="5.5" stroke="currentColor" stroke-width="1.8"/>
          <path d="M13 13l3.5 3.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
        </svg>

        <input
          type="text"
          class="search-input"
          bind:value={query}
          placeholder="Search your files…"
          disabled={phase === 'loading'}
          on:keydown={(e) => e.key === 'Enter' && search()}
          autocomplete="off"
          spellcheck="false"
        />

        <!-- Settings / spinner -->
        {#if phase === 'loading'}
          <span class="spinner" aria-label="Searching" />
        {:else}
          <button
            class="icon-btn"
            title="Settings"
            on:click={() => (showSettings = true)}
            aria-label="Open settings"
          >
            <svg viewBox="0 0 20 20" fill="none">
              <circle cx="10" cy="10" r="2.5" stroke="currentColor" stroke-width="1.6"/>
              <path d="M10 3v1.5M10 15.5V17M3 10h1.5M15.5 10H17M5.05 5.05l1.06 1.06M13.89 13.89l1.06 1.06M14.95 5.05l-1.06 1.06M6.11 13.89l-1.06 1.06"
                    stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>
            </svg>
          </button>
        {/if}
      </div>

      <!-- Status hint -->
      <p class="hint" class:error={phase === 'error'}>{hint}</p>
    </div>

    <!-- Results area -->
    {#if phase === 'results'}
      <div class="results-area">

        <!-- Answer -->
        {#if answer}
          <div class="answer-block">
            <span class="label">Answer</span>
            <p class="answer-text">{answer}</p>

            <!-- Source chips -->
            {#if sourcePDFs.length > 0}
              <div class="chips">
                {#each sourcePDFs as pdf}
                  <button
                    class="chip"
                    on:click={() => openResult({ docPath: pdf, chunkIdx: 0, pageNum: 1, text: '', score: 0 })}
                    title={pdf}
                  >
                    <svg viewBox="0 0 16 16" fill="none" aria-hidden="true" class="chip-icon">
                      <rect x="2" y="1" width="10" height="13" rx="2" stroke="currentColor" stroke-width="1.4"/>
                      <path d="M5 5h5M5 8h5M5 11h3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
                    </svg>
                    {filename(pdf)}
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {/if}

        <!-- Result list -->
        {#if results.length > 0}
          <div class="result-list">
            {#each results as r (r.docPath + r.chunkIdx)}
              {@const isPDF = r.docPath.toLowerCase().endsWith('.pdf')}
              <button
                class="result-card"
                class:clickable={isPDF}
                on:click={() => openResult(r)}
                disabled={!isPDF}
              >
                <!-- File icon -->
                <svg class="file-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                  <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
                  <path d="M14 2v6h6" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>

                <div class="file-info">
                  <span class="file-name">{filename(r.docPath)}</span>
                  <span class="file-path">{dirpath(r.docPath)}</span>
                </div>

                <span class="score">{r.score.toFixed(2)}</span>
              </button>
            {/each}
          </div>
        {/if}

      </div>
    {/if}

  </div><!-- /column -->

  <!-- Overlays -->
  <SettingsModal
    visible={showSettings}
    {indexing}
    {indexedCount}
    {totalFiles}
    {currentFile}
    {storeChunks}
    on:close={() => (showSettings = false)}
    on:engineReady={() => { engineReady = true; showSettings = false }}
    on:indexStart={handleIndexStart}
  />

  {#if showPDFViewer}
    <PDFViewer
      docPath={pdfViewerPath}
      initialPage={pdfViewerPage}
      onClose={() => (showPDFViewer = false)}
    />
  {/if}

  <StatusBar
    {indexing}
    {reindexing}
    {watcherActive}
    {indexedCount}
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

  /* ── Column ─────────────────────────────────────────────── */
  .column {
    width: 100%;
    max-width: 660px;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    padding-top: 18vh;
    transition: padding-top 0.3s ease;
  }

  /* When in results mode, collapse top padding so search stays near top */
  .column:has(.results-mode) {
    padding-top: 6vh;
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
    border-color: rgba(255, 255, 255, 0.28);
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
    caret-color: white;
  }

  .search-input::placeholder { color: #48484a; }

  .search-input:disabled { opacity: 0.5; }

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

  /* Loading spinner */
  .spinner {
    width: 18px;
    height: 18px;
    border: 2px solid rgba(255,255,255,0.15);
    border-top-color: rgba(255,255,255,0.7);
    border-radius: 50%;
    flex-shrink: 0;
    animation: spin 0.8s linear infinite;
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

  /* Answer */
  .answer-block {
    display: flex;
    flex-direction: column;
    gap: 0.85rem;
  }

  .label {
    font-size: 0.68rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: #636366;
  }

  .answer-text {
    font-size: 0.97rem;
    line-height: 1.75;
    color: #e5e5ea;
    white-space: pre-wrap;
  }

  /* Source chips */
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .chip {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0.75rem;
    border-radius: 20px;
    border: 1px solid rgba(255,255,255,0.12);
    background: rgba(255,255,255,0.05);
    color: #aeaeb2;
    font-size: 0.8rem;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
    white-space: nowrap;
  }

  .chip:hover {
    background: rgba(255,255,255,0.1);
    color: #f5f5f7;
  }

  .chip-icon {
    width: 13px;
    height: 13px;
    flex-shrink: 0;
  }

  /* Result cards */
  .result-list {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .result-card {
    display: flex;
    align-items: center;
    gap: 0.85rem;
    padding: 0.8rem 0.9rem;
    border-radius: 10px;
    background: rgba(255,255,255,0.03);
    border: 1px solid transparent;
    cursor: default;
    text-align: left;
    width: 100%;
    color: inherit;
    transition: background 0.15s, border-color 0.15s;
  }

  .result-card.clickable {
    cursor: pointer;
  }

  .result-card.clickable:hover {
    background: rgba(255,255,255,0.07);
    border-color: rgba(255,255,255,0.1);
  }

  .result-card:disabled {
    cursor: default;
  }

  .file-icon {
    width: 22px;
    height: 22px;
    color: #48484a;
    flex-shrink: 0;
  }

  .file-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .file-name {
    font-size: 0.88rem;
    font-weight: 500;
    color: #e5e5ea;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .file-path {
    font-size: 0.75rem;
    color: #48484a;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .score {
    font-size: 0.78rem;
    color: #48484a;
    font-variant-numeric: tabular-nums;
    flex-shrink: 0;
  }
</style>
