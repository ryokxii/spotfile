package indexing

import (
	"log"
	"os"
	"sort"
	"time"
)

// SortPathsByModTime sorts paths by modification time (most recent first).
// Paths that cannot be stat'd are logged and skipped.
func SortPathsByModTime(paths []string) []string {
	type pathTime struct {
		path    string
		modTime int64
	}

	var pathTimes []pathTime
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			log.Printf("sort: skip %s: %v", p, err)
			continue
		}
		pathTimes = append(pathTimes, pathTime{
			path:    p,
			modTime: info.ModTime().UnixNano(),
		})
	}

	// Sort by modification time (descending: most recent first)
	sort.Slice(pathTimes, func(i, j int) bool {
		return pathTimes[i].modTime > pathTimes[j].modTime
	})

	result := make([]string, len(pathTimes))
	for i, pt := range pathTimes {
		result[i] = pt.path
	}
	return result
}

// SplitByModTime partitions paths into those modified within the given window
// (recent) and the rest (older), each sorted most-recent-first. Paths that
// cannot be stat'd are logged and dropped. Used to feed recent files to the
// indexer at high priority and history at low priority.
func SplitByModTime(paths []string, within time.Duration) (recent, older []string) {
	type pathTime struct {
		path    string
		modTime int64
	}

	cutoff := time.Now().Add(-within).UnixNano()
	var recents, olders []pathTime
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			log.Printf("sort: skip %s: %v", p, err)
			continue
		}
		pt := pathTime{path: p, modTime: info.ModTime().UnixNano()}
		if pt.modTime >= cutoff {
			recents = append(recents, pt)
		} else {
			olders = append(olders, pt)
		}
	}

	sortDesc := func(pts []pathTime) []string {
		sort.Slice(pts, func(i, j int) bool { return pts[i].modTime > pts[j].modTime })
		out := make([]string, len(pts))
		for i, pt := range pts {
			out[i] = pt.path
		}
		return out
	}
	return sortDesc(recents), sortDesc(olders)
}
