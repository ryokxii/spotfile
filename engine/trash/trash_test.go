package trash

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeTrasher struct {
	trashed []string
	err     error
}

func (f *fakeTrasher) Trash(path string) error {
	f.trashed = append(f.trashed, path)
	return f.err
}

func write(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAllowed(t *testing.T) {
	root := t.TempDir()
	notes := filepath.Join(root, "notes")
	work := filepath.Join(root, "work")
	inNotes := write(t, filepath.Join(notes, "a.md"))
	inWork := write(t, filepath.Join(work, "sub", "b.md"))
	outside := write(t, filepath.Join(root, "elsewhere", "c.md"))
	link := filepath.Join(notes, "link.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	folders := []string{notes, work}
	indexed := func(p string) bool { return p != filepath.Join(notes, "unindexed.md") }
	write(t, filepath.Join(notes, "unindexed.md"))

	for _, tc := range []struct {
		name    string
		path    string
		wantErr string
	}{
		{"file in the first folder", inNotes, ""},
		{"file in the second folder", inWork, ""},
		{"relative path", "notes/a.md", "clean absolute path"},
		{"unclean path", notes + "/../notes/a.md", "clean absolute path"},
		{"not indexed", filepath.Join(notes, "unindexed.md"), "isn't in the index"},
		{"missing file", filepath.Join(notes, "gone.md"), "resolve"},
		{"directory", notes, "isn't a file"},
		{"outside every folder", outside, "outside the indexed folders"},
		{"symlink pointing outside", link, "outside the indexed folders"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Allowed(tc.path, folders, indexed)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Allowed: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestMove(t *testing.T) {
	root := t.TempDir()
	file := write(t, filepath.Join(root, "a.md"))
	indexed := func(string) bool { return true }

	refused := &fakeTrasher{}
	if err := Move(refused, file, []string{filepath.Join(root, "other")}, indexed); err == nil {
		t.Fatal("Move allowed a file outside the folders")
	}
	if len(refused.trashed) != 0 {
		t.Fatal("trasher called for a refused path")
	}

	ok := &fakeTrasher{}
	if err := Move(ok, file, []string{root}, indexed); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if len(ok.trashed) != 1 || ok.trashed[0] != file {
		t.Fatalf("trashed %v", ok.trashed)
	}

	failing := &fakeTrasher{err: errors.New("no Trash on this volume")}
	if err := Move(failing, file, []string{root}, indexed); err == nil || !strings.Contains(err.Error(), "no Trash on this volume") {
		t.Fatalf("err = %v", err)
	}
}
