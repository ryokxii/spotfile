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
