// Package pathutil holds path helpers shared by the settings, index and
// indexing packages.
package pathutil

import (
	"path/filepath"
	"strings"
)

// Within reports whether path is dir itself or lies below it. A sibling that
// merely shares the prefix ("/a/bc" for "/a/b") is not within it.
func Within(path, dir string) bool {
	path, dir = filepath.Clean(path), filepath.Clean(dir)
	return path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))
}
