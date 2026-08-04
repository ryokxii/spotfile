package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileAsBase64 reads a file from disk and returns its content as base64.
// Used by the viewers to load local files without a file:// URL.
func (a *App) ReadFileAsBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// indexableExtensions are the file types Spotfile indexes. PDFs are parsed
// page-by-page by the engine; every other type is read as UTF-8 text.
var indexableExtensions = map[string]bool{
	".pdf": true,
	// documents / notes
	".txt": true, ".md": true, ".markdown": true, ".rst": true, ".org": true,
	// data / config
	".csv": true, ".tsv": true, ".json": true, ".jsonc": true, ".yaml": true,
	".yml": true, ".toml": true, ".ini": true, ".cfg": true, ".conf": true,
	".env": true, ".log": true, ".xml": true,
	// web
	".html": true, ".htm": true, ".css": true, ".scss": true, ".svelte": true, ".vue": true,
	// code
	".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".go": true, ".py": true,
	".rb": true, ".rs": true, ".java": true, ".kt": true, ".c": true, ".h": true,
	".cpp": true, ".hpp": true, ".cc": true, ".cs": true, ".php": true, ".swift": true,
	".sh": true, ".bash": true, ".zsh": true, ".sql": true,
}

// isIndexable reports whether a path's extension is one Spotfile indexes.
func isIndexable(path string) bool {
	return indexableExtensions[strings.ToLower(filepath.Ext(path))]
}
