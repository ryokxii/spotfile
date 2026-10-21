# JSON and YAML Viewer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `docs/superpowers/specs/2026-10-21-json-yaml-viewer-design.md`.

**Architecture:** A pure tokenizer and fold finder in `lib/structured.ts`, a presentational `StructuredView` that keeps the DOM text identical to the source, and a whitespace-insensitive option for the existing search highlighter.

**Tech Stack:** Svelte 3 + TypeScript, Vitest 0.34 (jsdom). No new dependencies.

**Branch:** `feat/view-json-yaml`.

### Task 1: `lib/structured.ts`

- [ ] Write `lib/structured.test.ts` covering the cases listed under *Testing* in the spec; run it and see it fail on the missing module.
- [ ] Implement `structuredKindFor`, `formatJson`, `tokenizeJson`, `tokenizeYaml`, `jsonFolds`, `yamlFolds`, `parseStructured`.
- [ ] `npx vitest run src/lib/structured.test.ts` passes; commit `feat(viewer): tokenize and fold JSON and YAML`.

### Task 2: Whitespace-insensitive highlight

- [ ] Write `lib/textHighlight.test.ts`: a pretty-printed `<pre>` and a whitespace-free needle → `locateChunkRange` returns `null` by default and the full range with `{ ignoreWhitespace: true }`; see it fail.
- [ ] Add the option to `normalize`, `normalizeNeedle`, `locateChunkRange` and `highlightAndScroll`; export `locateChunkRange`.
- [ ] Tests pass; commit `feat(viewer): optional whitespace-insensitive search highlight`.

### Task 3: `StructuredView`

- [ ] Write `viewers/StructuredView.test.ts` (token classes, toggle, collapse/expand all, DOM text); see it fail.
- [ ] Implement `viewers/StructuredView.svelte`.
- [ ] Tests pass; commit `feat(viewer): collapsible highlighted source view`.

### Task 4: Wire into `TextViewer`

- [ ] For structured files (by extension, under 2 MB) render `StructuredView` with Collapse all / Expand all in the toolbar; expand all and highlight with `ignoreWhitespace` on search results.
- [ ] `npm run check`, `npx vitest run`, `npm run build`; commit `feat(viewer): JSON and YAML open highlighted and collapsible`.

### Task 5: Verify and document

- [ ] `wails dev` with `~/.spotfile` backed up and restored: minified JSON, `.jsonc`, nested YAML, both themes, a search result highlight.
- [ ] ROADMAP note; commit `docs: JSON and YAML viewer`.
