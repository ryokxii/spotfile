package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			return nil
		}
		if isIndexable(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no supported text or PDF files found in %s", filepath.Base(dir))
	}
	log.Printf("index: starting folder=%q files=%d", dir, len(paths))

	wailsruntime.EventsEmit(a.ctx, "index:start", map[string]any{
		"total": len(paths),
		"dir":   dir,
	})

	// Recent files index at high priority (searchable first); history backfills
	// in the background. Unreadable paths are dropped here, so count over the
	// exact set that will be indexed to keep the progress percentage accurate.
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

	// Start the watcher before enqueuing so edits during the scan queue as urgent.
	a.restartWatcher(dir)

	items := make([]indexing.PathPriority, 0, len(indexed))
	for _, p := range recent {
		items = append(items, indexing.PathPriority{Path: p, Priority: indexing.PriorityHigh})
	}
	for _, p := range older {
		items = append(items, indexing.PathPriority{Path: p, Priority: indexing.PriorityLow})
	}

	a.indexer.Enqueue(items, indexing.Hooks{
		OnChunk: func(chunk vectorstore.EmbeddedChunk, jobTotal int) {
			wailsruntime.EventsEmit(a.ctx, "index:chunk", map[string]any{
				"total": jobTotal,
				"path":  chunk.DocPath,
			})
		},
		OnDone: func(jobTotal int) {
			wailsruntime.EventsEmit(a.ctx, "index:done", jobTotal)
			log.Printf("index: complete folder=%q chunks=%d files=%d", dir, jobTotal, len(indexed))
			a.indexMu.Unlock()
		},
	})

	return nil
}

// restartWatcher (re)starts the recursive file watcher rooted at dir, wired to
// the shared indexer so changes re-index at urgent priority.
func (a *App) restartWatcher(dir string) {
	if a.watcher != nil {
		_ = a.watcher.Stop()
		a.watcher = nil
	}
	watcher, err := indexing.StartWatcher(a.ctx, collectWatchDirs(dir), a.indexer)
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
	a.indexer.Enqueue(items, indexing.Hooks{
		OnChunk: func(chunk vectorstore.EmbeddedChunk, jobTotal int) {
			wailsruntime.EventsEmit(a.ctx, "index:chunk", map[string]any{
				"total": jobTotal,
				"path":  chunk.DocPath,
			})
		},
		OnDone: func(jobTotal int) {
			wailsruntime.EventsEmit(a.ctx, "index:done", jobTotal)
			log.Printf("index: complete chunks=%d files=%d", jobTotal, len(paths))
		},
	})
	return nil
}
