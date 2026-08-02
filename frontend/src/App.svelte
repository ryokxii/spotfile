<script lang="ts">
  import { onMount } from 'svelte'
  import { fade, fly } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { Search, GenerateAnswer, SelectFolder, IndexFolder } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import PDFViewer from './PDFViewer.svelte'
  import StatusBar from './StatusBar.svelte'

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
  let answer = ''
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

  // PDF viewer
  let pdfViewerPath = ''
  let pdfViewerPage = 1
  let pdfViewerHighlight = ''
  let showPDFViewer = false

  const topK = 5

  $: sourcePDFs = [...new Set(
    results
      .filter(r => r.docPath.toLowerCase().endsWith('.pdf'))
      .map(r => r.docPath)
  )]

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
    answer = ''
    searchError = ''

    try {
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
      pdfViewerHighlight = result.text
      showPDFViewer = true
    }
  }

  function getFirstResult(pdf: string): SearchResult | undefined {
    return results.find(r => r.docPath === pdf)
  }

  function filename(path: string) {
    return path.split(/[/\\]/).pop() ?? path
  }

  function dirpath(path: string) {
    const parts = path.split(/[/\\]/)
    parts.pop()
    return parts.join('/') || path
  }
</script>

<!-- ── Root ───────────────────────────────────────────────────── -->
<div class="root">
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

        {#if answer}
          <div class="answer-block">
            <span class="label">Answer</span>
            <p class="answer-text">{answer}</p>

            {#if sourcePDFs.length > 0}
              <div class="chips">
                {#each sourcePDFs as pdf}
                  {@const first = getFirstResult(pdf)}
                  <button
                    class="chip"
                    on:click={() => first && openResult(first)}
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

        {#if results.length > 0}
          <div class="result-list">
            {#each results as r, i (r.docPath + r.chunkIdx)}
              {@const isPDF = r.docPath.toLowerCase().endsWith('.pdf')}
              <button
                class="result-card"
                class:clickable={isPDF}
                on:click={() => isPDF && openResult(r)}
                tabindex={isPDF ? 0 : -1}
                aria-disabled={!isPDF}
                in:fly={{ y: 12, duration: 300, delay: i * 70, easing: cubicOut }}
              >
                <svg class="file-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                  <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/>
                  <path d="M14 2v6h6" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>

                <div class="file-info">
                  <span class="file-name">{filename(r.docPath)}</span>
                  <span class="file-path">{dirpath(r.docPath)}</span>
                </div>

                {#if isPDF}
                  <span class="page-badge">p.{r.pageNum || 1}</span>
                {/if}

                <div class="rel-bar" style="--rel:{Math.min(r.score, 1)}" title="Relevance: {r.score.toFixed(2)}"></div>
              </button>
            {/each}
          </div>
        {/if}

      </div>
    {/if}

  </div>

  {#if showPDFViewer}
    <PDFViewer
      docPath={pdfViewerPath}
      initialPage={pdfViewerPage}
      highlightText={pdfViewerHighlight}
      onClose={() => (showPDFViewer = false)}
    />
  {/if}

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

  .column {
    width: 100%;
    max-width: 660px;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    padding-top: 18vh;
    transition: padding-top 0.3s ease;
  }

  .column.in-results {
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
    transition: background 0.15s, color 0.15s, border-color 0.15s;
    white-space: nowrap;
  }

  .chip:hover {
    background: rgba(202,138,4,0.12);
    border-color: rgba(202,138,4,0.35);
    color: #f5f5f7;
  }

  .chip-icon { width: 13px; height: 13px; flex-shrink: 0; }

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

  .result-card.clickable { cursor: pointer; }

  .result-card.clickable:hover {
    background: rgba(255,255,255,0.07);
    border-color: rgba(255,255,255,0.1);
  }

  .result-card[aria-disabled='true'] { cursor: default; opacity: 0.6; }

  .file-icon { width: 22px; height: 22px; color: #48484a; flex-shrink: 0; }

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

  .page-badge {
    font-size: 0.72rem;
    color: rgba(202,138,4,0.75);
    font-variant-numeric: tabular-nums;
    flex-shrink: 0;
    white-space: nowrap;
  }

  .rel-bar {
    width: 36px;
    height: 3px;
    background: rgba(255,255,255,0.08);
    border-radius: 2px;
    overflow: hidden;
    flex-shrink: 0;
    position: relative;
  }

  .rel-bar::after {
    content: '';
    position: absolute;
    inset: 0;
    width: calc(var(--rel) * 100%);
    background: rgba(202, 138, 4, 0.7);
    border-radius: 2px;
    transition: width 0.4s ease;
  }
</style>
