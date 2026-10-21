package indexing

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"spotfile/engine/vectorstore"
)

func at(sec int64, size int64) vectorstore.Stamp {
	return vectorstore.Stamp{ModTime: time.Unix(sec, 0), Size: size}
}

func TestReconcile(t *testing.T) {
	savedAt := time.Unix(1000, 0)
	for _, tc := range []struct {
		name       string
		onDisk     map[string]vectorstore.Stamp
		indexed    map[string]vectorstore.Stamp
		wantIndex  []string
		wantRemove []string
		wantAdopt  map[string]vectorstore.Stamp
	}{
		{
			name:       "file gone from disk",
			indexed:    map[string]vectorstore.Stamp{"/n/gone.md": at(10, 1)},
			wantRemove: []string{"/n/gone.md"},
		},
		{
			name:      "new file",
			onDisk:    map[string]vectorstore.Stamp{"/n/new.md": at(10, 1)},
			wantIndex: []string{"/n/new.md"},
		},
		{
			name:      "modification time changed",
			onDisk:    map[string]vectorstore.Stamp{"/n/a.md": at(20, 1)},
			indexed:   map[string]vectorstore.Stamp{"/n/a.md": at(10, 1)},
			wantIndex: []string{"/n/a.md"},
		},
		{
			name:      "size changed",
			onDisk:    map[string]vectorstore.Stamp{"/n/a.md": at(10, 2)},
			indexed:   map[string]vectorstore.Stamp{"/n/a.md": at(10, 1)},
			wantIndex: []string{"/n/a.md"},
		},
		{
			name:    "unchanged",
			onDisk:  map[string]vectorstore.Stamp{"/n/a.md": at(10, 1)},
			indexed: map[string]vectorstore.Stamp{"/n/a.md": at(10, 1)},
		},
		{
			name:      "unstamped, modified after the index was saved",
			onDisk:    map[string]vectorstore.Stamp{"/n/a.md": at(2000, 1)},
			indexed:   map[string]vectorstore.Stamp{"/n/a.md": {}},
			wantIndex: []string{"/n/a.md"},
		},
		{
			name:      "unstamped, older than the index",
			onDisk:    map[string]vectorstore.Stamp{"/n/a.md": at(500, 1)},
			indexed:   map[string]vectorstore.Stamp{"/n/a.md": {}},
			wantAdopt: map[string]vectorstore.Stamp{"/n/a.md": at(500, 1)},
		},
		{
			name:       "outputs are sorted",
			onDisk:     map[string]vectorstore.Stamp{"/n/c.md": at(1, 1), "/n/a.md": at(1, 1), "/n/b.md": at(1, 1)},
			indexed:    map[string]vectorstore.Stamp{"/n/z.md": at(1, 1), "/n/y.md": at(1, 1)},
			wantIndex:  []string{"/n/a.md", "/n/b.md", "/n/c.md"},
			wantRemove: []string{"/n/y.md", "/n/z.md"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Reconcile(tc.onDisk, tc.indexed, savedAt)
			if !reflect.DeepEqual(got.Index, tc.wantIndex) {
				t.Errorf("Index = %v, want %v", got.Index, tc.wantIndex)
			}
			if !reflect.DeepEqual(got.Remove, tc.wantRemove) {
				t.Errorf("Remove = %v, want %v", got.Remove, tc.wantRemove)
			}
			if len(got.Adopt) != len(tc.wantAdopt) {
				t.Fatalf("Adopt = %v, want %v", got.Adopt, tc.wantAdopt)
			}
			for p, want := range tc.wantAdopt {
				if got.Adopt[p] != want {
					t.Errorf("Adopt[%s] = %+v, want %+v", p, got.Adopt[p], want)
				}
			}
		})
	}
}

func TestReconcileDoesNotMutateInputs(t *testing.T) {
	onDisk := map[string]vectorstore.Stamp{"/n/a.md": at(10, 1)}
	indexed := map[string]vectorstore.Stamp{"/n/b.md": at(10, 1)}
	Reconcile(onDisk, indexed, time.Time{})
	if len(onDisk) != 1 || len(indexed) != 1 {
		t.Fatalf("inputs changed: %v %v", onDisk, indexed)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "alpha")
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "be")
	writeFile(t, filepath.Join(dir, "c.bin"), "x")
	include := func(p string) bool { return strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".txt") }

	got, err := Scan(dir, include)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Scan = %v, want 2 files", got)
	}
	if got[filepath.Join(dir, "a.md")].Size != 5 || got[filepath.Join(dir, "sub", "b.txt")].Size != 2 {
		t.Fatalf("sizes wrong: %v", got)
	}
}

func TestScanMissingDir(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "nope"), func(string) bool { return true }); err == nil {
		t.Fatal("Scan of a missing folder returned no error")
	}
}

func TestMissingPaths(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "kept.md")
	writeFile(t, kept, "k")
	gone := filepath.Join(dir, "gone.md")
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := MissingPaths([]string{kept, gone, sub})
	want := []string{gone, sub}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MissingPaths = %v, want %v", got, want)
	}
}

func TestStampsOf(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "a.md")
	writeFile(t, present, "hello")
	missing := filepath.Join(dir, "gone.md")
	stamps := stampsOf([]string{present, missing})

	if stamps[present].Size != 5 {
		t.Fatalf("stamp for present file = %+v", stamps[present])
	}
	if _, ok := stamps[missing]; ok {
		t.Fatal("missing file was stamped")
	}
}

func newForgetIndexer(t *testing.T) (*Indexer, string) {
	t.Helper()
	storeFile := filepath.Join(t.TempDir(), "store.gob")
	store := new(vectorstore.VectorStore)
	store.SetPersistPath(storeFile)
	store.Add(vectorstore.EmbeddedChunk{DocPath: "/n/old/a.md", Embedding: []float32{1}})
	store.Add(vectorstore.EmbeddedChunk{DocPath: "/n/old/b.md", Embedding: []float32{1}})
	store.Add(vectorstore.EmbeddedChunk{DocPath: "/n/keep.md", Embedding: []float32{1}})
	ix := &Indexer{store: store}
	ix.cond = sync.NewCond(&ix.mu)
	return ix, storeFile
}

func TestIndexerForgetFolder(t *testing.T) {
	ix, storeFile := newForgetIndexer(t)
	if n := ix.Forget("/n/old"); n != 2 {
		t.Fatalf("Forget = %d, want 2", n)
	}
	if ix.store.DocCount() != 1 {
		t.Fatalf("DocCount = %d, want 1", ix.store.DocCount())
	}
	if _, err := os.Stat(storeFile); err != nil {
		t.Fatalf("store not persisted: %v", err)
	}
}

func TestIndexerForgetUnknownPathWritesNothing(t *testing.T) {
	ix, storeFile := newForgetIndexer(t)
	if n := ix.Forget("/elsewhere"); n != 0 {
		t.Fatalf("Forget = %d, want 0", n)
	}
	if _, err := os.Stat(storeFile); !os.IsNotExist(err) {
		t.Fatalf("store written for a no-op: %v", err)
	}
}
