package vectorstore

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadAcceptsSameEmbedder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.gob")
	orig := new(VectorStore)
	orig.SetEmbedder("model-a")
	orig.Add(EmbeddedChunk{DocPath: "a.md", Embedding: []float32{1, 0}})
	if err := orig.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := new(VectorStore)
	loaded.SetEmbedder("model-a")
	if err := loaded.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Len() != 1 {
		t.Fatalf("Len = %d, want 1", loaded.Len())
	}
}

func TestLoadRejectsDifferentEmbedder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.gob")
	orig := new(VectorStore)
	orig.SetEmbedder("model-a")
	orig.Add(EmbeddedChunk{DocPath: "a.md", Embedding: []float32{1, 0}})
	orig.Add(EmbeddedChunk{DocPath: "b.md", Embedding: []float32{0, 1}})
	if err := orig.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := new(VectorStore)
	loaded.SetEmbedder("model-b")
	err := loaded.Load(path)

	var stale *StaleError
	if !errors.As(err, &stale) {
		t.Fatalf("Load error = %v, want *StaleError", err)
	}
	slices.Sort(stale.Paths)
	if !slices.Equal(stale.Paths, []string{"a.md", "b.md"}) {
		t.Fatalf("stale paths = %v", stale.Paths)
	}
	if loaded.Len() != 0 {
		t.Fatalf("stale chunks were loaded: Len = %d", loaded.Len())
	}
}

// TestLoadLegacyFormatIsStale covers stores written before the format carried
// an embedder ID: a bare gob-encoded map of chunks.
func TestLoadLegacyFormatIsStale(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.gob")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	legacy := map[string][]EmbeddedChunk{
		"notes.md": {{DocPath: "notes.md", Embedding: []float32{1, 0}}},
	}
	if err := gob.NewEncoder(f).Encode(legacy); err != nil {
		t.Fatal(err)
	}
	f.Close()

	loaded := new(VectorStore)
	loaded.SetEmbedder("model-a")
	err = loaded.Load(path)

	var stale *StaleError
	if !errors.As(err, &stale) {
		t.Fatalf("Load error = %v, want *StaleError", err)
	}
	if !slices.Equal(stale.Paths, []string{"notes.md"}) {
		t.Fatalf("stale paths = %v", stale.Paths)
	}
	if loaded.Len() != 0 {
		t.Fatalf("legacy chunks were loaded: Len = %d", loaded.Len())
	}
}
