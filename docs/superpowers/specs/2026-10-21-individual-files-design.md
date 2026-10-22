# Individual files — design

**Date:** 2026-10-21
**Branch:** `feat/index-files` (from `develop`)
**Status:** approved design

## Goal

Index individual files as well as folders, kept across restarts and in sync with the disk like folders are.

## Behaviour

- **Library:** **Add files…** beside **Add folder…** opens the system picker (multi-select, filtered to the supported types). Added files are listed under the folders as **Individual files**, each with **Remove…** (confirmation; the file stays on disk).
- **Chat:** an **Add files** button (file icon) next to the folder button. **Dashboard** empty state: **Add files** beside **Choose a folder**.
- **Rules:** a file inside an indexed folder is already covered and isn't listed. Adding a folder that contains listed files absorbs them. Unsupported or missing files are skipped; if nothing is left, the error reads "None of the selected files can be indexed".
- **Sync:** at launch a listed file that changed is re-indexed and one that's gone is forgotten (removed from the list and the index). Entries outside every saved folder and file are dropped, as before.
- **Watcher:** each listed file is followed through its parent folder (non-recursive), so editors that save by replacing the file keep working. Events for other files in that folder are ignored; in general the watcher now only re-indexes supported files inside a saved folder or listed file.
- **Status line:** "watching Notes", "watching 2 folders", "watching Notes and 3 files", "watching 2 files".
- **Dashboard / Move to Trash:** listed files count like files in folders; Trash accepts a listed file.

## Data

- `settings.json`: `"files": ["/a/b.md"]`. `Settings.Files`, `WithFileAdded(path) (Settings, bool)`, `WithFileRemoved(path) Settings`, `Roots() []string` (folders then files); `WithFolderAdded` also drops listed files inside the new folder.
- `EngineStatus.files []string`.
- Bindings: `SelectFiles() ([]string, error)`, `AddFiles(paths []string) error`, `RemoveFile(path string) error`. The unused `IndexFiles` is removed.
- `indexing.StartWatcher(ctx, dirs, indexer, onUpdated, accept)`: `accept(path)` decides whether a settled change is re-indexed.

## Testing

- `settings`: file add/remove, covered by a folder, duplicates, folder absorbs files, `Roots`, validation, legacy files without `files`.
- `indexing`: watcher action respects `accept`.
- Frontend: engine store `files`, `chooseFiles`, `removeFile`; status line wording.
- Manual (`wails dev`, `~/.spotfile` backed up and restored): add two files; edit one while running and while closed; delete one; add a folder containing a listed file; Remove…; restart.

## Known gaps

- A listed file deleted while Spotfile runs leaves the index at once (the watcher forgets it) but stays in the Library list until the next launch, when it is forgotten from settings too.
- Chat has an **Add files** button beside the folder button rather than a menu; it does the same with less focus handling.

