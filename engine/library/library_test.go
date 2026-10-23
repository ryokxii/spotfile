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
