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

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := Settings{Theme: ThemeDaylight, Motion: MotionReduce, Folders: []string{"/Users/me/Notes", "/Users/me/Work"}}
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
	want := Settings{Theme: ThemeSystem, Motion: MotionSystem, Folders: []string{"/notes"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadSetsAsideUnusableFiles(t *testing.T) {
	for name, content := range map[string]string{
		"corrupt json":  `{"theme":`,
		"unknown theme": `{"theme":"sepia","motion":"system"}`,
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
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("unusable file still in place: %v", err)
			}
		})
	}
}

func TestSaveRejectsInvalidValues(t *testing.T) {
	for name, s := range map[string]Settings{
		"theme":  {Theme: "sepia", Motion: MotionSystem},
		"motion": {Theme: ThemeSystem, Motion: "slow"},
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

func TestSaveCreatesParentDirAndLeavesNoTempFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")
	path := filepath.Join(dir, "settings.json")
	for range 2 {
		if err := Save(path, Defaults()); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "settings.json" {
		t.Fatalf("dir entries = %v, want only settings.json", entries)
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

func TestDefaultsHaveNoFolders(t *testing.T) {
	if d := Defaults(); d.Folders == nil || len(d.Folders) != 0 {
		t.Fatalf("Defaults().Folders = %#v, want an empty list", d.Folders)
	}
}

func TestSaveRejectsRelativeFolder(t *testing.T) {
	s := Defaults()
	s.Folders = []string{"notes"}
	if err := Save(filepath.Join(t.TempDir(), "settings.json"), s); err == nil {
		t.Fatal("Save accepted a relative folder")
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
	if got := got.WithFolderRemoved("/unknown"); !reflect.DeepEqual(got.Folders, []string{"/work"}) {
		t.Fatalf("removing an unknown folder changed the list: %v", got.Folders)
	}
}
