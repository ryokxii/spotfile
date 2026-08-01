package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// FileWatcher monitors filesystem for changes and triggers re-indexing.
type FileWatcher struct {
	watcher      *fsnotify.Watcher
	debounceMap  map[string]*time.Timer
	debounceTime time.Duration
	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	indexer      *Indexer
	closed       chan struct{}
}

// StartWatcher begins watching for filesystem changes on the given paths. Changed
// files are re-indexed through the shared indexer at urgent priority.
func StartWatcher(ctx context.Context, paths []string, indexer *Indexer) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	// Add all paths to the watcher
	for _, p := range paths {
		if err := watcher.Add(p); err != nil {
			watcher.Close()
			return nil, fmt.Errorf("add path %s to watcher: %w", p, err)
		}
	}

	watchCtx, cancel := context.WithCancel(ctx)
	fw := &FileWatcher{
		watcher:      watcher,
		debounceMap:  make(map[string]*time.Timer),
		debounceTime: 2 * time.Second,
		ctx:          watchCtx,
		cancel:       cancel,
		indexer:      indexer,
		closed:       make(chan struct{}),
	}

	// Start event loop
	fw.wg.Add(1)
	go fw.eventLoop()

	return fw, nil
}

// eventLoop handles fsnotify events and debounces re-indexing.
func (fw *FileWatcher) eventLoop() {
	defer fw.wg.Done()
	defer close(fw.closed)

	for {
		select {
		case <-fw.ctx.Done():
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

// handleEvent processes a filesystem event with debouncing.
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Only care about write and create events
	if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Cancel existing timer for this path
	if timer, exists := fw.debounceMap[event.Name]; exists {
		timer.Stop()
	}

	// Set new debounce timer
	fw.debounceMap[event.Name] = time.AfterFunc(fw.debounceTime, func() {
		fw.reindexPath(event.Name)
		fw.mu.Lock()
		delete(fw.debounceMap, event.Name)
		fw.mu.Unlock()
	})
}

// reindexPath enqueues a changed file at urgent priority so it jumps ahead of any
// background history scan. The indexer handles de-dup and persistence; the hooks
// emit watcher:reindexing / watcher:done so the StatusBar reflects live activity.
func (fw *FileWatcher) reindexPath(path string) {
	log.Printf("watcher: detected change in %s, re-indexing...", path)
	fw.indexer.Enqueue(
		[]PathPriority{{Path: path, Priority: PriorityUrgent}},
		IndexHooks{
			OnStart: func() {
				wailsruntime.EventsEmit(fw.ctx, "watcher:reindexing", map[string]any{"path": path})
			},
			OnDone: func(chunks int) {
				log.Printf("watcher: re-indexed %s (%d chunks)", path, chunks)
				wailsruntime.EventsEmit(fw.ctx, "watcher:done", nil)
			},
		},
	)
}

// Stop gracefully shuts down the file watcher.
func (fw *FileWatcher) Stop() error {
	fw.cancel()
	if err := fw.watcher.Close(); err != nil {
		return fmt.Errorf("close watcher: %w", err)
	}
	fw.wg.Wait()
	return nil
}

// WaitForClosed blocks until the watcher event loop exits.
func (fw *FileWatcher) WaitForClosed() {
	<-fw.closed
}
