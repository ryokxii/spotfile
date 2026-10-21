package vectorstore

import "testing"

func TestDocStats(t *testing.T) {
	store := new(VectorStore)
	store.Add(EmbeddedChunk{DocPath: "/n/b.md", Text: "hello", Embedding: []float32{1, 2}})
	store.Add(EmbeddedChunk{DocPath: "/n/b.md", Text: "abc", Embedding: []float32{1, 2}})
	store.Add(EmbeddedChunk{DocPath: "/n/a.md", Text: "", Embedding: []float32{1, 2, 3}})

	got := store.DocStats()
	want := []DocStat{
		{Path: "/n/a.md", Chunks: 1, Bytes: 12},
		{Path: "/n/b.md", Chunks: 2, Bytes: (8 + 5) + (8 + 3)},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("DocStats = %+v, want %+v", got, want)
	}
}
