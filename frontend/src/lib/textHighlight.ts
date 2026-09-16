// Locates a matched search chunk inside a rendered element and highlights it.
//
// The chunk text (`result.text`) is a whitespace-normalized window of words
// (the Go chunker splits on `strings.Fields` and rejoins with single spaces),
// so it is never a verbatim substring of the source. Matching therefore has to
// be whitespace-tolerant: we build a collapsed-whitespace projection of the
// element's text, search within that, then map the hit back to real DOM offsets.

const HIGHLIGHT_NAME = 'spotfile-match'
const HIGHLIGHT_STYLE_ID = 'spotfile-highlight-style'
const FALLBACK_MARK_CLASS = 'spotfile-match-mark'

interface NodeSpan {
  node: Text
  start: number // inclusive index of this node's first char within `full`
  end: number // exclusive index of this node's last char within `full`
}

interface Normalized {
  norm: string // whitespace-collapsed, lowercased projection of `full`
  map: number[] // norm index -> index within `full`
}

// Block-level tags whose boundaries imply whitespace in the rendered layout.
// The chunk text carries a space at every such boundary (Go's strings.Fields
// dropped the source newline), but the DOM concatenates block text nodes with
// no separator — so we inject one to keep the projection aligned. Inline tags
// (<strong>, <a>, <code>…) are intentionally absent: "wor<strong>d</strong>"
// must stay "word".
const BLOCK_TAGS = new Set([
  'ADDRESS', 'ARTICLE', 'ASIDE', 'BLOCKQUOTE', 'DD', 'DETAILS', 'DIV', 'DL', 'DT',
  'FIGCAPTION', 'FIGURE', 'FOOTER', 'FORM', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6',
  'HEADER', 'HR', 'LI', 'MAIN', 'NAV', 'OL', 'P', 'PRE', 'SECTION', 'TABLE',
  'TBODY', 'TD', 'TH', 'THEAD', 'TR', 'UL',
])

// Walk the DOM in document order, appending each text node's data to `full` and
// inserting a '\n' separator across block boundaries. `full` (separators
// included) is returned so the caller normalizes and maps against the same
// string the span offsets were built from.
function collectTextNodes(root: HTMLElement): { full: string; spans: NodeSpan[] } {
  const spans: NodeSpan[] = []
  let full = ''
  let pendingSeparator = false

  function walk(node: Node): void {
    if (node.nodeType === Node.TEXT_NODE) {
      const textNode = node as Text
      if (textNode.data.length === 0) return
      if (pendingSeparator && full.length > 0) {
        full += '\n'
        pendingSeparator = false
      }
      const start = full.length
      full += textNode.data
      spans.push({ node: textNode, start, end: full.length })
      return
    }
    if (node.nodeType !== Node.ELEMENT_NODE) return
    const element = node as Element
    const isBlock = BLOCK_TAGS.has(element.tagName) || element.tagName === 'BR'
    if (isBlock) pendingSeparator = true
    node.childNodes.forEach(walk)
    if (isBlock) pendingSeparator = true
  }

  walk(root)
  return { full, spans }
}

// Collapse every run of whitespace to a single space and lowercase, recording
// the source index of each surviving character so hits can be mapped back.
function normalize(full: string): Normalized {
  const out: string[] = []
  const map: number[] = []
  let inWhitespace = false
  for (let i = 0; i < full.length; i++) {
    const ch = full[i]
    if (/\s/.test(ch)) {
      if (!inWhitespace) {
        out.push(' ')
        map.push(i)
        inWhitespace = true
      }
    } else {
      out.push(ch.toLowerCase())
      map.push(i)
      inWhitespace = false
    }
  }
  return { norm: out.join(''), map }
}

function normalizeNeedle(needle: string): string {
  return needle.replace(/\s+/g, ' ').trim().toLowerCase()
}

// Truncate a normalized needle to at most `maxChars`, cutting on a word boundary.
function clampToWord(needle: string, maxChars: number): string {
  if (needle.length <= maxChars) return needle
  const cut = needle.lastIndexOf(' ', maxChars)
  return needle.slice(0, cut > 0 ? cut : maxChars)
}

// Full match first; fall back to progressively shorter prefixes so a partial
// mismatch (e.g. a stray rendered character) still scrolls near the region.
function findNormIndex(norm: string, needle: string): { index: number; length: number } | null {
  const candidates = [needle, clampToWord(needle, 300), clampToWord(needle, 120)]
  for (const candidate of candidates) {
    if (candidate.length < 4) continue
    const index = norm.indexOf(candidate)
    if (index !== -1) return { index, length: candidate.length }
  }
  return null
}

function locateInSpan(spans: NodeSpan[], fullIndex: number): { node: Text; offset: number } {
  for (const span of spans) {
    if (fullIndex >= span.start && fullIndex < span.end) {
      return { node: span.node, offset: fullIndex - span.start }
    }
  }
  const last = spans[spans.length - 1]
  return { node: last.node, offset: last.node.data.length }
}

function locateChunkRange(root: HTMLElement, needle: string): Range | null {
  const needleNorm = normalizeNeedle(needle)
  if (needleNorm.length < 4) return null

  const { full, spans } = collectTextNodes(root)
  if (spans.length === 0) return null

  const { norm, map } = normalize(full)

  const hit = findNormIndex(norm, needleNorm)
  if (!hit) return null

  const fullStart = map[hit.index]
  const fullEnd = map[hit.index + hit.length - 1] + 1

  const start = locateInSpan(spans, fullStart)
  const end = locateInSpan(spans, fullEnd)

  const range = document.createRange()
  range.setStart(start.node, Math.min(start.offset, start.node.data.length))
  range.setEnd(end.node, Math.min(end.offset, end.node.data.length))
  return range
}

function ensureHighlightStyle(): void {
  if (document.getElementById(HIGHLIGHT_STYLE_ID)) return
  const style = document.createElement('style')
  style.id = HIGHLIGHT_STYLE_ID
  style.textContent =
    `::highlight(${HIGHLIGHT_NAME}) { background: var(--color-primary-wash-strong); color: var(--color-ink); }` +
    `mark.${FALLBACK_MARK_CLASS} { background: var(--color-primary-wash-strong); color: var(--color-ink); border-radius: var(--radius-xs); }`
  document.head.appendChild(style)
}

function supportsHighlightApi(): boolean {
  return typeof CSS !== 'undefined' && 'highlights' in CSS && typeof (window as any).Highlight === 'function'
}

function applyHighlight(range: Range): void {
  if (supportsHighlightApi()) {
    ensureHighlightStyle()
    const highlight = new (window as any).Highlight(range)
    ;(CSS as any).highlights.set(HIGHLIGHT_NAME, highlight)
    return
  }
  // Fallback for older WebViews: DOM-wrap only works within a single text node.
  if (range.startContainer === range.endContainer && range.startContainer.nodeType === Node.TEXT_NODE) {
    ensureHighlightStyle()
    const mark = document.createElement('mark')
    mark.className = FALLBACK_MARK_CLASS
    try {
      range.surroundContents(mark)
    } catch {
      /* cross-boundary range — skip the visual highlight, scrolling still works */
    }
  }
}

function scrollRangeIntoView(range: Range, scroller: HTMLElement): void {
  const rects = range.getClientRects()
  const rect = rects.length > 0 ? rects[0] : range.getBoundingClientRect()
  const scrollerRect = scroller.getBoundingClientRect()
  const delta = rect.top - scrollerRect.top - scroller.clientHeight / 2 + rect.height / 2
  scroller.scrollTo({ top: scroller.scrollTop + delta, behavior: 'smooth' })
}

// Remove any active highlight so switching results/files doesn't leave stragglers.
export function clearHighlight(): void {
  if (supportsHighlightApi()) {
    ;(CSS as any).highlights.delete(HIGHLIGHT_NAME)
  }
  document.querySelectorAll<HTMLElement>(`mark.${FALLBACK_MARK_CLASS}`).forEach((mark) => {
    const parent = mark.parentNode
    if (!parent) return
    while (mark.firstChild) parent.insertBefore(mark.firstChild, mark)
    parent.removeChild(mark)
    if (parent instanceof Element) parent.normalize()
  })
}

// Highlight `needle` within `root` and scroll `scroller` to center it.
// Returns whether a match was found. Any previous highlight is cleared first.
export function highlightAndScroll(root: HTMLElement, scroller: HTMLElement, needle: string): boolean {
  clearHighlight()
  if (!needle || !needle.trim()) return false

  const range = locateChunkRange(root, needle)
  if (!range) return false

  applyHighlight(range)
  scrollRangeIntoView(range, scroller)
  return true
}
