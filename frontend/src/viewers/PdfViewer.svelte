<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte'
  import * as pdfjsLib from 'pdfjs-dist'
  import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
  import { ReadFileAsBase64 } from '../../wailsjs/go/main/App.js'
  import { base64ToBytes } from '../lib/base64'
  import ViewerShell from './ViewerShell.svelte'

  pdfjsLib.GlobalWorkerOptions.workerSrc = workerUrl

  export let docPath: string
  export let initialPage: number = 1
  export let highlightText: string = '' // matched chunk text to highlight
  export let onClose: () => void = () => {}

  let canvas: HTMLCanvasElement
  let hlCanvas: HTMLCanvasElement // highlight overlay canvas
  let canvasWrap: HTMLElement // scrollable container
  let currentPage = initialPage || 1
  let totalPages = 0
  let loading = true
  let error = ''
  let scale = 1.25
  let matchTop = -1 // y of the topmost highlight (canvas px), -1 = none

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let pdfDoc: any = null

  async function loadPDF() {
    loading = true
    error = ''
    try {
      const bytes = base64ToBytes(await ReadFileAsBase64(docPath))
      const loadingTask = pdfjsLib.getDocument({ data: bytes })
      pdfDoc = await loadingTask.promise
      totalPages = pdfDoc.numPages
      if (currentPage > totalPages) currentPage = 1
      // Reveal the canvas (it lives in the {:else} branch) and let it mount
      // before the first render, or renderPage bails on an unbound canvas.
      loading = false
      await tick()
      await renderPage(currentPage)
    } catch (e: any) {
      error = `Failed to load PDF: ${e?.message || e}`
      loading = false
    }
  }

  async function renderPage(num: number) {
    if (!pdfDoc || !canvas) return

    let page: any
    let viewport: any
    try {
      page = await pdfDoc.getPage(num)
      viewport = page.getViewport({ scale })

      canvas.width = viewport.width
      canvas.height = viewport.height
      const ctx = canvas.getContext('2d')!
      await page.render({ canvasContext: ctx, viewport }).promise
      error = ''
    } catch (e: any) {
      error = `Render error: ${e?.message || e}`
      return
    }

    // Highlighting is best-effort — a failure here must never blank the page.
    try {
      if (highlightText) await renderHighlights(page, viewport)
      else clearHighlights(viewport)
    } catch (e: any) {
      console.warn('PDF highlight render failed:', e)
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

    let topY = Infinity
    for (const item of (textContent?.items ?? []) as any[]) {
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
      if (y - h < topY) topY = y - h
    }
    matchTop = topY === Infinity ? -1 : topY
  }

  function clearHighlights(viewport: any) {
    matchTop = -1
    if (!hlCanvas) return
    hlCanvas.width = viewport.width
    hlCanvas.height = viewport.height
  }

  // Scroll the preview so the topmost highlighted match is near the top.
  function scrollToMatch() {
    if (matchTop < 0 || !canvas || !canvasWrap) return
    const canvasTop = canvas.getBoundingClientRect().top
    const wrapTop = canvasWrap.getBoundingClientRect().top
    const target = canvasWrap.scrollTop + (canvasTop - wrapTop) + matchTop - 60
    canvasWrap.scrollTo({ top: Math.max(0, target), behavior: 'smooth' })
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

  // React to prop changes so opening a different result switches the preview
  // without a full remount: a new document reloads; a new page (same document)
  // just navigates. Manual page nav changes currentPage but not the props, so
  // it never fights this.
  let mounted = false
  let loadedPath = ''
  let appliedRequest = ''
  $: request = `${docPath} ${initialPage} ${highlightText}`
  $: if (mounted && request !== appliedRequest) applyRequest()

  async function applyRequest() {
    appliedRequest = request
    if (docPath !== loadedPath) {
      loadedPath = docPath
      currentPage = initialPage || 1
      await loadPDF()
    } else if (initialPage && initialPage !== currentPage) {
      await goToPage(initialPage)
    } else {
      await renderPage(currentPage) // same page, highlight may have changed
    }
  }

  onMount(() => {
    mounted = true
    window.addEventListener('keydown', handleKeydown)
  })

  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown)
    if (pdfDoc) pdfDoc.destroy()
  })
</script>

<ViewerShell {docPath} {onClose}>
  <div slot="toolbar" class="toolbar">
    <div class="nav">
      <button class="nav-btn" on:click={() => goToPage(currentPage - 1)} disabled={currentPage <= 1}>&lsaquo;</button>
      <span class="page-info">Page {currentPage} / {totalPages}</span>
      <button class="nav-btn" on:click={() => goToPage(currentPage + 1)} disabled={currentPage >= totalPages}>&rsaquo;</button>
      <button class="zoom-btn" on:click={() => { scale = Math.max(0.5, scale - 0.25); renderPage(currentPage) }}>−</button>
      <span class="zoom-label">{Math.round(scale * 100)}%</span>
      <button class="zoom-btn" on:click={() => { scale = Math.min(4, scale + 0.25); renderPage(currentPage) }}>+</button>
    </div>
    {#if highlightText}
      <button
        class="highlight-badge"
        on:click={scrollToMatch}
        disabled={matchTop < 0}
        title={matchTop < 0 ? 'Match not found on this page' : 'Jump to the highlighted match'}
      >
        Highlighted match
      </button>
    {/if}
  </div>

  <div class="canvas-wrap" bind:this={canvasWrap}>
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
</ViewerShell>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-shrink: 0;
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
  .nav-btn:not(:disabled):hover, .zoom-btn:hover { background: rgba(255, 255, 255, 0.13); }

  .page-info, .zoom-label { min-width: 80px; text-align: center; color: #636366; }

  .highlight-badge {
    font-size: 0.72rem;
    padding: 0.25rem 0.6rem;
    border-radius: 20px;
    border: 1px solid rgba(202, 138, 4, 0.35);
    background: rgba(202, 138, 4, 0.15);
    color: rgba(202, 138, 4, 0.9);
    white-space: nowrap;
    flex-shrink: 0;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }

  .highlight-badge:hover:not(:disabled) {
    background: rgba(202, 138, 4, 0.28);
    border-color: rgba(202, 138, 4, 0.6);
  }

  .highlight-badge:disabled {
    cursor: default;
    opacity: 0.5;
  }

  .canvas-wrap {
    flex: 1;
    min-width: 0;
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
