# Individual Files Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `docs/superpowers/specs/2026-10-21-individual-files-design.md`.

**Architecture:** Listed files become a second kind of saved root next to folders. Everything that scoped work to folders (startup sync, the outside-roots prune, the watcher, Trash checks) takes `Settings.Roots()` instead.

**Tech Stack:** Go 1.26, Wails v2.12, Svelte 3 + TypeScript, Vitest 0.34.

### Task 1: `settings.Files`
- [ ] Tests (see spec *Testing*), see them fail; implement `Files`, `WithFileAdded`, `WithFileRemoved`, `Roots`, folder absorption; pass; commit `feat(settings): individual indexed files`.

### Task 2: Watcher `accept`
- [ ] Test that a regular file `accept` rejects is ignored; implement the parameter; pass; commit `feat(watcher): only re-index files inside saved roots`.

### Task 3: App
- [ ] `SelectFiles`, `AddFiles`, `RemoveFile`; startup sync over files; watcher on file parents; `EngineStatus.Files`; Trash and pruning use `Roots()`; remove `IndexFiles`; build, vet, regenerate bindings, Go tests; commit `feat(app): add and remove individual files`.

### Task 4: Frontend
- [ ] Engine store and status-line tests, see them fail; implement store, status line, Library list and buttons, Chat button, Dashboard empty state; check and test; commit `feat(ui): add individual files from Library, Chat and the Dashboard`.

### Task 5: Verify and document
- [ ] Manual run per spec; ROADMAP; commit `docs: individual files`.
