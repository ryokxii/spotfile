// Package trash moves files to the system Trash (macOS) or Recycle Bin
// (Windows), where they can be restored. It never deletes permanently.
package trash

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"spotfile/engine/pathutil"
)

// ErrUnsupported is returned where there is no system Trash to move files to.
var ErrUnsupported = errors.New("moving files to the Trash isn't supported on this system")

// Trasher moves one file to the system Trash.
type Trasher interface {
	Trash(path string) error
}

// System is the platform's Trash.
type System struct{}

// Allowed checks that path may go to the Trash: an absolute, clean path to a
// regular file that is indexed and, once symlinks are resolved, inside one of
// folders.
func Allowed(path string, folders []string, indexed func(string) bool) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("%q isn't a clean absolute path", path)
	}
	if !indexed(path) {
		return fmt.Errorf("%s isn't in the index", filepath.Base(path))
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", filepath.Base(path), err)
	}
	if info, err := os.Stat(real); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s isn't a file", filepath.Base(path))
	}
	for _, f := range folders {
		if realFolder, err := filepath.EvalSymlinks(f); err == nil && pathutil.Within(real, realFolder) {
			return nil
		}
	}
	return fmt.Errorf("%s is outside the indexed folders", filepath.Base(path))
}

// Move checks path with Allowed, then moves it to the Trash with t.
func Move(t Trasher, path string, folders []string, indexed func(string) bool) error {
	if err := Allowed(path, folders, indexed); err != nil {
		return err
	}
	if err := t.Trash(path); err != nil {
		return fmt.Errorf("move %s to the Trash: %w", filepath.Base(path), err)
	}
	return nil
}
