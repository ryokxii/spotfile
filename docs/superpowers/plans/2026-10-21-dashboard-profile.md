# Dashboard and Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `docs/superpowers/specs/2026-10-21-dashboard-profile-design.md`: a Bento Dashboard as the launch page, a Profile page with the display name, Search renamed Chat at `/chat`, and a Privacy section in Settings.

**Architecture:** Go owns a local activity log (`engine/activity`, `~/.spotfile/activity.json`) and computes the Dashboard in one binding from the index, the log and the settings. Pure engine packages (`jsonfile`, `activity`, `library`, `trash`) hold the logic and are CI-tested; `App` stays thin. The frontend adds a dashboard store and a profile store, presentational tiles tested in jsdom, and thin container views.

**Tech Stack:** Go 1.26, Wails v2.12, Svelte 3 + TypeScript, Vitest 0.34 (jsdom), svelte-spa-router.

**Branch:** `feat/settings-impl`, after the Settings and Multiple folders plans.

## Deviations from the spec

1. **Shared JSON file helper:** `settings` and the new `activity` both need "load, set aside a bad file, save atomically". That logic moves into `engine/jsonfile`, and `settings` uses it.
2. **`Stale` and file types live in `engine/library`**, not `engine/activity`: they describe indexed files, not history.
3. **Profile and privacy preferences get their own store** (`stores/profile.ts`) instead of growing the appearance store, so each store keeps one save call.
4. **`GetDashboard` returns `Dashboard` without an error.** Nothing in it can fail; a failed IPC call still rejects the promise and the UI shows its error state.
5. **Five file types** (PDF, Markdown, Text, Code, Data) map onto `--color-identity-0…4`, so the "Other" bucket is never needed.
6. **Recently opened rows always offer Forget** (a small × on hover), since a missing file is only discovered when the viewer fails.
7. **Untested in jsdom:** `ViewerPanel` (it mounts pdf.js) and the container views. They are checked in `wails dev` in Task 17.

## File map

| File | Responsibility |
|---|---|
| `engine/jsonfile/jsonfile.go` (new) | `Load`, `SetAside`, `Save` |
| `engine/settings/settings.go` | Uses `jsonfile`; `Name`, `History`, `StaleDays`, `WithName`, `WithPrivacy` |
| `engine/activity/activity.go` (new) | History log and its pure updates, load/save |
| `engine/library/library.go` (new) | `TypeOf`, `CountTypes`, `Stale` |
| `engine/vectorstore/docstats.go` (new) | `DocStats` |
| `engine/trash/*.go` (new) | `Allowed`, `Move`, platform `System.Trash` |
| `engine/indexing/watcher.go` | `onUpdated` callback |
| `activity.go` (new, main) | Activity bindings and recording |
| `dashboard.go` (new, main) | `GetDashboard`, `RemoveFromIndex`, `MoveToTrash`, `SetProfile`, `SetPrivacy` |
| `app.go`, `indexing.go`, `search.go` | Wiring |
| `frontend/src/lib/greeting.ts`, `lib/time.ts` (new) | Greeting, initials, relative times |
| `frontend/src/stores/profile.ts`, `stores/dashboard.ts` (new) | Stores |
| `frontend/src/stores/search.ts`, `stores/app.ts`, `main.ts` | `opened` hook, wiring, boot |
| `frontend/src/ui/ProfileButton.svelte` (new), `ui/Sidebar.svelte`, `ui/icons.ts`, `routes.ts`, `App.svelte` | Navigation |
| `frontend/src/views/profile/*` (new) | `NameField`, `ProfileView` |
| `frontend/src/views/dashboard/*` (new) | Search bar, tiles, stale list, dialogs, viewer panel, view |
| `frontend/src/views/settings/PrivacySection.svelte` (new) | Privacy controls |

Commands (repo root unless noted): `go test ./engine/...`, `cd frontend && npx vitest run <path>`, `cd frontend && npm run check`.

---

### Task 1: `engine/jsonfile`

**Files:**
- Create: `engine/jsonfile/jsonfile.go`, `engine/jsonfile/jsonfile_test.go`

- [x] **Step 1: Write the failing tests**

```go
package jsonfile

import (
	"os"
	"path/filepath"
	"testing"
)

type doc struct {
	A string `json:"a"`
	B int    `json:"b"`
}

func TestLoadMissingFileLeavesValue(t *testing.T) {
	v := doc{A: "kept"}
	if err := Load(filepath.Join(t.TempDir(), "x.json"), &v); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v.A != "kept" {
		t.Fatalf("v = %+v", v)
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "x.json")
	if err := Save(path, doc{A: "a", B: 2}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var got doc
	if err := Load(path, &got); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != (doc{A: "a", B: 2}) {
		t.Fatalf("got %+v", got)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("dir holds %v, want only x.json", entries)
	}
}

func TestLoadSetsAsideCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.json")
	if err := os.WriteFile(path, []byte(`{"a":`), 0o644); err != nil {
		t.Fatal(err)
	}
	var v doc
	if err := Load(path, &v); err == nil {
		t.Fatal("Load returned no error for a corrupt file")
	}
	if _, err := os.Stat(path + ".bad"); err != nil {
		t.Fatalf("not set aside: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("corrupt file still in place: %v", err)
	}
}
```

- [x] **Step 2: Run to confirm failure**

Run: `go test ./engine/jsonfile/`
Expected: FAIL, `undefined: Load`.

- [x] **Step 3: Implement**

```go
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
```

- [x] **Step 4: Run to confirm pass**

Run: `go test ./engine/jsonfile/`
Expected: `ok`.

- [x] **Step 5: Commit**

```bash
git add engine/jsonfile
git commit -m "feat(jsonfile): shared atomic JSON load and save"
```

---

### Task 2: Settings: profile and privacy fields, on `jsonfile`

**Files:**
- Modify: `engine/settings/settings.go` (replace whole file), `engine/settings/settings_test.go` (replace whole file)

- [x] **Step 1: Replace the tests**

```go
package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, Defaults()) {
		t.Fatalf("got %+v, want defaults", got)
	}
}

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.Folders == nil || len(d.Folders) != 0 || d.Name != "" || !d.History || d.StaleDays != 90 {
		t.Fatalf("Defaults() = %+v", d)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := Defaults()
	want.Theme, want.Motion = ThemeDaylight, MotionReduce
	want.Folders = []string{"/Users/me/Notes", "/Users/me/Work"}
	want.Name, want.History, want.StaleDays = "Zafer Acar", false, 180
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadMigratesTheOldFolderAndFillsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"folder":"/notes"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Defaults()
	want.Folders = []string{"/notes"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLegacyFolderIsNotSavedBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"folder":"/notes"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"folders": [`) || strings.Contains(string(data), `"folder":`) {
		t.Fatalf("saved %s", data)
	}
}

func TestLoadSetsAsideUnusableFiles(t *testing.T) {
	for name, content := range map[string]string{
		"corrupt json":  `{"theme":`,
		"unknown theme": `{"theme":"sepia","motion":"system"}`,
		"stale days":    `{"staleDays":45}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := Load(path)
			if err == nil {
				t.Fatal("Load returned no error for an unusable file")
			}
			if !reflect.DeepEqual(got, Defaults()) {
				t.Fatalf("got %+v, want defaults", got)
			}
			if _, err := os.Stat(path + ".bad"); err != nil {
				t.Fatalf("unusable file not kept as .bad: %v", err)
			}
		})
	}
}

func TestSaveRejectsInvalidValues(t *testing.T) {
	invalid := func(change func(*Settings)) Settings {
		s := Defaults()
		change(&s)
		return s
	}
	for name, s := range map[string]Settings{
		"theme":           invalid(func(s *Settings) { s.Theme = "sepia" }),
		"motion":          invalid(func(s *Settings) { s.Motion = "slow" }),
		"relative folder": invalid(func(s *Settings) { s.Folders = []string{"notes"} }),
		"stale days":      invalid(func(s *Settings) { s.StaleDays = 45 }),
		"long name":       invalid(func(s *Settings) { s.Name = strings.Repeat("é", MaxNameLength+1) }),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := Save(path, s); err == nil {
				t.Fatal("Save accepted invalid settings")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("file written for invalid settings: %v", err)
			}
		})
	}
}

func TestNameAtTheLimitIsValid(t *testing.T) {
	s := Defaults().WithName(strings.Repeat("é", MaxNameLength))
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestWithName(t *testing.T) {
	start := Defaults()
	got := start.WithName("  Zafer Acar \n")
	if got.Name != "Zafer Acar" || start.Name != "" {
		t.Fatalf("got %q, receiver %q", got.Name, start.Name)
	}
}

func TestWithPrivacy(t *testing.T) {
	start := Defaults()
	got := start.WithPrivacy(false, 30)
	if got.History || got.StaleDays != 30 || !start.History || start.StaleDays != 90 {
		t.Fatalf("got %+v, receiver %+v", got, start)
	}
}

func TestWithFolderAdded(t *testing.T) {
	for _, tc := range []struct {
		name        string
		from        []string
		dir         string
		want        []string
		wantChanged bool
	}{
		{"first folder", []string{}, "/notes", []string{"/notes"}, true},
		{"another folder", []string{"/notes"}, "/work", []string{"/notes", "/work"}, true},
		{"same folder, trailing slash", []string{"/notes"}, "/notes/", []string{"/notes"}, false},
		{"inside an indexed folder", []string{"/notes"}, "/notes/2026", []string{"/notes"}, false},
		{"contains indexed folders", []string{"/notes/a", "/work", "/notes/b"}, "/notes", []string{"/work", "/notes"}, true},
		{"sibling sharing a prefix", []string{"/notes"}, "/notes-old", []string{"/notes", "/notes-old"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := Defaults()
			start.Folders = append([]string{}, tc.from...)
			got, changed := start.WithFolderAdded(tc.dir)
			if changed != tc.wantChanged {
				t.Fatalf("changed = %v, want %v", changed, tc.wantChanged)
			}
			if !reflect.DeepEqual(got.Folders, tc.want) {
				t.Fatalf("Folders = %v, want %v", got.Folders, tc.want)
			}
			if !reflect.DeepEqual(start.Folders, tc.from) {
				t.Fatalf("receiver modified: %v", start.Folders)
			}
		})
	}
}

func TestWithFolderRemoved(t *testing.T) {
	start := Defaults()
	start.Folders = []string{"/notes", "/work"}
	got := start.WithFolderRemoved("/notes/")
	if !reflect.DeepEqual(got.Folders, []string{"/work"}) {
		t.Fatalf("Folders = %v", got.Folders)
	}
	if len(start.Folders) != 2 {
		t.Fatalf("receiver modified: %v", start.Folders)
	}
}
```

- [x] **Step 2: Run to confirm failure**

Run: `go test ./engine/settings/`
Expected: FAIL, `s.Name undefined`, `undefined: MaxNameLength`.

- [x] **Step 3: Replace `settings.go`**

```go
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
```

- [x] **Step 4: Run to confirm pass**

Run: `go test ./engine/settings/ && go build ./...`
Expected: `ok`, and the app still builds.

- [x] **Step 5: Commit**

```bash
git add engine/settings
git commit -m "feat(settings): display name, history switch and stale threshold"
```

---

### Task 3: `engine/activity`

**Files:**
- Create: `engine/activity/activity.go`, `engine/activity/activity_test.go`

- [x] **Step 1: Write the failing tests**

```go
package activity

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)

func at(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }

func queries(l Log) []string {
	out := make([]string, len(l.Searches))
	for i, s := range l.Searches {
		out[i] = s.Query
	}
	return out
}

func TestWithSearch(t *testing.T) {
	l := WithSearch(Empty(), "grant deadlines", at(1))
	l = WithSearch(l, "invoice", at(2))
	l = WithSearch(l, "  Grant Deadlines ", at(3)) // a repeat, ignoring case and spaces
	l = WithSearch(l, "   ", at(4))                // blank: ignored

	if got, want := queries(l), []string{"Grant Deadlines", "invoice"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("queries = %v, want %v", got, want)
	}
	if !l.Searches[0].At.Equal(at(3)) {
		t.Fatalf("top search at %v", l.Searches[0].At)
	}
}

func TestListsAreCapped(t *testing.T) {
	l := Empty()
	for i := range MaxEntries + 5 {
		l = WithSearch(l, fmt.Sprintf("q%d", i), at(i))
		l = WithOpened(l, fmt.Sprintf("/n/%d.md", i), at(i))
	}
	if len(l.Searches) != MaxEntries || len(l.Opened) != MaxEntries {
		t.Fatalf("lengths %d, %d", len(l.Searches), len(l.Opened))
	}
	if l.Searches[0].Query != fmt.Sprintf("q%d", MaxEntries+4) {
		t.Fatalf("newest first expected, got %q", l.Searches[0].Query)
	}
}

func TestWithOpenedKeepsOneEntryPerPath(t *testing.T) {
	l := WithOpened(Empty(), "/n/a.md", at(1))
	l = WithOpened(l, "/n/b.md", at(2))
	l = WithOpened(l, "/n/a.md", at(3))
	if len(l.Opened) != 2 || l.Opened[0].Path != "/n/a.md" || !l.Opened[0].At.Equal(at(3)) {
		t.Fatalf("opened = %+v", l.Opened)
	}
}

func TestWithoutOpenedAndLastOpened(t *testing.T) {
	l := WithOpened(WithOpened(Empty(), "/n/a.md", at(1)), "/n/b.md", at(2))
	l = WithoutOpened(l, "/n/a.md")
	got := LastOpened(l)
	if len(got) != 1 || !got["/n/b.md"].Equal(at(2)) {
		t.Fatalf("LastOpened = %v", got)
	}
}

func TestClearedKeepsLastUpdate(t *testing.T) {
	l := WithUpdate(WithOpened(WithSearch(Empty(), "q", at(1)), "/n/a.md", at(1)), "/n/a.md", at(2))
	c := Cleared(l)
	if len(c.Searches) != 0 || len(c.Opened) != 0 || c.LastUpdate == nil || c.LastUpdate.Path != "/n/a.md" {
		t.Fatalf("Cleared = %+v", c)
	}
}

func TestUpdatesNeverChangeTheirInput(t *testing.T) {
	l := WithUpdate(WithOpened(WithSearch(Empty(), "q", at(1)), "/n/a.md", at(1)), "/n/a.md", at(1))
	before := l.clone()
	_ = WithSearch(l, "other", at(2))
	_ = WithOpened(l, "/n/b.md", at(2))
	_ = WithoutOpened(l, "/n/a.md")
	_ = WithUpdate(l, "/n/b.md", at(2))
	_ = Cleared(l)
	if !reflect.DeepEqual(l, before) {
		t.Fatalf("input changed: %+v", l)
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity.json")
	want := WithUpdate(WithOpened(WithSearch(Empty(), "q", at(1)), "/n/a.md", at(2)), "/n/a.md", at(3))
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Searches) != 1 || got.Opened[0].Path != "/n/a.md" || !got.LastUpdate.At.Equal(at(3)) {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadMissingOrBadFile(t *testing.T) {
	dir := t.TempDir()
	got, err := Load(filepath.Join(dir, "none.json"))
	if err != nil || got.Searches == nil || got.Opened == nil || got.LastUpdate != nil {
		t.Fatalf("missing: %+v, %v", got, err)
	}

	bad := filepath.Join(dir, "activity.json")
	if err := os.WriteFile(bad, []byte(`{"searches":`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(bad)
	if err == nil || len(got.Searches) != 0 {
		t.Fatalf("bad: %+v, %v", got, err)
	}
	if _, err := os.Stat(bad + ".bad"); err != nil {
		t.Fatalf("not set aside: %v", err)
	}
}
```

- [x] **Step 2: Run to confirm failure**

Run: `go test ./engine/activity/`
Expected: FAIL, `undefined: Empty`.

- [x] **Step 3: Implement**

```go
// Package activity keeps the Dashboard's short local history: recent
// searches, recently opened files and the last file the index updated. It is
// stored only on this computer, in ~/.spotfile/activity.json.
package activity

import (
	"strings"
	"time"

	"spotfile/engine/jsonfile"
)

// MaxEntries caps each history list.
const MaxEntries = 50

// Search is a query that returned results.
type Search struct {
	Query string    `json:"query"`
	At    time.Time `json:"at"`
}

// Opened is a file opened in a viewer.
type Opened struct {
	Path string    `json:"path"`
	At   time.Time `json:"at"`
}

// Update is the last file the index (re)indexed.
type Update struct {
	Path string    `json:"path"`
	At   time.Time `json:"at"`
}

// Log is the whole history. Every function in this package returns a new Log
// and leaves its argument untouched.
type Log struct {
	Searches   []Search `json:"searches"`   // newest first
	Opened     []Opened `json:"opened"`     // newest first, one entry per path
	LastUpdate *Update  `json:"lastUpdate"` // nil until the index first changes
}

// Empty is a log with no history.
func Empty() Log {
	return Log{Searches: []Search{}, Opened: []Opened{}}
}

func (l Log) clone() Log {
	out := Log{Searches: append([]Search{}, l.Searches...), Opened: append([]Opened{}, l.Opened...)}
	if l.LastUpdate != nil {
		u := *l.LastUpdate
		out.LastUpdate = &u
	}
	return out
}

// WithSearch puts query at the top of the searches. A blank query is ignored;
// a repeat (ignoring case and surrounding spaces) moves to the top.
func WithSearch(l Log, query string, now time.Time) Log {
	out := l.clone()
	query = strings.TrimSpace(query)
	if query == "" {
		return out
	}
	next := []Search{{Query: query, At: now}}
	for _, s := range l.Searches {
		if len(next) == MaxEntries {
			break
		}
		if !strings.EqualFold(s.Query, query) {
			next = append(next, s)
		}
	}
	out.Searches = next
	return out
}

// WithOpened puts path at the top of the opened files.
func WithOpened(l Log, path string, now time.Time) Log {
	out := l.clone()
	next := []Opened{{Path: path, At: now}}
	for _, o := range l.Opened {
		if len(next) == MaxEntries {
			break
		}
		if o.Path != path {
			next = append(next, o)
		}
	}
	out.Opened = next
	return out
}

// WithoutOpened forgets path from the opened files.
func WithoutOpened(l Log, path string) Log {
	out := l.clone()
	next := make([]Opened, 0, len(l.Opened))
	for _, o := range l.Opened {
		if o.Path != path {
			next = append(next, o)
		}
	}
	out.Opened = next
	return out
}

// WithUpdate records path as the last file the index updated.
func WithUpdate(l Log, path string, now time.Time) Log {
	out := l.clone()
	out.LastUpdate = &Update{Path: path, At: now}
	return out
}

// Cleared forgets searches and opened files. The last update describes the
// library rather than the user, so it is kept.
func Cleared(l Log) Log {
	out := l.clone()
	out.Searches = []Search{}
	out.Opened = []Opened{}
	return out
}

// LastOpened maps each opened file to when it was last opened.
func LastOpened(l Log) map[string]time.Time {
	out := make(map[string]time.Time, len(l.Opened))
	for _, o := range l.Opened {
		out[o.Path] = o.At
	}
	return out
}

// Load reads the log. A missing file gives an empty log; an unusable one is
// set aside and an empty log is returned with the error.
func Load(path string) (Log, error) {
	l := Empty()
	if err := jsonfile.Load(path, &l); err != nil {
		return Empty(), err
	}
	if l.Searches == nil {
		l.Searches = []Search{}
	}
	if l.Opened == nil {
		l.Opened = []Opened{}
	}
	return l, nil
}

// Save writes the log atomically.
func Save(path string, l Log) error {
	return jsonfile.Save(path, l)
}
```

- [x] **Step 4: Run to confirm pass**

Run: `go test -race ./engine/activity/`
Expected: `ok`.

- [x] **Step 5: Commit**

```bash
git add engine/activity
git commit -m "feat(activity): local history of searches, opened files and updates"
```

---

### Task 4: `engine/library`

**Files:**
- Create: `engine/library/library.go`, `engine/library/library_test.go`

- [x] **Step 1: Write the failing tests**

```go
package library

import (
	"reflect"
	"testing"
	"time"
)

func TestTypeOf(t *testing.T) {
	for path, want := range map[string]string{
		"/n/a.PDF":      "PDF",
		"/n/b.md":       "Markdown",
		"/n/c.markdown": "Markdown",
		"/n/d.go":       "Code",
		"/n/e.html":     "Code",
		"/n/f.yaml":     "Data",
		"/n/g.txt":      "Text",
		"/n/h.rst":      "Text",
	} {
		if got := TypeOf(path); got != want {
			t.Errorf("TypeOf(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestCountTypes(t *testing.T) {
	got := CountTypes([]string{"/a.pdf", "/b.pdf", "/c.md", "/d.txt", "/e.go", "/f.pdf"})
	want := []TypeCount{{"PDF", 3}, {"Code", 1}, {"Markdown", 1}, {"Text", 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CountTypes = %v, want %v", got, want)
	}
	if got := CountTypes(nil); len(got) != 0 {
		t.Fatalf("CountTypes(nil) = %v", got)
	}
}

func TestStale(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	daysAgo := func(d int) time.Time { return now.AddDate(0, 0, -d) }
	files := []File{
		{Path: "/n/fresh.md", Bytes: 900, Modified: daysAgo(10)},
		{Path: "/n/old-small.md", Bytes: 10, Modified: daysAgo(200)},
		{Path: "/n/old-big.pdf", Bytes: 500, Modified: daysAgo(400)},
		{Path: "/n/old-opened.md", Bytes: 700, Modified: daysAgo(300)},
		{Path: "/n/old-opened-long-ago.md", Bytes: 20, Modified: daysAgo(300)},
		{Path: "/n/edge.md", Bytes: 30, Modified: daysAgo(90)},
	}
	opened := map[string]time.Time{
		"/n/old-opened.md":          daysAgo(5),
		"/n/old-opened-long-ago.md": daysAgo(120),
	}

	got := Stale(files, opened, 90, now)
	var paths []string
	for _, f := range got {
		paths = append(paths, f.Path)
	}
	want := []string{"/n/old-big.pdf", "/n/old-opened-long-ago.md", "/n/old-small.md"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("Stale = %v, want %v", paths, want)
	}
	if files[0].Path != "/n/fresh.md" {
		t.Fatal("input reordered")
	}
}
```

- [x] **Step 2: Run to confirm failure**

Run: `go test ./engine/library/`
Expected: FAIL, `undefined: TypeOf`.

- [x] **Step 3: Implement**

```go
// Package library computes the Dashboard's figures about indexed files: how
// many there are of each type, and which haven't been used in a while.
package library

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TypeCount is how many indexed files are of one type.
type TypeCount struct {
	Type  string `json:"type"`
	Files int    `json:"files"`
}

// typeByExt groups the indexed extensions (see isIndexable in files.go).
// Anything not listed is Text.
var typeByExt = map[string]string{
	".pdf": "PDF",
	".md":  "Markdown", ".markdown": "Markdown",
	".csv": "Data", ".tsv": "Data", ".json": "Data", ".jsonc": "Data", ".yaml": "Data", ".yml": "Data",
	".toml": "Data", ".ini": "Data", ".cfg": "Data", ".conf": "Data", ".env": "Data", ".log": "Data", ".xml": "Data",
	".html": "Code", ".htm": "Code", ".css": "Code", ".scss": "Code", ".svelte": "Code", ".vue": "Code",
	".js": "Code", ".jsx": "Code", ".ts": "Code", ".tsx": "Code", ".go": "Code", ".py": "Code",
	".rb": "Code", ".rs": "Code", ".java": "Code", ".kt": "Code", ".c": "Code", ".h": "Code",
	".cpp": "Code", ".hpp": "Code", ".cc": "Code", ".cs": "Code", ".php": "Code", ".swift": "Code",
	".sh": "Code", ".bash": "Code", ".zsh": "Code", ".sql": "Code",
}

// TypeOf names a file's type for the Dashboard: PDF, Markdown, Code, Data or Text.
func TypeOf(path string) string {
	if t, ok := typeByExt[strings.ToLower(filepath.Ext(path))]; ok {
		return t
	}
	return "Text"
}

// CountTypes counts paths per type, most common first, ties by name.
func CountTypes(paths []string) []TypeCount {
	counts := make(map[string]int)
	for _, p := range paths {
		counts[TypeOf(p)]++
	}
	out := make([]TypeCount, 0, len(counts))
	for t, n := range counts {
		out = append(out, TypeCount{Type: t, Files: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Files != out[j].Files {
			return out[i].Files > out[j].Files
		}
		return out[i].Type < out[j].Type
	})
	return out
}

// File is an indexed file as Free up space sees it.
type File struct {
	Path     string    `json:"path"`
	Bytes    int64     `json:"bytes"`    // estimated share of the index
	Modified time.Time `json:"modified"` // on disk
}

// Stale returns the files neither modified nor opened in Spotfile within the
// last days, largest first, ties by path. The input is not modified.
func Stale(files []File, lastOpened map[string]time.Time, days int, now time.Time) []File {
	cutoff := now.AddDate(0, 0, -days)
	out := []File{}
	for _, f := range files {
		if !f.Modified.Before(cutoff) {
			continue
		}
		if opened, ok := lastOpened[f.Path]; ok && !opened.Before(cutoff) {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes != out[j].Bytes {
			return out[i].Bytes > out[j].Bytes
		}
		return out[i].Path < out[j].Path
	})
	return out
}
```

- [x] **Step 4: Run to confirm pass**

Run: `go test ./engine/library/`
Expected: `ok`.

- [x] **Step 5: Commit**

```bash
git add engine/library
git commit -m "feat(library): file types and stale files for the Dashboard"
```

---

### Task 5: `VectorStore.DocStats`

**Files:**
- Create: `engine/vectorstore/docstats.go`, `engine/vectorstore/docstats_test.go`

- [x] **Step 1: Write the failing test**

```go
package vectorstore

import "testing"

func TestDocStats(t *testing.T) {
	store := new(VectorStore)
	store.Add(EmbeddedChunk{DocPath: "/n/b.md", Text: "hello", Embedding: []float32{1, 2}})
	store.Add(EmbeddedChunk{DocPath: "/n/b.md", Text: "abc", Embedding: []float32{1, 2}})
	store.Add(EmbeddedChunk{DocPath: "/n/a.md", Text: "", Embedding: []float32{1, 2, 3}})

	got := store.DocStats()
	want := []DocStat{
		{Path: "/n/a.md", Chunks: 1, Bytes: 12},
		{Path: "/n/b.md", Chunks: 2, Bytes: (8 + 5) + (8 + 3)},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("DocStats = %+v, want %+v", got, want)
	}
}
```

- [x] **Step 2: Run to confirm failure**

Run: `go test ./engine/vectorstore/ -run DocStats`
Expected: FAIL, `store.DocStats undefined`.

- [x] **Step 3: Implement**

```go
package vectorstore

import "sort"

// DocStat is one indexed file's share of the index.
type DocStat struct {
	Path   string
	Chunks int
	Bytes  int64 // estimate: 4 bytes per vector component plus the chunk text
}

// DocStats returns every document with chunks, sorted by path.
func (vs *VectorStore) DocStats() []DocStat {
	vs.mu.RLock()
	out := make([]DocStat, 0, len(vs.docs))
	for p, chunks := range vs.docs {
		var b int64
		for _, c := range chunks {
			b += int64(len(c.Embedding))*4 + int64(len(c.Text))
		}
		out = append(out, DocStat{Path: p, Chunks: len(chunks), Bytes: b})
	}
	vs.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
```

- [x] **Step 4: Run to confirm pass**

Run: `go test ./engine/vectorstore/`
Expected: `ok`.

- [x] **Step 5: Commit**

```bash
git add engine/vectorstore
git commit -m "feat(vectorstore): per-file chunk counts and index size"
```

---

### Task 6: `engine/trash`

**Files:**
- Create: `engine/trash/trash.go`, `engine/trash/trash_darwin.go`, `engine/trash/trash_windows.go`, `engine/trash/trash_other.go`, `engine/trash/trash_test.go`

- [x] **Step 1: Write the failing tests**

```go
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
```

- [x] **Step 2: Run to confirm failure**

Run: `go test ./engine/trash/`
Expected: FAIL, `undefined: Allowed`.

- [x] **Step 3: Implement `trash.go`**

```go
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
```

- [x] **Step 4: Implement `trash_darwin.go`**

```go
//go:build darwin

package trash

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>

// trashItem moves path to the user's Trash. It returns NULL on success, or a
// malloc'd error message the caller frees.
static char *trashItem(const char *path) {
	@autoreleasepool {
		NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSError *error = nil;
		if ([[NSFileManager defaultManager] trashItemAtURL:url resultingItemURL:nil error:&error]) {
			return NULL;
		}
		NSString *message = error.localizedDescription ?: @"unknown error";
		return strdup(message.UTF8String);
	}
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

// Trash moves path to the Trash through NSFileManager, which needs no
// permission prompt and lets the user put the file back from Finder.
func (System) Trash(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	if msg := C.trashItem(cpath); msg != nil {
		defer C.free(unsafe.Pointer(msg))
		return errors.New(C.GoString(msg))
	}
	return nil
}
```

- [x] **Step 5: Implement `trash_windows.go`**

```go
//go:build windows

package trash

import (
	"fmt"
	"syscall"
	"unsafe"
)

// SHFileOperationW flags: delete with undo (Recycle Bin), no UI.
const (
	foDelete          = 0x0003
	fofSilent         = 0x0004
	fofNoConfirmation = 0x0010
	fofAllowUndo      = 0x0040
	fofNoErrorUI      = 0x0400
)

// shFileOpStruct mirrors SHFILEOPSTRUCTW (natural alignment on 64-bit Windows).
type shFileOpStruct struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 *uint16
	pTo                   *uint16
	fFlags                uint16
	fAnyOperationsAborted int32
	hNameMappings         uintptr
	lpszProgressTitle     *uint16
}

var procSHFileOperationW = syscall.NewLazyDLL("shell32.dll").NewProc("SHFileOperationW")

// Trash moves path to the Recycle Bin. With FOF_ALLOWUNDO the shell refuses
// rather than deleting permanently when the volume has no Recycle Bin.
func (System) Trash(path string) error {
	from, err := syscall.UTF16FromString(path)
	if err != nil {
		return err
	}
	from = append(from, 0) // pFrom is a list ending with an extra NUL
	op := shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI,
	}
	if code, _, _ := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&op))); code != 0 {
		return fmt.Errorf("Recycle Bin refused the file (code %d)", code)
	}
	if op.fAnyOperationsAborted != 0 {
		return fmt.Errorf("moving to the Recycle Bin was cancelled")
	}
	return nil
}
```

- [x] **Step 6: Implement `trash_other.go`**

```go
//go:build !darwin && !windows

package trash

// Trash is unavailable here; nothing is deleted.
func (System) Trash(string) error {
	return ErrUnsupported
}
```

- [x] **Step 7: Run to confirm pass, and cross-compile**

```bash
go test -race ./engine/trash/
GOOS=windows GOARCH=amd64 go vet ./engine/trash/
GOOS=linux GOARCH=amd64 go vet ./engine/trash/
```

Expected: `ok`, and both vets print nothing.

- [x] **Step 8: Commit**

```bash
git add engine/trash
git commit -m "feat(trash): move indexed files to the system Trash, never delete"
```

---

### Task 7: Watcher reports updated files

**Files:**
- Modify: `engine/indexing/watcher.go`, `indexing.go`

- [x] **Step 1: Add the callback**

In `engine/indexing/watcher.go`:
- Add a field to `FileWatcher` after `indexer *Indexer`: `onUpdated func(path string) // called after a changed file is re-indexed; may be nil`
- Change the signature to `func StartWatcher(ctx context.Context, paths []string, indexer *Indexer, onUpdated func(path string)) (*FileWatcher, error)` and set `onUpdated: onUpdated,` in the struct literal; update the doc comment's last sentence to `Changed files are re-indexed through the shared indexer at urgent priority, and onUpdated (if set) hears about each one.`
- In `reindexPath`'s `OnDone`, after the log line, add:

```go
				if fw.onUpdated != nil {
					fw.onUpdated(path)
				}
```

In `indexing.go`'s `restartWatcher`, pass `a.recordUpdate` as the fourth argument. (`recordUpdate` arrives in Task 8; build after Task 8.)

- [x] **Step 2: Run the indexing tests**

Run: `go test ./engine/indexing/`
Expected: `ok`.

- [x] **Step 3: Commit with Task 8** (the app package builds only once `recordUpdate` exists).

---

### Task 8: App activity bindings

**Files:**
- Create: `activity.go`
- Modify: `app.go`, `indexing.go`, `search.go`

- [x] **Step 1: Create `activity.go`**

```go
package main

// Activity: the Dashboard's local history (recent searches, opened files and
// the last file the index updated), kept in ~/.spotfile/activity.json.

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"spotfile/engine/activity"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func activityPath() string {
	return filepath.Join(spotfileDir(), "activity.json")
}

// loadActivity reads the history at startup. An unusable file is set aside and
// history starts empty, so it never blocks launch.
func (a *App) loadActivity() {
	l, err := activity.Load(activityPath())
	if err != nil {
		log.Printf("activity: %v; starting empty", err)
	}
	a.activityMu.Lock()
	a.activity = l
	a.activityMu.Unlock()
}

func (a *App) activitySnapshot() activity.Log {
	a.activityMu.Lock()
	defer a.activityMu.Unlock()
	return a.activity
}

// changeActivity applies change, saves the history and tells the Dashboard.
// The change stays in memory even if saving fails; the error is returned.
// Emits: "activity:changed".
func (a *App) changeActivity(change func(activity.Log) activity.Log) error {
	a.activityMu.Lock()
	a.activity = change(a.activity)
	err := activity.Save(activityPath(), a.activity) // under the lock: writes stay in order
	a.activityMu.Unlock()
	wailsruntime.EventsEmit(a.ctx, "activity:changed", nil)
	if err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	return nil
}

// recordSearch remembers a successful search when history is on.
func (a *App) recordSearch(query string) {
	if !a.GetSettings().History {
		return
	}
	err := a.changeActivity(func(l activity.Log) activity.Log { return activity.WithSearch(l, query, time.Now()) })
	if err != nil {
		log.Printf("activity: %v", err)
	}
}

// recordUpdate remembers the last file the index updated. It is kept even with
// history off: it describes the library, not the user.
func (a *App) recordUpdate(path string) {
	err := a.changeActivity(func(l activity.Log) activity.Log { return activity.WithUpdate(l, path, time.Now()) })
	if err != nil {
		log.Printf("activity: %v", err)
	}
}

// RecordOpened remembers a file opened in a viewer. Nothing is kept with
// history off.
func (a *App) RecordOpened(path string) error {
	if !a.GetSettings().History {
		return nil
	}
	return a.changeActivity(func(l activity.Log) activity.Log { return activity.WithOpened(l, path, time.Now()) })
}

// ForgetOpened removes a file from the recently opened list.
func (a *App) ForgetOpened(path string) error {
	return a.changeActivity(func(l activity.Log) activity.Log { return activity.WithoutOpened(l, path) })
}

// ClearHistory forgets recent searches and opened files.
func (a *App) ClearHistory() error {
	return a.changeActivity(activity.Cleared)
}
```

- [x] **Step 2: Wire `app.go`**

- Add imports `"spotfile/engine/activity"` and `"spotfile/engine/trash"`.
- Add to the end of `App`:

```go
	activityMu sync.Mutex // guards activity
	activity   activity.Log
	trasher    trash.Trasher
```

- In `NewApp`, return `&App{store: store, settings: settings.Defaults(), activity: activity.Empty(), trasher: trash.System{}}`.
- In `startup`, call `a.loadActivity()` right after `a.loadSettings()`.

- [x] **Step 3: Record searches and index updates**

In `search.go`, in `Search`, directly before the final `return results, nil`, add `a.recordSearch(query)`.

In `indexing.go`, replace `progressHooks` with:

```go
// progressHooks emits index:chunk per stored chunk and index:done at the end,
// records the job's last file as the latest update, then runs onDone.
func (a *App) progressHooks(onDone func(jobTotal int)) indexing.Hooks {
	var last string // hooks run one at a time on the indexer's worker
	return indexing.Hooks{
		OnChunk: func(chunk vectorstore.EmbeddedChunk, jobTotal int) {
			last = chunk.DocPath
			wailsruntime.EventsEmit(a.ctx, "index:chunk", map[string]any{
				"total": jobTotal,
				"path":  chunk.DocPath,
			})
		},
		OnDone: func(jobTotal int) {
			wailsruntime.EventsEmit(a.ctx, "index:done", jobTotal)
			if last != "" {
				a.recordUpdate(last)
			}
			onDone(jobTotal)
		},
	}
}
```

- [x] **Step 4: Build, vet and test**

```bash
gofmt -l . ; go vet . ./engine/... && go build ./... && go test ./engine/...
```

Expected: nothing from `gofmt`, `ok` for all packages.

- [x] **Step 5: Commit (Tasks 7 and 8)**

```bash
git add engine/indexing/watcher.go activity.go app.go indexing.go search.go
git commit -m "feat(app): record searches, opened files and index updates"
```

---

### Task 9: App Dashboard, profile and privacy bindings

**Files:**
- Create: `dashboard.go`
- Regenerate: `frontend/wailsjs/go/**`

- [x] **Step 1: Create `dashboard.go`**

```go
package main

// Dashboard bindings: the launch page's figures and history, the actions on
// stale files, and the profile and privacy preferences.

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"spotfile/engine/activity"
	"spotfile/engine/library"
	"spotfile/engine/settings"
	"spotfile/engine/trash"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// recentLimit is how many recent searches and opened files the Dashboard shows.
const recentLimit = 5

// Dashboard is everything the launch page shows, computed in one call.
type Dashboard struct {
	Folders        []FolderStat        `json:"folders"`
	IndexBytes     int64               `json:"indexBytes"`
	Files          int                 `json:"files"`
	Chunks         int                 `json:"chunks"`
	Types          []library.TypeCount `json:"types"`
	LastUpdate     *activity.Update    `json:"lastUpdate"`
	RecentSearches []activity.Search   `json:"recentSearches"`
	RecentFiles    []activity.Opened   `json:"recentFiles"`
	Stale          []library.File      `json:"stale"`
	StaleBytes     int64               `json:"staleBytes"`
	History        bool                `json:"history"`
	StaleDays      int                 `json:"staleDays"`
}

// GetDashboard computes the Dashboard from the index, the history and the
// settings. Files missing from disk are left out of Free up space.
func (a *App) GetDashboard() Dashboard {
	s := a.GetSettings()
	history := a.activitySnapshot()
	docs := a.store.DocStats()

	paths := make([]string, len(docs))
	files := make([]library.File, 0, len(docs))
	for i, d := range docs {
		paths[i] = d.Path
		if info, err := os.Stat(d.Path); err == nil {
			files = append(files, library.File{Path: d.Path, Bytes: d.Bytes, Modified: info.ModTime()})
		}
	}
	stale := library.Stale(files, activity.LastOpened(history), s.StaleDays, time.Now())
	var staleBytes int64
	for _, f := range stale {
		staleBytes += f.Bytes
	}

	d := Dashboard{
		Folders:        a.folderStats(),
		IndexBytes:     a.store.DiskSize(),
		Files:          len(docs),
		Chunks:         a.store.Len(),
		Types:          library.CountTypes(paths),
		LastUpdate:     history.LastUpdate,
		RecentSearches: []activity.Search{},
		RecentFiles:    []activity.Opened{},
		Stale:          stale,
		StaleBytes:     staleBytes,
		History:        s.History,
		StaleDays:      s.StaleDays,
	}
	if s.History {
		d.RecentSearches = first(history.Searches, recentLimit)
		d.RecentFiles = first(history.Opened, recentLimit)
	}
	return d
}

// first returns a copy of at most n leading items.
func first[T any](items []T, n int) []T {
	if len(items) > n {
		items = items[:n]
	}
	return append([]T{}, items...)
}

func (a *App) isIndexed(path string) bool {
	_, ok := a.store.Stamps()[path]
	return ok
}

// RemoveFromIndex drops one file from the index. The file stays on disk, and
// comes back if it changes or its folder is added again.
// Emits: "index:changed".
func (a *App) RemoveFromIndex(path string) error {
	if !a.indexMu.TryLock() {
		return fmt.Errorf("indexing is in progress")
	}
	defer a.indexMu.Unlock()
	if !a.isIndexed(path) {
		return fmt.Errorf("%s isn't in the index", filepath.Base(path))
	}
	a.store.RemoveDocs([]string{path})
	a.persistChanged()
	log.Printf("index: removed file %q", path)
	return nil
}

// MoveToTrash moves an indexed file to the system Trash and drops it from the
// index. Anything outside the indexed folders is refused, and it never falls
// back to deleting permanently.
// Emits: "index:changed", "activity:changed".
func (a *App) MoveToTrash(path string) error {
	if !a.indexMu.TryLock() {
		return fmt.Errorf("indexing is in progress")
	}
	defer a.indexMu.Unlock()
	if err := trash.Move(a.trasher, path, a.GetSettings().Folders, a.isIndexed); err != nil {
		log.Printf("trash: %v", err)
		return err
	}
	a.store.RemoveDocs([]string{path})
	a.persistChanged()
	if err := a.changeActivity(func(l activity.Log) activity.Log { return activity.WithoutOpened(l, path) }); err != nil {
		log.Printf("activity: %v", err)
	}
	log.Printf("trash: moved %q", path)
	return nil
}

// SetProfile saves the display name, trimmed.
func (a *App) SetProfile(name string) error {
	return a.updateSettings(func(s settings.Settings) settings.Settings { return s.WithName(name) })
}

// SetPrivacy saves the history switch and the stale threshold. Turning history
// off keeps what was recorded; ClearHistory forgets it.
// Emits: "activity:changed", so the Dashboard reloads.
func (a *App) SetPrivacy(history bool, staleDays int) error {
	err := a.updateSettings(func(s settings.Settings) settings.Settings { return s.WithPrivacy(history, staleDays) })
	if err != nil {
		return err
	}
	wailsruntime.EventsEmit(a.ctx, "activity:changed", nil)
	return nil
}
```

- [x] **Step 2: Build, vet, regenerate bindings and test**

```bash
gofmt -l . ; go vet . ./engine/... && go build ./... && wails generate module && chmod 644 frontend/wailsjs/runtime/*
grep -n "GetDashboard\|RemoveFromIndex\|MoveToTrash\|RecordOpened\|ForgetOpened\|ClearHistory\|SetProfile\|SetPrivacy" frontend/wailsjs/go/main/App.d.ts
go test ./engine/...
```

Expected: all eight functions listed; tests `ok`.

- [x] **Step 3: Commit**

```bash
git add dashboard.go frontend/wailsjs
git commit -m "feat(app): Dashboard, Free up space, profile and privacy bindings"
```

---

### Task 10: `lib/greeting.ts` and `lib/time.ts`

**Files:**
- Create: `frontend/src/lib/greeting.ts`, `frontend/src/lib/greeting.test.ts`, `frontend/src/lib/time.ts`, `frontend/src/lib/time.test.ts`

- [x] **Step 1: Write the failing tests**

`greeting.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { greeting, initials, partOfDay } from './greeting'

describe('partOfDay', () => {
  it.each([
    [4, 'evening'],
    [5, 'morning'],
    [11, 'morning'],
    [12, 'afternoon'],
    [17, 'afternoon'],
    [18, 'evening'],
    [23, 'evening'],
  ])('%s h → %s', (hour, part) => {
    expect(partOfDay(hour)).toBe(part)
  })
})

describe('greeting', () => {
  it('uses the first name', () => {
    expect(greeting(20, 'Zafer Acar')).toBe('Good evening, Zafer')
  })

  it('has no name part without a name', () => {
    expect(greeting(9, '   ')).toBe('Good morning')
  })
})

describe('initials', () => {
  it.each([
    ['Zafer Acar', 'ZA'],
    ['zafer', 'Z'],
    ['Ana María de la Cruz', 'AC'],
    ['  élise   ', 'É'],
    ['', ''],
  ])('%s → %s', (name, letters) => {
    expect(initials(name)).toBe(letters)
  })
})
```

`time.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { age, ago } from './time'

const now = new Date(2026, 9, 21, 12, 0) // Wed 21 Oct 2026, local time
const minutesAgo = (m: number) => new Date(now.getTime() - m * 60_000)

describe('ago', () => {
  it.each([
    [0.5, 'just now'],
    [3, '3 min ago'],
    [59, '59 min ago'],
    [60, '1h ago'],
    [23 * 60, '23h ago'],
    [30 * 60, 'yesterday'],
    [3 * 24 * 60, 'Sun'],
    [20 * 24 * 60, 'Oct 1'],
  ])('%s minutes → %s', (m, text) => {
    expect(ago(minutesAgo(m), now)).toBe(text)
  })
})

describe('age', () => {
  it.each([
    [1, '1 day'],
    [45, '45 days'],
    [59, '59 days'],
    [61, '2 months'],
    [430, '14 months'],
  ])('%s days → %s', (days, text) => {
    expect(age(minutesAgo(days * 24 * 60), now)).toBe(text)
  })
})
```

- [x] **Step 2: Run to confirm failure**

Run: `cd frontend && npx vitest run src/lib/greeting.test.ts src/lib/time.test.ts`
Expected: FAIL, `Failed to resolve import`.

- [x] **Step 3: Implement `greeting.ts`**

```ts
// The Dashboard greeting and the sidebar initials, from the display name.

export type PartOfDay = 'morning' | 'afternoon' | 'evening'

// 5:00–11:59 is morning, 12:00–17:59 afternoon, the rest evening.
export function partOfDay(hour: number): PartOfDay {
  if (hour >= 5 && hour < 12) return 'morning'
  if (hour >= 12 && hour < 18) return 'afternoon'
  return 'evening'
}

const words = (name: string): string[] => name.trim().split(/\s+/).filter(Boolean)

// "Good evening, Zafer" — first name only; "Good evening" without a name.
export function greeting(hour: number, name: string): string {
  const salutation = `Good ${partOfDay(hour)}`
  const firstName = words(name)[0]
  return firstName ? `${salutation}, ${firstName}` : salutation
}

const firstLetter = (word: string): string => (Array.from(word)[0] ?? '').toUpperCase()

// "ZA" for "Zafer Acar", "Z" for "Zafer", "" without a name.
export function initials(name: string): string {
  const parts = words(name)
  if (parts.length === 0) return ''
  if (parts.length === 1) return firstLetter(parts[0])
  return firstLetter(parts[0]) + firstLetter(parts[parts.length - 1])
}
```

- [x] **Step 4: Implement `time.ts`**

```ts
// Short human times for the Dashboard: "3 min ago", "yesterday", "14 months".

const MINUTE = 60_000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

// When something happened, relative to now.
export function ago(when: Date, now: Date): string {
  const ms = now.getTime() - when.getTime()
  if (ms < MINUTE) return 'just now'
  if (ms < HOUR) return `${Math.floor(ms / MINUTE)} min ago`
  if (ms < DAY) return `${Math.floor(ms / HOUR)}h ago`
  if (ms < 2 * DAY) return 'yesterday'
  if (ms < 7 * DAY) return WEEKDAYS[when.getDay()]
  return `${MONTHS[when.getMonth()]} ${when.getDate()}`
}

// How long something has gone unused: days under 60, then whole months.
export function age(since: Date, now: Date): string {
  const days = Math.floor((now.getTime() - since.getTime()) / DAY)
  if (days < 60) return `${days} ${days === 1 ? 'day' : 'days'}`
  return `${Math.floor(days / 30)} months`
}
```

- [x] **Step 5: Run to confirm pass**

Run: `cd frontend && npx vitest run src/lib/greeting.test.ts src/lib/time.test.ts`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add frontend/src/lib/greeting.ts frontend/src/lib/greeting.test.ts frontend/src/lib/time.ts frontend/src/lib/time.test.ts
git commit -m "feat(ui): greeting, initials and relative times"
```

---

### Task 11: Profile store

**Files:**
- Create: `frontend/src/stores/profile.ts`, `frontend/src/stores/profile.test.ts`

- [x] **Step 1: Write the failing tests**

```ts
import { get } from 'svelte/store'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanName, createProfileStore, NAME_MAX, type Profile, type ProfileRuntime } from './profile'

function fakeRuntime(overrides: Partial<ProfileRuntime> = {}): ProfileRuntime {
  return {
    get: vi.fn(async (): Promise<Profile> => ({ name: '', history: true, staleDays: 90 })),
    setProfile: vi.fn(async () => undefined),
    setPrivacy: vi.fn(async () => undefined),
    clearHistory: vi.fn(async () => undefined),
    ...overrides,
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('cleanName', () => {
  it('trims and caps at NAME_MAX characters', () => {
    expect(cleanName('  Zafer Acar  ')).toBe('Zafer Acar')
    expect(Array.from(cleanName('é'.repeat(NAME_MAX + 5)))).toHaveLength(NAME_MAX)
  })
})

describe('createProfileStore', () => {
  it('loads saved values', async () => {
    const profile = createProfileStore(fakeRuntime({ get: async () => ({ name: 'Zafer', history: false, staleDays: 30 }) }))
    await profile.load()
    expect(get(profile)).toEqual({ name: 'Zafer', history: false, staleDays: 30, nameError: '', privacyError: '' })
  })

  it('keeps defaults when loading fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const profile = createProfileStore(fakeRuntime({ get: () => Promise.reject(new Error('no bridge')) }))
    await profile.load()
    expect(get(profile)).toMatchObject({ name: '', history: true, staleDays: 90 })
  })

  it('saves a cleaned name and skips an unchanged one', async () => {
    const runtime = fakeRuntime()
    const profile = createProfileStore(runtime)
    await profile.setName('  Zafer  ')
    expect(get(profile).name).toBe('Zafer')
    expect(runtime.setProfile).toHaveBeenCalledWith('Zafer')
    await profile.setName('Zafer ')
    expect(runtime.setProfile).toHaveBeenCalledOnce()
  })

  it('restores the name when saving fails', async () => {
    const profile = createProfileStore(fakeRuntime({ setProfile: () => Promise.reject(new Error('disk full')) }))
    await profile.setName('Zafer')
    expect(get(profile)).toMatchObject({ name: '', nameError: "Couldn't save — disk full" })
  })

  it('saves both privacy values together', async () => {
    const runtime = fakeRuntime()
    const profile = createProfileStore(runtime)
    await profile.setStaleDays(180)
    await profile.setHistory(false)
    expect(runtime.setPrivacy).toHaveBeenNthCalledWith(1, true, 180)
    expect(runtime.setPrivacy).toHaveBeenNthCalledWith(2, false, 180)
  })

  it('restores privacy values when saving fails', async () => {
    const profile = createProfileStore(fakeRuntime({ setPrivacy: () => Promise.reject(new Error('disk full')) }))
    await profile.setHistory(false)
    expect(get(profile)).toMatchObject({ history: true, privacyError: "Couldn't save — disk full" })
  })

  it('reports a failed history clear', async () => {
    const profile = createProfileStore(fakeRuntime({ clearHistory: () => Promise.reject(new Error('locked')) }))
    await profile.clearHistory()
    expect(get(profile).privacyError).toBe("Couldn't clear history — locked")
  })
})
```

- [x] **Step 2: Run to confirm failure**

Run: `cd frontend && npx vitest run src/stores/profile.test.ts`
Expected: FAIL, `Failed to resolve import "./profile"`.

- [x] **Step 3: Implement**

```ts
// Profile and privacy preferences, saved by the Go backend in settings.json:
// the display name, whether history is kept, and when a file counts as stale.
// A change applies at once and rolls back if the save fails.

import { get, writable, type Readable } from 'svelte/store'

export type StaleDays = 30 | 90 | 180
export const STALE_DAY_CHOICES: StaleDays[] = [30, 90, 180]
export const NAME_MAX = 60

export interface Profile {
  name: string
  history: boolean
  staleDays: StaleDays
}

export interface ProfileState extends Profile {
  nameError: string // last failed name save, '' when none
  privacyError: string // last failed privacy save or history clear
}

export const defaultProfile: Profile = { name: '', history: true, staleDays: 90 }

// The backend surface the store needs; the app passes the Wails bindings.
export interface ProfileRuntime {
  get(): Promise<Profile>
  setProfile(name: string): Promise<void>
  setPrivacy(history: boolean, staleDays: number): Promise<void>
  clearHistory(): Promise<void>
}

export interface ProfileStore extends Readable<ProfileState> {
  load(): Promise<void>
  setName(name: string): Promise<void>
  setHistory(on: boolean): Promise<void>
  setStaleDays(days: StaleDays): Promise<void>
  clearHistory(): Promise<void>
}

// Trimmed and capped at NAME_MAX characters (not UTF-16 units).
export function cleanName(raw: string): string {
  return Array.from(raw.trim()).slice(0, NAME_MAX).join('').trim()
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

const saveFailed = (err: unknown): string => `Couldn't save — ${messageOf(err)}`

export function createProfileStore(runtime: ProfileRuntime): ProfileStore {
  const store = writable<ProfileState>({ ...defaultProfile, nameError: '', privacyError: '' })

  async function setPrivacy(change: Partial<Pick<Profile, 'history' | 'staleDays'>>): Promise<void> {
    const before = get(store)
    const next = { ...before, ...change }
    if (next.history === before.history && next.staleDays === before.staleDays) return
    store.set(next)
    try {
      await runtime.setPrivacy(next.history, next.staleDays)
      store.update((s) => ({ ...s, privacyError: '' }))
    } catch (err) {
      // Roll back only if no newer choice has replaced these values meanwhile.
      store.update((s) =>
        s.history === next.history && s.staleDays === next.staleDays
          ? { ...s, history: before.history, staleDays: before.staleDays, privacyError: saveFailed(err) }
          : { ...s, privacyError: saveFailed(err) },
      )
    }
  }

  return {
    subscribe: store.subscribe,
    async load() {
      try {
        const saved = await runtime.get()
        store.update((s) => ({ ...s, name: saved.name, history: saved.history, staleDays: saved.staleDays }))
      } catch (err) {
        console.error('profile: could not load, using defaults', err)
      }
    },
    async setName(raw) {
      const name = cleanName(raw)
      const previous = get(store).name
      if (name === previous) return
      store.update((s) => ({ ...s, name }))
      try {
        await runtime.setProfile(name)
        store.update((s) => ({ ...s, nameError: '' }))
      } catch (err) {
        store.update((s) => (s.name === name ? { ...s, name: previous, nameError: saveFailed(err) } : { ...s, nameError: saveFailed(err) }))
      }
    },
    setHistory: (history) => setPrivacy({ history }),
    setStaleDays: (staleDays) => setPrivacy({ staleDays }),
    async clearHistory() {
      try {
        await runtime.clearHistory()
        store.update((s) => ({ ...s, privacyError: '' }))
      } catch (err) {
        store.update((s) => ({ ...s, privacyError: `Couldn't clear history — ${messageOf(err)}` }))
      }
    },
  }
}
```

- [x] **Step 4: Run to confirm pass**

Run: `cd frontend && npx vitest run src/stores/profile.test.ts`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add frontend/src/stores/profile.ts frontend/src/stores/profile.test.ts
git commit -m "feat(ui): profile store for the display name and privacy"
```

---

### Task 12: Dashboard store and the search store's `opened` hook

**Files:**
- Create: `frontend/src/stores/dashboard.ts`, `frontend/src/stores/dashboard.test.ts`
- Modify: `frontend/src/stores/search.ts`, `frontend/src/stores/search.test.ts`

- [x] **Step 1: Write the failing tests**

`dashboard.test.ts`:

```ts
import { get } from 'svelte/store'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createDashboardStore, RELOAD_DELAY_MS, type DashboardData, type DashboardRuntime } from './dashboard'

const data = (files: number): DashboardData => ({
  folders: [{ path: '/n', files }],
  indexBytes: 2048,
  files,
  chunks: files * 10,
  types: [{ type: 'PDF', files }],
  lastUpdate: null,
  recentSearches: [],
  recentFiles: [],
  stale: [],
  staleBytes: 0,
  history: true,
  staleDays: 90,
})

function fakeRuntime(overrides: Partial<DashboardRuntime> = {}) {
  const handlers = new Map<string, () => void>()
  const runtime: DashboardRuntime = {
    on: (event, cb) => {
      handlers.set(event, cb as () => void)
      return () => handlers.delete(event)
    },
    get: vi.fn(async () => data(3)),
    removeFromIndex: vi.fn(async () => undefined),
    moveToTrash: vi.fn(async () => undefined),
    forgetOpened: vi.fn(async () => undefined),
    recordOpened: vi.fn(async () => undefined),
    ...overrides,
  }
  return { runtime, emit: (event: string) => handlers.get(event)?.(), handlers }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
})

describe('createDashboardStore', () => {
  it('loads on start', async () => {
    const { runtime } = fakeRuntime()
    const dashboard = createDashboardStore(runtime)
    expect(get(dashboard).phase).toBe('loading')
    await dashboard.start()
    expect(get(dashboard)).toMatchObject({ phase: 'ready', error: '' })
    expect(get(dashboard).data?.files).toBe(3)
  })

  it('shows an error when the first load fails', async () => {
    const { runtime } = fakeRuntime({ get: () => Promise.reject(new Error('bridge down')) })
    const dashboard = createDashboardStore(runtime)
    await dashboard.start()
    expect(get(dashboard)).toEqual({ phase: 'error', data: null, error: 'bridge down' })
  })

  it('keeps the figures shown when a reload fails', async () => {
    const get_ = vi.fn().mockResolvedValueOnce(data(3)).mockRejectedValueOnce(new Error('bridge down'))
    const dashboard = createDashboardStore(fakeRuntime({ get: get_ }).runtime)
    await dashboard.start()
    await dashboard.reload()
    expect(get(dashboard)).toMatchObject({ phase: 'ready', error: 'bridge down' })
    expect(get(dashboard).data?.files).toBe(3)
  })

  it.each(['index:done', 'watcher:done', 'index:cleared', 'index:changed', 'activity:changed'])(
    'reloads after %s',
    async (event) => {
      const { runtime, emit } = fakeRuntime()
      const dashboard = createDashboardStore(runtime)
      await dashboard.start()
      emit(event)
      await vi.advanceTimersByTimeAsync(RELOAD_DELAY_MS)
      expect(runtime.get).toHaveBeenCalledTimes(2)
    },
  )

  it('reloads once for a burst of events', async () => {
    const { runtime, emit } = fakeRuntime()
    const dashboard = createDashboardStore(runtime)
    await dashboard.start()
    emit('watcher:done')
    emit('index:changed')
    emit('activity:changed')
    await vi.advanceTimersByTimeAsync(RELOAD_DELAY_MS)
    expect(runtime.get).toHaveBeenCalledTimes(2)
  })

  it('stops listening on stop', async () => {
    const { runtime, emit, handlers } = fakeRuntime()
    const dashboard = createDashboardStore(runtime)
    await dashboard.start()
    dashboard.stop()
    emit('index:changed')
    await vi.advanceTimersByTimeAsync(RELOAD_DELAY_MS)
    expect(runtime.get).toHaveBeenCalledOnce()
    expect(handlers.size).toBe(0)
  })

  it('row actions resolve to an error message, or empty on success', async () => {
    const { runtime } = fakeRuntime({ moveToTrash: () => Promise.reject(new Error('refused')) })
    const dashboard = createDashboardStore(runtime)
    expect(await dashboard.removeFromIndex('/n/a.md')).toBe('')
    expect(runtime.removeFromIndex).toHaveBeenCalledWith('/n/a.md')
    expect(await dashboard.moveToTrash('/n/a.md')).toBe('refused')
    expect(await dashboard.forget('/n/a.md')).toBe('')
    expect(runtime.forgetOpened).toHaveBeenCalledWith('/n/a.md')
  })

  it('records opened files and only logs a failure', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const { runtime } = fakeRuntime({ recordOpened: vi.fn(() => Promise.reject(new Error('locked'))) })
    createDashboardStore(runtime).opened('/n/a.md')
    await vi.runAllTimersAsync()
    expect(runtime.recordOpened).toHaveBeenCalledWith('/n/a.md')
    expect(error).toHaveBeenCalled()
  })
})
```

Append inside `describe('createSearchStore', ...)` in `search.test.ts`:

```ts
  it('reports a file opened in the preview', () => {
    const opened = vi.fn()
    const search = createSearchStore({ search: async () => [], opened })
    search.open({ docPath: '/n/a.md', chunkIdx: 0, pageNum: 0, text: 'alpha', score: 1 })
    expect(opened).toHaveBeenCalledWith('/n/a.md')
  })
```

- [x] **Step 2: Run to confirm failure**

Run: `cd frontend && npx vitest run src/stores/dashboard.test.ts src/stores/search.test.ts`
Expected: FAIL, `Failed to resolve import "./dashboard"`, and `opened` not called.

- [x] **Step 3: Implement `dashboard.ts`**

```ts
// The Dashboard's figures and history from the Go backend, reloaded when the
// index or the history changes.

import { writable, type Readable } from 'svelte/store'
import type { FolderStat } from './engine'

export interface TypeCount {
  type: string
  files: number
}

export interface RecentSearch {
  query: string
  at: string // RFC 3339
}

export interface RecentFile {
  path: string
  at: string // RFC 3339
}

export interface StaleFile {
  path: string
  bytes: number // estimated share of the index
  modified: string // RFC 3339
}

export interface DashboardData {
  folders: FolderStat[]
  indexBytes: number
  files: number
  chunks: number
  types: TypeCount[]
  lastUpdate: RecentFile | null
  recentSearches: RecentSearch[]
  recentFiles: RecentFile[]
  stale: StaleFile[] // largest first
  staleBytes: number
  history: boolean
  staleDays: number
}

export type DashboardPhase = 'loading' | 'ready' | 'error'

export interface DashboardState {
  phase: DashboardPhase
  data: DashboardData | null
  error: string
}

// The backend surface the store needs; the app passes the Wails bindings.
export interface DashboardRuntime {
  on(event: string, callback: (...data: unknown[]) => void): () => void
  get(): Promise<DashboardData>
  removeFromIndex(path: string): Promise<void>
  moveToTrash(path: string): Promise<void>
  forgetOpened(path: string): Promise<void>
  recordOpened(path: string): Promise<void>
}

export interface DashboardStore extends Readable<DashboardState> {
  start(): Promise<void>
  stop(): void
  reload(): Promise<void>
  // Row actions resolve to an error message, or '' on success.
  removeFromIndex(path: string): Promise<string>
  moveToTrash(path: string): Promise<string>
  forget(path: string): Promise<string>
  opened(path: string): void
}

// Events after which the figures may have changed.
export const RELOAD_EVENTS = ['index:done', 'watcher:done', 'index:cleared', 'index:changed', 'activity:changed']
// A burst of events (a folder of watcher updates) reloads once.
export const RELOAD_DELAY_MS = 250

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

export function createDashboardStore(runtime: DashboardRuntime): DashboardStore {
  const store = writable<DashboardState>({ phase: 'loading', data: null, error: '' })
  let unsubscribers: (() => void)[] = []
  let timer: ReturnType<typeof setTimeout> | undefined
  let latest = 0 // only the most recent load may write

  async function reload(): Promise<void> {
    const id = ++latest
    try {
      const data = await runtime.get()
      if (id === latest) store.set({ phase: 'ready', data, error: '' })
    } catch (err) {
      if (id !== latest) return
      // Figures already shown stay; only an empty Dashboard shows the error state.
      store.update((s) => (s.data ? { ...s, error: messageOf(err) } : { phase: 'error', data: null, error: messageOf(err) }))
    }
  }

  function scheduleReload(): void {
    clearTimeout(timer)
    timer = setTimeout(() => void reload(), RELOAD_DELAY_MS)
  }

  async function act(call: () => Promise<void>): Promise<string> {
    try {
      await call()
      return ''
    } catch (err) {
      return messageOf(err)
    }
  }

  return {
    subscribe: store.subscribe,
    async start() {
      unsubscribers = RELOAD_EVENTS.map((event) => runtime.on(event, scheduleReload))
      await reload()
    },
    stop() {
      clearTimeout(timer)
      unsubscribers.forEach((off) => off())
      unsubscribers = []
    },
    reload,
    removeFromIndex: (path) => act(() => runtime.removeFromIndex(path)),
    moveToTrash: (path) => act(() => runtime.moveToTrash(path)),
    forget: (path) => act(() => runtime.forgetOpened(path)),
    opened(path) {
      runtime.recordOpened(path).catch((err) => console.error('dashboard: could not record an opened file', err))
    },
  }
}
```

- [x] **Step 4: Add the `opened` hook to `search.ts`**

- In `SearchApi`, add after `search(...)`:

```ts
  // Told when a result's file opens in the preview (for recent files).
  opened?(path: string): void
```

- In `open(result)`, add as the first line of the method body: `api.opened?.(result.docPath)`

- [x] **Step 5: Run to confirm pass**

Run: `cd frontend && npx vitest run src/stores/dashboard.test.ts src/stores/search.test.ts`
Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add frontend/src/stores/dashboard.ts frontend/src/stores/dashboard.test.ts frontend/src/stores/search.ts frontend/src/stores/search.test.ts
git commit -m "feat(ui): dashboard store, and recent files from the Chat preview"
```

---

### Task 13: Navigation: Dashboard and Chat routes, profile button

**Files:**
- Create: `frontend/src/ui/ProfileButton.svelte`, `frontend/src/ui/ProfileButton.test.ts`, `frontend/src/views/dashboard/DashboardView.svelte` (placeholder replaced in Task 15), `frontend/src/views/profile/ProfileView.svelte` (placeholder replaced in Task 14)
- Modify: `frontend/src/ui/icons.ts`, `frontend/src/ui/Sidebar.svelte`, `frontend/src/routes.ts`, `frontend/src/App.svelte`, `frontend/src/stores/app.ts`, `frontend/src/main.ts`, `frontend/src/views/Redirect.svelte`

- [x] **Step 1: Write the failing test**

```ts
import { afterEach, describe, expect, it } from 'vitest'
import ProfileButton from './ProfileButton.svelte'

afterEach(() => {
  document.body.innerHTML = ''
})

function mount(props: { name?: string; compact?: boolean; selected?: boolean }) {
  const target = document.createElement('div')
  document.body.append(target)
  new ProfileButton({ target, props })
  return target.querySelector('a') as HTMLAnchorElement
}

describe('ProfileButton', () => {
  it('links to Profile with initials and the name', () => {
    const link = mount({ name: 'Zafer Acar' })
    expect(link.getAttribute('href')).toBe('#/profile')
    expect(link.querySelector('.avatar')?.textContent).toBe('ZA')
    expect(link.textContent).toContain('Zafer Acar')
  })

  it('invites a name when there is none', () => {
    const link = mount({ name: '' })
    expect(link.textContent).toContain('Add your name')
  })

  it('shows only the initials in the compact rail, labelled for screen readers', () => {
    const link = mount({ name: 'Zafer Acar', compact: true })
    expect(link.textContent?.trim()).toBe('ZA')
    expect(link.getAttribute('aria-label')).toBe('Profile — Zafer Acar')
  })
})
```

- [x] **Step 2: Run to confirm failure**

Run: `cd frontend && npx vitest run src/ui/ProfileButton.test.ts`
Expected: FAIL, `Failed to resolve import "./ProfileButton.svelte"`.

- [x] **Step 3: Implement `ProfileButton.svelte`**

```svelte
<script lang="ts">
  // The display name at the bottom of the sidebar; opens Profile.
  import { initials } from '../lib/greeting'
  import PressableSurface from './PressableSurface.svelte'

  export let name = ''
  export let compact = false
  export let selected = false

  $: letters = initials(name)
  $: label = name ? `Profile — ${name}` : 'Add your name'
</script>

<PressableSurface
  href="#/profile"
  {selected}
  label={compact ? label : undefined}
  title={compact ? label : undefined}
  class="profile-button {compact ? 'compact' : ''}"
>
  <span class="avatar" aria-hidden={compact ? undefined : 'true'}>{letters || '+'}</span>
  {#if !compact}<span class="name" class:empty={!name}>{name || 'Add your name'}</span>{/if}
</PressableSurface>

<style>
  :global(.profile-button) {
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-xs) var(--space-sm);
    color: var(--color-ink-secondary);
  }

  :global(.profile-button.compact) {
    justify-content: center;
    padding: var(--space-xs) 0;
  }

  :global(.profile-button.selected) {
    color: var(--color-primary);
  }

  .avatar {
    display: grid;
    place-items: center;
    flex-shrink: 0;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background-color: var(--color-secondary-wash-strong);
    color: var(--color-secondary);
    font-size: var(--text-micro-size);
    font-weight: var(--text-micro-weight);
  }

  .name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--text-body-size);
    font-weight: var(--text-body-strong-weight);
  }

  .name.empty {
    color: var(--color-ink-faint);
    font-weight: var(--text-body-weight);
  }
</style>
```

- [x] **Step 4: Icons**

In `ui/icons.ts`, add `| 'dashboard' | 'chat'` to `IconName` and to `ICONS`:

```ts
  dashboard: ['M3.5 4.5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1h-4a1 1 0 0 1-1-1v-6Z', 'M10.5 4.5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2a1 1 0 0 1-1 1h-4a1 1 0 0 1-1-1v-2Z', 'M10.5 10.5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v5a1 1 0 0 1-1 1h-4a1 1 0 0 1-1-1v-5Z', 'M3.5 14.5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v1a1 1 0 0 1-1 1h-4a1 1 0 0 1-1-1v-1Z'],
  chat: ['M4.5 4h11A1.5 1.5 0 0 1 17 5.5v7a1.5 1.5 0 0 1-1.5 1.5H10l-3.5 3v-3h-2A1.5 1.5 0 0 1 3 12.5v-7A1.5 1.5 0 0 1 4.5 4Z'],
```

- [x] **Step 5: Sidebar**

In `ui/Sidebar.svelte`:
- Import `ProfileButton from './ProfileButton.svelte'` and add `export let name = ''`.
- After `</ul>`, add:

```svelte
  <div class="profile">
    <ProfileButton {name} {compact} selected={$location === '/profile'} />
  </div>
```

- In the styles, add:

```css
  .profile {
    margin-top: auto;
    width: 100%;
  }
```

- [x] **Step 6: Placeholder views**

`views/dashboard/DashboardView.svelte` and `views/profile/ProfileView.svelte` each start as:

```svelte
<h1>Coming in the next task</h1>
```

- [x] **Step 7: Routes**

Replace `routes.ts` with:

```ts
// Route table and sidebar registry. A destination is listed in the sidebar only
// once it exists here, so the nav never shows unbuilt pages. Profile is routed
// but reached from the profile button at the foot of the sidebar.

import type { SvelteComponent } from 'svelte'
import type { IconName } from './ui/icons'
import DashboardView from './views/dashboard/DashboardView.svelte'
import ProfileView from './views/profile/ProfileView.svelte'
import Redirect from './views/Redirect.svelte'
import SearchView from './views/search/SearchView.svelte'
import SettingsView from './views/settings/SettingsView.svelte'

export interface Destination {
  path: string
  label: string
  icon: IconName
  component: typeof SvelteComponent
}

export const destinations: Destination[] = [
  { path: '/', label: 'Dashboard', icon: 'dashboard', component: DashboardView },
  { path: '/chat', label: 'Chat', icon: 'chat', component: SearchView },
  { path: '/settings', label: 'Settings', icon: 'settings', component: SettingsView },
]

export const routes: Record<string, typeof SvelteComponent> = {
  ...Object.fromEntries(destinations.map((d) => [d.path, d.component])),
  '/profile': ProfileView,
  '*': Redirect,
}
```

In `views/Redirect.svelte`, change the comment to `// Unknown routes land on the Dashboard.`

- [x] **Step 8: Stores and boot**

Replace `stores/app.ts` with:

```ts
// App-wide store instances wired to the Wails bindings. Components import from
// here; tests construct the stores directly with their own runtimes.

import { derived } from 'svelte/store'
import {
  ClearHistory,
  ClearIndex,
  EngineStatus,
  ForgetOpened,
  GetDashboard,
  GetSettings,
  IndexFolder,
  MoveToTrash,
  RecordOpened,
  RemoveFolder,
  RemoveFromIndex,
  Search,
  SelectFolder,
  SetAppearance,
  SetPrivacy,
  SetProfile,
} from '../../wailsjs/go/main/App.js'
import { EventsOn } from '../../wailsjs/runtime/runtime.js'
import { createDashboardStore, type DashboardData } from './dashboard'
import { createEngineStore } from './engine'
import { createMotionStore } from './motion'
import { createProfileStore, type StaleDays } from './profile'
import { createSearchStore } from './search'
import { createSettingsStore, type MotionPreference, type ThemePreference } from './settings'
import { createThemeStore } from './theme'

const logFailure = (what: string) => (err: unknown) => console.error(what, err)

export const settings = createSettingsStore({
  // Go validates these values before saving, so the casts only narrow the type.
  get: async () => {
    const saved = await GetSettings()
    return { theme: saved.theme as ThemePreference, motion: saved.motion as MotionPreference }
  },
  setAppearance: SetAppearance,
})

export const profile = createProfileStore({
  get: async () => {
    const saved = await GetSettings()
    return { name: saved.name, history: saved.history, staleDays: saved.staleDays as StaleDays }
  },
  setProfile: SetProfile,
  setPrivacy: SetPrivacy,
  clearHistory: ClearHistory,
})

export const engine = createEngineStore({
  on: EventsOn,
  status: EngineStatus,
  selectFolder: SelectFolder,
  indexFolder: IndexFolder,
  clearIndex: ClearIndex,
  removeFolder: RemoveFolder,
})

export const search = createSearchStore({
  search: Search,
  opened: (path) => void RecordOpened(path).catch(logFailure('search: could not record an opened file')),
})

export const dashboard = createDashboardStore({
  on: EventsOn,
  // Go's time.Time fields arrive as RFC 3339 strings; the generated model types them loosely.
  get: async () => (await GetDashboard()) as unknown as DashboardData,
  removeFromIndex: RemoveFromIndex,
  moveToTrash: MoveToTrash,
  forgetOpened: ForgetOpened,
  recordOpened: RecordOpened,
})

const root = document.documentElement

export const theme = createThemeStore(
  window.matchMedia('(prefers-color-scheme: light)'),
  root,
  derived(settings, (s) => s.theme),
)

export const motion = createMotionStore(
  root,
  derived(settings, (s) => s.motion),
)
```

In `main.ts`, import `profile` beside `settings` and replace the `settings.load()` call with `Promise.all([settings.load(), profile.load()])`.

In `App.svelte`, change the store import to `import { engine, motion, profile, search, theme } from './stores/app'` and pass the name: `<Sidebar {destinations} {compact} name={$profile.name} />`.

- [x] **Step 9: Check, test and commit**

```bash
cd frontend && npm run check && npx vitest run
git add frontend/src
git commit -m "feat(ui): Dashboard and Chat in the sidebar, profile button"
```

Expected: 0 errors, all tests pass.

---

### Task 14: Profile page

**Files:**
- Create: `frontend/src/views/profile/NameField.svelte`, `frontend/src/views/profile/NameField.test.ts`
- Replace: `frontend/src/views/profile/ProfileView.svelte`

- [x] **Step 1: Write the failing test**

```ts
import { tick } from 'svelte'
import { afterEach, describe, expect, it } from 'vitest'
import NameField from './NameField.svelte'

afterEach(() => {
  document.body.innerHTML = ''
})

function mount(value = '') {
  const target = document.createElement('div')
  document.body.append(target)
  const component = new NameField({ target, props: { value } })
  const saved: string[] = []
  component.$on('save', (e: CustomEvent<string>) => saved.push(e.detail))
  const input = target.querySelector('input') as HTMLInputElement
  const type = (text: string) => {
    input.focus()
    input.value = text
    input.dispatchEvent(new Event('input'))
  }
  const key = (k: string) => input.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true }))
  return { component, input, saved, type, key }
}

describe('NameField', () => {
  it('saves the trimmed name on blur', async () => {
    const { input, saved, type } = mount()
    type('  Zafer Acar  ')
    input.blur()
    await tick()
    expect(saved).toEqual(['Zafer Acar'])
  })

  it('saves on Enter', async () => {
    const { saved, type, key } = mount()
    type('Zafer')
    key('Enter')
    await tick()
    expect(saved).toEqual(['Zafer'])
  })

  it('does not save an unchanged name', async () => {
    const { input, saved, type } = mount('Zafer')
    type('Zafer ')
    input.blur()
    await tick()
    expect(saved).toEqual([])
  })

  it('Escape restores the saved name without saving', async () => {
    const { input, saved, type, key } = mount('Zafer')
    type('Somebody else')
    key('Escape')
    await tick()
    expect(saved).toEqual([])
    expect(input.value).toBe('Zafer')
  })

  it('limits typing to 60 characters', () => {
    const { input } = mount()
    expect(input.maxLength).toBe(60)
  })
})
```

- [x] **Step 2: Run to confirm failure**

Run: `cd frontend && npx vitest run src/views/profile/NameField.test.ts`
Expected: FAIL, `Failed to resolve import "./NameField.svelte"`.

- [x] **Step 3: Implement `NameField.svelte`**

```svelte
<script lang="ts">
  // The display name input. Saves the cleaned name on blur or Enter; Escape
  // puts back the saved name. Follows `value` whenever it isn't being edited,
  // so a rolled-back save shows the saved name again.
  import { createEventDispatcher } from 'svelte'
  import { cleanName, NAME_MAX } from '../../stores/profile'

  export let value = ''
  export let id = 'display-name'

  const dispatch = createEventDispatcher<{ save: string }>()
  let draft = value
  let editing = false

  $: if (!editing) draft = value

  function commit() {
    editing = false
    const name = cleanName(draft)
    if (name !== value) dispatch('save', name)
  }

  function onKeydown(e: KeyboardEvent) {
    const input = e.currentTarget as HTMLInputElement
    if (e.key === 'Enter') {
      e.preventDefault()
      input.blur()
    } else if (e.key === 'Escape') {
      draft = value
      editing = false
      input.blur()
    }
  }
</script>

<input
  {id}
  class="name-field"
  type="text"
  maxlength={NAME_MAX}
  autocomplete="name"
  placeholder="Your name"
  bind:value={draft}
  on:focus={() => (editing = true)}
  on:blur={() => editing && commit()}
  on:keydown={onKeydown}
/>

<style>
  .name-field {
    width: 100%;
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-sm);
    border: 1px solid var(--color-hairline-strong);
    background-color: var(--color-surface);
    color: var(--color-ink);
    font-size: var(--text-body-size);
    transition: border-color var(--motion-quick) var(--ease-standard);
  }

  .name-field:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px var(--color-primary-wash);
  }
</style>
```

- [x] **Step 4: Replace `ProfileView.svelte`**

```svelte
<script lang="ts">
  // Profile: the display name, kept on this computer.
  import { fade } from 'svelte/transition'
  import { duration } from '../../lib/motion'
  import { profile } from '../../stores/app'
  import NameField from './NameField.svelte'
</script>

<div class="profile-view" in:fade={{ duration: duration('normal') }}>
  <div class="column">
    <header>
      <h1 class="display">Profile</h1>
      <p class="lede">Kept only on this computer.</p>
    </header>
    <div class="field">
      <label class="eyebrow" for="display-name">Display name</label>
      <NameField value={$profile.name} on:save={(e) => profile.setName(e.detail)} />
      <p class="hint">Used for your initials in the sidebar and the Dashboard greeting.</p>
      {#if $profile.nameError}
        <p class="error" role="alert">{$profile.nameError}</p>
      {/if}
    </div>
  </div>
</div>

<style>
  .profile-view {
    height: 100%;
    overflow-y: auto;
  }

  .column {
    max-width: var(--settings-column-max);
    margin: 0 auto;
    padding: var(--space-huge) var(--space-xl) var(--space-xxl);
    display: flex;
    flex-direction: column;
    gap: var(--space-xxl);
  }

  .display {
    font-family: var(--font-display);
    font-size: var(--text-display-size);
    line-height: var(--text-display-line);
    font-weight: var(--text-display-weight);
    letter-spacing: var(--text-display-tracking);
  }

  .lede,
  .hint {
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
  }

  .lede {
    margin-top: var(--space-xs);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .eyebrow {
    font-size: var(--text-micro-size);
    font-weight: var(--text-micro-weight);
    letter-spacing: var(--text-micro-tracking);
    text-transform: uppercase;
    color: var(--color-ink-faint);
  }

  .error {
    font-size: var(--text-small-size);
    color: var(--color-urgent);
  }
</style>
```

- [x] **Step 5: Run, check and commit**

```bash
cd frontend && npx vitest run src/views/profile && npm run check
git add frontend/src/views/profile
git commit -m "feat(ui): Profile page with the display name"
```

Expected: PASS, 0 errors.

---

### Task 15: Dashboard tiles

**Files:**
- Create in `frontend/src/views/dashboard/`: `DashboardSearch.svelte`, `DiskTile.svelte`, `RecentFilesTile.svelte`, `RecentSearchesTile.svelte`, `StaleFileList.svelte`, `FreeSpaceTile.svelte`, `StaleFilesDialog.svelte`, `ViewerPanel.svelte`, `tiles.test.ts`
- Replace: `DashboardView.svelte`

- [x] **Step 1: Write the failing tests** (`tiles.test.ts`)

```ts
import { tick } from 'svelte'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import type { DashboardData, StaleFile } from '../../stores/dashboard'
import DashboardSearch from './DashboardSearch.svelte'
import DiskTile from './DiskTile.svelte'
import FreeSpaceTile from './FreeSpaceTile.svelte'
import RecentFilesTile from './RecentFilesTile.svelte'
import RecentSearchesTile from './RecentSearchesTile.svelte'
import StaleFileList from './StaleFileList.svelte'

beforeAll(() => {
  // jsdom has <dialog> but not its modal methods.
  HTMLDialogElement.prototype.showModal = function (this: HTMLDialogElement) {
    this.open = true
  }
  HTMLDialogElement.prototype.close = function (this: HTMLDialogElement) {
    this.open = false
  }
})

afterEach(() => {
  document.body.innerHTML = ''
})

const now = new Date(2026, 9, 21, 12, 0)
const minutesAgo = (m: number) => new Date(now.getTime() - m * 60_000).toISOString()

function mount<P extends Record<string, unknown>>(Component: any, props: P) {
  const target = document.createElement('div')
  document.body.append(target)
  const component = new Component({ target, props })
  const events: [string, unknown][] = []
  for (const name of ['search', 'open', 'forget', 'remove', 'trash', 'showAll']) {
    component.$on(name, (e: CustomEvent) => events.push([name, e.detail]))
  }
  const button = (text: string) =>
    [...target.querySelectorAll('button')].find((b) => b.textContent?.trim() === text) as HTMLButtonElement
  return { target, events, button }
}

const data: DashboardData = {
  folders: [{ path: '/n', files: 412 }],
  indexBytes: 2_200_000,
  files: 412,
  chunks: 9860,
  types: [
    { type: 'Markdown', files: 121 },
    { type: 'PDF', files: 214 },
    { type: 'Text', files: 77 },
  ],
  lastUpdate: { path: '/n/meeting-notes.md', at: minutesAgo(3) },
  recentSearches: [],
  recentFiles: [],
  stale: [],
  staleBytes: 0,
  history: true,
  staleDays: 90,
}

const staleFile = (name: string, bytes: number): StaleFile => ({ path: `/n/${name}`, bytes, modified: minutesAgo(200 * 24 * 60) })

describe('DashboardSearch', () => {
  it('dispatches the trimmed query on Enter, and nothing for a blank one', () => {
    const { target, events } = mount(DashboardSearch, {})
    const input = target.querySelector('input') as HTMLInputElement
    input.value = '   '
    input.dispatchEvent(new Event('input'))
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    input.value = '  grant deadlines '
    input.dispatchEvent(new Event('input'))
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }))
    expect(events).toEqual([['search', 'grant deadlines']])
  })

  it('Ctrl+K focuses the search bar', () => {
    const { target } = mount(DashboardSearch, {})
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))
    expect(document.activeElement).toBe(target.querySelector('input'))
  })
})

describe('DiskTile', () => {
  it('shows the index size, counts, types in a fixed order and the last update', () => {
    const { target } = mount(DiskTile, { data, now })
    expect(target.querySelector('.value')?.textContent).toBe('2.1')
    expect(target.textContent).toContain('MB index')
    expect(target.textContent).toContain('412 files · 9,860 chunks')
    const legend = [...target.querySelectorAll('.legend li')].map((li) => li.textContent?.replace(/\s+/g, ' ').trim())
    expect(legend).toEqual(['PDF 214', 'Markdown 121', 'Text 77'])
    expect(target.textContent).toContain('meeting-notes.md')
    expect(target.textContent).toContain('3 min ago')
  })

  it('says so when nothing has been updated yet', () => {
    const { target } = mount(DiskTile, { data: { ...data, lastUpdate: null, types: [] }, now })
    expect(target.textContent).toContain('Nothing updated yet')
    expect(target.querySelector('.bar')).toBeNull()
  })
})

describe('RecentFilesTile', () => {
  const files = [
    { path: '/n/thesis.pdf', at: minutesAgo(120) },
    { path: '/n/notes.md', at: minutesAgo(30 * 60) },
  ]

  it('opens and forgets files', () => {
    const { target, events } = mount(RecentFilesTile, { files, history: true, now })
    ;(target.querySelector('.open') as HTMLButtonElement).click()
    ;(target.querySelector('.forget') as HTMLButtonElement).click()
    expect(events).toEqual([
      ['open', '/n/thesis.pdf'],
      ['forget', '/n/thesis.pdf'],
    ])
    expect(target.textContent).toContain('2h ago')
    expect(target.textContent).toContain('yesterday')
  })

  it('has quiet empty states', () => {
    expect(mount(RecentFilesTile, { files: [], history: true, now }).target.textContent).toContain('Files you open will appear here')
    expect(mount(RecentFilesTile, { files: [], history: false, now }).target.textContent).toContain('History is off')
  })
})

describe('RecentSearchesTile', () => {
  it('runs a recent search again', () => {
    const { button, events } = mount(RecentSearchesTile, { searches: [{ query: 'invoice march', at: minutesAgo(5) }], history: true })
    button('invoice march').click()
    expect(events).toEqual([['search', 'invoice march']])
  })

  it('has quiet empty states', () => {
    expect(mount(RecentSearchesTile, { searches: [], history: true }).target.textContent).toContain('Your searches will appear here')
    expect(mount(RecentSearchesTile, { searches: [], history: false }).target.textContent).toContain('History is off')
  })
})

describe('StaleFileList', () => {
  const files = [staleFile('old-syllabus.pdf', 48_000), staleFile('draft.md', 12_000)]

  it('removes from the index straight away', () => {
    const { target, events } = mount(StaleFileList, { files, busy: false, errors: {}, now })
    ;(target.querySelector('.remove') as HTMLButtonElement).click()
    expect(events).toEqual([['remove', '/n/old-syllabus.pdf']])
  })

  it('moves to the Trash only after confirming', async () => {
    const { target, events, button } = mount(StaleFileList, { files, busy: false, errors: {}, now })
    ;(target.querySelector('.trash') as HTMLButtonElement).click()
    await tick()
    expect(events).toEqual([])
    expect(target.querySelector('dialog')?.open).toBe(true)
    expect(target.textContent).toContain('Move old-syllabus.pdf to the Trash?')
    button('Move to Trash').click()
    await tick()
    expect(events).toEqual([['trash', '/n/old-syllabus.pdf']])
  })

  it('shows a row error and disables actions while busy', () => {
    const { target } = mount(StaleFileList, { files, busy: true, errors: { '/n/draft.md': 'Nothing was deleted.' }, now })
    expect(target.textContent).toContain('Nothing was deleted.')
    expect((target.querySelector('.remove') as HTMLButtonElement).disabled).toBe(true)
    expect(target.textContent).toContain('6 months')
  })
})

describe('FreeSpaceTile', () => {
  it('lists the three largest and offers the rest', () => {
    const stale = ['a', 'b', 'c', 'd'].map((n, i) => staleFile(`${n}.md`, 4000 - i))
    const { target, events, button } = mount(FreeSpaceTile, { stale, staleBytes: 15_994, staleDays: 90, busy: false, errors: {}, now })
    expect(target.querySelectorAll('.stale-row')).toHaveLength(3)
    expect(target.textContent).toContain('4 files untouched for 90+ days')
    button('Show all 4 →').click()
    expect(events).toEqual([['showAll', null]])
  })

  it('says when nothing is stale', () => {
    const { target } = mount(FreeSpaceTile, { stale: [], staleBytes: 0, staleDays: 30, busy: false, errors: {}, now })
    expect(target.textContent).toContain('Nothing stale. Everything was used in the last 30 days.')
  })
})
```

- [x] **Step 2: Run to confirm failure**

Run: `cd frontend && npx vitest run src/views/dashboard/tiles.test.ts`
Expected: FAIL, `Failed to resolve import "./DashboardSearch.svelte"`.

- [x] **Step 3: `DashboardSearch.svelte`**

```svelte
<script lang="ts">
  // The Dashboard's search bar. Enter dispatches `search` with the query; the
  // Dashboard then shows the results in Chat. ⌘K / Ctrl+K focuses it.
  import { createEventDispatcher } from 'svelte'
  import Icon from '../../ui/Icon.svelte'

  export let disabled = false
  export let placeholder = 'Search your files…'

  const dispatch = createEventDispatcher<{ search: string }>()
  const shortcut = /Mac|iPhone|iPad/.test(navigator.userAgent) ? '⌘K' : 'Ctrl K'
  let input: HTMLInputElement
  let query = ''

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && query.trim()) dispatch('search', query.trim())
  }

  function onWindowKeydown(e: KeyboardEvent) {
    if (e.key.toLowerCase() === 'k' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      input?.focus()
    }
  }
</script>

<svelte:window on:keydown={onWindowKeydown} />

<label class="dashboard-search" class:disabled>
  <Icon name="search" />
  <input bind:this={input} bind:value={query} type="search" {placeholder} {disabled} aria-label="Search your files" on:keydown={onKeydown} />
  <kbd aria-hidden="true">{shortcut}</kbd>
</label>

<style>
  .dashboard-search {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-hairline-strong);
    background-color: var(--color-surface);
    box-shadow: var(--shadow-raised);
    color: var(--color-ink-faint);
    transition: border-color var(--motion-quick) var(--ease-standard);
  }

  .dashboard-search:focus-within {
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px var(--color-primary-wash);
  }

  .dashboard-search.disabled {
    opacity: 0.6;
  }

  input {
    flex: 1;
    min-width: 0;
    border: none;
    background: none;
    color: var(--color-ink);
    font-size: var(--text-heading-size);
  }

  input:focus {
    outline: none;
  }

  kbd {
    padding: 0 var(--space-xxs);
    border-radius: var(--radius-xs);
    border: 1px solid var(--color-hairline-strong);
    font-family: var(--font-sans);
    font-size: var(--text-micro-size);
  }
</style>
```

- [x] **Step 4: `DiskTile.svelte`**

```svelte
<script lang="ts">
  // Disk: the index size, its files and chunks, files by type, and the last
  // file the index updated.
  import { formatBytes } from '../../lib/format'
  import { filename } from '../../lib/path'
  import { ago } from '../../lib/time'
  import type { DashboardData } from '../../stores/dashboard'

  export let data: DashboardData
  export let now = new Date()

  // A fixed order keeps each type on its colour as the counts change.
  const TYPE_ORDER = ['PDF', 'Markdown', 'Text', 'Code', 'Data']
  const rank = (type: string) => (TYPE_ORDER.includes(type) ? TYPE_ORDER.indexOf(type) : TYPE_ORDER.length)
  const colorOf = (type: string) => `var(--color-identity-${rank(type) % 6})`
  const plural = (n: number, word: string) => `${n.toLocaleString('en-US')} ${word}${n === 1 ? '' : 's'}`

  $: size = formatBytes(data.indexBytes)
  $: value = size.slice(0, size.lastIndexOf(' '))
  $: unit = size.slice(size.lastIndexOf(' ') + 1)
  $: segments = [...data.types].sort((a, b) => rank(a.type) - rank(b.type))
</script>

<section class="tile disk" aria-labelledby="disk-title">
  <h2 id="disk-title" class="eyebrow">Disk</h2>
  <p class="big"><span class="value">{value}</span><span class="unit">{unit} index</span></p>
  <p class="muted">{plural(data.files, 'file')} · {plural(data.chunks, 'chunk')}</p>

  {#if segments.length > 0}
    <div class="bar" role="img" aria-label={segments.map((s) => `${s.type} ${s.files}`).join(', ')}>
      {#each segments as s (s.type)}
        <span class="segment" style="flex-grow: {s.files}; background-color: {colorOf(s.type)}" />
      {/each}
    </div>
    <ul class="legend">
      {#each segments as s (s.type)}
        <li><i style="background-color: {colorOf(s.type)}" aria-hidden="true" />{s.type} <span class="count">{s.files.toLocaleString('en-US')}</span></li>
      {/each}
    </ul>
  {/if}

  <div class="well">
    <span class="muted">Last update</span>
    {#if data.lastUpdate}
      <span title={data.lastUpdate.path}>
        <strong>{filename(data.lastUpdate.path)}</strong>
        <span class="muted">· {ago(new Date(data.lastUpdate.at), now)}</span>
      </span>
    {:else}
      <span class="muted">Nothing updated yet</span>
    {/if}
  </div>
</section>

<style>
  .big {
    display: flex;
    align-items: baseline;
    gap: var(--space-xs);
    margin-top: var(--space-sm);
  }

  .value {
    font-family: var(--font-display);
    font-size: var(--text-display-size);
    line-height: var(--text-display-line);
    font-weight: var(--text-display-weight);
    letter-spacing: var(--text-display-tracking);
    font-variant-numeric: tabular-nums;
  }

  .unit {
    color: var(--color-ink-secondary);
    font-size: var(--text-heading-size);
  }

  .bar {
    display: flex;
    gap: 2px;
    height: 10px;
    margin: var(--space-md) 0 var(--space-sm);
    border-radius: 5px;
    overflow: hidden;
  }

  .segment {
    min-width: 4px;
  }

  .legend {
    list-style: none;
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs) var(--space-md);
    font-size: var(--text-small-size);
    color: var(--color-ink-secondary);
  }

  .legend i {
    display: inline-block;
    width: 8px;
    height: 8px;
    margin-right: var(--space-xxs);
    border-radius: 2px;
  }

  .count {
    font-variant-numeric: tabular-nums;
  }

  .well {
    display: flex;
    flex-direction: column;
    gap: var(--space-xxs);
    margin-top: var(--space-lg);
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-sm);
    background-color: var(--color-surface-sunken);
  }

  strong {
    font-weight: var(--text-body-strong-weight);
  }
</style>
```

- [x] **Step 5: `RecentFilesTile.svelte`**

```svelte
<script lang="ts">
  // Recently opened files: click to open in the viewer, × to forget.
  import { createEventDispatcher } from 'svelte'
  import { filename } from '../../lib/path'
  import { ago } from '../../lib/time'
  import type { RecentFile } from '../../stores/dashboard'
  import Icon from '../../ui/Icon.svelte'

  export let files: RecentFile[]
  export let history: boolean
  export let now = new Date()

  const dispatch = createEventDispatcher<{ open: string; forget: string }>()
</script>

<section class="tile recent-files" aria-labelledby="recent-files-title">
  <h2 id="recent-files-title" class="eyebrow">Recently opened</h2>
  {#if !history}
    <p class="muted">History is off · <a href="#/settings">Turn on in Settings</a></p>
  {:else if files.length === 0}
    <p class="muted">Files you open will appear here</p>
  {:else}
    <ul class="rows">
      {#each files as file (file.path)}
        <li class="row">
          <button class="open" type="button" title={file.path} on:click={() => dispatch('open', file.path)}>
            <span class="name">{filename(file.path)}</span>
            <span class="when">{ago(new Date(file.at), now)}</span>
          </button>
          <button class="forget" type="button" aria-label="Forget {filename(file.path)}" title="Forget" on:click={() => dispatch('forget', file.path)}>
            <Icon name="close" size={14} />
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .rows {
    list-style: none;
    margin-top: var(--space-sm);
  }

  .row {
    display: flex;
    align-items: center;
    border-top: 1px solid var(--color-hairline);
  }

  .row:first-child {
    border-top: none;
  }

  .open {
    flex: 1;
    min-width: 0;
    display: flex;
    justify-content: space-between;
    gap: var(--space-sm);
    padding: var(--space-xs) var(--space-xs);
    border-radius: var(--radius-xs);
    color: var(--color-ink);
    text-align: left;
    transition: background-color var(--motion-quick) var(--ease-standard);
  }

  .open:hover,
  .open:focus-visible {
    background-color: var(--color-surface-raised);
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .when {
    flex-shrink: 0;
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
    font-variant-numeric: tabular-nums;
  }

  .forget {
    display: grid;
    place-items: center;
    width: 24px;
    height: 24px;
    border-radius: var(--radius-xs);
    color: var(--color-ink-faint);
    opacity: 0;
    transition: opacity var(--motion-quick) var(--ease-standard);
  }

  .row:hover .forget,
  .forget:focus-visible {
    opacity: 1;
  }

  a {
    color: var(--color-primary);
  }
</style>
```

- [x] **Step 6: `RecentSearchesTile.svelte`**

```svelte
<script lang="ts">
  // Recent searches as chips; clicking one runs it again in Chat.
  import { createEventDispatcher } from 'svelte'
  import type { RecentSearch } from '../../stores/dashboard'

  export let searches: RecentSearch[]
  export let history: boolean

  const dispatch = createEventDispatcher<{ search: string }>()
</script>

<section class="tile recent-searches" aria-labelledby="recent-searches-title">
  <h2 id="recent-searches-title" class="eyebrow">Recent searches</h2>
  {#if !history}
    <p class="muted">History is off · <a href="#/settings">Turn on in Settings</a></p>
  {:else if searches.length === 0}
    <p class="muted">Your searches will appear here</p>
  {:else}
    <ul class="chips">
      {#each searches as s (s.query)}
        <li><button type="button" class="chip" on:click={() => dispatch('search', s.query)}>{s.query}</button></li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .chips {
    list-style: none;
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
    margin-top: var(--space-sm);
  }

  .chip {
    padding: var(--space-xxs) var(--space-sm);
    border-radius: 999px;
    border: 1px solid var(--color-hairline);
    background-color: var(--color-surface-raised);
    color: var(--color-ink-secondary);
    font-size: var(--text-small-size);
    transition:
      color var(--motion-quick) var(--ease-standard),
      border-color var(--motion-quick) var(--ease-standard);
  }

  .chip:hover,
  .chip:focus-visible {
    color: var(--color-primary);
    border-color: var(--color-primary);
  }

  a {
    color: var(--color-primary);
  }
</style>
```

- [x] **Step 7: `StaleFileList.svelte`**

```svelte
<script lang="ts">
  // Stale files with their actions. Remove from index acts at once; Trash asks
  // first. Each row shows its own error.
  import { createEventDispatcher } from 'svelte'
  import { formatBytes } from '../../lib/format'
  import { filename } from '../../lib/path'
  import { age } from '../../lib/time'
  import type { StaleFile } from '../../stores/dashboard'
  import ConfirmDialog from '../../ui/ConfirmDialog.svelte'

  export let files: StaleFile[]
  export let busy = false // indexing: actions wait until it finishes
  export let errors: Record<string, string> = {}
  export let now = new Date()

  const dispatch = createEventDispatcher<{ remove: string; trash: string }>()
  let confirming: StaleFile | null = null

  function confirmTrash() {
    if (!confirming) return
    const path = confirming.path
    confirming = null
    dispatch('trash', path)
  }
</script>

<ul class="stale-list">
  {#each files as file (file.path)}
    <li class="stale-row">
      <div class="line">
        <span class="name" title={file.path}>{filename(file.path)}</span>
        <span class="meta">~{formatBytes(file.bytes)} · {age(new Date(file.modified), now)}</span>
        <span class="actions">
          <button type="button" class="remove" disabled={busy} on:click={() => dispatch('remove', file.path)}>Remove from index</button>
          <button type="button" class="trash" disabled={busy} on:click={() => (confirming = file)}>Trash…</button>
        </span>
      </div>
      {#if errors[file.path]}
        <p class="error" role="alert">{errors[file.path]}</p>
      {/if}
    </li>
  {/each}
</ul>

<ConfirmDialog
  open={confirming !== null}
  title={`Move ${confirming ? filename(confirming.path) : ''} to the Trash?`}
  confirmLabel="Move to Trash"
  on:confirm={confirmTrash}
  on:cancel={() => (confirming = null)}
>
  <p>You can restore it from the Trash.</p>
</ConfirmDialog>

<style>
  .stale-list {
    list-style: none;
  }

  .stale-row {
    padding: var(--space-xs) 0;
    border-top: 1px solid var(--color-hairline);
  }

  .stale-row:first-child {
    border-top: none;
  }

  .line {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .meta {
    flex-shrink: 0;
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
    font-variant-numeric: tabular-nums;
  }

  .actions {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    flex-shrink: 0;
  }

  .remove,
  .trash {
    padding: var(--space-xxs) var(--space-sm);
    border-radius: var(--radius-xs);
    font-size: var(--text-small-size);
    font-weight: var(--text-body-strong-weight);
    transition: background-color var(--motion-quick) var(--ease-standard);
  }

  .remove {
    background-color: var(--color-secondary-wash);
    color: var(--color-secondary);
  }

  .remove:hover:not(:disabled),
  .remove:focus-visible {
    background-color: var(--color-secondary-wash-strong);
  }

  .trash {
    color: var(--color-urgent);
  }

  .trash:hover:not(:disabled),
  .trash:focus-visible {
    background-color: var(--color-urgent-wash);
  }

  .remove:disabled,
  .trash:disabled {
    opacity: 0.5;
  }

  .error {
    margin-top: var(--space-xxs);
    font-size: var(--text-small-size);
    color: var(--color-urgent);
  }
</style>
```

- [x] **Step 8: `FreeSpaceTile.svelte`**

```svelte
<script lang="ts">
  // Free up space: files untouched for a while, largest first. Shows three;
  // "Show all" opens the full list.
  import { createEventDispatcher } from 'svelte'
  import { formatBytes } from '../../lib/format'
  import type { StaleFile } from '../../stores/dashboard'
  import StaleFileList from './StaleFileList.svelte'

  export let stale: StaleFile[]
  export let staleBytes: number
  export let staleDays: number
  export let busy = false
  export let errors: Record<string, string> = {}
  export let now = new Date()

  const SHOWN = 3
  const dispatch = createEventDispatcher<{ showAll: null }>()
  $: count = `${stale.length} ${stale.length === 1 ? 'file' : 'files'}`
</script>

<section class="tile free-space" aria-labelledby="free-space-title">
  <header class="head">
    <h2 id="free-space-title" class="eyebrow">Free up space</h2>
    {#if stale.length > 0}
      <span class="muted">{count} untouched for {staleDays}+ days · {formatBytes(staleBytes)} of index</span>
    {/if}
  </header>
  {#if stale.length === 0}
    <p class="muted">Nothing stale. Everything was used in the last {staleDays} days.</p>
  {:else}
    <StaleFileList files={stale.slice(0, SHOWN)} {busy} {errors} {now} on:remove on:trash />
    {#if stale.length > SHOWN}
      <button type="button" class="show-all" on:click={() => dispatch('showAll', null)}>Show all {stale.length} →</button>
    {/if}
  {/if}
</section>

<style>
  .head {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: var(--space-md);
    flex-wrap: wrap;
    margin-bottom: var(--space-xs);
  }

  .show-all {
    margin-top: var(--space-xs);
    color: var(--color-primary);
    font-size: var(--text-small-size);
  }

  .show-all:hover,
  .show-all:focus-visible {
    text-decoration: underline;
  }
</style>
```

- [x] **Step 9: `StaleFilesDialog.svelte`**

```svelte
<script lang="ts">
  // Every stale file, in a modal. Controlled by `open`; Esc and Close dispatch `close`.
  import { createEventDispatcher, tick } from 'svelte'
  import type { StaleFile } from '../../stores/dashboard'
  import Button from '../../ui/Button.svelte'
  import StaleFileList from './StaleFileList.svelte'

  export let open = false
  export let files: StaleFile[]
  export let busy = false
  export let errors: Record<string, string> = {}
  export let now = new Date()

  const dispatch = createEventDispatcher<{ close: void }>()
  let dialog: HTMLDialogElement

  $: if (dialog) void sync(open)

  async function sync(shouldOpen: boolean) {
    if (shouldOpen && !dialog.open) {
      dialog.showModal()
      await tick()
    } else if (!shouldOpen && dialog.open) {
      dialog.close()
    }
  }

  function onEscape(e: Event) {
    e.preventDefault()
    dispatch('close')
  }
</script>

<dialog bind:this={dialog} class="stale-dialog" aria-labelledby="stale-dialog-title" on:cancel={onEscape}>
  <header class="head">
    <h2 id="stale-dialog-title">Free up space</h2>
    <Button on:click={() => dispatch('close')}>Close</Button>
  </header>
  <div class="body">
    <StaleFileList {files} {busy} {errors} {now} on:remove on:trash />
  </div>
</dialog>

<style>
  .stale-dialog {
    width: min(720px, calc(100vw - 2 * var(--space-xl)));
    max-height: calc(100vh - 2 * var(--space-xxl));
    padding: 0;
    border-radius: var(--radius-md);
    border: 1px solid var(--color-hairline-strong);
    background-color: var(--color-surface-raised);
    color: var(--color-ink);
    box-shadow: var(--shadow-raised);
  }

  .stale-dialog::backdrop {
    background: rgb(0 0 0 / 0.45);
  }

  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md) var(--space-lg);
    border-bottom: 1px solid var(--color-hairline);
  }

  h2 {
    font-size: var(--text-title-size);
    line-height: var(--text-title-line);
    font-weight: var(--text-title-weight);
  }

  .body {
    padding: var(--space-sm) var(--space-lg) var(--space-lg);
    overflow-y: auto;
  }
</style>
```

- [x] **Step 10: `ViewerPanel.svelte`**

```svelte
<script lang="ts">
  // A file viewer sliding over the Dashboard. Esc or the close button dismisses it.
  import { createEventDispatcher } from 'svelte'
  import { fly } from 'svelte/transition'
  import { viewerKindFor } from '../../lib/fileTypes'
  import { duration } from '../../lib/motion'
  import PdfViewer from '../../viewers/PdfViewer.svelte'
  import TextViewer from '../../viewers/TextViewer.svelte'

  export let path: string | null = null

  const dispatch = createEventDispatcher<{ close: void }>()
  const close = () => dispatch('close')

  $: kind = path ? viewerKindFor(path) : 'text'

  function onKeydown(e: KeyboardEvent) {
    if (path && e.key === 'Escape') close()
  }
</script>

<svelte:window on:keydown={onKeydown} />

{#if path}
  <aside class="viewer-panel" aria-label="Preview" transition:fly={{ x: 48, duration: duration('normal') }}>
    {#key path}
      {#if kind === 'pdf'}
        <PdfViewer docPath={path} onClose={close} />
      {:else}
        <TextViewer docPath={path} mode={kind === 'markdown' ? 'markdown' : 'text'} onClose={close} />
      {/if}
    {/key}
  </aside>
{/if}

<style>
  .viewer-panel {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(720px, 62%);
    border-left: 1px solid var(--color-hairline-strong);
    box-shadow: var(--shadow-raised);
    z-index: 2;
  }
</style>
```

- [x] **Step 11: Replace `DashboardView.svelte`**

```svelte
<script lang="ts">
  // The launch page: greeting, search, and the Bento tiles. Searches open in
  // Chat; recent files open in a panel over the page.
  import { onDestroy, onMount } from 'svelte'
  import { push } from 'svelte-spa-router'
  import { fade } from 'svelte/transition'
  import { greeting } from '../../lib/greeting'
  import { statusLine } from '../../lib/indexStatus'
  import { duration } from '../../lib/motion'
  import { dashboard, engine, profile, search } from '../../stores/app'
  import Button from '../../ui/Button.svelte'
  import DashboardSearch from './DashboardSearch.svelte'
  import DiskTile from './DiskTile.svelte'
  import FreeSpaceTile from './FreeSpaceTile.svelte'
  import RecentFilesTile from './RecentFilesTile.svelte'
  import RecentSearchesTile from './RecentSearchesTile.svelte'
  import StaleFilesDialog from './StaleFilesDialog.svelte'
  import ViewerPanel from './ViewerPanel.svelte'

  const CLOCK_MS = 60_000 // relative times and the greeting move on each minute
  const TRASH_FAILED = "Couldn't move to Trash. Nothing was deleted."

  let now = new Date()
  let viewing: string | null = null
  let showAll = false
  let errors: Record<string, string> = {}
  const clock = setInterval(() => (now = new Date()), CLOCK_MS)

  onMount(() => {
    void dashboard.start()
  })

  onDestroy(() => {
    dashboard.stop()
    clearInterval(clock)
  })

  $: data = $dashboard.data
  $: status = statusLine($engine)
  $: indexing = $engine.indexing !== null
  $: hasLibrary = $engine.folders.length > 0 || $engine.documents > 0
  $: canSearch = $engine.ready && $engine.documents > 0
  $: placeholder = canSearch ? 'Search your files…' : hasLibrary ? 'Indexing — search opens when it finishes…' : 'Index a folder to start searching…'

  function runSearch(query: string) {
    void search.submit(query)
    void push('/chat')
  }

  function open(path: string) {
    viewing = path
    dashboard.opened(path)
  }

  async function removeFromIndex(path: string) {
    errors = { ...errors, [path]: '' }
    errors = { ...errors, [path]: await dashboard.removeFromIndex(path) }
  }

  async function moveToTrash(path: string) {
    errors = { ...errors, [path]: '' }
    const error = await dashboard.moveToTrash(path)
    if (error) console.error('dashboard: move to Trash failed', error)
    errors = { ...errors, [path]: error ? TRASH_FAILED : '' }
  }
</script>

<div class="dashboard-view" in:fade={{ duration: duration('normal') }}>
  <div class="content">
    <header class="top">
      <h1 class="display">{greeting(now.getHours(), $profile.name)}</h1>
      <p class="status" class:active={status.tone === 'active'} class:error={status.tone === 'error'}>{status.text}</p>
    </header>

    <DashboardSearch disabled={!canSearch} {placeholder} on:search={(e) => runSearch(e.detail)} />

    {#if !hasLibrary}
      <section class="tile empty">
        <p>Choose a folder to index, and your library's figures appear here.</p>
        <Button variant="primary" on:click={() => push('/settings')}>Choose a folder</Button>
      </section>
    {:else if !data && $dashboard.phase === 'error'}
      <section class="tile empty">
        <p class="error" role="alert">Couldn't load your library figures — {$dashboard.error}</p>
        <Button on:click={dashboard.reload}>Retry</Button>
      </section>
    {:else if data}
      <div class="grid">
        <DiskTile {data} {now} />
        <RecentFilesTile files={data.recentFiles} history={data.history} {now} on:open={(e) => open(e.detail)} on:forget={(e) => dashboard.forget(e.detail)} />
        <RecentSearchesTile searches={data.recentSearches} history={data.history} on:search={(e) => runSearch(e.detail)} />
        <FreeSpaceTile
          stale={data.stale}
          staleBytes={data.staleBytes}
          staleDays={data.staleDays}
          busy={indexing}
          {errors}
          {now}
          on:remove={(e) => removeFromIndex(e.detail)}
          on:trash={(e) => moveToTrash(e.detail)}
          on:showAll={() => (showAll = true)}
        />
      </div>
    {/if}
  </div>

  <StaleFilesDialog
    open={showAll}
    files={data?.stale ?? []}
    busy={indexing}
    {errors}
    {now}
    on:close={() => (showAll = false)}
    on:remove={(e) => removeFromIndex(e.detail)}
    on:trash={(e) => moveToTrash(e.detail)}
  />
  <ViewerPanel path={viewing} on:close={() => (viewing = null)} />
</div>

<style>
  .dashboard-view {
    position: relative;
    height: 100%;
    overflow: hidden;
  }

  .content {
    height: 100%;
    max-width: var(--content-max);
    margin: 0 auto;
    padding: var(--space-xxl) var(--space-xl);
    display: flex;
    flex-direction: column;
    gap: var(--space-lg);
    overflow-y: auto;
  }

  .top {
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    gap: var(--space-md);
    flex-wrap: wrap;
  }

  .display {
    font-family: var(--font-display);
    font-size: var(--text-display-size);
    line-height: var(--text-display-line);
    font-weight: var(--text-display-weight);
    letter-spacing: var(--text-display-tracking);
  }

  .status {
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
  }

  .status.active {
    color: var(--color-primary);
  }

  .status.error,
  .error {
    color: var(--color-urgent);
  }

  .grid {
    display: grid;
    grid-template-columns: minmax(0, 1.35fr) minmax(0, 1fr);
    grid-template-areas:
      'disk files'
      'disk searches'
      'free free';
    gap: var(--space-md);
  }

  .grid > :global(.disk) {
    grid-area: disk;
  }

  .grid > :global(.recent-files) {
    grid-area: files;
  }

  .grid > :global(.recent-searches) {
    grid-area: searches;
  }

  .grid > :global(.free-space) {
    grid-area: free;
  }

  /* About 900px of content beside the sidebar. */
  @media (max-width: 1160px) {
    .grid {
      grid-template-columns: minmax(0, 1fr);
      grid-template-areas: 'disk' 'files' 'searches' 'free';
    }
  }

  .dashboard-view :global(.tile) {
    padding: var(--space-md) var(--space-lg);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-hairline);
    background-color: var(--color-surface);
    box-shadow: var(--shadow-raised);
  }

  .dashboard-view :global(.tile .eyebrow) {
    font-size: var(--text-micro-size);
    font-weight: var(--text-micro-weight);
    letter-spacing: var(--text-micro-tracking);
    text-transform: uppercase;
    color: var(--color-ink-faint);
  }

  .dashboard-view :global(.tile .muted) {
    color: var(--color-ink-faint);
    font-size: var(--text-small-size);
  }

  .empty {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-md);
  }

  @media (prefers-reduced-motion: no-preference) {
    .grid > :global(.tile) {
      animation: rise var(--motion-slow) var(--ease-enter) both;
    }

    .grid > :global(.tile:nth-child(2)) {
      animation-delay: var(--motion-stagger);
    }

    .grid > :global(.tile:nth-child(3)) {
      animation-delay: calc(2 * var(--motion-stagger));
    }

    .grid > :global(.tile:nth-child(4)) {
      animation-delay: calc(3 * var(--motion-stagger));
    }
  }

  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(var(--motion-rise));
    }
  }
</style>
```

- [x] **Step 12: Run, check and commit**

```bash
cd frontend && npx vitest run src/views/dashboard && npm run check && npx vitest run
git add frontend/src/views/dashboard
git commit -m "feat(ui): Bento Dashboard with search, disk, recent and Free up space"
```

Expected: PASS, 0 errors.

---

### Task 16: Privacy section in Settings

**Files:**
- Create: `frontend/src/views/settings/PrivacySection.svelte`
- Modify: `frontend/src/views/settings/SettingsView.svelte`

- [x] **Step 1: Implement**

```svelte
<script lang="ts">
  // Privacy: whether history is kept, when files count as stale, clearing history.
  import { profile } from '../../stores/app'
  import { STALE_DAY_CHOICES, type StaleDays } from '../../stores/profile'
  import Button from '../../ui/Button.svelte'
  import ConfirmDialog from '../../ui/ConfirmDialog.svelte'
  import SegmentedControl from '../../ui/SegmentedControl.svelte'
  import SettingRow from './SettingRow.svelte'
  import SettingsSection from './SettingsSection.svelte'

  const HISTORY = [
    { value: 'on', label: 'On' },
    { value: 'off', label: 'Off' },
  ]
  const STALE = STALE_DAY_CHOICES.map((d) => ({ value: String(d), label: `${d} days` }))

  let confirming = false

  const onHistory = (e: CustomEvent<string>) => profile.setHistory(e.detail === 'on')
  const onStale = (e: CustomEvent<string>) => profile.setStaleDays(Number(e.detail) as StaleDays)

  async function clear() {
    confirming = false
    await profile.clearHistory()
  }
</script>

<SettingsSection title="Privacy">
  <SettingRow label="Remember searches and opened files" hint="Shown on the Dashboard. Stored only on this computer in ~/.spotfile">
    <SegmentedControl label="Remember searches and opened files" options={HISTORY} value={$profile.history ? 'on' : 'off'} on:change={onHistory} />
  </SettingRow>
  <SettingRow label="Stale after" hint="Files not changed or opened for this long appear in Free up space">
    <SegmentedControl label="Stale after" options={STALE} value={String($profile.staleDays)} on:change={onStale} />
  </SettingRow>
  <SettingRow label="Clear history" hint="Forgets recent searches and opened files">
    <Button variant="destructive" on:click={() => (confirming = true)}>Clear history…</Button>
  </SettingRow>
  {#if $profile.privacyError}
    <p class="error" role="alert">{$profile.privacyError}</p>
  {/if}
</SettingsSection>

<ConfirmDialog open={confirming} title="Clear your history?" confirmLabel="Clear history" on:confirm={clear} on:cancel={() => (confirming = false)}>
  <p>Recent searches and opened files are forgotten. Your files and the index stay as they are.</p>
</ConfirmDialog>

<style>
  .error {
    font-size: var(--text-small-size);
    color: var(--color-urgent);
  }
</style>
```

In `SettingsView.svelte`, import `PrivacySection from './PrivacySection.svelte'` and render `<PrivacySection />` after `<LibrarySection />`.

- [x] **Step 2: Check and commit**

```bash
cd frontend && npm run check && npx vitest run
git add frontend/src/views/settings
git commit -m "feat(ui): Privacy section for history and the stale threshold"
```

Expected: 0 errors, all tests pass.

---

### Task 17: Docs and verification

**Files:**
- Modify: `context/ROADMAP.md`, `context/PERFORMANCE.md`, `docs/superpowers/specs/2026-10-21-dashboard-profile-design.md`

- [x] **Step 1: Docs**

- ROADMAP Phase 6: mark `5. Dashboard and Profile` `[x]` and replace "implementation not started" with `Plan: docs/superpowers/plans/2026-10-21-dashboard-profile.md`.
- PERFORMANCE §6: add a bullet: `**Dashboard.** GetDashboard walks the index once (DocStats), stats each indexed file for Free up space, and reads the in-memory history; it runs on page load and at most once per 250 ms burst of index or history events. activity.json is rewritten on each search, opened file and index update — a few KB, atomically.`
- Spec: add `## Implementation notes` listing the seven deviations from the top of this plan.

- [x] **Step 2: Full automated run**

```bash
gofmt -l . ; go vet . ./engine/... && go test -race ./engine/...
GOOS=windows GOARCH=amd64 go vet ./engine/...
cd frontend && npm run check && npx vitest run && npm run build
```

Expected: `gofmt` prints nothing, all `ok`, 0 type errors, all Vitest tests pass, build succeeds.

- [x] **Step 3: Manual verification with `wails dev`**

Back up `~/.spotfile/store.gob` and note whether `settings.json`/`activity.json` exist; restore them afterwards. Use a scratch folder with a few files, two of them given old modification times (`touch -t 202401010000`). Check, in Nocturne and Daylight:
1. Launch lands on the Dashboard; the sidebar reads Dashboard, Chat, Settings, and "Add your name" at the foot.
2. Profile: type a name, press Enter → sidebar initials and the greeting update; restart → kept.
3. Dashboard search → Chat opens with results; the query appears under Recent searches.
4. Open a result's preview in Chat → it appears under Recently opened; clicking it on the Dashboard opens the slide-over viewer; Esc closes it.
5. Free up space lists the two old files; Remove from index → the row goes and Disk figures drop; Trash… → confirm → the file is in the macOS Trash (restore it from Finder afterwards) and gone from search.
6. Settings → Privacy: history Off → Recent tiles say "History is off"; Stale after 30 days → the list changes; Clear history → recent lists empty.
7. A narrow window collapses the grid to one column.

- [x] **Step 4: Commit**

```bash
git add context docs
git commit -m "docs: Dashboard and Profile done, performance notes"
```
