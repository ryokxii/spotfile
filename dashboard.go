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
	if err := trash.Move(a.trasher, path, a.GetSettings().Roots(), a.isIndexed); err != nil {
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
