<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import * as pdfjsLib from 'pdfjs-dist'
  import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
  import { ReadFileAsBase64 } from '../wailsjs/go/main/App.js'

  pdfjsLib.GlobalWorkerOptions.workerSrc = workerUrl

  export let docPath: string
  export let initialPage: number = 1
  export let onClose: () => void = () => {}

  let canvas: HTMLCanvasElement
  let currentPage = initialPage || 1
  let totalPages = 0
  let loading = true
  let error = ''
  let scale = 1.5

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let pdfDoc: any = null

  async function loadPDF() {
    loading = true
    error = ''
    try {
      const b64 = await ReadFileAsBase64(docPath)
      const binary = atob(b64)
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i)
      }
      const loadingTask = pdfjsLib.getDocument({ data: bytes })
      pdfDoc = await loadingTask.promise
      totalPages = pdfDoc.numPages
      if (currentPage > totalPages) currentPage = 1
      await renderPage(currentPage)
    } catch (e: any) {
      error = `Failed to load PDF: ${e?.message || e}`
    } finally {
      loading = false
    }
  }

  async function renderPage(num: number) {
    if (!pdfDoc || !canvas) return
    try {
      const page = await pdfDoc.getPage(num)
      const viewport = page.getViewport({ scale })
      canvas.width = viewport.width
      canvas.height = viewport.height
      const ctx = canvas.getContext('2d')!
      await page.render({ canvasContext: ctx, viewport }).promise
    } catch (e: any) {
      error = `Render error: ${e?.message || e}`
    }
  }

  async function goToPage(num: number) {
    if (num < 1 || num > totalPages) return
    currentPage = num
    await renderPage(currentPage)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') goToPage(currentPage + 1)
    else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') goToPage(currentPage - 1)
    else if (e.key === 'Escape') onClose()
  }

  onMount(() => {
    loadPDF()
    window.addEventListener('keydown', handleKeydown)
  })

  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown)
    if (pdfDoc) pdfDoc.destroy()
  })
</script>

<div
  class="viewer-backdrop"
  role="dialog"
  aria-modal="true"
  on:click|self={onClose}
  on:keydown={(e) => e.key === 'Escape' && onClose()}
>
  <div class="viewer-container">
    <header class="viewer-header">
      <span class="doc-name" title={docPath}>{docPath.split('/').pop()}</span>
      <div class="nav">
        <button class="nav-btn" on:click={() => goToPage(currentPage - 1)} disabled={currentPage <= 1}>&lsaquo;</button>
        <span class="page-info">Page {currentPage} / {totalPages}</span>
        <button class="nav-btn" on:click={() => goToPage(currentPage + 1)} disabled={currentPage >= totalPages}>&rsaquo;</button>
        <button class="zoom-btn" on:click={() => { scale = Math.max(0.5, scale - 0.25); renderPage(currentPage) }}>−</button>
        <span class="zoom-label">{Math.round(scale * 100)}%</span>
        <button class="zoom-btn" on:click={() => { scale = Math.min(4, scale + 0.25); renderPage(currentPage) }}>+</button>
      </div>
      <button class="close-btn" on:click={onClose}>&times;</button>
    </header>

    <div class="canvas-wrap">
      {#if loading}
        <div class="loading">Loading PDF…</div>
      {:else if error}
        <div class="viewer-error">{error}</div>
      {:else}
        <canvas bind:this={canvas} class="pdf-canvas" />
      {/if}
    </div>
  </div>
</div>

<style>
  .viewer-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.75);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .viewer-container {
    background: #1a2a3a;
    border: 1px solid rgba(100, 181, 246, 0.3);
    border-radius: 8px;
    display: flex;
    flex-direction: column;
    width: 90vw;
    max-width: 1000px;
    height: 90vh;
    overflow: hidden;
  }

  .viewer-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 1rem;
    background: rgba(15, 28, 46, 0.8);
    border-bottom: 1px solid rgba(100, 181, 246, 0.15);
    flex-shrink: 0;
  }

  .doc-name {
    flex: 1;
    font-size: 0.85rem;
    color: #90caf9;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .nav {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
    color: #b0bec5;
  }

  .nav-btn, .zoom-btn {
    background: rgba(100, 181, 246, 0.15);
    border: 1px solid rgba(100, 181, 246, 0.25);
    color: #90caf9;
    border-radius: 4px;
    padding: 0.2rem 0.6rem;
    cursor: pointer;
    font-size: 1rem;
    line-height: 1;
  }

  .nav-btn:disabled {
    opacity: 0.3;
    cursor: default;
  }

  .nav-btn:not(:disabled):hover, .zoom-btn:hover {
    background: rgba(100, 181, 246, 0.3);
  }

  .page-info, .zoom-label {
    min-width: 80px;
    text-align: center;
  }

  .close-btn {
    background: none;
    border: none;
    color: #90caf9;
    font-size: 1.5rem;
    cursor: pointer;
    line-height: 1;
    padding: 0 0.25rem;
  }

  .close-btn:hover {
    color: #ef5350;
  }

  .canvas-wrap {
    flex: 1;
    overflow: auto;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding: 1rem;
    background: #111;
  }

  .pdf-canvas {
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.5);
    border-radius: 2px;
  }

  .loading, .viewer-error {
    color: #90caf9;
    font-size: 0.95rem;
    padding: 2rem;
    text-align: center;
  }

  .viewer-error {
    color: #ef9a9a;
  }
</style>
