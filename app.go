package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"spotfile/engine/embedder"
	"spotfile/engine/indexing"
	"spotfile/engine/llm"
	"spotfile/engine/vectorstore"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	engMu    sync.Mutex // serialises engine init so IndexFolder waits if startup is mid-init
	indexMu  sync.Mutex // prevents overlapping folder-indexing jobs
	embedder *embedder.Model
	store    *vectorstore.VectorStore
	indexer  *indexing.Indexer
	watcher  *indexing.FileWatcher
	llm      *llm.Model
}

// recentWindow bounds which files are treated as "recent" and indexed at high
// priority on startup; older files backfill in the background.
const recentWindow = 7 * 24 * time.Hour

func NewApp() *App {
	return &App{store: new(vectorstore.VectorStore)}
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
	if a.embedder != nil {
		a.embedder.Close()
	}
	// Tear down the process-global ORT environment once, after the engine
	// (and its session) is closed.
	embedder.ShutdownRuntime()
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
	if a.embedder != nil {
		a.embedder.Close()
		a.embedder = nil
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
	a.embedder, err = embedder.New(embedder.Config{
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
	a.llm, _ = llm.New(llm.Config{
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
	a.indexer = indexing.NewIndexer(a.ctx, a.embedder, a.store)

	return nil
}

// SelectFolder opens a native directory picker and returns the chosen path.
// Returns an empty string if the user cancels.
func (a *App) SelectFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Choose a folder to index",
	})
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
