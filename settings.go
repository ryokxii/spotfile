package main

// Settings bindings: appearance preferences and the saved folder, persisted to
// ~/.spotfile/settings.json.

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"spotfile/engine/indexing"
	"spotfile/engine/settings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func settingsPath() string {
	return filepath.Join(spotfileDir(), "settings.json")
}

// loadSettings reads saved settings at startup. An unusable file is set aside
// and defaults are used, so settings never block launch.
func (a *App) loadSettings() {
	s, err := settings.Load(settingsPath())
	if err != nil {
		log.Printf("settings: %v; using defaults", err)
	}
	a.settingsMu.Lock()
	a.settings = s
	a.settingsMu.Unlock()
}

// GetSettings returns the current settings.
func (a *App) GetSettings() settings.Settings {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return a.settings
}

// SetAppearance saves the theme and reduce-motion preferences.
func (a *App) SetAppearance(theme, motion string) error {
	return a.updateSettings(func(s settings.Settings) settings.Settings {
		s.Theme = theme
		s.Motion = motion
		return s
	})
}

// updateSettings saves change(current) and keeps it only if the save succeeds.
func (a *App) updateSettings(change func(settings.Settings) settings.Settings) error {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	next := change(a.settings)
	if err := settings.Save(settingsPath(), next); err != nil {
		return err
	}
	a.settings = next
	return nil
}

// useFolder records dir as the indexed folder. Switching to a different folder
// empties the index first, so the index only ever covers one folder.
func (a *App) useFolder(dir string) error {
	if _, changed := a.GetSettings().WithFolder(dir); !changed {
		return nil
	}
	a.store.Clear()
	if err := a.store.Persist(); err != nil {
		return fmt.Errorf("empty the index for the new folder: %w", err)
	}
	return a.updateSettings(func(s settings.Settings) settings.Settings {
		next, _ := s.WithFolder(dir)
		return next
	})
}

// syncSavedFolder runs once the engine is ready. It brings the loaded index in
// line with the disk: files deleted while Spotfile was closed are dropped, new
// and changed files are indexed, and the watcher resumes on the saved folder.
func (a *App) syncSavedFolder() {
	dir := a.GetSettings().Folder
	if dir == "" {
		a.dropMissing()
		return
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		log.Printf("settings: saved folder %q is gone; forgetting it and its index", dir)
		forget := func(s settings.Settings) settings.Settings { s.Folder = ""; return s }
		if err := a.updateSettings(forget); err != nil {
			log.Printf("settings: %v", err)
		}
		a.store.Clear()
		if err := a.store.Persist(); err != nil {
			log.Printf("store: %v", err)
		}
		wailsruntime.EventsEmit(a.ctx, "index:changed", nil)
		return
	}
	if !a.indexMu.TryLock() {
		return // a folder index is starting and will start its own watcher
	}
	onDisk, err := indexing.Scan(dir, isIndexable)
	if err != nil {
		log.Printf("index: catch-up scan of %q failed: %v", dir, err)
		a.restartWatcher(dir)
		a.indexMu.Unlock()
		return
	}
	plan := indexing.Reconcile(onDisk, a.store.Stamps(), a.store.SavedAt())
	a.store.RemoveDocs(plan.Remove)
	a.store.AdoptStamps(plan.Adopt)
	if len(plan.Remove) > 0 || len(plan.Adopt) > 0 {
		if err := a.store.Persist(); err != nil {
			log.Printf("store: %v", err)
		}
		wailsruntime.EventsEmit(a.ctx, "index:changed", nil)
	}
	log.Printf("index: catch-up folder=%q index=%d remove=%d", dir, len(plan.Index), len(plan.Remove))
	a.restartWatcher(dir)
	if len(plan.Index) == 0 {
		a.indexMu.Unlock()
		return
	}
	if err := a.enqueueIndexJob(dir, plan.Index); err != nil {
		a.indexMu.Unlock()
		wailsruntime.EventsEmit(a.ctx, "index:error", err.Error())
	}
}

// dropMissing removes entries whose files no longer exist. Used when no folder
// is saved, so there is nothing to scan.
func (a *App) dropMissing() {
	gone := indexing.MissingPaths(keys(a.store.Stamps()))
	if len(gone) == 0 {
		return
	}
	a.store.RemoveDocs(gone)
	if err := a.store.Persist(); err != nil {
		log.Printf("store: %v", err)
	}
	log.Printf("index: dropped %d missing files", len(gone))
	wailsruntime.EventsEmit(a.ctx, "index:changed", nil)
}

// ClearIndex empties the search index and deletes its file. Models, the saved
// folder and the watcher are left alone.
// Emits: "index:cleared".
func (a *App) ClearIndex() error {
	if !a.indexMu.TryLock() {
		return fmt.Errorf("indexing is in progress")
	}
	defer a.indexMu.Unlock()
	err := a.store.Wipe()
	log.Printf("index: cleared by user")
	wailsruntime.EventsEmit(a.ctx, "index:cleared", nil)
	if err != nil {
		return fmt.Errorf("index cleared, but the file couldn't be removed: %w", err)
	}
	return nil
}
