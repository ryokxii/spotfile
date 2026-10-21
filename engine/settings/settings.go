// Package settings stores the user's preferences in a small JSON file.
package settings

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"spotfile/engine/jsonfile"
	"spotfile/engine/pathutil"
)

const (
	ThemeSystem   = "system"
	ThemeNocturne = "nocturne"
	ThemeDaylight = "daylight"

	MotionSystem = "system"
	MotionReduce = "reduce"
	MotionFull   = "full"

	// MaxNameLength is the longest display name, in characters.
	MaxNameLength = 60
	// DefaultStaleDays is how long a file goes unused before Free up space lists it.
	DefaultStaleDays = 90
)

// StaleDayChoices are the thresholds offered in Settings → Privacy.
var StaleDayChoices = []int{30, 90, 180}

// Settings is everything Spotfile remembers between launches.
type Settings struct {
	Theme     string   `json:"theme"`
	Motion    string   `json:"motion"`
	Folders   []string `json:"folders"`   // indexed folders, in the order they were added
	Files     []string `json:"files"`     // individually indexed files outside those folders
	Name      string   `json:"name"`      // display name; "" when unset
	History   bool     `json:"history"`   // remember searches and opened files
	StaleDays int      `json:"staleDays"` // one of StaleDayChoices
}

// Defaults follow the OS for appearance, have no folders or name, keep
// history, and list files unused for 90 days.
func Defaults() Settings {
	return Settings{
		Theme:     ThemeSystem,
		Motion:    MotionSystem,
		Folders:   []string{},
		Files:     []string{},
		History:   true,
		StaleDays: DefaultStaleDays,
	}
}

// Validate reports a value Spotfile can't use.
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
	for _, f := range s.Files {
		if !filepath.IsAbs(f) {
			return fmt.Errorf("file %q is not an absolute path", f)
		}
	}
	if utf8.RuneCountInString(s.Name) > MaxNameLength {
		return fmt.Errorf("name is longer than %d characters", MaxNameLength)
	}
	if !validStaleDays(s.StaleDays) {
		return fmt.Errorf("stale after %d days isn't one of %v", s.StaleDays, StaleDayChoices)
	}
	return nil
}

func validStaleDays(days int) bool {
	for _, c := range StaleDayChoices {
		if c == days {
			return true
		}
	}
	return false
}

// WithFolderAdded returns a copy that also indexes dir, and whether the list
// changed. A folder inside an indexed one is already covered; a folder that
// contains indexed folders or listed files replaces them.
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
	s.Files = without(s.Files, func(f string) bool { return pathutil.Within(f, dir) })
	return s, true
}

// WithFileAdded returns a copy that also indexes path, and whether the list
// changed. A file inside an indexed folder, or already listed, is covered.
func (s Settings) WithFileAdded(path string) (Settings, bool) {
	path = filepath.Clean(path)
	for _, root := range s.Roots() {
		if pathutil.Within(path, root) {
			s.Files = append([]string{}, s.Files...)
			return s, false
		}
	}
	s.Files = append(append([]string{}, s.Files...), path)
	return s, true
}

// WithFileRemoved returns a copy without path in the file list.
func (s Settings) WithFileRemoved(path string) Settings {
	path = filepath.Clean(path)
	s.Files = without(s.Files, func(f string) bool { return filepath.Clean(f) == path })
	return s
}

// Roots are everything indexed: the folders, then the listed files.
func (s Settings) Roots() []string {
	return append(append([]string{}, s.Folders...), s.Files...)
}

// without returns a new slice of the items drop rejects.
func without(items []string, drop func(string) bool) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if !drop(item) {
			out = append(out, item)
		}
	}
	return out
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

// WithName returns a copy with the display name, trimmed.
func (s Settings) WithName(name string) Settings {
	s.Name = strings.TrimSpace(name)
	return s
}

// WithPrivacy returns a copy with the history switch and the stale threshold.
func (s Settings) WithPrivacy(history bool, staleDays int) Settings {
	s.History = history
	s.StaleDays = staleDays
	return s
}

// Load reads settings from path. A missing file gives Defaults. A file that
// can't be used is renamed to path+".bad" and Defaults are returned with an
// error describing the problem, so a bad file never stops the app starting.
func Load(path string) (Settings, error) {
	// Fields missing from the file keep their defaults. Files written before
	// several folders were supported hold a single "folder".
	file := struct {
		Settings
		Folder string `json:"folder"`
	}{Settings: Defaults()}
	if err := jsonfile.Load(path, &file); err != nil {
		return Defaults(), err
	}
	s := file.Settings
	if s.Folders == nil {
		s.Folders = []string{}
	}
	if s.Files == nil {
		s.Files = []string{}
	}
	if len(s.Folders) == 0 && file.Folder != "" {
		s.Folders = []string{file.Folder}
	}
	if err := s.Validate(); err != nil {
		return Defaults(), jsonfile.SetAside(path, err)
	}
	return s, nil
}

// Save validates s and writes it to path atomically.
func Save(path string, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	return jsonfile.Save(path, s)
}
