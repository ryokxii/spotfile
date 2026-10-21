# Dashboard and Profile — design

**Date:** 2026-10-21
**Piece:** 5 of 5, redefined from "Profile" to "Dashboard and Profile". Built on `feat/settings-impl` **after** the Settings plan (`docs/superpowers/plans/2026-09-17-settings.md`) is complete; it depends on that plan's `settings.json`, folder state, `DiskSize`, `ConfirmDialog`, `SegmentedControl` and `SettingRow`.
**Status:** approved design, ready for an implementation plan

## Goal

1. A **Dashboard** becomes the launch page: search bar, library disk usage, files by type, the last updated file, recent searches, recently opened files, and a **Free up space** list of stale files.
2. A **Profile** page holding one thing: the display name.
3. The **Search** destination is renamed **Chat** and moves to `/chat`. It keeps today's search page until the chat engine (piece 3) replaces it.
4. A **Privacy** section in Settings controls the history the Dashboard records.

## Non-goals

- Accounts, sign-up, sync or any network feature. (Future: see *Roadmap changes*.)
- Avatars or images on the profile.
- Multiple chats or chat folders (piece 4).
- Deleting files permanently. Files only ever go to the system Trash.
- Usage charts over time; search counts per query.
- Catching up on edits made while the app was closed (Gap 6, unchanged).

## Decisions

| Question | Decision |
|---|---|
| Dashboard layout | Bento: greeting and search across the top, Disk tile largest, Recent files and Recent searches beside it, Free up space full width at the bottom. |
| Where Dashboard searches show results | The Chat page (`/chat`), which today is the search page. |
| Opening a recent file | A viewer panel that slides over the Dashboard; the user stays on the Dashboard. |
| "Remove" on a stale file | Primary: remove from the index only. Secondary: Move to Trash…, behind a confirmation. |
| "Stale" | Not modified **and** not opened in Spotfile for `staleDays` (30 / 90 / 180, default 90). macOS access times aren't reliable, so they are not used. |
| Profile | Display name only, stored in `settings.json`. Reached from the name at the bottom of the sidebar, not a nav item. |
| History storage | Go owns `~/.spotfile/activity.json` through a new `engine/activity` package. |
| History controls | Settings → Privacy: on/off switch, Clear history…, stale threshold. |

## Pages and navigation

| Route | Page | Sidebar |
|---|---|---|
| `/` | Dashboard | 1st item, label "Dashboard", icon `dashboard`; the logo also links here |
| `/chat` | Chat (current `SearchView`, unchanged apart from the route) | 2nd item, label "Chat", icon `chat` |
| `/settings` | Settings | 3rd item (from the Settings plan) |
| `/profile` | Profile | `ProfileButton` pinned to the sidebar bottom: initials circle + name, or "Add your name" when empty. In compact (rail) mode, initials only. |

- Dashboard search: Enter runs `search.run(query)` on the shared search store, then `push('/chat')`. `SearchView` already reads `$search.query` on mount, so the results are there on arrival.
- Clicking a recent search does the same with that query.
- `⌘K` / `Ctrl+K` on the Dashboard focuses its search bar.
- Clicking a recent file opens `ViewerPanel` over the Dashboard; `Esc` or the close button dismisses it.

## Layout

```
Good evening, Zafer                         ● Watching ~/Documents/Research
[ Search your files…                                              ⌘K ]

┌ DISK ─────────────────────────┐ ┌ RECENTLY OPENED ──────────┐
│ 2.1 MB index                  │ │ thesis-draft-v3.pdf    2h │
│ 412 files · 9,860 chunks      │ │ meeting-notes.md       1d │
│ ████████████▓▓▓▓▓░░░          │ │ budget-2026.txt       Mon │
│ ■ PDF 214 ■ Markdown 121 ■ Text 77 └───────────────────────┘
│ ┌ Last update ──────────────┐ │ ┌ RECENT SEARCHES ──────────┐
│ │ meeting-notes.md · 3m ago │ │ │ (grant deadlines) (…)     │
│ └───────────────────────────┘ │ └───────────────────────────┘
└───────────────────────────────┘
┌ FREE UP SPACE          23 files untouched for 90+ days · 310 KB of index ┐
│ old-syllabus-2024.pdf   ~48 KB · 14 months   [Remove from index] Trash… │
│ draft-old.md            ~12 KB · 7 months    [Remove from index] Trash… │
│ Show all 23 →                                                            │
└──────────────────────────────────────────────────────────────────────────┘
```

- Greeting: "Good morning / afternoon / evening, {name}" by local hour (5–12, 12–18, else evening); "Good evening" alone when the name is empty. Fraunces display type, the only display type on the page.
- Status line beside the greeting reuses the existing `indexStatus` text (watching / indexing / no folder).
- The disk figure uses tabular numerals and the Settings plan's `lib/format.ts`. File-type bar segments use `--color-identity-0…5` in a fixed type order; types beyond six are grouped as "Other".
- Free up space shows the 3 largest stale files; "Show all N →" opens `StaleFilesDialog` with the full list and the same row actions.
- Below ~900 px of content width the grid collapses to one column in the order: Disk, Recently opened, Recent searches, Free up space.
- Tiles use `--color-surface`, `--radius-md`, hairline borders; Daylight gets `--shadow-raised`. All colours come from tokens.

### Profile

```
Profile
Kept only on this computer.

DISPLAY NAME
[ Zafer Acar                       ]
Used for your initials in the sidebar and the Dashboard greeting.
```

Saves on blur or Enter. Trimmed; at most 60 characters (the input enforces `maxlength`, Go validates). Initials: first letter of the first and last words, uppercased; one letter for a single word.

### Settings → Privacy (new section after Library)

```
PRIVACY
Remember searches and opened files      [On|Off]
  Shown on the Dashboard. Stored only on this computer in ~/.spotfile
Stale after                         [30 days|90 days|180 days]
  Files not changed or opened for this long appear in Free up space
Clear history                         [Clear history…]
```

## Backend (Go)

### `engine/activity` package

```go
type Search struct { Query string    `json:"query"`; At time.Time `json:"at"` }
type Opened struct { Path  string    `json:"path"`;  At time.Time `json:"at"` }
type Update struct { Path  string    `json:"path"`;  At time.Time `json:"at"` }

type Log struct {
    Searches   []Search `json:"searches"`   // newest first, max 50
    Opened     []Opened `json:"opened"`     // newest first, max 50
    LastUpdate *Update  `json:"lastUpdate"` // nil until the first index or watcher update
}

const MaxEntries = 50

func WithSearch(l Log, query string, now time.Time) Log  // trims; ignores empty; a repeat moves to the top
func WithOpened(l Log, path string, now time.Time) Log   // one entry per path; a repeat moves to the top
func WithUpdate(l Log, path string, now time.Time) Log
func WithoutOpened(l Log, path string) Log               // "Forget" on a missing file
func Cleared(l Log) Log                                  // empties Searches and Opened, keeps LastUpdate
func LastOpened(l Log) map[string]time.Time
func Load(path string) (Log, error)
func Save(path string, l Log) error
```

- All `With*` functions return a new `Log` and never mutate their input.
- `Load` behaves like `settings.Load`: missing file → empty log; unreadable or invalid → renamed to `activity.json.bad`, logged, empty log returned. The app never fails to start because of activity.
- `Save` writes `path.tmp` and renames it over `path`.
- File: `~/.spotfile/activity.json` (`activityPath()` beside `settingsPath()`).

### `engine/activity` — staleness

```go
type StaleInput struct { Path string; Modified time.Time; Bytes int64 }
func Stale(files []StaleInput, lastOpened map[string]time.Time, days int, now time.Time) []StaleInput
```

A file is stale when both its modification time and its last open (if any) are older than `now - days`. The result is sorted by `Bytes` descending, then path. Pure; the caller supplies `now` and modification times.

### `engine/vectorstore`

```go
type DocStat struct { Path string; Chunks int; Bytes int64 }
func (vs *VectorStore) DocStats() []DocStat
```

`Bytes` is an estimate: `len(Embedding)*4 + len(Text)` summed over the file's chunks. The UI labels per-file sizes "~". The headline total stays the real `store.gob` size from `DiskSize`.

### `engine/settings` (extends the Settings plan)

```go
Name      string `json:"name"`      // trimmed, ≤ 60 runes, "" when unset
History   bool   `json:"history"`   // default true
StaleDays int    `json:"staleDays"` // 30 | 90 | 180, default 90
```

`Validate` rejects other `StaleDays` values and names over 60 runes. A `settings.json` written before these fields existed loads with the defaults for them (`history` must default to `true`, so `Load` starts from `Defaults()` before unmarshalling).

### `App`

`App` gains `activityMu sync.Mutex` and `activity activity.Log`, loaded in `startup` beside settings. Every change saves through `activity.Save` and emits `activity:changed`.

- **Recording**
  - `Search(query, topK)`: after a successful search, if `settings.History`, record the query.
  - `RecordOpened(path string) error`: new binding, called by the frontend when the viewer opens a file (Chat preview or Dashboard panel). No-op when history is off.
  - Watcher re-index finishing a path, and an index run finishing, record `LastUpdate` (the last file processed). Recorded even when history is off: it describes the library, not the user.
- `GetDashboard() (Dashboard, error)`:

  ```go
  type TypeCount struct { Type string `json:"type"`; Files int `json:"files"` }
  type StaleFile struct { Path string `json:"path"`; Bytes int64 `json:"bytes"`; Modified time.Time `json:"modified"` }
  type Dashboard struct {
      Folder         string            `json:"folder"`
      IndexBytes     int64             `json:"indexBytes"`
      Files          int               `json:"files"`
      Chunks         int               `json:"chunks"`
      Types          []TypeCount       `json:"types"`          // sorted by Files desc
      LastUpdate     *activity.Update  `json:"lastUpdate"`
      RecentSearches []activity.Search `json:"recentSearches"` // first 5
      RecentFiles    []activity.Opened `json:"recentFiles"`    // first 5
      Stale          []StaleFile       `json:"stale"`          // all, largest first
      StaleBytes     int64             `json:"staleBytes"`
      History        bool              `json:"history"`
      StaleDays      int               `json:"staleDays"`
  }
  ```

  Types come from the existing `lib/fileTypes` mapping, mirrored in Go by extension (PDF, Markdown, Text, …). Modification times come from `os.Stat`; a file that no longer exists on disk is skipped. With history off, `RecentSearches` and `RecentFiles` are empty.
- `RemoveFromIndex(path string) error`: `store.RemoveDoc(path)`, `store.Persist()`, emit `index:changed`. Refused with `indexing is in progress` while `indexMu` is held. If the file is edited later the watcher re-indexes it, which is correct: it is no longer stale.
- `MoveToTrash(path string) error`:
  - Refused unless `path` is absolute, cleaned, inside the saved folder (no `..` escape, symlinks resolved) **and** present in the index.
  - Calls a `Trasher` interface (`Trash(path string) error`). The real implementation uses the macOS Finder Trash and the Windows Recycle Bin; it never falls back to a permanent delete. If trashing fails, the error is returned and nothing else changes.
  - On success: remove from the index as above, drop the path from `Opened`, emit `index:changed`.
- `ForgetOpened(path string) error`: `activity.WithoutOpened`, save, emit `activity:changed`.
- `ClearHistory() error`: `activity.Cleared`, save, emit `activity:changed`.
- `SetProfile(name string) error` and `SetPrivacy(history bool, staleDays int) error`: validate, save settings, update the in-memory copy; on error the copy is unchanged. Turning history off does not delete existing history; Clear history does.

### Events

| Event | New? | Payload |
|---|---|---|
| `activity:changed` | new | none |
| `index:changed` | new | none (a single file was removed from the index) |

## Frontend (Svelte)

### Stores

- **`stores/dashboard.ts`**: `createDashboardStore(api, events)`. State: `{ phase: 'loading' | 'ready' | 'error', data, error }`. Loads on first subscribe and reloads on `index:done`, `watcher:done`, `index:cleared`, `index:changed` and `activity:changed`, coalescing bursts into one call. Actions `removeFromIndex(path)`, `moveToTrash(path)`, `forget(path)` return a per-row error string instead of throwing.
- **`stores/settings.ts`** (from the Settings plan) gains `name`, `history`, `staleDays`, with `setName`, `setHistory`, `setStaleDays` and the same optimistic-then-restore behaviour.
- **`stores/search.ts`**: when a preview opens, call `RecordOpened(path)`; failures are logged, never shown.

### Routes and components

- `routes.ts`: destinations become Dashboard (`/`), Chat (`/chat`, `SearchView`), Settings (`/settings`). `/profile` is a route without a sidebar entry.
- `ui/icons.ts`: add `dashboard` and `chat` icons.
- `ui/ProfileButton.svelte`: bottom of `Sidebar`.
- `lib/greeting.ts`: `greeting(hour, name)` and `initials(name)`, pure.
- `views/dashboard/`: `DashboardView`, `DashboardSearch`, `DiskTile`, `RecentFilesTile`, `RecentSearchesTile`, `FreeSpaceTile`, `StaleFileRow`, `StaleFilesDialog`, `ViewerPanel` (wraps the existing `ViewerShell`).
- `views/profile/ProfileView.svelte`.
- `views/settings/PrivacySection.svelte`, added to `SettingsView` after Library.

Motion: tiles rise in with `--motion-stagger`; the viewer panel slides with `--motion-normal` / `--ease-enter`. Only `transform` and `opacity` animate, and the Settings plan's reduce-motion override applies.

## Error handling

| Situation | What the user sees |
|---|---|
| `GetDashboard` fails | Tiles replaced by one urgent line "Couldn't load your library figures" and **Retry**. Search bar still works. |
| Remove from index fails | Urgent line under that row; the row stays. |
| Remove while indexing | Row actions disabled with the hint "Available when indexing finishes". |
| Move to Trash | `ConfirmDialog`: "Move *{name}* to the Trash? You can restore it from the Trash." Destructive button "Move to Trash". |
| Trash fails | Urgent line under the row: "Couldn't move to Trash. Nothing was deleted." |
| Trash path refused (outside folder / not indexed) | Same urgent line; the Go error is logged. |
| Recent file missing | The viewer's existing "can't open" state; the row offers **Forget**. |
| Saving name / privacy fails | Inline urgent line in that section; previous value restored. |
| No folder | Dashboard shows "Choose a folder to index" with a button to Settings; search bar disabled with the existing placeholder. |
| No history yet / history off | Recent tiles: "Your searches will appear here" / "History is off · Turn on in Settings". |
| Nothing stale | "Nothing stale. Everything was used in the last {staleDays} days." |

## Testing

TDD throughout; 80% coverage on new code.

**Go (`go test ./engine/...`)**
- `activity`: caps at 50; repeats move to top; empty queries ignored; `Cleared` keeps `LastUpdate`; inputs never mutated; `Load` missing/invalid file paths; atomic `Save`.
- `activity.Stale`: table tests with an injected `now` covering modified-only, opened-recently, both old, missing open entry, ordering.
- `vectorstore.DocStats`: per-file chunk counts and byte estimates.
- `settings`: new field defaults, validation, old files without the fields.
- Trash path checks extracted to a pure function (`trashAllowed(path, folder string, indexed func(string) bool) error`) and tested for `..`, symlink escape, outside folder, not indexed; `MoveToTrash` tested with a fake `Trasher`.

**Frontend (Vitest)**
- `greeting` and `initials`.
- Dashboard store: initial load, reload on each event, burst coalescing, per-row action errors.
- `DashboardView`: loading, error with Retry, no folder, populated.
- Dashboard search: Enter runs the query and navigates to `/chat`.
- Recent file click opens the panel and calls `RecordOpened`; `Esc` closes it.
- Stale rows: Remove calls the API; Trash opens the confirm dialog and only calls the API on confirm.
- `ProfileView`: trimming, 60-character limit, save on blur and Enter, restore on failure.
- `PrivacySection`: toggles and Clear history confirmation.
- Sidebar: Dashboard and Chat entries, `ProfileButton` with and without a name.

**Manual (`wails dev`)**: both themes, a narrow window (one-column collapse), reduce motion on, moving a real file to the Trash and restoring it.

## Roadmap changes

- Piece 5 "Profile" → **Dashboard and Profile** (this spec).
- Piece 4 gains **multiple chats and folders of chats**.
- New future item: **Account sign-up with opt-in usage metrics** — the app's only online feature, off unless the user chooses to share, with full transparency about what is sent.

## Known gaps

- Per-file index sizes are estimates; they won't sum exactly to the `store.gob` size.
- A file removed from the index comes back on a full re-index of the folder. No exclusion list is kept (intentional for now).
- Staleness ignores opens in other apps, since macOS doesn't reliably record access times.
