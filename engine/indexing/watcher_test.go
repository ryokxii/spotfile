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
		name string
		path string
		want watchAction
	}{
		{"regular file is re-indexed", file, actReindex},
		{"missing path is forgotten", filepath.Join(dir, "gone.md"), actForget},
		{"deleted folder is forgotten", filepath.Join(dir, "gone-dir"), actForget},
		{"directory is ignored", sub, actIgnore},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := actionFor(tc.path); got != tc.want {
				t.Fatalf("actionFor(%s) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
