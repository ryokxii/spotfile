package pathutil

import (
	"path/filepath"
	"testing"
)

func TestWithin(t *testing.T) {
	p := filepath.FromSlash
	for _, tc := range []struct {
		name      string
		path, dir string
		want      bool
	}{
		{"same path", "/a/b", "/a/b", true},
		{"child", "/a/b/x.md", "/a/b", true},
		{"grandchild", "/a/b/c/x.md", "/a/b", true},
		{"sibling sharing a prefix", "/a/bc", "/a/b", false},
		{"trailing separator on dir", "/a/b/x.md", "/a/b/", true},
		{"parent", "/a", "/a/b", false},
		{"unrelated", "/z/x.md", "/a/b", false},
		{"unclean path", "/a/b/../b/x.md", "/a/b", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Within(p(tc.path), p(tc.dir)); got != tc.want {
				t.Fatalf("Within(%q, %q) = %v, want %v", tc.path, tc.dir, got, tc.want)
			}
		})
	}
}
