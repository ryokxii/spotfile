package engine

import (
	"log"
	"os"
	"sort"
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
