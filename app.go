package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"spotfile/engine"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	engMu   sync.Mutex // serialises engine init so IndexFolder waits if startup is mid-init
	indexMu sync.Mutex // prevents overlapping folder-indexing jobs
	eng     *engine.Engine
	store   *engine.VectorStore
	indexer *engine.Indexer
	watcher *engine.FileWatcher
	llm     *engine.LLM
}

// recentWindow bounds which files are treated as "recent" and indexed at high
// priority on startup; older files backfill in the background.
const recentWindow = 7 * 24 * time.Hour

func NewApp() *App {
	return &App{store: new(engine.VectorStore)}
}

// startup auto-initialises the engine with platform defaults so users
// don't have to configure anything before indexing.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go func() {
		if err := a.InitEngine(EngineConfig{}); err != nil {
			wailsruntime.EventsEmit(ctx, "engine:error", err.Error())
			return
		}
		wailsruntime.EventsEmit(ctx, "engine:ready", nil)
	}()
}

func (a *App) shutdown(_ context.Context) {
	if a.watcher != nil {
		if err := a.watcher.Stop(); err != nil {
			fmt.Printf("error stopping watcher: %v\n", err)
		}
	}
	if a.indexer != nil {
		a.indexer.Stop()
	}
	if a.llm != nil {
		if err := a.llm.Close(); err != nil {
			fmt.Printf("error closing LLM: %v\n", err)
		}
	}
	if a.eng != nil {
		a.eng.Close()
	}
	// Tear down the process-global ORT environment once, after the engine
	// (and its session) is closed.
	engine.ShutdownRuntime()
}

// EngineConfig is the payload for initialising the engine.
// All fields are optional — empty strings fall back to platform defaults.
type EngineConfig struct {
	LibraryPath string `json:"libraryPath"`
	ModelPath   string `json:"modelPath"`
	VocabPath   string `json:"vocabPath"`
	Workers     int    `json:"workers"`
}

// InitEngine initialises (or re-initialises) the ONNX engine.
func (a *App) InitEngine(cfg EngineConfig) error {
	a.engMu.Lock()
	defer a.engMu.Unlock()
	return a.initEngineLocked(cfg)
}

func (a *App) initEngineLocked(cfg EngineConfig) error {
	// Stop the indexer before tearing down the engine it runs on.
	if a.indexer != nil {
		a.indexer.Stop()
		a.indexer = nil
	}
	if a.eng != nil {
		a.eng.Close()
		a.eng = nil
	}

	if cfg.LibraryPath == "" {
		cfg.LibraryPath = defaultLibraryPath()
	}
	if cfg.ModelPath == "" {
		cfg.ModelPath = filepath.Join(spotfileDir(), "model.onnx")
	}
	if cfg.VocabPath == "" {
		cfg.VocabPath = filepath.Join(spotfileDir(), "vocab.txt")
	}

	if err := os.MkdirAll(spotfileDir(), 0755); err != nil {
		return fmt.Errorf("create spotfile dir: %w", err)
	}

	// A missing model/vocab data file is a different problem from a missing ONNX
	// runtime library — check assets up front so we give an accurate, actionable
	// error instead of conflating it with a dlopen failure.
	for _, asset := range []struct{ path, name string }{
		{cfg.ModelPath, "model.onnx"},
		{cfg.VocabPath, "vocab.txt"},
	} {
		if _, statErr := os.Stat(asset.path); os.IsNotExist(statErr) {
			return fmt.Errorf("missing %s — download the bge-small-en-v1.5 model files into %s (see README)", asset.name, spotfileDir())
		}
	}

	var err error
	a.eng, err = engine.New(engine.Config{
		LibraryPath: cfg.LibraryPath,
		ModelPath:   cfg.ModelPath,
		VocabPath:   cfg.VocabPath,
		Workers:     cfg.Workers,
	})
	if err != nil {
		// Only a genuine shared-library load failure means the runtime is absent.
		if strings.Contains(err.Error(), "dlopen") {
			return fmt.Errorf("ONNX Runtime not found — run: brew install onnxruntime")
		}
		return err
	}

	llmModelPath := filepath.Join(spotfileDir(), "model.gguf")
	a.llm, _ = engine.NewLLM(engine.LLMConfig{
		ModelPath: llmModelPath,
		ModelType: "llama",
	})

	// Persist to ~/.spotfile/store.gob and reload a prior index on launch. Only
	// load when the in-memory store is empty so a re-init can't clobber chunks
	// indexed since startup.
	a.store.SetPersistPath(storePath())
	if a.store.Len() == 0 {
		if err := a.store.Load(storePath()); err != nil {
			log.Printf("store: load failed (starting empty): %v", err)
		} else if n := a.store.Len(); n > 0 {
			log.Printf("store: loaded %d chunks from %s", n, storePath())
		}
	}

	// Background indexer drains prioritized work on a single worker.
	a.indexer = engine.NewIndexer(a.ctx, a.eng, a.store)

	return nil
}

// SelectFolder opens a native directory picker and returns the chosen path.
// Returns an empty string if the user cancels.
func (a *App) SelectFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Choose a folder to index",
	})
}

// IndexFolder starts a background job that recursively walks dir, indexes all
// .txt, .md, and .pdf files, and (re)starts the file watcher on the directory
// tree. It returns immediately so Wails can deliver progress events while the
// indexing work continues.
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
	if a.eng == nil {
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
		switch strings.ToLower(filepath.Ext(path)) {
		case ".txt", ".md", ".pdf":
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("no supported files (.txt, .md, .pdf) found in %s", filepath.Base(dir))
	}
	log.Printf("index: starting folder=%q files=%d", dir, len(paths))

	wailsruntime.EventsEmit(a.ctx, "index:start", map[string]any{
		"total": len(paths),
		"dir":   dir,
	})

	// Recent files index at high priority (searchable first); history backfills
	// in the background. Unreadable paths are dropped here, so count over the
	// exact set that will be indexed to keep the progress percentage accurate.
	recent, older := engine.SplitByModTime(paths, recentWindow)
	indexed := append(append([]string{}, recent...), older...)
	totalChunks, err := engine.CountChunks(a.ctx, indexed)
	if err != nil {
		return fmt.Errorf("count chunks: %w", err)
	}
	log.Printf("index: prepared folder=%q chunks=%d recent=%d history=%d", dir, totalChunks, len(recent), len(older))
	wailsruntime.EventsEmit(a.ctx, "index:prepared", map[string]any{
		"totalChunks": totalChunks,
	})

	// Start the watcher before enqueuing so edits during the scan queue as urgent.
	a.restartWatcher(dir)

	items := make([]engine.PathPriority, 0, len(indexed))
	for _, p := range recent {
		items = append(items, engine.PathPriority{Path: p, Priority: engine.PriorityHigh})
	}
	for _, p := range older {
		items = append(items, engine.PathPriority{Path: p, Priority: engine.PriorityLow})
	}

	a.indexer.Enqueue(items, engine.IndexHooks{
		OnChunk: func(chunk engine.EmbeddedChunk, jobTotal int) {
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
	watcher, err := engine.StartWatcher(a.ctx, collectWatchDirs(dir), a.indexer)
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
	items := make([]engine.PathPriority, len(paths))
	for i, p := range paths {
		items[i] = engine.PathPriority{Path: p, Priority: engine.PriorityHigh}
	}
	a.indexer.Enqueue(items, engine.IndexHooks{
		OnChunk: func(chunk engine.EmbeddedChunk, jobTotal int) {
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

// Search embeds query and returns the topK nearest stored chunks.
func (a *App) Search(query string, topK int) ([]engine.SearchResult, error) {
	started := time.Now()
	if a.eng == nil {
		log.Printf("search: rejected because engine is not initialized")
		return nil, fmt.Errorf("engine not initialised — call InitEngine first")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		log.Printf("search: rejected empty query")
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if topK <= 0 {
		topK = 5
	}
	log.Printf("search: started query=%q topK=%d indexedChunks=%d", query, topK, a.store.Len())
	if a.store.Len() == 0 {
		log.Printf("search: completed results=0 duration=%s (no indexed chunks)", time.Since(started).Round(time.Millisecond))
		return []engine.SearchResult{}, nil
	}

	// Embed the query with the locally loaded BGE ONNX model. No query text or
	// document content leaves this process.
	vec, err := a.eng.Embed(a.ctx, query)
	if err != nil {
		log.Printf("search: embedding failed query=%q duration=%s error=%v", query, time.Since(started).Round(time.Millisecond), err)
		return nil, fmt.Errorf("embed local search query: %w", err)
	}
	results := a.store.Search(vec, topK)
	log.Printf("search: completed results=%d duration=%s", len(results), time.Since(started).Round(time.Millisecond))
	for i, result := range results {
		log.Printf(
			"search: result=%d score=%.4f path=%q chunk=%d page=%d content=%q",
			i+1,
			result.Score,
			result.DocPath,
			result.ChunkIdx,
			result.PageNum,
			result.Text,
		)
	}
	return results, nil
}

// GenerateAnswer searches for relevant chunks and uses LLM to generate an answer.
func (a *App) GenerateAnswer(query string, topK int) (string, error) {
	if a.eng == nil {
		return "", fmt.Errorf("engine not initialised — call InitEngine first")
	}
	if a.llm == nil {
		return "", fmt.Errorf("LLM not initialized")
	}

	results, err := a.Search(query, topK)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	if len(results) == 0 {
		return "No relevant documents found to generate an answer from.", nil
	}

	context := engine.FormatContext(results, 2000)
	systemPrompt := "You are a helpful search assistant. Answer based on the provided documents. If information is not in the documents, say so clearly."
	answer, err := a.llm.Generate(a.ctx, systemPrompt, query, context)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}
	return answer, nil
}

// StoreSize returns the number of indexed chunks.
func (a *App) StoreSize() int {
	return a.store.Len()
}

// ReadFileAsBase64 reads a file from disk and returns its content as base64.
// Used by the PDF viewer to load local files without a file:// URL.
func (a *App) ReadFileAsBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func spotfileDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".spotfile")
}

// storePath is where the persistent vector store lives on disk.
func storePath() string {
	return filepath.Join(spotfileDir(), "store.gob")
}

func defaultLibraryPath() string {
	switch runtime.GOOS {
	case "darwin":
		for _, p := range []string{
			"/opt/homebrew/lib/libonnxruntime.dylib",
			"/usr/local/lib/libonnxruntime.dylib",
		} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	default:
		return "libonnxruntime.so"
	}
}
