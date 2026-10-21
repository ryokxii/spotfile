# Settings page — design

**Date:** 2026-09-17
**Piece:** 2 of 5 (shell → **settings** → chat engine → chat sessions → profile)
**Status:** approved design, ready for an implementation plan

## Goal

Add a Settings destination to the app shell where the user can:

1. Pick a theme: System / Nocturne / Daylight.
2. Pick reduced motion: System / On / Off.
3. See and change the single indexed folder.
4. See storage details (documents, chunks, index size on disk).
5. Clear the search index.

All choices persist across restarts.

## Non-goals

- Deleting or managing model files in `~/.spotfile/models/`.
- Multiple indexed folders.
- Engine info or editing model/server paths.
- Clearing chat history (arrives with chat sessions, piece 4).

## Decisions

| Question | Decision |
|---|---|
| What "Clear index" deletes | `store.gob` and the in-memory index only. Models, saved folder and watcher are untouched. |
| Folder model | One folder. Choosing a different folder empties the index first. |
| Persistence | One `~/.spotfile/settings.json`, owned by Go. |
| Layout | Single-column list: Appearance section, then Library section. |

## Layout

Single column, capped at a readable width, in the content area to the right of the sidebar.

```
Settings                                  ← display type, the only one on the page

APPEARANCE                                ← micro eyebrow
Theme            [System|Nocturne|Daylight]
  Nocturne at night, Daylight by day
Reduce motion    [System|On|Off]
  Turns off slides and fades
  (inline urgent error line if a save fails)

LIBRARY
Indexed folder                  [Change…]
  ~/Documents/…/Research                  ← middle-truncated, sunken well
412 documents · 9,860 chunks · 2.1 MB on disk   ← tabular numerals
Clear index                [Clear index…]  ← urgent-coloured, opens ConfirmDialog
  Search is empty until you re-index
  (inline urgent error line if clearing fails)
```

With no folder, the folder row reads "No folder yet" and the button reads "Choose…".
While indexing, "Clear index…" is disabled with the hint "Available when indexing finishes".

Pieces 3–5 add sections below Library.

## Backend (Go)

### `settings` package — `settings/settings.go`

```go
type Settings struct {
    Theme  string `json:"theme"`  // "system" | "nocturne" | "daylight"
    Motion string `json:"motion"` // "system" | "reduce" | "full"
    Folder string `json:"folder"` // absolute path, "" when none
}

func Defaults() Settings
func (s Settings) Validate() error
func Load(path string) (Settings, error)
func Save(path string, s Settings) error
```

- `Load`: a missing file returns `Defaults()` and no error. If the file is unreadable or its JSON or values are invalid, `Load` renames it to `settings.json.bad`, logs the reason, and returns `Defaults()` with no error. The app never fails to start because of settings.
- `Save`: runs `Validate()` first, then writes `path.tmp` and renames it over `path`, as `store.gob` does.
- File location: `~/.spotfile/settings.json` (`settingsPath()` beside `storePath()` in `app.go`).

### `vectorstore`

- New `func (vs *VectorStore) Clear()` empties `docs` under the write lock.

### `App`

`App` gains `settingsMu sync.Mutex` and `settings settings.Settings`, loaded in `startup` before engine init.

- `GetSettings() settings.Settings` returns a copy.
- `SetAppearance(theme, motion string) error` validates, saves, then updates the in-memory copy. On error it returns the error and the in-memory copy stays unchanged.
- `IndexFolder(dir)`: inside the job, before walking, `a.switchFolder(dir)`:
  - If `dir` equals the saved `Folder`, do nothing. Re-indexing replaces each file's chunks as it already does.
  - Otherwise: `store.Clear()`, `store.Persist()`, set and save `Folder = dir`.
  - `switchFolder` takes the store and a save function so it can be unit-tested without Wails.
  - Both Search's folder button and Settings' "Change…" go through this path.
- **Startup:** after engine init succeeds, if `Folder != ""`:
  - the folder exists → `restartWatcher(Folder)`
  - it doesn't → clear `Folder`, save, and log it.
- `ClearIndex() error`:
  - `indexMu.TryLock()` fails → return `indexing is in progress`.
  - Otherwise `store.Clear()`, then remove `store.gob` (ignore not-exist), unlock, and emit `index:cleared`.
  - If removing the file fails, memory stays cleared and the error is returned as `index cleared, but the file couldn't be removed: <err>`. The event is still emitted.
  - The watcher keeps running. A watched file edited afterwards is re-indexed as usual.
- `EngineStatus` gains:
  - `Folder string \`json:"folder"\``: the saved folder.
  - `IndexBytes int64 \`json:"indexBytes"\``: the size of `store.gob`, 0 if absent.

### Events

| Event | New? | Payload |
|---|---|---|
| `index:cleared` | new | none |

## Keeping the index in sync

*Added 2026-10-21 after a bug report: entries for files from a deleted or previously indexed folder stayed in the index, and moving a document left its old path searchable.*

**Root cause.** Entries were only ever removed for a path that was about to be indexed again (`Indexer.process`). Indexing a folder only added files, the watcher ignored deletes and renames, and startup loaded `store.gob` without checking it.

**Rule.** Compare what is on disk with what is indexed: drop entries whose file is gone, index files that are new or whose modification time or size changed.

### `vectorstore`

- `type Stamp struct { ModTime time.Time; Size int64 }`. The store keeps one per indexed path (`stamps map[string]Stamp`, saved in `storeFile.Stamps`). `formatVersion` stays 2: gob leaves the field empty when reading an older file, so existing indexes load without a rebuild.
- `SetStamp(path, Stamp)`, `AdoptStamps(map[string]Stamp)`, `Stamps() map[string]Stamp` (every indexed path; the zero `Stamp` when none is recorded), `SavedAt() time.Time` (modification time of the store file, zero when absent).
- `RemoveDocs` and `Clear` also drop stamps.
- `RemoveUnder(path string) int` removes `path` and everything below it (`path + separator` prefix, so `/a/b` never touches `/a/bc`) and returns how many documents went.

### `indexing`

- `Scan(dir string, include func(string) bool) (map[string]vectorstore.Stamp, error)`: walks `dir`, unreadable entries skipped.
- `Reconcile(onDisk, indexed map[string]vectorstore.Stamp, savedAt time.Time) SyncPlan` with `SyncPlan{Index, Remove []string; Adopt map[string]vectorstore.Stamp}`, both slices sorted:
  - indexed but not on disk → `Remove`;
  - on disk but not indexed → `Index`;
  - both, with a recorded stamp → `Index` when the modification time or size differs;
  - both, without a stamp (an index from before this change) → `Index` when the file was modified after `savedAt`, otherwise `Adopt` its current stamp so the next launch compares exactly.
- `MissingPaths(paths []string) []string`: paths that no longer exist as regular files.
- The indexer stamps each path it processes (`os.Stat` before embedding). A file that can't be stat'ed gets no stamp; its old chunks were already removed.
- `Indexer.Forget(path) int`: `store.RemoveUnder(path)`, persisting when anything was removed.
- **Watcher:** reacts to remove and rename as well as write and create, through the same 2-second debounce. When the timer fires it checks the path: a regular file is re-indexed as before; a missing path is forgotten (with everything under it, which covers a deleted folder) and `index:changed` is emitted; a directory is ignored as before. Editors that save by replacing the file therefore end with a normal re-index.

### `App`

- **Index a folder:** after the walk, entries not in the walk are removed before the job is enqueued. Together with the folder switch this keeps the index to exactly one folder.
- **Startup** (`syncSavedFolder`, replacing `resumeWatcher`), after the engine is ready:
  - no saved folder → remove entries whose files are missing (`MissingPaths`), persist if any;
  - saved folder gone → forget the folder **and** empty the index (`store.Clear`, persist);
  - otherwise, holding `indexMu`: `Scan`, `Reconcile`, apply `Remove` and `Adopt`, persist if anything changed, restart the watcher, then enqueue `Index` as an ordinary progress job (recent files high priority, the rest low). Nothing changed → no events, nothing shown.
- The three enqueue paths (folder, stale rebuild, startup catch-up) share one helper, `enqueueIndexJob(dir string, paths []string)`.
- **Automatic, no prompt.** Only new and changed files are processed, usually a few seconds; the status line reports it like any indexing run.

### Events

| Event | New? | Payload |
|---|---|---|
| `index:changed` | new | none; entries were removed outside an indexing job. The engine store refreshes `EngineStatus`. |

## Frontend (Svelte)

### Boot — `main.ts`

Before `new App(...)`:

```ts
const initial = await GetSettings().catch((err) => { console.error(err); return defaultSettings })
applyAppearance(document.documentElement, initial, matchMedia('(prefers-color-scheme: light)').matches)
```

This sets `data-theme` and `data-motion` before first paint, so the wrong theme never flashes. `initial` seeds the settings store.

### Stores

- **`stores/settings.ts`** — `createSettingsStore(initial, { setAppearance })`
  - State: `{ theme, motion, error }`. The folder isn't kept here: the engine store owns it through `EngineStatus.folder`, so it has a single source.
  - `setTheme(v)` and `setMotion(v)` update immediately, then call Go. If Go rejects, the previous value is restored and `error` is set. The next successful save clears `error`.
- **`stores/theme.ts`** — `createThemeStore(query, root, preference: Readable<ThemePreference>)`
  - `system` follows the media query as it does today.
  - `nocturne` or `daylight` pin the theme and ignore OS changes.
- **`stores/motion.ts`** — `createMotionStore(root, preference: Readable<MotionPreference>)`
  - `reduce` → `data-motion="reduce"`, `full` → `data-motion="full"`, `system` → attribute removed.
- **`stores/engine.ts`**
  - `EngineStatus` and `EngineState` gain `folder` (already in state; now also restored from the status snapshot) and `indexBytes`.
  - `index:cleared` → `documents: 0, chunks: 0, indexBytes: 0`.
  - After `index:done` and `index:cleared`, the store calls `status()` again to refresh `indexBytes`, `documents` and `chunks`.
  - New action `clearIndex()` calls `ClearIndex` and returns the error message, or `''`.
- **`stores/app.ts`** wires the new stores to the Wails bindings.

### Tokens — `styles/tokens.css`

```css
:root[data-motion='reduce'] { /* motion tokens = 0 (same list as the media query) */ }

@media (prefers-reduced-motion: reduce) {
  :root:not([data-motion='full']) { /* motion tokens = 0 */ }
}
```

`lib/motion.ts` reads the tokens, so Svelte's JS transitions follow automatically.

### Routes and components

- `routes.ts`: add `{ path: '/settings', label: 'Settings', icon: 'settings', component: SettingsView }`. Add a `settings` icon to `ui/icons.ts`.
- `views/settings/SettingsView.svelte`: the page, with a display title and sections.
- `views/settings/SettingsSection.svelte`: eyebrow plus a slot for rows.
- `views/settings/SettingRow.svelte`: label, optional hint, control slot, hairline separator.
- `views/settings/LibrarySection.svelte`: folder row, storage figures, Clear index row and confirm dialog.
- `ui/SegmentedControl.svelte`:
  - `role="radiogroup"`, with segments as `role="radio"` and `aria-checked`.
  - Tab focuses the group, and the arrow keys move and select, wrapping at the ends.
  - The selected segment sits on `surface-raised`, and the focus ring uses tokens.
- `ui/ConfirmDialog.svelte`:
  - Built on the native `<dialog>` with `showModal()`, never `window.confirm`.
  - Title, body, Cancel, and a confirm action in the `urgent` color.
  - Esc and Cancel close it without confirming. Focus starts on Cancel.
- `lib/format.ts`: `formatBytes(n)` (B / KB / MB / GB, one decimal from KB up) and `truncateMiddle(path, max)`.
- `views/search/SearchView.svelte`: the folder action label "Index another folder" becomes "Change folder".

All styling uses tokens from `tokens.css` and follows DESIGN.md: one display element, three planes, `urgent` for destructive actions only.

## Error handling

| Failure | Behaviour |
|---|---|
| `settings.json` corrupt | Renamed to `.bad`, logged, defaults used. Nothing shown in the UI. |
| `SetAppearance` fails | Control rolls back. Inline `urgent` line under Appearance: "Couldn't save — ‹reason›". |
| `GetSettings` fails at boot | Defaults applied, error logged to the console, app mounts. |
| Clear during indexing | Button disabled. If a race still reaches Go, its error shows inline under Library. |
| `store.gob` removal fails | Memory cleared. Inline error: "Index cleared, but the file couldn't be removed: ‹reason›". |
| Saved folder missing on startup | `Folder` cleared and saved, index emptied, logged. Library shows "No folder yet". |
| Catch-up scan fails (folder unreadable) | Logged; the loaded index is kept and the watcher still starts. |
| Folder switch then indexing fails | Existing `index:error` handling. The old index is already empty, as switching intends. |

## Testing

TDD: write each test first and watch it fail.

**Go**
- `settings`: missing file → defaults. Save and Load round trip. Corrupt JSON → `.bad` file plus defaults. Invalid theme or motion → `Validate` error and `Save` refuses. No `.tmp` left after `Save`.
- `vectorstore`: `Clear` empties `Len` and `DocCount`. `Save` after `Clear` loads back empty.
- `vectorstore` stamps: saved and loaded; an older file without stamps loads with none; `RemoveDocs` and `Clear` drop stamps; `RemoveUnder` removes a folder's files but not a sibling sharing its prefix.
- `indexing.Reconcile`: table tests for missing, new, changed time, changed size, unchanged, unstamped before and after `savedAt`. `Scan` and `MissingPaths` against a temp dir. `Indexer.Forget` persists only when something was removed.
- `App` helpers: `ClearIndex` returns an error while `indexMu` is held. `switchFolder` with a new folder clears and saves. With the same folder it doesn't clear or save.

**Vitest**
- `settings` store: optimistic update, rollback plus `error` on rejection, `error` cleared on the next success.
- `theme`: pinned preference ignores media-query changes. `system` follows them. Changing preference re-applies.
- `motion`: attribute set or removed for each preference.
- `tokens.test.ts`: every motion token is zeroed under `data-motion='reduce'`.
- `engine` reducer: `index:cleared` zeroes counts and bytes, and the status snapshot carries `indexBytes`.
- `SegmentedControl`: arrow keys move the selection and wrap. `aria-checked` reflects the value.
- `ConfirmDialog`: Esc and Cancel don't dispatch `confirm`. The confirm button does.
- `format`: `formatBytes` boundaries and `truncateMiddle` keeping the last path segment.

**Visual verification**
- Run `wails dev` against the real backend.
- Screenshot Settings in Nocturne and Daylight, with the wide sidebar and the compact rail.
- With Reduce motion on, search results appear without the staggered slide-in.
- Clear index → counts drop to 0 and Search shows the empty state.

## Known gaps

- **New subfolders aren't watched until the next launch.** The watcher registers the folder tree when it starts; files added to a subfolder created afterwards are picked up by the startup catch-up.

## Implementation notes

Deviations from this design, made in the implementation plan:

1. **Package location:** the settings package lives at `engine/settings`, not `settings/`. CI only runs `go test ./engine/...`, and package `main` tests would need a built `frontend/dist`.
2. **Folder-reset logic:** it lives in `settings.WithFolder` (returns the new settings plus whether the folder changed). Clearing the index file lives in `vectorstore.Wipe` / `DiskSize`. `App` stays thin.
3. **When a folder switch empties the index:** after the folder walk finds supported files. A folder with nothing to index leaves the old index alone.
4. **Boot:** `main.ts` awaits `settings.load()` and then mounts, instead of calling a separate `applyAppearance`. The theme and motion stores stamp `<html>` during App init, before first paint.
5. **Wording:** the Library figures say "files", matching the existing status line, not "documents".
6. **Index sync:** stamps are recorded after a batch is embedded (stat taken before), so a store saved mid-batch never marks an unembedded file current. The folder walk is `indexing.Scan`, which counts only regular files, so symlinked files are no longer indexed. With a saved folder, a stale store (different embedding model) is rebuilt by the startup sync rather than `rebuildStaleIndex`.
