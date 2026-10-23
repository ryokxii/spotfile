# Readable, collapsible JSON and YAML — design

**Date:** 2026-10-21
**Branch:** `feat/view-json-yaml` (from `feat/settings-impl`)
**Status:** approved design

## Goal

`.json`, `.jsonc`, `.yaml` and `.yml` files open in the viewer (Chat preview and the Dashboard panel) as highlighted, collapsible source instead of raw text.

## Behaviour

- **Highlighting** with theme tokens: keys `--color-secondary`, strings `--color-identity-1`, numbers `--color-primary`, `true`/`false`/`null` (and YAML `yes`/`no`/`~`) `--color-identity-4`, punctuation and comments `--color-ink-faint` (comments italic). Works in Nocturne and Daylight.
- **Line numbers** in a gutter.
- **Folding:** every multi-line JSON object/array and every YAML block (a line followed by more-indented lines, or a `key:` followed by `- ` items at the same indent) gets a ▸/▾ toggle. Folded, the first line shows a summary — `{ … 12 keys }`, `[ … 5 items ]` (JSON, followed by the closing line's text) or `… 8 lines` (YAML). The viewer header has **Collapse all** and **Expand all**. Everything starts expanded.
- **Formatting:** a JSON file that is a single line and parses is pretty-printed with 2 spaces. Anything else (already formatted, comments, invalid) is shown exactly as written; invalid JSON still highlights and folds as far as its brackets allow.
- **Search highlight:** opening a result expands every fold first, then highlights. For these files the match ignores whitespace, because reformatting adds spaces that the indexed text doesn't have. Other file types match as before.
- **Size cap:** files over 2 MB open as plain text.

## Design

- `lib/structured.ts` (pure, unit-tested): `structuredKindFor(path)`, `formatJson(source)`, `tokenizeJson(source)`, `tokenizeYaml(source)` (both return tokens per line), `jsonFolds(lines)`, `yamlFolds(lines)`, `parseStructured(kind, source)`.
- `viewers/StructuredView.svelte`: renders lines, gutter and toggles; owns the folded set; exposes `expandAll()` and `collapseAll()`. Line numbers are CSS `::before` content and the toggles carry no text, so the DOM text is exactly the source text.
- `viewers/TextViewer.svelte`: for structured files renders `StructuredView`, puts Collapse all / Expand all in the header toolbar, and before highlighting calls `expandAll()` and highlights with `{ ignoreWhitespace: true }`.
- `lib/textHighlight.ts`: `highlightAndScroll` gains an optional `{ ignoreWhitespace }`; `locateChunkRange` is exported for tests.
- No new dependencies.

## Non-goals

Editing, find-in-file, a tree view, TOML/XML, remembering folds between openings.

## Testing

- `structured`: kinds by extension; formatting rules; token kinds (key vs string, numbers, literals, punctuation, `//` and multi-line `/* */` comments; YAML keys, quoted values containing `#`, comments, list dashes, block scalars); fold ranges and labels (nested, empty, same-line brackets, YAML lists under keys, blank lines, comment lines).
- `textHighlight`: a whitespace-free needle finds pretty-printed text only with `ignoreWhitespace`.
- `StructuredView`: token classes; toggling hides inner lines and shows the summary; Collapse all / Expand all; the DOM text has no line numbers or summaries when expanded.
- Manual: a minified JSON, a commented `.jsonc`, a nested YAML, in both themes; open from a search result and check the highlight.
