package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"spotfile/engine/embedder"
	"spotfile/engine/indexing"
	"spotfile/engine/vectorstore"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// IndexFolder starts a background job that recursively walks dir, indexes all
// supported files, and (re)starts the file watcher on the directory tree. It
// returns immediately so Wails can deliver progress events while the indexing
// work continues.
// Emits: "index:start" (with total file count), "index:chunk", "index:done",
// "index:error", and "watcher:started" once the watcher is running.
func (a *App) IndexFolder(dir string) error {
	if !a.indexMu.TryLock() {
		return fmt.Errorf("indexing is already in progress")
	}

	go func() {
		if err := a.indexFolder(dir); err != nil {
			a.indexMu.Unlock() // enqueue never happened; release now
			log.Printf("index: failed folder=%q: %v", dir, err)
			wailsruntime.EventsEmit(a.ctx, "index:error", err.Error())
		}
		// On success the lock is released by the enqueued job's OnDone hook.
	}()
	return nil
}

func (a *App) indexFolder(dir string) error {
	// If startup's init goroutine is still running, this blocks until it
	// finishes. If it failed or hasn't started yet, we init here instead.
	a.engMu.Lock()
	if a.embedder == nil {
		if err := a.initEngineLocked(EngineConfig{}); err != nil {
			a.engMu.Unlock()
			return fmt.Errorf("engine init: %w", err)
		}
	}
	a.engMu.Unlock()

	onDisk, err := indexing.Scan(dir, isIndexable)
	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}
	if len(onDisk) == 0 {
		return fmt.Errorf("no supported text or PDF files found in %s", filepath.Base(dir))
	}
	// Only a folder with something to index is added. Other folders stay as
	// they are; within this one, only new and changed files are indexed.
	if err := a.addFolder(dir); err != nil {
		return err
	}
	paths := a.reconcileFolder(dir, onDisk)
	a.watchFolders()
	if len(paths) == 0 {
		a.indexMu.Unlock() // nothing new: no job will release it
		return nil
	}
	log.Printf("index: starting folder=%q files=%d", dir, len(paths))
	return a.enqueueIndexJob(dir, paths)
}

// enqueueIndexJob indexes paths as one progress-reporting job and releases
// indexMu when it finishes; the caller must hold indexMu. Recent files index at
// high priority (searchable first); history backfills in the background. dir
// names the folder in "index:start" ("" for several).
// Emits: "index:start", "index:prepared", then the job's "index:chunk" and
// "index:done".
func (a *App) enqueueIndexJob(dir string, paths []string) error {
	wailsruntime.EventsEmit(a.ctx, "index:start", map[string]any{
		"total": len(paths),
		"dir":   dir,
	})

	// Unreadable paths are dropped here, so count over the exact set that will
	// be indexed to keep the progress percentage accurate.
	recent, older := indexing.SplitByModTime(paths, recentWindow)
	indexed := append(append([]string{}, recent...), older...)
	totalChunks, err := indexing.CountChunks(a.ctx, indexed)
	if err != nil {
		return fmt.Errorf("count chunks: %w", err)
	}
	log.Printf("index: prepared folder=%q chunks=%d recent=%d history=%d", dir, totalChunks, len(recent), len(older))
	wailsruntime.EventsEmit(a.ctx, "index:prepared", map[string]any{
		"totalChunks": totalChunks,
	})

	items := make([]indexing.PathPriority, 0, len(indexed))
	for _, p := range recent {
		items = append(items, indexing.PathPriority{Path: p, Priority: indexing.PriorityHigh})
	}
	for _, p := range older {
		items = append(items, indexing.PathPriority{Path: p, Priority: indexing.PriorityLow})
	}

	a.indexer.Enqueue(items, a.progressHooks(func(jobTotal int) {
		log.Printf("index: complete folder=%q chunks=%d files=%d", dir, jobTotal, len(indexed))
		a.indexMu.Unlock()
	}))
	return nil
}

// rebuildStaleIndex re-indexes documents from a store built by an older format
// or embedding model, reporting progress like a folder index. Files that no
// longer exist are dropped.
func (a *App) rebuildStaleIndex(paths []string) {
	gone := make(map[string]bool)
	for _, p := range indexing.MissingPaths(paths) {
		gone[p] = true
	}
	existing := make([]string, 0, len(paths))
	for _, p := range paths {
		if !gone[p] {
			existing = append(existing, p)
		}
	}
	if len(existing) == 0 {
		return
	}
	if !a.indexMu.TryLock() {
		log.Printf("index: skip rebuild of %d documents: indexing already in progress", len(existing))
		return
	}
	log.Printf("index: rebuilding %d documents for embedding model %s", len(existing), embedder.ID)
	if err := a.enqueueIndexJob("", existing); err != nil {
		a.indexMu.Unlock()
		log.Printf("index: rebuild aborted: %v", err)
		wailsruntime.EventsEmit(a.ctx, "index:error", err.Error())
	}
}

// keys returns the paths of a stamp map, sorted.
func keys(m map[string]vectorstore.Stamp) []string {
	out := make([]string, 0, len(m))
	for p := range m {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// progressHooks emits index:chunk per stored chunk and index:done at the end,
// records the job's last file as the latest update, then runs onDone.
func (a *App) progressHooks(onDone func(jobTotal int)) indexing.Hooks {
	var last string // hooks run one at a time on the indexer's worker
	return indexing.Hooks{
		OnChunk: func(chunk vectorstore.EmbeddedChunk, jobTotal int) {
			last = chunk.DocPath
			wailsruntime.EventsEmit(a.ctx, "index:chunk", map[string]any{
				"total": jobTotal,
				"path":  chunk.DocPath,
			})
		},
		OnDone: func(jobTotal int) {
			wailsruntime.EventsEmit(a.ctx, "index:done", jobTotal)
			if last != "" {
				a.recordUpdate(last)
			}
			onDone(jobTotal)
		},
	}
}

// restartWatcher (re)starts one recursive file watcher over the given folders,
// wired to the shared indexer so changes re-index at urgent priority. With no
// folders it only stops the current watcher.
func (a *App) restartWatcher(folders []string) {
	if a.watcher != nil {
		_ = a.watcher.Stop()
		a.watcher = nil
	}
	if len(folders) == 0 {
		return
	}
	var dirs []string
	for _, f := range folders {
		dirs = append(dirs, collectWatchDirs(f)...)
	}
	watcher, err := indexing.StartWatcher(a.ctx, dirs, a.indexer, a.recordUpdate)
	if err != nil {
		log.Printf("warning: failed to start watcher: %v", err)
		return
	}
	a.watcher = watcher
	wailsruntime.EventsEmit(a.ctx, "watcher:started", nil)
}

// collectWatchDirs returns root and all its subdirectories for recursive watching.
func collectWatchDirs(root string) []string {
	var dirs []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs
}

// IndexFiles enqueues an explicit set of files at high priority. De-duplication,
// persistence, and progress events are handled by the shared indexer.
func (a *App) IndexFiles(paths []string) error {
	if a.indexer == nil {
		return fmt.Errorf("engine not initialised — call InitEngine first")
	}
	if len(paths) == 0 {
		return nil
	}
	log.Printf("index: enqueue files=%d", len(paths))
	items := make([]indexing.PathPriority, len(paths))
	for i, p := range paths {
		items[i] = indexing.PathPriority{Path: p, Priority: indexing.PriorityHigh}
	}
	a.indexer.Enqueue(items, a.progressHooks(func(jobTotal int) {
		log.Printf("index: complete chunks=%d files=%d", jobTotal, len(paths))
	}))
	return nil
}
