<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import * as pdfjsLib from 'pdfjs-dist'
  import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
  import { ReadFileAsBase64 } from '../wailsjs/go/main/App.js'

  pdfjsLib.GlobalWorkerOptions.workerSrc = workerUrl

  export let docPath: string
  export let initialPage: number = 1
  export let highlightText: string = ''   // matched chunk text to highlight
  export let onClose: () => void = () => {}

  let canvas: HTMLCanvasElement
  let hlCanvas: HTMLCanvasElement          // highlight overlay canvas
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
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
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

      // Draw text highlights on overlay canvas
      if (highlightText) {
        await renderHighlights(page, viewport)
      } else {
        clearHighlights(viewport)
      }
    } catch (e: any) {
      error = `Render error: ${e?.message || e}`
    }
  }

  // Draws amber highlight rectangles over text items that appear in highlightText.
  async function renderHighlights(page: any, viewport: any) {
    if (!hlCanvas) return

    hlCanvas.width = viewport.width
    hlCanvas.height = viewport.height
    const ctx = hlCanvas.getContext('2d')!
    ctx.clearRect(0, 0, viewport.width, viewport.height)

    const textContent = await page.getTextContent()
    const normalTarget = normalize(highlightText)

    ctx.fillStyle = 'rgba(202, 138, 4, 0.28)'

    for (const item of textContent.items as any[]) {
      const str: string = item.str ?? ''
      if (str.trim().length < 3) continue
      if (!normalTarget.includes(normalize(str))) continue

      // item.transform = [a, b, c, d, tx, ty] — (tx, ty) is PDF-space origin (baseline)
      const tx: number = item.transform[4]
      const ty: number = item.transform[5]
      const [x, y] = viewport.convertToViewportPoint(tx, ty)

      // Height of the glyph box in screen pixels
      const h = Math.abs(item.height * viewport.scale)
      // Width derived from item.width (PDF units) scaled to screen
      const w = item.width * viewport.scale

      // y is the baseline in screen coords (top-down); text box sits above it
      ctx.fillRect(x, y - h, w, h)
    }
  }

  function clearHighlights(viewport: any) {
    if (!hlCanvas) return
    hlCanvas.width = viewport.width
    hlCanvas.height = viewport.height
  }

  function normalize(text: string): string {
    return text.toLowerCase().replace(/\s+/g, ' ').trim()
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
      {#if highlightText}
        <span class="highlight-badge">Highlighted match</span>
      {/if}
      <button class="close-btn" on:click={onClose} aria-label="Close PDF viewer">&times;</button>
    </header>

    <div class="canvas-wrap">
      {#if loading}
        <div class="loading">Loading PDF…</div>
      {:else if error}
        <div class="viewer-error">{error}</div>
      {:else}
        <!-- PDF canvas + highlight overlay, stacked via CSS -->
        <div class="canvas-stack">
          <canvas bind:this={canvas} class="pdf-canvas" />
          <canvas bind:this={hlCanvas} class="hl-canvas" aria-hidden="true" />
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .viewer-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .viewer-container {
    background: #141414;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
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
    padding: 0.65rem 1rem;
    background: #1c1c1e;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    flex-shrink: 0;
  }

  .doc-name {
    flex: 1;
    font-size: 0.85rem;
    color: #aeaeb2;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .nav {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
    color: #636366;
  }

  .nav-btn, .zoom-btn {
    background: rgba(255, 255, 255, 0.07);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #aeaeb2;
    border-radius: 5px;
    padding: 0.2rem 0.6rem;
    cursor: pointer;
    font-size: 1rem;
    line-height: 1;
    transition: background 0.15s;
  }

  .nav-btn:disabled { opacity: 0.25; cursor: default; }
  .nav-btn:not(:disabled):hover, .zoom-btn:hover { background: rgba(255,255,255,0.13); }

  .page-info, .zoom-label { min-width: 80px; text-align: center; color: #636366; }

  .highlight-badge {
    font-size: 0.72rem;
    padding: 0.2rem 0.55rem;
    border-radius: 20px;
    background: rgba(202, 138, 4, 0.15);
    color: rgba(202, 138, 4, 0.9);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .close-btn {
    background: rgba(255,255,255,0.08);
    border: none;
    color: #aeaeb2;
    font-size: 1.3rem;
    cursor: pointer;
    line-height: 1;
    padding: 0.15rem 0.5rem;
    border-radius: 5px;
    transition: background 0.15s, color 0.15s;
    flex-shrink: 0;
  }
  .close-btn:hover { background: rgba(255,69,58,0.2); color: #ff453a; }

  .canvas-wrap {
    flex: 1;
    overflow: auto;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding: 1.25rem;
    background: #0c0c0f;
  }

  /* Stack PDF canvas and highlight overlay on top of each other */
  .canvas-stack {
    position: relative;
    display: inline-block;
    line-height: 0;
  }

  .pdf-canvas {
    display: block;
    box-shadow: 0 4px 32px rgba(0, 0, 0, 0.6);
    border-radius: 2px;
  }

  /* Overlay canvas sits exactly on top of the PDF canvas, pointer-events off */
  .hl-canvas {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none;
    border-radius: 2px;
  }

  .loading, .viewer-error {
    color: #636366;
    font-size: 0.95rem;
    padding: 2rem;
    text-align: center;
    align-self: center;
  }

  .viewer-error { color: #ff453a; }
</style>
