package vectorstore

import (
	"path/filepath"
	"testing"
)

func TestVectorStoreSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.gob")

	orig := new(VectorStore)
	orig.Add(EmbeddedChunk{DocPath: "a.md", ChunkIdx: 0, PageNum: 1, Text: "alpha", Embedding: []float32{1, 0}})
	orig.Add(EmbeddedChunk{DocPath: "a.md", ChunkIdx: 1, PageNum: 1, Text: "beta", Embedding: []float32{0.9, 0.1}})
	orig.Add(EmbeddedChunk{DocPath: "b.md", ChunkIdx: 0, PageNum: 0, Text: "gamma", Embedding: []float32{0, 1}})

	if err := orig.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := new(VectorStore)
	if err := loaded.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := loaded.Len(), orig.Len(); got != want {
		t.Fatalf("loaded Len = %d, want %d", got, want)
	}

	// Search should behave identically against the reloaded store.
	results := loaded.Search([]float32{1, 0}, 1)
	if len(results) != 1 || results[0].DocPath != "a.md" || results[0].Text != "alpha" {
		t.Fatalf("unexpected search after load: %#v", results)
	}
}

func TestVectorStoreLoadMissingFileIsNoError(t *testing.T) {
	store := new(VectorStore)
	if err := store.Load(filepath.Join(t.TempDir(), "does-not-exist.gob")); err != nil {
		t.Fatalf("Load of missing file returned error: %v", err)
	}
	if store.Len() != 0 {
		t.Fatalf("expected empty store, got Len = %d", store.Len())
	}
}

func TestVectorStoreRemoveDocReplacesChunks(t *testing.T) {
	store := new(VectorStore)
	store.Add(EmbeddedChunk{DocPath: "a.md", ChunkIdx: 0, Embedding: []float32{1, 0}})
	store.Add(EmbeddedChunk{DocPath: "a.md", ChunkIdx: 1, Embedding: []float32{0, 1}})
	store.Add(EmbeddedChunk{DocPath: "b.md", ChunkIdx: 0, Embedding: []float32{1, 0}})

	store.RemoveDoc("a.md")
	if got := store.Len(); got != 1 {
		t.Fatalf("after RemoveDoc, Len = %d, want 1", got)
	}

	// Re-adding a.md must not resurrect the removed chunks.
	store.Add(EmbeddedChunk{DocPath: "a.md", ChunkIdx: 0, Embedding: []float32{1, 0}})
	if got := store.Len(); got != 2 {
		t.Fatalf("after re-add, Len = %d, want 2", got)
	}
}
