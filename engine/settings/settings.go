// Package settings stores the user's preferences in a small JSON file.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	ThemeSystem   = "system"
	ThemeNocturne = "nocturne"
	ThemeDaylight = "daylight"

	MotionSystem = "system"
	MotionReduce = "reduce"
	MotionFull   = "full"
)

// Settings is everything Spotfile remembers between launches.
type Settings struct {
	Theme  string `json:"theme"`
	Motion string `json:"motion"`
	Folder string `json:"folder"` // the indexed folder; "" when none
}

// Defaults follow the OS for appearance and have no folder.
func Defaults() Settings {
	return Settings{Theme: ThemeSystem, Motion: MotionSystem}
}

// Validate reports an unknown theme or motion value.
func (s Settings) Validate() error {
	switch s.Theme {
	case ThemeSystem, ThemeNocturne, ThemeDaylight:
	default:
		return fmt.Errorf("unknown theme %q", s.Theme)
	}
	switch s.Motion {
	case MotionSystem, MotionReduce, MotionFull:
	default:
		return fmt.Errorf("unknown motion setting %q", s.Motion)
	}
	return nil
}

// WithFolder returns a copy that uses dir as the indexed folder, and whether
// dir is a different folder from the current one.
func (s Settings) WithFolder(dir string) (Settings, bool) {
	changed := s.Folder == "" || filepath.Clean(s.Folder) != filepath.Clean(dir)
	s.Folder = dir
	return s, changed
}

// Load reads settings from path. A missing file gives Defaults. A file that
// can't be used is renamed to path+".bad" and Defaults are returned with an
// error describing the problem, so a bad file never stops the app starting.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Defaults(), nil
	}
	if err != nil {
		return Defaults(), setAside(path, err)
	}
	s := Defaults() // fields missing from the file keep their defaults
	if err := json.Unmarshal(data, &s); err != nil {
		return Defaults(), setAside(path, fmt.Errorf("decode settings: %w", err))
	}
	if err := s.Validate(); err != nil {
		return Defaults(), setAside(path, err)
	}
	return s, nil
}

func setAside(path string, cause error) error {
	if err := os.Rename(path, path+".bad"); err != nil {
		return fmt.Errorf("%w (moving the file aside also failed: %v)", cause, err)
	}
	return fmt.Errorf("%w (moved to %s.bad)", cause, filepath.Base(path))
}

// Save validates s and writes it to path via a temp file and rename, so a
// crash mid-write never leaves a truncated file.
func Save(path string, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp settings: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp settings: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}
	return nil
}
