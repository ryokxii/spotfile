package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"spotfile/engine"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx   context.Context
	eng   *engine.Engine
	store *engine.VectorStore
}

func NewApp() *App {
	return &App{store: new(engine.VectorStore)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(_ context.Context) {
	if a.eng != nil {
		a.eng.Close()
	}
}

// EngineConfig is the payload sent from the frontend when initialising the engine.
type EngineConfig struct {
	LibraryPath string `json:"libraryPath"`
	ModelPath   string `json:"modelPath"`
	VocabPath   string `json:"vocabPath"`
	Workers     int    `json:"workers"`
}

// InitEngine initialises (or re-initialises) the ONNX engine. Call this once
// from the frontend before IndexFiles or Search.
func (a *App) InitEngine(cfg EngineConfig) error {
	if a.eng != nil {
		a.eng.Close()
		a.eng = nil
	}

	// Fill in any blanks with platform defaults.
	if cfg.LibraryPath == "" {
		cfg.LibraryPath = defaultLibraryPath()
	}
	if cfg.ModelPath == "" {
		cfg.ModelPath = filepath.Join(spotfileDir(), "model.onnx")
	}
	if cfg.VocabPath == "" {
		cfg.VocabPath = filepath.Join(spotfileDir(), "vocab.txt")
	}

	// Ensure .spotfile directory exists
	if err := os.MkdirAll(spotfileDir(), 0755); err != nil {
		return fmt.Errorf("create spotfile dir: %w", err)
	}

	var err error
	a.eng, err = engine.New(engine.Config{
		LibraryPath: cfg.LibraryPath,
		ModelPath:   cfg.ModelPath,
		VocabPath:   cfg.VocabPath,
		Workers:     cfg.Workers,
	})
	return err
}

// IndexFiles embeds every file in paths and stores the result vectors.
// Progress events ("index:chunk") are emitted after each chunk is stored.
func (a *App) IndexFiles(paths []string) error {
	if a.eng == nil {
		return fmt.Errorf("engine not initialised — call InitEngine first")
	}

	// Sort by modification time (recent files first)
	paths = engine.SortPathsByModTime(paths)

	chunks := a.eng.IndexFiles(a.ctx, paths)
	var n int
	for chunk := range chunks {
		a.store.Add(chunk)
		n++
		wailsruntime.EventsEmit(a.ctx, "index:chunk", map[string]any{
			"total": n,
			"path":  chunk.DocPath,
		})
	}
	wailsruntime.EventsEmit(a.ctx, "index:done", n)
	return nil
}

// Search embeds query and returns the topK nearest stored chunks.
func (a *App) Search(query string, topK int) ([]engine.SearchResult, error) {
	if a.eng == nil {
		return nil, fmt.Errorf("engine not initialised — call InitEngine first")
	}
	vec, err := a.eng.Embed(a.ctx, query)
	if err != nil {
		return nil, err
	}
	return a.store.Search(vec, topK), nil
}

// StoreSize returns the number of indexed chunks (useful for UI status).
func (a *App) StoreSize() int {
	return a.store.Len()
}

// Greet returns a greeting message.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s!", name)
}

func spotfileDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".spotfile")
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
