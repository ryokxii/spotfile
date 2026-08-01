package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbedSmoke runs a real end-to-end embedding against the installed
// bge-small-en-v1.5 assets in ~/.spotfile. It is skipped when the model or
// vocab files are absent so it never blocks CI without the assets.
func TestEmbedSmoke(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	dir := filepath.Join(home, ".spotfile")
	modelPath := filepath.Join(dir, "model.onnx")
	vocabPath := filepath.Join(dir, "vocab.txt")
	for _, p := range []string{modelPath, vocabPath} {
		if _, statErr := os.Stat(p); statErr != nil {
			t.Skipf("asset missing: %s", p)
		}
	}

	eng, err := New(Config{
		LibraryPath: testLibraryPath(t),
		ModelPath:   modelPath,
		VocabPath:   vocabPath,
		Workers:     1,
	})
	if err != nil {
		t.Fatalf("engine init failed with real assets: %v", err)
	}
	defer eng.Close()

	vec, err := eng.Embed(context.Background(), "the quick brown fox")
	if err != nil {
		t.Fatalf("embed failed: %v", err)
	}
	// bge-small-en-v1.5 produces 384-dimensional embeddings.
	if len(vec) != 384 {
		t.Fatalf("unexpected embedding dimension: got %d, want 384", len(vec))
	}
}
