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
