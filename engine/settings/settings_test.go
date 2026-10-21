package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
	want.Folders = []string{abs("/Users/me/Notes"), abs("/Users/me/Work")}
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
	if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"folder":%q}`, abs("/notes"))), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Defaults()
	want.Folders = []string{abs("/notes")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLegacyFolderIsNotSavedBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"folder":%q}`, abs("/notes"))), 0o644); err != nil {
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
		{"first folder", []string{}, abs("/notes"), []string{abs("/notes")}, true},
		{"another folder", []string{abs("/notes")}, abs("/work"), []string{abs("/notes"), abs("/work")}, true},
		{"same folder, trailing slash", []string{abs("/notes")}, abs("/notes/"), []string{abs("/notes")}, false},
		{"inside an indexed folder", []string{abs("/notes")}, abs("/notes/2026"), []string{abs("/notes")}, false},
		{"contains indexed folders", []string{abs("/notes/a"), abs("/work"), abs("/notes/b")}, abs("/notes"), []string{abs("/work"), abs("/notes")}, true},
		{"sibling sharing a prefix", []string{abs("/notes")}, abs("/notes-old"), []string{abs("/notes"), abs("/notes-old")}, true},
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
	start.Folders = []string{abs("/notes"), abs("/work")}
	got := start.WithFolderRemoved(abs("/notes/"))
	if !reflect.DeepEqual(got.Folders, []string{abs("/work")}) {
		t.Fatalf("Folders = %v", got.Folders)
	}
	if len(start.Folders) != 2 {
		t.Fatalf("receiver modified: %v", start.Folders)
	}
}

func TestDefaultsHaveNoFiles(t *testing.T) {
	if d := Defaults(); d.Files == nil || len(d.Files) != 0 {
		t.Fatalf("Defaults().Files = %#v, want an empty list", d.Files)
	}
}

func TestOlderFileWithoutFilesLoadsEmptyList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"folders":[%q],"files":null}`, abs("/notes"))), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Files == nil || len(got.Files) != 0 {
		t.Fatalf("Files = %#v", got.Files)
	}
}

func TestFilesRoundTripAndValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := Defaults()
	want.Files = []string{abs("/a/b.md"), abs("/c/d.json")}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v", got, err)
	}
	bad := Defaults()
	bad.Files = []string{"relative.md"}
	if err := bad.Validate(); err == nil {
		t.Fatal("Validate accepted a relative file")
	}
}

func TestWithFileAdded(t *testing.T) {
	for _, tc := range []struct {
		name        string
		folders     []string
		files       []string
		path        string
		want        []string
		wantChanged bool
	}{
		{"first file", nil, nil, abs("/a/b.md"), []string{abs("/a/b.md")}, true},
		{"another file", nil, []string{abs("/a/b.md")}, abs("/c/d.md"), []string{abs("/a/b.md"), abs("/c/d.md")}, true},
		{"duplicate", nil, []string{abs("/a/b.md")}, abs("/a/./b.md"), []string{abs("/a/b.md")}, false},
		{"inside an indexed folder", []string{abs("/a")}, nil, abs("/a/x/b.md"), []string{}, false},
		{"beside an indexed folder", []string{abs("/a")}, nil, abs("/ab/b.md"), []string{abs("/ab/b.md")}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := Defaults()
			start.Folders = append([]string{}, tc.folders...)
			start.Files = append([]string{}, tc.files...)
			got, changed := start.WithFileAdded(tc.path)
			if changed != tc.wantChanged || !reflect.DeepEqual(got.Files, tc.want) {
				t.Fatalf("got %v (changed %v), want %v (%v)", got.Files, changed, tc.want, tc.wantChanged)
			}
			if !reflect.DeepEqual(start.Files, append([]string{}, tc.files...)) {
				t.Fatalf("receiver modified: %v", start.Files)
			}
		})
	}
}

func TestWithFileRemoved(t *testing.T) {
	start := Defaults()
	start.Files = []string{abs("/a/b.md"), abs("/c/d.md")}
	got := start.WithFileRemoved(abs("/a/b.md"))
	if !reflect.DeepEqual(got.Files, []string{abs("/c/d.md")}) || len(start.Files) != 2 {
		t.Fatalf("got %v, receiver %v", got.Files, start.Files)
	}
}

func TestAddingAFolderAbsorbsItsFiles(t *testing.T) {
	start := Defaults()
	start.Files = []string{abs("/notes/a.md"), abs("/work/b.md"), abs("/notes/sub/c.md")}
	got, _ := start.WithFolderAdded(abs("/notes"))
	if !reflect.DeepEqual(got.Files, []string{abs("/work/b.md")}) {
		t.Fatalf("Files = %v", got.Files)
	}
	if len(start.Files) != 3 {
		t.Fatalf("receiver modified: %v", start.Files)
	}
}

func TestRoots(t *testing.T) {
	s := Defaults()
	s.Folders = []string{abs("/notes")}
	s.Files = []string{abs("/work/b.md")}
	if got := s.Roots(); !reflect.DeepEqual(got, []string{abs("/notes"), abs("/work/b.md")}) {
		t.Fatalf("Roots = %v", got)
	}
}

// abs makes a slash path like "/notes" absolute on this OS — "C:\notes" on
// Windows — so the same cases run everywhere.
func abs(p string) string {
	p = filepath.FromSlash(p)
	if runtime.GOOS == "windows" {
		return "C:" + p
	}
	return p
}
