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
	"spotfile/engine/vectorstore"

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

// addFolder records dir as an indexed folder (see settings.WithFolderAdded).
func (a *App) addFolder(dir string) error {
	return a.updateSettings(func(s settings.Settings) settings.Settings {
		next, _ := s.WithFolderAdded(dir)
		return next
	})
}

// reconcileFolder brings the entries under dir in line with onDisk, the
// folder's files as just scanned, and returns the files to index: new ones and
// changed ones. Other folders' entries are untouched.
func (a *App) reconcileFolder(dir string, onDisk map[string]vectorstore.Stamp) []string {
	plan := indexing.Reconcile(onDisk, indexing.StampsUnder(a.store.Stamps(), dir), a.store.SavedAt())
	a.store.RemoveDocs(plan.Remove)
	a.store.AdoptStamps(plan.Adopt)
	if len(plan.Remove) > 0 || len(plan.Adopt) > 0 {
		a.persistChanged()
	}
	log.Printf("index: reconcile folder=%q index=%d remove=%d", dir, len(plan.Index), len(plan.Remove))
	return plan.Index
}

// persistChanged saves the index after entries changed outside an indexing job
// and tells the UI to refresh its figures.
// Emits: "index:changed".
func (a *App) persistChanged() {
	if err := a.store.Persist(); err != nil {
		log.Printf("store: %v", err)
	}
	wailsruntime.EventsEmit(a.ctx, "index:changed", nil)
}

// watchFolders restarts the watcher on every saved folder.
func (a *App) watchFolders() {
	a.restartWatcher(a.GetSettings().Folders)
}

// RemoveFolder stops indexing dir: its files leave the index and it is no
// longer watched. The files themselves stay on disk.
// Emits: "index:changed".
func (a *App) RemoveFolder(dir string) error {
	if !a.indexMu.TryLock() {
		return fmt.Errorf("indexing is in progress")
	}
	defer a.indexMu.Unlock()
	if err := a.updateSettings(func(s settings.Settings) settings.Settings { return s.WithFolderRemoved(dir) }); err != nil {
		return err
	}
	n := a.store.RemoveUnder(dir)
	a.persistChanged()
	a.watchFolders()
	log.Printf("index: removed folder=%q files=%d", dir, n)
	return nil
}

// syncSavedFolders runs once the engine is ready. It brings the loaded index in
// line with the disk: saved folders that are gone are forgotten with their
// files, entries outside every saved folder are dropped, each folder's new and
// changed files are indexed in one job, and the watcher resumes.
func (a *App) syncSavedFolders() {
	folders := a.GetSettings().Folders
	if len(folders) == 0 {
		a.dropMissing() // an index from before folders were saved: keep it, minus missing files
		return
	}
	if !a.indexMu.TryLock() {
		return // a folder index is starting and will start the watcher itself
	}
	var kept []string
	for _, dir := range folders {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			kept = append(kept, dir)
			continue
		}
		log.Printf("settings: saved folder %q is gone; forgetting it and its files", dir)
		if err := a.updateSettings(func(s settings.Settings) settings.Settings { return s.WithFolderRemoved(dir) }); err != nil {
			log.Printf("settings: %v", err)
		}
	}
	if gone := indexing.Outside(a.store.Stamps(), kept); len(gone) > 0 {
		a.store.RemoveDocs(gone)
		a.persistChanged()
		log.Printf("index: dropped %d files outside the saved folders", len(gone))
	}
	var toIndex []string
	for _, dir := range kept {
		onDisk, err := indexing.Scan(dir, isIndexable)
		if err != nil {
			log.Printf("index: catch-up scan of %q failed: %v", dir, err)
			continue
		}
		toIndex = append(toIndex, a.reconcileFolder(dir, onDisk)...)
	}
	a.watchFolders()
	if len(toIndex) == 0 {
		a.indexMu.Unlock()
		return
	}
	if err := a.enqueueIndexJob(singleFolder(kept), toIndex); err != nil {
		a.indexMu.Unlock()
		wailsruntime.EventsEmit(a.ctx, "index:error", err.Error())
	}
}

// singleFolder is the folder's path when there is exactly one, else "".
func singleFolder(folders []string) string {
	if len(folders) == 1 {
		return folders[0]
	}
	return ""
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
// folders and the watcher are left alone.
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
