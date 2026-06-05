package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"spotfile/engine"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	eng     *engine.Engine
	store   *engine.VectorStore
	watcher *engine.FileWatcher
	llm     *engine.LLM
}

func NewApp() *App {
	return &App{store: new(engine.VectorStore)}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(_ context.Context) {
	if a.watcher != nil {
		if err := a.watcher.Stop(); err != nil {
			fmt.Printf("error stopping watcher: %v\n", err)
		}
	}
	if a.llm != nil {
		if err := a.llm.Close(); err != nil {
			fmt.Printf("error closing LLM: %v\n", err)
		}
	}
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
	if err != nil {
		return err
	}

	// Initialize LLM (gracefully handles missing model)
	llmModelPath := filepath.Join(spotfileDir(), "model.gguf")
	a.llm, _ = engine.NewLLM(engine.LLMConfig{
		ModelPath: llmModelPath,
		ModelType: "llama",
	})

	return nil
}

// IndexFiles embeds every file in paths and stores the result vectors.
// Progress events ("index:chunk") are emitted after each chunk is stored.
// Starts the file watcher on first call to auto-reindex on file changes.
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

	// Start watcher on first indexing (if not already started)
	if a.watcher == nil && len(paths) > 0 {
		watcher, err := engine.StartWatcher(a.ctx, paths, a.eng, a.store)
		if err != nil {
			fmt.Printf("warning: failed to start file watcher: %v\n", err)
		} else {
			a.watcher = watcher
			wailsruntime.EventsEmit(a.ctx, "watcher:started", nil)
		}
	}

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

// GenerateAnswer searches for relevant chunks and uses LLM to generate an answer.
func (a *App) GenerateAnswer(query string, topK int) (string, error) {
	if a.eng == nil {
		return "", fmt.Errorf("engine not initialised — call InitEngine first")
	}
	if a.llm == nil {
		return "", fmt.Errorf("LLM not initialized")
	}

	// Search for relevant chunks
	results, err := a.Search(query, topK)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return "No relevant documents found to generate an answer from.", nil
	}

	// Format context from search results
	context := engine.FormatContext(results, 2000)

	// Generate answer using LLM
	systemPrompt := "You are a helpful search assistant. Answer based on the provided documents. If information is not in the documents, say so clearly."
	answer, err := a.llm.Generate(a.ctx, systemPrompt, query, context)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	return answer, nil
}

// StoreSize returns the number of indexed chunks (useful for UI status).
func (a *App) StoreSize() int {
	return a.store.Len()
}

// ReadFileAsBase64 reads a file from disk and returns its content as a
// base64-encoded string. Used by the frontend PDF viewer to load local PDFs.
func (a *App) ReadFileAsBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
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
