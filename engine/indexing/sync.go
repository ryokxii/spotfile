package indexing

// Keeping the index in sync with the disk: what is indexed is compared with
// what is on disk, so deleted files are dropped and new or changed files are
// indexed — at startup, after a folder walk, and when the watcher sees a path
// disappear.

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"spotfile/engine/pathutil"
	"spotfile/engine/vectorstore"
)

// SyncPlan is what it takes to bring the index in line with the disk.
type SyncPlan struct {
	Index  []string                     // new or changed on disk
	Remove []string                     // indexed but gone from disk
	Adopt  map[string]vectorstore.Stamp // unstamped and judged unchanged; record their stamp
}

// Reconcile compares the files on disk with the indexed ones. A file indexed
// before stamps were recorded (zero Stamp) counts as changed when it was
// modified after the index was last saved; otherwise its current stamp is
// adopted so the next comparison is exact. Inputs are not modified.
func Reconcile(onDisk, indexed map[string]vectorstore.Stamp, savedAt time.Time) SyncPlan {
	var plan SyncPlan
	for p := range indexed {
		if _, ok := onDisk[p]; !ok {
			plan.Remove = append(plan.Remove, p)
		}
	}
	for p, disk := range onDisk {
		was, ok := indexed[p]
		switch {
		case !ok:
			plan.Index = append(plan.Index, p)
		case was == (vectorstore.Stamp{}):
			if disk.ModTime.After(savedAt) {
				plan.Index = append(plan.Index, p)
			} else {
				if plan.Adopt == nil {
					plan.Adopt = make(map[string]vectorstore.Stamp)
				}
				plan.Adopt[p] = disk
			}
		case !was.ModTime.Equal(disk.ModTime) || was.Size != disk.Size:
			plan.Index = append(plan.Index, p)
		}
	}
	sort.Strings(plan.Index)
	sort.Strings(plan.Remove)
	return plan
}

// Scan walks dir and stamps every regular file include accepts. Unreadable
// entries below dir are skipped; dir itself must be readable.
func Scan(dir string, include func(string) bool) (map[string]vectorstore.Stamp, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("scan folder: %w", err)
	}
	out := make(map[string]vectorstore.Stamp)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.Mode().IsRegular() || !include(path) {
			return nil
		}
		out[path] = vectorstore.StampOf(info)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan folder: %w", err)
	}
	return out, nil
}

// MissingPaths returns, in order, the paths that are no longer regular files.
func MissingPaths(paths []string) []string {
	var gone []string
	for _, p := range paths {
		if info, err := os.Stat(p); err != nil || !info.Mode().IsRegular() {
			gone = append(gone, p)
		}
	}
	return gone
}

// stampsOf returns the current version of each path. A path that can't be
// stat'ed gets no stamp.
func stampsOf(paths []string) map[string]vectorstore.Stamp {
	out := make(map[string]vectorstore.Stamp, len(paths))
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil {
			out[p] = vectorstore.StampOf(info)
		}
	}
	return out
}

// Forget removes path and everything under it from the index, persisting when
// anything was removed, and returns how many files went.
func (ix *Indexer) Forget(path string) int {
	n := ix.store.RemoveUnder(path)
	if n == 0 {
		return 0
	}
	if err := ix.store.Persist(); err != nil {
		log.Printf("indexer: persist after forgetting %s failed: %v", path, err)
	}
	return n
}

// StampsUnder returns the entries within dir.
func StampsUnder(all map[string]vectorstore.Stamp, dir string) map[string]vectorstore.Stamp {
	out := make(map[string]vectorstore.Stamp)
	for p, s := range all {
		if pathutil.Within(p, dir) {
			out[p] = s
		}
	}
	return out
}

// Outside returns, sorted, the paths that lie within none of dirs.
func Outside(all map[string]vectorstore.Stamp, dirs []string) []string {
	var out []string
	for p := range all {
		inside := false
		for _, d := range dirs {
			if pathutil.Within(p, d) {
				inside = true
				break
			}
		}
		if !inside {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
