<script lang="ts">
  // Highlighted, collapsible JSON or YAML. Line numbers come from CSS and the
  // fold toggles carry no text, so the rendered text is exactly the source and
  // search highlighting can find chunks in it. Folded blocks are not rendered;
  // callers expand everything before highlighting.
  import { parseStructured, type Fold, type StructuredKind, type Token } from '../lib/structured'

  export let kind: StructuredKind
  export let source: string

  interface Row {
    index: number
    tokens: Token[]
    fold: Fold | undefined
    collapsed: boolean
    summary: string // shown after a collapsed line, '' otherwise
    closing: string // a collapsed JSON block's closing line, e.g. "},"
  }

  let collapsed = new Set<number>() // start lines of folded blocks

  $: parsed = parseStructured(kind, source)
  $: foldAt = new Map(parsed.folds.map((f) => [f.start, f]))
  $: parsed, (collapsed = new Set()) // a new file opens fully expanded
  $: rows = visibleRows(parsed.lines, foldAt, collapsed)

  export function expandAll(): void {
    collapsed = new Set()
  }

  export function collapseAll(): void {
    collapsed = new Set(parsed.folds.map((f) => f.start))
  }

  function toggle(start: number): void {
    const next = new Set(collapsed)
    if (next.has(start)) next.delete(start)
    else next.add(start)
    collapsed = next
  }

  const lineText = (tokens: Token[]): string => tokens.map((t) => t.text).join('')

  function visibleRows(lines: Token[][], folds: Map<number, Fold>, folded: Set<number>): Row[] {
    const out: Row[] = []
    for (let index = 0; index < lines.length; index++) {
      const fold = folds.get(index)
      const isCollapsed = fold !== undefined && folded.has(index)
      const closing = isCollapsed && kind === 'json' ? lineText(lines[fold.end]).trim() : ''
      const summary = isCollapsed ? ` … ${fold.label}${closing ? ' ' : ''}` : ''
      out.push({ index, tokens: lines[index], fold, collapsed: isCollapsed, summary, closing })
      if (isCollapsed) index = fold.end
    }
    return out
  }
</script>

<!-- No whitespace between tags: it would become text and shift search matches. -->
<div class="structured">{#each rows as row (row.index)}<div class="line" data-line={row.index + 1}>{#if row.fold}<button
          type="button"
          class="toggle"
          aria-expanded={!row.collapsed}
          aria-label="{row.collapsed ? 'Expand' : 'Collapse'} {row.fold.label}"
          title="{row.collapsed ? 'Expand' : 'Collapse'} {row.fold.label}"
          on:click={() => toggle(row.index)}
        />{/if}<span class="code">{#each row.tokens as token}<span class="tok-{token.kind}">{token.text}</span>{/each}{#if row.collapsed}<span class="summary">{row.summary}</span><span class="tok-punct">{row.closing}</span>{/if}</span></div>{/each}</div>

<style>
  .structured {
    --gutter: 52px;
    font-family: var(--font-code);
    font-size: var(--text-small-size);
    line-height: var(--text-body-line);
    color: var(--color-ink);
  }

  .line {
    position: relative;
    display: flex;
    min-height: calc(var(--text-body-line) * 1em);
  }

  /* Line numbers live in CSS so they are never part of the text. */
  .line::before {
    content: attr(data-line);
    flex: 0 0 var(--gutter);
    padding-right: var(--space-lg);
    text-align: right;
    color: var(--color-ink-faint);
    opacity: 0.6;
    font-variant-numeric: tabular-nums;
    user-select: none;
  }

  .code {
    flex: 1;
    min-width: 0;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .toggle {
    position: absolute;
    left: calc(var(--gutter) - var(--space-lg) + 2px);
    top: 0;
    width: 16px;
    height: calc(var(--text-body-line) * 1em);
    display: grid;
    place-items: center;
    border-radius: var(--radius-xs);
    color: var(--color-ink-faint);
    transition: color var(--motion-quick) var(--ease-standard);
  }

  /* A chevron drawn with borders: ▾ when expanded, ▸ when folded. */
  .toggle::before {
    content: '';
    width: 6px;
    height: 6px;
    border-right: 1.5px solid currentColor;
    border-bottom: 1.5px solid currentColor;
    transform: translateY(-2px) rotate(45deg);
    transition: transform var(--motion-quick) var(--ease-standard);
  }

  .toggle[aria-expanded='false']::before {
    transform: translateX(-2px) rotate(-45deg);
  }

  .toggle:hover,
  .toggle:focus-visible {
    color: var(--color-primary);
  }

  .summary {
    margin: 0 var(--space-xxs);
    padding: 0 var(--space-xs);
    border-radius: var(--radius-xs);
    background-color: var(--color-secondary-wash);
    color: var(--color-secondary);
  }

  .tok-key {
    color: var(--color-secondary);
  }

  .tok-string {
    color: var(--color-identity-1);
  }

  .tok-number {
    color: var(--color-primary);
  }

  .tok-literal {
    color: var(--color-identity-4);
  }

  .tok-punct {
    color: var(--color-ink-faint);
  }

  .tok-comment {
    color: var(--color-ink-faint);
    font-style: italic;
  }
</style>
