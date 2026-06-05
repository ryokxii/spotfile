package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	if a.llm != nil {
		if err := a.llm.Close(); err != nil {
			fmt.Printf("error closing LLM: %v\n", err)
		}
	}
	if a.eng != nil {
		a.eng.Close()
	}
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

	llmModelPath := filepath.Join(spotfileDir(), "model.gguf")
	a.llm, _ = engine.NewLLM(engine.LLMConfig{
		ModelPath: llmModelPath,
		ModelType: "llama",
	})

	return nil
}

// SelectFolder opens a native directory picker and returns the chosen path.
// Returns an empty string if the user cancels.
func (a *App) SelectFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Choose a folder to index",
	})
}

// IndexFolder recursively walks dir, indexes all .txt, .md, and .pdf files,
// and (re)starts the file watcher on the directory tree.
// Emits: "index:start" (with total file count), "index:chunk", "index:done",
// and "watcher:started" once the watcher is running.
func (a *App) IndexFolder(dir string) error {
	if a.eng == nil {
		return fmt.Errorf("engine not initialised")
	}

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

	wailsruntime.EventsEmit(a.ctx, "index:start", map[string]any{
		"total": len(paths),
		"dir":   dir,
	})

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

	// (Re)start watcher on all directories under root for recursive monitoring.
	if a.watcher != nil {
		_ = a.watcher.Stop()
		a.watcher = nil
	}
	watchDirs := collectWatchDirs(dir)
	watcher, err := engine.StartWatcher(a.ctx, watchDirs, a.eng, a.store)
	if err != nil {
		fmt.Printf("warning: failed to start watcher: %v\n", err)
	} else {
		a.watcher = watcher
		wailsruntime.EventsEmit(a.ctx, "watcher:started", nil)
	}

	return nil
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

// IndexFiles is kept for internal use by the watcher's reindex path.
func (a *App) IndexFiles(paths []string) error {
	if a.eng == nil {
		return fmt.Errorf("engine not initialised — call InitEngine first")
	}

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
