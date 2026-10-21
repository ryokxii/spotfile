package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != Defaults() {
		t.Fatalf("got %+v, want defaults", got)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := Settings{Theme: ThemeDaylight, Motion: MotionReduce, Folder: "/Users/me/Notes"}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadFillsMissingFieldsWithDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"folder":"/notes"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Settings{Theme: ThemeSystem, Motion: MotionSystem, Folder: "/notes"}
	if got != want {
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
			if got != Defaults() {
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

func TestWithFolder(t *testing.T) {
	base := Settings{Theme: ThemeSystem, Motion: MotionSystem, Folder: "/notes"}
	for _, tc := range []struct {
		name        string
		from, dir   string
		wantChanged bool
	}{
		{name: "same folder", from: "/notes", dir: "/notes", wantChanged: false},
		{name: "same folder, trailing slash", from: "/notes", dir: "/notes/", wantChanged: false},
		{name: "different folder", from: "/notes", dir: "/work", wantChanged: true},
		{name: "first folder", from: "", dir: "/notes", wantChanged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := base
			start.Folder = tc.from
			got, changed := start.WithFolder(tc.dir)
			if changed != tc.wantChanged {
				t.Fatalf("changed = %v, want %v", changed, tc.wantChanged)
			}
			if got.Folder != tc.dir || got.Theme != base.Theme || got.Motion != base.Motion {
				t.Fatalf("got %+v", got)
			}
			if start.Folder != tc.from {
				t.Fatalf("WithFolder modified its receiver: %+v", start)
			}
		})
	}
}
