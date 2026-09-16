// Package embeddertest provides helpers for tests that need ONNX Runtime or the
// real embedding model. Every helper skips the test when its dependency is not
// installed, so suites never fail on machines without the assets.
package embeddertest

import (
	"os"
	"path/filepath"
	"testing"

	"spotfile/engine/embedder"
)

// LibraryPath returns a loadable onnxruntime shared library path, or skips.
func LibraryPath(tb testing.TB) string {
	tb.Helper()
	for _, p := range []string{
		"/opt/homebrew/lib/libonnxruntime.dylib",
		"/usr/local/lib/libonnxruntime.dylib",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	tb.Skip("onnxruntime shared library not found")
	return ""
}

// RealModel loads bge-small-en-v1.5 from ~/.spotfile and closes it when the
// test ends, or skips when the assets are absent. workers 0 means NumCPU.
func RealModel(tb testing.TB, workers int) *embedder.Model {
	tb.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		tb.Skipf("no home dir: %v", err)
	}
	dir := filepath.Join(home, ".spotfile")
	modelPath := filepath.Join(dir, "model.onnx")
	vocabPath := filepath.Join(dir, "vocab.txt")
	for _, p := range []string{modelPath, vocabPath} {
		if _, statErr := os.Stat(p); statErr != nil {
			tb.Skipf("asset missing: %s", p)
		}
	}
	m, err := embedder.New(embedder.Config{
		LibraryPath: LibraryPath(tb),
		ModelPath:   modelPath,
		VocabPath:   vocabPath,
		Workers:     workers,
	})
	if err != nil {
		tb.Fatalf("embedder init: %v", err)
	}
	tb.Cleanup(m.Close)
	return m
}

// WriteVocab writes a minimal BERT vocab into a temp dir and returns its path.
func WriteVocab(tb testing.TB) string {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "vocab.txt")
	content := "[PAD]\n[unused0]\n[unused1]\n[UNK]\n[CLS]\n[SEP]\nhello\nworld\nfoo\nbar\nbaz\ntest\ntext\nanother\nsample\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatalf("write test vocab: %v", err)
	}
	return path
}
