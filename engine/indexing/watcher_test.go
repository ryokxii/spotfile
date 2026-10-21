package indexing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActionFor(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.md")
	writeFile(t, file, "alpha")
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name   string
		path   string
		accept func(string) bool
		want   watchAction
	}{
		{"regular file is re-indexed", file, nil, actReindex},
		{"accepted file is re-indexed", file, func(p string) bool { return p == file }, actReindex},
		{"file outside the saved roots is ignored", file, func(string) bool { return false }, actIgnore},
		{"missing path is forgotten", filepath.Join(dir, "gone.md"), nil, actForget},
		{"deleted folder is forgotten", filepath.Join(dir, "gone-dir"), nil, actForget},
		{"directory is ignored", sub, nil, actIgnore},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := actionFor(tc.path, tc.accept); got != tc.want {
				t.Fatalf("actionFor(%s) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
