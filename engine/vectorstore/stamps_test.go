package vectorstore

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func stamp(sec int64, size int64) Stamp {
	return Stamp{ModTime: time.Unix(sec, 0).UTC(), Size: size}
}

func TestStampsRoundTripThroughSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.gob")
	orig := new(VectorStore)
	orig.SetEmbedder("m1")
	orig.Add(EmbeddedChunk{DocPath: "/n/a.md", Text: "alpha", Embedding: []float32{1, 0}})
	orig.Add(EmbeddedChunk{DocPath: "/n/b.md", Text: "beta", Embedding: []float32{0, 1}})
	orig.SetStamp("/n/a.md", stamp(100, 5))
	orig.SetStamp("/n/b.md", stamp(200, 4))
	if err := orig.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := new(VectorStore)
	loaded.SetEmbedder("m1")
	if err := loaded.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := loaded.Stamps()
	for p, want := range map[string]Stamp{"/n/a.md": stamp(100, 5), "/n/b.md": stamp(200, 4)} {
		if !got[p].ModTime.Equal(want.ModTime) || got[p].Size != want.Size {
			t.Fatalf("stamp %s = %+v, want %+v", p, got[p], want)
		}
	}
}

func TestLoadOlderFileWithoutStamps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.gob")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	older := struct {
		Version  int
		Embedder string
		Docs     map[string][]EmbeddedChunk
	}{formatVersion, "m1", map[string][]EmbeddedChunk{"/n/a.md": {{DocPath: "/n/a.md", Embedding: []float32{1}}}}}
	if err := gob.NewEncoder(f).Encode(older); err != nil {
		t.Fatal(err)
	}
	f.Close()

	store := new(VectorStore)
	store.SetEmbedder("m1")
	if err := store.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if store.DocCount() != 1 {
		t.Fatalf("DocCount = %d, want 1", store.DocCount())
	}
	got, ok := store.Stamps()["/n/a.md"]
	if !ok || got != (Stamp{}) {
		t.Fatalf("stamp = %+v (present %v), want zero stamp", got, ok)
	}
}

func TestStampsCoverEveryIndexedPath(t *testing.T) {
	store := new(VectorStore)
	store.Add(EmbeddedChunk{DocPath: "/n/unstamped.md", Embedding: []float32{1}})
	store.SetStamp("/n/empty.md", stamp(1, 0)) // indexed, but produced no chunks

	got := store.Stamps()
	if len(got) != 2 {
		t.Fatalf("Stamps = %v, want 2 paths", got)
	}
	if got["/n/unstamped.md"] != (Stamp{}) {
		t.Fatalf("unstamped doc = %+v, want zero", got["/n/unstamped.md"])
	}
	if got["/n/empty.md"].Size != 0 || got["/n/empty.md"].ModTime.IsZero() {
		t.Fatalf("empty file stamp = %+v", got["/n/empty.md"])
	}
}

func TestStampsReturnsACopy(t *testing.T) {
	store := new(VectorStore)
	store.SetStamp("/n/a.md", stamp(1, 1))
	store.Stamps()["/n/b.md"] = stamp(2, 2)
	if _, ok := store.Stamps()["/n/b.md"]; ok {
		t.Fatal("writing to the returned map changed the store")
	}
}

func TestRemoveDocsAndClearDropStamps(t *testing.T) {
	store := new(VectorStore)
	store.Add(EmbeddedChunk{DocPath: "/n/a.md", Embedding: []float32{1}})
	store.SetStamp("/n/a.md", stamp(1, 1))
	store.SetStamp("/n/b.md", stamp(2, 2))

	store.RemoveDocs([]string{"/n/a.md"})
	if _, ok := store.Stamps()["/n/a.md"]; ok {
		t.Fatal("RemoveDocs kept the stamp")
	}
	store.Clear()
	if n := len(store.Stamps()); n != 0 {
		t.Fatalf("after Clear: %d stamps", n)
	}
}

func TestRemoveUnder(t *testing.T) {
	sep := string(filepath.Separator)
	dir := sep + filepath.Join("a", "b")
	paths := []string{
		filepath.Join(dir, "x.md"),
		filepath.Join(dir, "y", "z.md"),
		dir, // a file named exactly like the removed path
		sep + filepath.Join("a", "bc.md"),
	}
	store := new(VectorStore)
	for i, p := range paths {
		store.Add(EmbeddedChunk{DocPath: p, Embedding: []float32{1}})
		store.SetStamp(p, stamp(int64(i+1), 1))
	}

	if n := store.RemoveUnder(dir); n != 3 {
		t.Fatalf("RemoveUnder = %d, want 3", n)
	}
	got := store.Stamps()
	if len(got) != 1 || store.DocCount() != 1 {
		t.Fatalf("left %v (DocCount %d), want only the sibling", got, store.DocCount())
	}
	if _, ok := got[paths[3]]; !ok {
		t.Fatalf("sibling %s was removed", paths[3])
	}
}

func TestRemoveUnderStampOnlyPath(t *testing.T) {
	store := new(VectorStore)
	store.SetStamp("/n/empty.md", stamp(1, 0))
	if n := store.RemoveUnder("/n"); n != 1 {
		t.Fatalf("RemoveUnder = %d, want 1", n)
	}
}

func TestSavedAt(t *testing.T) {
	store := new(VectorStore)
	store.SetPersistPath(filepath.Join(t.TempDir(), "store.gob"))
	if !store.SavedAt().IsZero() {
		t.Fatal("SavedAt non-zero with no file")
	}
	if err := store.Persist(); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(store.SavedAt()); d < 0 || d > 5*time.Second {
		t.Fatalf("SavedAt %v ago", d)
	}
}

func TestAdoptStamps(t *testing.T) {
	store := new(VectorStore)
	store.Add(EmbeddedChunk{DocPath: "/n/a.md", Embedding: []float32{1}})
	store.AdoptStamps(map[string]Stamp{"/n/a.md": stamp(7, 3)})
	if got := store.Stamps()["/n/a.md"]; got != stamp(7, 3) {
		t.Fatalf("stamp = %+v", got)
	}
	if store.Len() != 1 {
		t.Fatalf("Len = %d, chunks changed", store.Len())
	}
}

func TestStampOf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	got := StampOf(info)
	if got.Size != 5 || !got.ModTime.Equal(info.ModTime()) {
		t.Fatalf("StampOf = %+v", got)
	}
}
