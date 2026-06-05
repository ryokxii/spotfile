<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { InitEngine, IndexFiles } from '../wailsjs/go/main/App.js'

  export let visible = false
  export let indexing = false
  export let indexedCount = 0
  export let totalFiles = 0
  export let currentFile = ''
  export let storeChunks = 0

  const dispatch = createEventDispatcher<{
    close: void
    engineReady: void
    indexStart: { paths: string[]; total: number }
  }>()

  let libraryPath = ''
  let modelPath = ''
  let vocabPath = ''
  let workers = 0

  let initState: 'idle' | 'busy' | 'ok' | 'err' = 'idle'
  let initError = ''

  // Bind directly to the hidden file input for reliable cross-platform triggering
  let fileInput: HTMLInputElement

  async function initEngine() {
    initState = 'busy'
    initError = ''
    try {
      await InitEngine({ libraryPath, modelPath, vocabPath, workers })
      initState = 'ok'
      dispatch('engineReady')
    } catch (e: any) {
      initState = 'err'
      initError = e?.message || String(e)
    }
  }

  async function handleFiles(event: Event) {
    const input = event.target as HTMLInputElement
    const files = input.files
    if (!files || files.length === 0) return
    const paths = Array.from(files).map(f => (f as any).path || f.name)
    dispatch('indexStart', { paths, total: files.length })
    input.value = ''
    try {
      await IndexFiles(paths)
    } catch {
      /* errors surface through watcher/done events handled in App */
    }
  }

  $: shortFile = currentFile ? currentFile.split('/').pop() ?? currentFile : ''
  $: percent = totalFiles > 0 ? Math.round((indexedCount / totalFiles) * 100) : 0
  $: workersLabel = workers === 0 ? 'Auto' : String(workers)
</script>

{#if visible}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="backdrop" on:click|self={() => dispatch('close')} role="dialog" aria-modal="true">
    <div class="sheet">
      <header>
        <span class="title">Settings</span>
        <button class="close-btn" on:click={() => dispatch('close')} aria-label="Close settings">
          <svg viewBox="0 0 20 20" fill="none"><path d="M5 5l10 10M15 5L5 15" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/></svg>
        </button>
      </header>

      <!-- Engine -->
      <section>
        <h3>Engine</h3>
        <label>
          <span>ONNX Runtime Library <span class="required">required</span></span>
          <input type="text" bind:value={libraryPath} placeholder="e.g. /opt/homebrew/lib/libonnxruntime.dylib" />
        </label>
        <label>
          <span>Embedding Model (.onnx)</span>
          <input type="text" bind:value={modelPath} placeholder="~/.spotfile/model.onnx (default)" />
        </label>
        <label>
          <span>Vocabulary (.txt)</span>
          <input type="text" bind:value={vocabPath} placeholder="~/.spotfile/vocab.txt (default)" />
        </label>
        <label class="inline">
          <span>Worker threads</span>
          <div class="workers-wrap">
            <input
              type="number"
              bind:value={workers}
              min="0"
              placeholder="0"
              aria-label="Worker threads (0 = auto)"
            />
            <span class="workers-hint">{workersLabel}</span>
          </div>
        </label>

        <div class="init-row">
          <button
            class="btn-primary"
            on:click={initEngine}
            disabled={initState === 'busy' || !libraryPath.trim()}
            title={!libraryPath.trim() ? 'Enter the ONNX Runtime library path first' : undefined}
          >
            {initState === 'busy' ? 'Initializing…' : 'Initialize Engine'}
          </button>
          {#if initState === 'ok'}
            <span class="badge ok">Ready</span>
          {:else if initState === 'err'}
            <span class="badge err" title={initError}>{initError}</span>
          {/if}
        </div>
      </section>

      <!-- Indexing -->
      <section>
        <h3>Index Files</h3>
        <p class="hint">Select text, Markdown, or PDF files to embed and index.</p>

        <!-- Hidden real input — triggered programmatically for Wails compatibility -->
        <input
          type="file"
          bind:this={fileInput}
          multiple
          accept=".txt,.md,.pdf"
          on:change={handleFiles}
          disabled={indexing}
          style="display:none"
          aria-hidden="true"
          tabindex="-1"
        />

        <div class="file-row">
          <button
            class="file-btn"
            class:disabled={indexing}
            disabled={indexing}
            on:click={() => fileInput?.click()}
          >
            {indexing ? `Indexing… ${percent}%` : 'Choose files'}
          </button>
          {#if indexing && shortFile}
            <span class="current-file" title={currentFile}>{shortFile}</span>
          {/if}
        </div>

        {#if indexing && totalFiles > 0}
          <div class="prog-track">
            <div class="prog-fill" style="width:{percent}%" />
          </div>
        {/if}

        {#if storeChunks > 0}
          <p class="store-info">{storeChunks.toLocaleString()} chunks in index</p>
        {/if}
      </section>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(6px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 800;
  }

  .sheet {
    background: #1c1c1e;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 16px;
    width: min(560px, 94vw);
    max-height: 90vh;
    overflow-y: auto;
    padding: 0 0 1.5rem;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1.25rem 1.5rem 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    position: sticky;
    top: 0;
    background: #1c1c1e;
    z-index: 1;
  }

  .title {
    font-size: 1rem;
    font-weight: 600;
    color: #f5f5f7;
  }

  .close-btn {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: none;
    background: rgba(255, 255, 255, 0.1);
    color: #aeaeb2;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.15s;
  }
  .close-btn svg { width: 14px; height: 14px; }
  .close-btn:hover { background: rgba(255, 255, 255, 0.18); }

  section {
    padding: 1.25rem 1.5rem 0;
  }

  h3 {
    font-size: 0.68rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: #636366;
    margin: 0 0 1rem;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    margin-bottom: 0.85rem;
  }

  label.inline {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }

  label span {
    font-size: 0.8rem;
    color: #aeaeb2;
  }

  .required {
    font-size: 0.7rem;
    color: rgba(202, 138, 4, 0.8);
    font-weight: 500;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    margin-left: 0.4rem;
  }

  input[type='text'],
  input[type='number'] {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: #f5f5f7;
    font-size: 0.85rem;
    padding: 0.55rem 0.75rem;
    outline: none;
    transition: border-color 0.15s;
  }

  input[type='text']:focus,
  input[type='number']:focus {
    border-color: rgba(202, 138, 4, 0.5);
  }

  input[type='number'] {
    width: 60px;
    text-align: center;
  }

  /* Workers: number input + "Auto" label side by side */
  .workers-wrap {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .workers-hint {
    font-size: 0.78rem;
    color: #636366;
    min-width: 2.5rem;
  }

  .init-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.25rem;
  }

  .btn-primary {
    padding: 0.55rem 1.1rem;
    border-radius: 8px;
    border: none;
    background: white;
    color: #0c0c0f;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.15s;
  }
  .btn-primary:hover:not(:disabled) { opacity: 0.85; }
  .btn-primary:disabled { opacity: 0.35; cursor: default; }

  .badge {
    font-size: 0.78rem;
    padding: 0.25rem 0.6rem;
    border-radius: 20px;
    font-weight: 500;
  }
  .badge.ok { background: rgba(48, 209, 88, 0.15); color: #30d158; }
  .badge.err {
    background: rgba(255, 69, 58, 0.15);
    color: #ff453a;
    max-width: 280px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: help;
  }

  .hint {
    font-size: 0.8rem;
    color: #636366;
    margin: 0 0 0.85rem;
  }

  /* File picker row */
  .file-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .file-btn {
    display: inline-flex;
    align-items: center;
    padding: 0.55rem 1.1rem;
    border-radius: 8px;
    border: none;
    background: rgba(255, 255, 255, 0.1);
    color: #f5f5f7;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
    white-space: nowrap;
  }

  .file-btn:hover:not(.disabled) { background: rgba(255,255,255,0.18); }
  .file-btn.disabled { opacity: 0.4; cursor: default; }

  .current-file {
    font-size: 0.78rem;
    color: #636366;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 220px;
  }

  .prog-track {
    width: 100%;
    height: 3px;
    background: rgba(255,255,255,0.08);
    border-radius: 2px;
    overflow: hidden;
    margin-top: 0.75rem;
  }
  .prog-fill {
    height: 100%;
    background: rgba(202, 138, 4, 0.75);
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .store-info {
    font-size: 0.78rem;
    color: #636366;
    margin-top: 0.6rem;
  }
</style>
