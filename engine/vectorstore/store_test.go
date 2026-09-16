package vectorstore

import "testing"

func TestVectorStoreSearchRanksAndLimitsResults(t *testing.T) {
	store := new(VectorStore)
	store.Add(EmbeddedChunk{DocPath: "closest.md", ChunkIdx: 0, Embedding: []float32{1, 0}})
	store.Add(EmbeddedChunk{DocPath: "second.md", ChunkIdx: 1, Embedding: []float32{0.8, 0.2}})
	store.Add(EmbeddedChunk{DocPath: "incompatible.md", ChunkIdx: 2, Embedding: []float32{1, 0, 0}})

	results := store.Search([]float32{1, 0}, 2)
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	if results[0].DocPath != "closest.md" || results[1].DocPath != "second.md" {
		t.Fatalf("unexpected ranking: %#v", results)
	}
}

func TestVectorStoreSearchRejectsInvalidQuery(t *testing.T) {
	store := new(VectorStore)
	store.Add(EmbeddedChunk{DocPath: "document.md", Embedding: []float32{1, 0}})

	for _, tc := range []struct {
		name  string
		query []float32
		topK  int
	}{
		{name: "empty query", query: nil, topK: 5},
		{name: "non-positive limit", query: []float32{1, 0}, topK: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := store.Search(tc.query, tc.topK); len(got) != 0 {
				t.Fatalf("result count = %d, want 0", len(got))
			}
		})
	}
}

func TestVectorStoreDocCount(t *testing.T) {
	store := new(VectorStore)
	if got := store.DocCount(); got != 0 {
		t.Fatalf("empty store DocCount = %d, want 0", got)
	}
	store.Add(EmbeddedChunk{DocPath: "a.md", ChunkIdx: 0, Embedding: []float32{1, 0}})
	store.Add(EmbeddedChunk{DocPath: "a.md", ChunkIdx: 1, Embedding: []float32{0, 1}})
	store.Add(EmbeddedChunk{DocPath: "b.pdf", ChunkIdx: 0, Embedding: []float32{1, 0}})
	if got := store.DocCount(); got != 2 {
		t.Fatalf("DocCount = %d, want 2 (documents, not chunks)", got)
	}
	store.RemoveDoc("a.md")
	if got := store.DocCount(); got != 1 {
		t.Fatalf("after RemoveDoc, DocCount = %d, want 1", got)
	}
}
