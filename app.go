package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"spotfile/engine/embedder"
	"spotfile/engine/indexing"
	"spotfile/engine/llamaserver"
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
}

// context returns the Wails app context, or Background before startup.
func (a *App) context() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// EngineConfig is the payload for initialising the engine.
// All fields are optional — empty strings fall back to platform defaults.
type EngineConfig struct {
	ModelPath  string `json:"modelPath"`  // embedding model GGUF
	ServerPath string `json:"serverPath"` // llama-server executable
}

// InitEngine initialises (or re-initialises) the embedding engine.
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

	if cfg.ModelPath == "" {
		cfg.ModelPath = filepath.Join(modelsDir(), embedder.ModelFile)
	}
	if err := os.MkdirAll(modelsDir(), 0755); err != nil {
		return fmt.Errorf("create models dir: %w", err)
	}
	if _, err := os.Stat(cfg.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("missing %s — download it into %s (see README)", filepath.Base(cfg.ModelPath), modelsDir())
	}

	model, err := embedder.New(embedder.Config{ModelPath: cfg.ModelPath, ServerPath: cfg.ServerPath})
	if err != nil {
		return err
	}
	// Start llama-server now so a missing binary or a bad model is reported at
	// launch rather than on the first search.
	if err := model.Start(a.context()); err != nil {
		model.Close()
		if errors.Is(err, llamaserver.ErrNotFound) {
			return fmt.Errorf("llama.cpp not found — install it (macOS: brew install llama.cpp) or place llama-server next to Spotfile")
		}
		return fmt.Errorf("start embedding server: %w", err)
	}
	a.embedder = model

	llmModelPath := filepath.Join(spotfileDir(), "model.gguf")
	a.llm, _ = llm.New(llm.Config{
		ModelPath: llmModelPath,
		ModelType: "llama",
	})

	// Persist to ~/.spotfile/store.gob and reload a prior index on launch. Only
	// load when the in-memory store is empty so a re-init can't clobber chunks
	// indexed since startup.
	a.store.SetPersistPath(storePath())
	a.store.SetEmbedder(embedder.ID)
	var stalePaths []string
	if a.store.Len() == 0 {
		var stale *vectorstore.StaleError
		switch err := a.store.Load(storePath()); {
		case errors.As(err, &stale):
			log.Printf("store: %v", err)
			stalePaths = stale.Paths
		case err != nil:
			log.Printf("store: load failed (starting empty): %v", err)
		default:
			if n := a.store.Len(); n > 0 {
				log.Printf("store: loaded %d chunks from %s", n, storePath())
			}
		}
	}

	// Background indexer drains prioritized work on a single worker.
	a.indexer = indexing.NewIndexer(a.context(), a.embedder, a.store)
	if len(stalePaths) > 0 {
		go a.rebuildStaleIndex(stalePaths)
	}

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

// modelsDir holds downloaded model files.
func modelsDir() string {
	return filepath.Join(spotfileDir(), "models")
}

// storePath is where the persistent vector store lives on disk.
func storePath() string {
	return filepath.Join(spotfileDir(), "store.gob")
}
