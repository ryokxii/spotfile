<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte'
  import * as pdfjsLib from 'pdfjs-dist'
  import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
  import { ReadFileAsBase64 } from '../../wailsjs/go/main/App.js'
  import { base64ToBytes } from '../lib/base64'
  import Icon from '../ui/Icon.svelte'
  import PressableSurface from '../ui/PressableSurface.svelte'
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

  let pdfDoc: any = null

  const HIGHLIGHT_ALPHA = 0.28
  const ZOOM_STEP = 0.25
  const ZOOM_MIN = 0.5
  const ZOOM_MAX = 4

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

  // Draws primary-coloured highlight rectangles over text items that appear in
  // highlightText. Canvas can't read CSS variables, so resolve the token here.
  async function renderHighlights(page: any, viewport: any) {
    if (!hlCanvas) return

    hlCanvas.width = viewport.width
    hlCanvas.height = viewport.height
    const ctx = hlCanvas.getContext('2d')!
    ctx.clearRect(0, 0, viewport.width, viewport.height)

    const textContent = await page.getTextContent()
    const normalTarget = normalize(highlightText)

    ctx.fillStyle = getComputedStyle(document.documentElement).getPropertyValue('--color-primary').trim()
    ctx.globalAlpha = HIGHLIGHT_ALPHA

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
    {#if highlightText}
      <PressableSurface
        class="pdf-control"
        label="Jump to match"
        title={matchTop < 0 ? 'Match not found on this page' : 'Jump to the highlighted match'}
        disabled={matchTop < 0}
        on:click={scrollToMatch}
      >
        <Icon name="target" size={16} />
      </PressableSurface>
    {/if}
    <PressableSurface class="pdf-control" label="Previous page" disabled={currentPage <= 1} on:click={() => goToPage(currentPage - 1)}>
      <Icon name="chevron-left" size={16} />
    </PressableSurface>
    <span class="readout">{currentPage} / {totalPages}</span>
    <PressableSurface class="pdf-control" label="Next page" disabled={currentPage >= totalPages} on:click={() => goToPage(currentPage + 1)}>
      <Icon name="chevron-right" size={16} />
    </PressableSurface>
    <span class="divider" aria-hidden="true" />
    <PressableSurface class="pdf-control" label="Zoom out" disabled={scale <= ZOOM_MIN} on:click={() => { scale = Math.max(ZOOM_MIN, scale - ZOOM_STEP); renderPage(currentPage) }}>
      <Icon name="minus" size={16} />
    </PressableSurface>
    <span class="readout">{Math.round(scale * 100)}%</span>
    <PressableSurface class="pdf-control" label="Zoom in" disabled={scale >= ZOOM_MAX} on:click={() => { scale = Math.min(ZOOM_MAX, scale + ZOOM_STEP); renderPage(currentPage) }}>
      <Icon name="plus" size={16} />
    </PressableSurface>
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
    gap: var(--space-xxs);
    flex-shrink: 0;
  }

  :global(.pdf-control) {
    padding: var(--space-xs);
    color: var(--color-ink-secondary);
  }

  .readout {
    min-width: calc(var(--space-xxl) + var(--space-xs));
    text-align: center;
    font-size: var(--text-small-size);
    font-weight: var(--text-numeric-weight);
    font-variant-numeric: tabular-nums;
    color: var(--color-ink-faint);
  }

  .divider {
    width: 1px;
    height: var(--space-md);
    margin: 0 var(--space-xs);
    background-color: var(--color-hairline);
  }

  .canvas-wrap {
    flex: 1;
    min-width: 0;
    overflow: auto;
    display: flex;
    align-items: flex-start;
    /* No justify-content: center — it pushes a page wider than the pane past
       the left scroll edge. Auto margins centre without clipping. */
    padding: var(--space-lg);
    background-color: var(--color-surface-sunken);
  }

  /* Stack PDF canvas and highlight overlay on top of each other */
  .canvas-stack {
    position: relative;
    display: inline-block;
    line-height: 0;
    margin: 0 auto;
  }

  .pdf-canvas {
    display: block;
    border-radius: var(--radius-xs);
    border: 1px solid var(--color-hairline);
  }

  /* Overlay canvas sits exactly on top of the PDF canvas, pointer-events off */
  .hl-canvas {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none;
    border-radius: var(--radius-xs);
  }

  .loading,
  .viewer-error {
    align-self: center;
    padding: var(--space-xl);
    text-align: center;
    color: var(--color-ink-faint);
  }

  .viewer-error {
    color: var(--color-urgent);
  }
</style>
