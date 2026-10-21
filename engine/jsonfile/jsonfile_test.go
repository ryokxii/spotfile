package jsonfile

import (
	"os"
	"path/filepath"
	"testing"
)

type doc struct {
	A string `json:"a"`
	B int    `json:"b"`
}

func TestLoadMissingFileLeavesValue(t *testing.T) {
	v := doc{A: "kept"}
	if err := Load(filepath.Join(t.TempDir(), "x.json"), &v); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v.A != "kept" {
		t.Fatalf("v = %+v", v)
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "x.json")
	if err := Save(path, doc{A: "a", B: 2}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var got doc
	if err := Load(path, &got); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != (doc{A: "a", B: 2}) {
		t.Fatalf("got %+v", got)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("dir holds %v, want only x.json", entries)
	}
}

func TestLoadSetsAsideCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.json")
	if err := os.WriteFile(path, []byte(`{"a":`), 0o644); err != nil {
		t.Fatal(err)
	}
	var v doc
	if err := Load(path, &v); err == nil {
		t.Fatal("Load returned no error for a corrupt file")
	}
	if _, err := os.Stat(path + ".bad"); err != nil {
		t.Fatalf("not set aside: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("corrupt file still in place: %v", err)
	}
}
