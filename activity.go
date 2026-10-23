package main

// Activity: the Dashboard's local history (recent searches, opened files and
// the last file the index updated), kept in ~/.spotfile/activity.json.

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"spotfile/engine/activity"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func activityPath() string {
	return filepath.Join(spotfileDir(), "activity.json")
}

// loadActivity reads the history at startup. An unusable file is set aside and
// history starts empty, so it never blocks launch.
func (a *App) loadActivity() {
	l, err := activity.Load(activityPath())
	if err != nil {
		log.Printf("activity: %v; starting empty", err)
	}
	a.activityMu.Lock()
	a.activity = l
	a.activityMu.Unlock()
}

func (a *App) activitySnapshot() activity.Log {
	a.activityMu.Lock()
	defer a.activityMu.Unlock()
	return a.activity
}

// changeActivity applies change, saves the history and tells the Dashboard.
// The change stays in memory even if saving fails; the error is returned.
// Emits: "activity:changed".
func (a *App) changeActivity(change func(activity.Log) activity.Log) error {
	a.activityMu.Lock()
	a.activity = change(a.activity)
	err := activity.Save(activityPath(), a.activity) // under the lock: writes stay in order
	a.activityMu.Unlock()
	wailsruntime.EventsEmit(a.ctx, "activity:changed", nil)
	if err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	return nil
}

// recordSearch remembers a successful search when history is on.
func (a *App) recordSearch(query string) {
	if !a.GetSettings().History {
		return
	}
	err := a.changeActivity(func(l activity.Log) activity.Log { return activity.WithSearch(l, query, time.Now()) })
	if err != nil {
		log.Printf("activity: %v", err)
	}
}

// recordUpdate remembers the last file the index updated. It is kept even with
// history off: it describes the library, not the user.
func (a *App) recordUpdate(path string) {
	err := a.changeActivity(func(l activity.Log) activity.Log { return activity.WithUpdate(l, path, time.Now()) })
	if err != nil {
		log.Printf("activity: %v", err)
	}
}

// RecordOpened remembers a file opened in a viewer. Nothing is kept with
// history off.
func (a *App) RecordOpened(path string) error {
	if !a.GetSettings().History {
		return nil
	}
	return a.changeActivity(func(l activity.Log) activity.Log { return activity.WithOpened(l, path, time.Now()) })
}

// ForgetOpened removes a file from the recently opened list.
func (a *App) ForgetOpened(path string) error {
	return a.changeActivity(func(l activity.Log) activity.Log { return activity.WithoutOpened(l, path) })
}

// ClearHistory forgets recent searches and opened files.
func (a *App) ClearHistory() error {
	return a.changeActivity(activity.Cleared)
}
