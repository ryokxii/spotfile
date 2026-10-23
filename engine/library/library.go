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
