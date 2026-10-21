package vectorstore

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Stamp identifies the version of a file that was indexed. Comparing it with
// the file on disk tells whether the file changed since.
type Stamp struct {
	ModTime time.Time
	Size    int64
}

// StampOf returns the stamp of a file's current version.
func StampOf(info os.FileInfo) Stamp {
	return Stamp{ModTime: info.ModTime(), Size: info.Size()}
}

// SetStamp records the version of path that was indexed.
func (vs *VectorStore) SetStamp(path string, s Stamp) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	if vs.stamps == nil {
		vs.stamps = make(map[string]Stamp)
	}
	vs.stamps[path] = s
}

// AdoptStamps records stamps for paths already indexed, leaving chunks alone.
func (vs *VectorStore) AdoptStamps(stamps map[string]Stamp) {
	for p, s := range stamps {
		vs.SetStamp(p, s)
	}
}

// Stamps returns a copy covering every indexed path: each document with
// chunks, plus files indexed without producing any. A document indexed before
// stamps were recorded has the zero Stamp.
func (vs *VectorStore) Stamps() map[string]Stamp {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	out := make(map[string]Stamp, len(vs.docs)+len(vs.stamps))
	for p := range vs.docs {
		out[p] = Stamp{}
	}
	for p, s := range vs.stamps {
		out[p] = s
	}
	return out
}

// RemoveUnder removes path and everything below it, returning how many
// indexed paths were removed. A sibling that merely shares the prefix
// ("/a/bc" for "/a/b") is kept.
func (vs *VectorStore) RemoveUnder(path string) int {
	path = filepath.Clean(path)
	prefix := path + string(filepath.Separator)
	under := func(p string) bool { return p == path || strings.HasPrefix(p, prefix) }

	vs.mu.Lock()
	defer vs.mu.Unlock()
	removed := make(map[string]bool)
	for p := range vs.docs {
		if under(p) {
			delete(vs.docs, p)
			removed[p] = true
		}
	}
	for p := range vs.stamps {
		if under(p) {
			delete(vs.stamps, p)
			removed[p] = true
		}
	}
	return len(removed)
}

// SavedAt is when the store file was last written, or the zero time when
// there is none.
func (vs *VectorStore) SavedAt() time.Time {
	vs.mu.RLock()
	path := vs.persistPath
	vs.mu.RUnlock()
	if path == "" {
		return time.Time{}
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return time.Time{}
	}
	return info.ModTime()
}
