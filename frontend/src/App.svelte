<script lang="ts">
  import { InitEngine, IndexFiles, Search, StoreSize } from '../wailsjs/go/main/App.js'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'

  interface SearchResult {
    docPath: string
    chunkIdx: number
    text: string
    score: number
  }

  let libraryPath: string = ''
  let modelPath: string = ''
  let vocabPath: string = ''
  let workers: number = 0

  let indexedFiles: string[] = []
  let indexedCount: number = 0
  let totalFiles: number = 0

  let query: string = ''
  let results: SearchResult[] = []
  let searching: boolean = false
  let indexing: boolean = false
  let error: string = ''
  let message: string = ''

  const topK: number = 5

  async function initEngine() {
    try {
      error = ''
      message = ''
      await InitEngine({
        libraryPath,
        modelPath,
        vocabPath,
        workers,
      })
      message = 'Engine initialized successfully'
    } catch (e: any) {
      error = `Init failed: ${e.message || e}`
    }
  }

  async function handleFileSelect(event: Event) {
    const target = event.target as HTMLInputElement
    const files = target.files
    if (!files || files.length === 0) return

    try {
      error = ''
      message = ''
      indexing = true
      indexedFiles = []
      indexedCount = 0
      totalFiles = files.length

      const paths = Array.from(files).map((f) => (f as any).path || f.name)

      // Listen for indexing events
      EventsOn('index:chunk', (data: any) => {
        indexedCount = data.total
        indexedFiles = Array.from(new Set([...indexedFiles, data.path]))
      })

      EventsOn('index:done', (total: number) => {
        indexing = false
        message = `Indexed ${total} chunks successfully`
        const store = StoreSize()
        message += ` (Store now contains ${store} total chunks)`
      })

      await IndexFiles(paths)
    } catch (e: any) {
      error = `Index failed: ${e.message || e}`
      indexing = false
    }
  }

  async function performSearch() {
    if (!query.trim()) {
      error = 'Please enter a search query'
      return
    }

    try {
      error = ''
      message = ''
      searching = true
      const res = await Search(query, topK)
      results = res || []

      if (results.length === 0) {
        message = 'No results found'
      } else {
        message = `Found ${results.length} result(s)`
      }
    } catch (e: any) {
      error = `Search failed: ${e.message || e}`
    } finally {
      searching = false
    }
  }

  function truncateText(text: string, maxLen: number = 100): string {
    return text.length > maxLen ? text.substring(0, maxLen) + '...' : text
  }
</script>

<main>
  <div class="container">
    <h1>Spotfile — Local Search</h1>

    <!-- Engine Configuration -->
    <section class="panel">
      <h2>Engine Configuration</h2>
      <div class="form-group">
        <label for="libraryPath">ONNX Runtime Library Path</label>
        <input
          id="libraryPath"
          type="text"
          bind:value={libraryPath}
          placeholder="e.g., /opt/homebrew/lib/libonnxruntime.dylib"
        />
      </div>
      <div class="form-group">
        <label for="modelPath">Model Path</label>
        <input
          id="modelPath"
          type="text"
          bind:value={modelPath}
          placeholder="e.g., ~/.spotfile/model.onnx"
        />
      </div>
      <div class="form-group">
        <label for="vocabPath">Vocab Path</label>
        <input
          id="vocabPath"
          type="text"
          bind:value={vocabPath}
          placeholder="e.g., ~/.spotfile/vocab.txt"
        />
      </div>
      <div class="form-group">
        <label for="workers">Workers (0 = auto)</label>
        <input
          id="workers"
          type="number"
          bind:value={workers}
          min="0"
          placeholder="0"
        />
      </div>
      <button on:click={initEngine} class="btn btn-primary">Initialize Engine</button>
    </section>

    <!-- File Indexing -->
    <section class="panel">
      <h2>Index Files</h2>
      <input
        type="file"
        multiple
        on:change={handleFileSelect}
        disabled={indexing}
        accept=".txt,.md,.pdf"
      />
      {#if indexing}
        <div class="progress">
          <p>Indexing... {indexedCount} / {totalFiles} files</p>
          <div class="progress-bar">
            <div
              class="progress-fill"
              style={`width: ${totalFiles > 0 ? (indexedCount / totalFiles) * 100 : 0}%`}
            />
          </div>
        </div>
      {/if}
    </section>

    <!-- Search -->
    <section class="panel">
      <h2>Search</h2>
      <div class="search-box">
        <input
          type="text"
          bind:value={query}
          placeholder="Enter search query..."
          on:keydown={(e) => e.key === 'Enter' && performSearch()}
          disabled={searching}
        />
        <button on:click={performSearch} disabled={searching} class="btn btn-primary">
          {searching ? 'Searching...' : 'Search'}
        </button>
      </div>
    </section>

    <!-- Results -->
    {#if results.length > 0}
      <section class="panel">
        <h2>Results</h2>
        <table class="results-table">
          <thead>
            <tr>
              <th>Score</th>
              <th>Document</th>
              <th>Chunk</th>
              <th>Text Preview</th>
            </tr>
          </thead>
          <tbody>
            {#each results as result (result.docPath + result.chunkIdx)}
              <tr>
                <td class="score">{(result.score * 100).toFixed(1)}%</td>
                <td class="mono">{result.docPath}</td>
                <td>{result.chunkIdx}</td>
                <td>{truncateText(result.text)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>
    {/if}

    <!-- Status Messages -->
    {#if error}
      <div class="alert alert-error">{error}</div>
    {/if}
    {#if message}
      <div class="alert alert-info">{message}</div>
    {/if}
  </div>
</main>

<style>
  * {
    box-sizing: border-box;
  }

  main {
    width: 100%;
    height: 100vh;
    background: linear-gradient(135deg, #1e3a5f 0%, #0f1c2e 100%);
    color: #e0e0e0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    overflow-y: auto;
    padding: 2rem 0;
  }

  .container {
    max-width: 1000px;
    margin: 0 auto;
    padding: 0 2rem;
  }

  h1 {
    text-align: center;
    margin-bottom: 3rem;
    font-size: 2.5rem;
    background: linear-gradient(135deg, #64b5f6 0%, #42a5f5 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }

  h2 {
    font-size: 1.5rem;
    margin-bottom: 1rem;
    color: #64b5f6;
    border-bottom: 2px solid #42a5f5;
    padding-bottom: 0.5rem;
  }

  .panel {
    background: rgba(30, 58, 95, 0.4);
    border: 1px solid rgba(100, 181, 246, 0.2);
    border-radius: 8px;
    padding: 1.5rem;
    margin-bottom: 2rem;
    backdrop-filter: blur(10px);
  }

  .form-group {
    margin-bottom: 1rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    font-weight: 500;
    color: #b0bec5;
  }

  input[type='text'],
  input[type='number'],
  input[type='file'] {
    width: 100%;
    padding: 0.75rem;
    background: rgba(15, 28, 46, 0.6);
    border: 1px solid rgba(100, 181, 246, 0.3);
    border-radius: 4px;
    color: #e0e0e0;
    font-size: 0.95rem;
    transition: all 0.2s;
  }

  input[type='text']:focus,
  input[type='number']:focus,
  input[type='file']:focus {
    outline: none;
    border-color: #42a5f5;
    background: rgba(15, 28, 46, 0.8);
    box-shadow: 0 0 8px rgba(66, 165, 245, 0.3);
  }

  input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn {
    padding: 0.75rem 1.5rem;
    border: none;
    border-radius: 4px;
    font-size: 0.95rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-primary {
    background: linear-gradient(135deg, #42a5f5 0%, #1e88e5 100%);
    color: white;
  }

  .btn-primary:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(66, 165, 245, 0.4);
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .search-box {
    display: flex;
    gap: 0.5rem;
  }

  .search-box input {
    flex: 1;
  }

  .search-box button {
    flex-shrink: 0;
  }

  .progress {
    margin-top: 1rem;
  }

  .progress p {
    margin-bottom: 0.5rem;
    font-size: 0.9rem;
    color: #90caf9;
  }

  .progress-bar {
    width: 100%;
    height: 8px;
    background: rgba(15, 28, 46, 0.6);
    border-radius: 4px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #42a5f5 0%, #64b5f6 100%);
    transition: width 0.3s ease;
  }

  .results-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9rem;
  }

  .results-table th {
    background: rgba(15, 28, 46, 0.4);
    padding: 0.75rem;
    text-align: left;
    font-weight: 600;
    color: #64b5f6;
    border-bottom: 1px solid rgba(100, 181, 246, 0.2);
  }

  .results-table td {
    padding: 0.75rem;
    border-bottom: 1px solid rgba(100, 181, 246, 0.1);
  }

  .results-table tr:hover {
    background: rgba(100, 181, 246, 0.05);
  }

  .score {
    font-weight: 600;
    color: #81c784;
    min-width: 60px;
  }

  .mono {
    font-family: 'Monaco', 'Courier New', monospace;
    color: #90caf9;
    word-break: break-word;
    max-width: 250px;
  }

  .alert {
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 1rem;
    border-left: 4px solid;
  }

  .alert-error {
    background: rgba(229, 57, 53, 0.1);
    border-left-color: #ef5350;
    color: #ef9a9a;
  }

  .alert-info {
    background: rgba(66, 165, 245, 0.1);
    border-left-color: #42a5f5;
    color: #90caf9;
  }

  @media (max-width: 768px) {
    .container {
      padding: 0 1rem;
    }

    h1 {
      font-size: 1.75rem;
      margin-bottom: 2rem;
    }

    .results-table {
      font-size: 0.8rem;
    }

    .results-table th,
    .results-table td {
      padding: 0.5rem;
    }
  }
</style>
