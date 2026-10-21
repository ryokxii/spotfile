# Multiple Folders Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Index several folders side by side, added and removed independently, as specified in *Multiple folders* in `docs/superpowers/specs/2026-09-17-settings-design.md`.

**Architecture:** `settings.Folders` replaces `Folder`. Adding a folder reconciles only the entries under it, so other folders are untouched and re-adding is a catch-up. Startup reconciles every saved folder in one job and drops entries outside them. A shared `pathutil.Within` decides containment everywhere.

**Branch:** `feat/settings-impl`, after the Settings plan.

## File map

| File | Change |
|---|---|
| `engine/pathutil/pathutil.go` (new) | `Within(path, dir)` |
| `engine/vectorstore/stamps.go` | `RemoveUnder` uses `Within` |
| `engine/settings/settings.go` | `Folders`, legacy `folder` migration, `WithFolderAdded`, `WithFolderRemoved` |
| `engine/indexing/sync.go` | `StampsUnder`, `Outside` |
| `settings.go`, `indexing.go`, `app.go` | Add/remove folder, startup sync over all folders, `EngineStatus.Folders` |
| `frontend/src/stores/engine.ts` | `folders`, `removeFolder` |
| `frontend/src/lib/indexStatus.ts` | "watching N folders" |
| `frontend/src/views/settings/LibrarySection.svelte` | Folder list, Add and Remove |
| `frontend/src/views/search/SearchView.svelte` | "Add folder" |

---

### Task M1: `pathutil.Within`

- [ ] Test (table): equal path; child; grandchild; sibling sharing a prefix (`/a/bc` vs `/a/b`) is outside; trailing separators; parent is outside.
- [ ] Implement: `Within(path, dir string) bool` on cleaned paths; `vectorstore.RemoveUnder` uses it. Existing `TestRemoveUnder` still passes.
- [ ] Commit: `feat(pathutil): shared folder containment check`

### Task M2: `settings.Folders`

- [ ] Tests: legacy `{"folder":"/notes"}` loads as `Folders: ["/notes"]` and saves without `folder`; `WithFolderAdded` for new, duplicate (trailing slash), inside an existing folder (unchanged, `false`), containing existing folders (replaces them); `WithFolderRemoved`; receivers never modified; `Defaults().Folders` is empty. Comparisons use `reflect.DeepEqual` now that `Settings` holds a slice.
- [ ] Implement; remove `WithFolder`.
- [ ] Commit: `feat(settings): several indexed folders`

### Task M3: `StampsUnder`, `Outside`

- [ ] Tests: `StampsUnder(all, dir)` keeps entries within `dir`; `Outside(all, dirs)` lists sorted paths within none of `dirs`.
- [ ] Implement in `engine/indexing/sync.go`.
- [ ] Commit: `feat(indexing): scope reconciliation to a folder`

### Task M4: App

- [ ] `IndexFolder(dir)` (add): init engine; `Scan`; none → error; `WithFolderAdded` and save; `Reconcile(onDisk, StampsUnder(stamps, dir), SavedAt)`; apply `Remove`/`Adopt` (persist and `index:changed` if any); restart the watcher on all folders; enqueue `plan.Index` with `dir` in `index:start`, or release `indexMu` when empty (emit `index:done` with 0 so the UI settles).
- [ ] `RemoveFolder(dir)`: refuse while indexing; `WithFolderRemoved`, save; `RemoveUnder(dir)`, persist; restart or stop the watcher; emit `index:changed`.
- [ ] Startup `syncSavedFolders`: drop missing folders from the list (and their entries); with at least one folder, drop `Outside` entries; reconcile each folder; one combined job; watcher on all. With none: `dropMissing` as before.
- [ ] `restartWatcher(dirs []string)`; `EngineStatus.Folders []FolderStat{Path, Files}`; stale-store rebuild runs only when no folder is saved.
- [ ] `go vet`, `go build`, `wails generate module`, `go test ./engine/...`.
- [ ] Commit: `feat(app): add and remove folders independently`

### Task M5: Frontend

- [ ] Engine store tests: status carries `folders`; `index:start` with a new `dir` doesn't drop existing folders; `removeFolder` returns '' or the error message and calls the runtime.
- [ ] `indexStatus` tests: one folder → "watching Notes"; several → "watching 3 folders".
- [ ] Implement store, status line, `LibrarySection` (list, Add folder…, Remove… with `ConfirmDialog`, disabled while indexing), Search button "Add folder".
- [ ] `npm run check`, `npx vitest run`.
- [ ] Commit: `feat(ui): folder list with add and remove`

### Task M6: Docs and verification

- [ ] Dashboard spec: status line reads "Watching N folders" (or the folder name when one); `MoveToTrash` accepts a path within any saved folder; `Dashboard.Folders []FolderStat` replaces `Folder`.
- [ ] ROADMAP Settings line: several folders.
- [ ] Full run: `gofmt -l .`, `go vet . ./engine/...`, `go test -race ./engine/...`, `npm run check`, `npx vitest run`, `npm run build`.
- [ ] `wails dev` with `~/.spotfile` backed up and restored: add two folders → both indexed; re-add one → no re-embedding; remove one → its files leave search, the other stays; restart → both watched, catch-up runs.
- [ ] Commit: `docs: multiple folders in Settings and Dashboard specs`
