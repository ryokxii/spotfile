// Package jsonfile reads and writes the small JSON files Spotfile keeps in
// ~/.spotfile. Writes are atomic, and a file that can't be read is set aside
// rather than stopping the app.
package jsonfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Load decodes the file at path into v. A missing file is not an error and
// leaves v as it was. A file that can't be read or decoded is renamed to
// path+".bad" and the returned error says why.
func Load(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return SetAside(path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return SetAside(path, fmt.Errorf("decode %s: %w", filepath.Base(path), err))
	}
	return nil
}

// SetAside renames an unusable file to path+".bad", keeping it for inspection,
// and returns cause annotated with what happened to the file.
func SetAside(path string, cause error) error {
	if err := os.Rename(path, path+".bad"); err != nil {
		return fmt.Errorf("%w (moving the file aside also failed: %v)", cause, err)
	}
	return fmt.Errorf("%w (moved to %s.bad)", cause, filepath.Base(path))
}

// Save writes v as indented JSON to path through a temp file and rename, so a
// crash mid-write never leaves a truncated file. The directory is created.
func Save(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	return nil
}
