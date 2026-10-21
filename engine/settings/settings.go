// Package settings stores the user's preferences in a small JSON file.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"spotfile/engine/pathutil"
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
	Theme   string   `json:"theme"`
	Motion  string   `json:"motion"`
	Folders []string `json:"folders"` // indexed folders, in the order they were added
}

// Defaults follow the OS for appearance and have no folders.
func Defaults() Settings {
	return Settings{Theme: ThemeSystem, Motion: MotionSystem, Folders: []string{}}
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
	for _, f := range s.Folders {
		if !filepath.IsAbs(f) {
			return fmt.Errorf("folder %q is not an absolute path", f)
		}
	}
	return nil
}

// WithFolderAdded returns a copy that also indexes dir, and whether the list
// changed. A folder inside an indexed one is already covered; a folder that
// contains indexed ones replaces them.
func (s Settings) WithFolderAdded(dir string) (Settings, bool) {
	dir = filepath.Clean(dir)
	for _, f := range s.Folders {
		if pathutil.Within(dir, f) {
			s.Folders = append([]string{}, s.Folders...)
			return s, false
		}
	}
	next := make([]string, 0, len(s.Folders)+1)
	for _, f := range s.Folders {
		if !pathutil.Within(f, dir) {
			next = append(next, f)
		}
	}
	s.Folders = append(next, dir)
	return s, true
}

// WithFolderRemoved returns a copy without dir.
func (s Settings) WithFolderRemoved(dir string) Settings {
	dir = filepath.Clean(dir)
	next := make([]string, 0, len(s.Folders))
	for _, f := range s.Folders {
		if filepath.Clean(f) != dir {
			next = append(next, f)
		}
	}
	s.Folders = next
	return s
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
	// Fields missing from the file keep their defaults. Files written before
	// several folders were supported hold a single "folder".
	file := struct {
		Settings
		Folder string `json:"folder"`
	}{Settings: Defaults()}
	if err := json.Unmarshal(data, &file); err != nil {
		return Defaults(), setAside(path, fmt.Errorf("decode settings: %w", err))
	}
	s := file.Settings
	if s.Folders == nil {
		s.Folders = []string{}
	}
	if len(s.Folders) == 0 && file.Folder != "" {
		s.Folders = []string{file.Folder}
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
