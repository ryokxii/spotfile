package vectorstore

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
)

// SetPersistPath configures the file Persist() writes to. Call once at startup.
func (vs *VectorStore) SetPersistPath(path string) {
	vs.mu.Lock()
	vs.persistPath = path
	vs.mu.Unlock()
}

// Persist writes the store to its configured path. No-op when unset.
func (vs *VectorStore) Persist() error {
	vs.mu.RLock()
	path := vs.persistPath
	vs.mu.RUnlock()
	if path == "" {
		return nil
	}
	return vs.Save(path)
}

// Save gob-encodes the store to path via a temp file + atomic rename, so a
// crash mid-write can never leave a truncated store on disk.
func (vs *VectorStore) Save(path string) error {
	vs.mu.RLock()
	snapshot := make(map[string][]EmbeddedChunk, len(vs.docs))
	for k, chunks := range vs.docs {
		cp := make([]EmbeddedChunk, len(chunks))
		copy(cp, chunks) // decouple slice from concurrent appends; embeddings are never mutated in place
		snapshot[k] = cp
	}
	vs.mu.RUnlock()

	tmp, err := os.CreateTemp(filepath.Dir(path), ".store-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp store: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if err := gob.NewEncoder(tmp).Encode(snapshot); err != nil {
		tmp.Close()
		return fmt.Errorf("encode store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp store: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace store: %w", err)
	}
	return nil
}

// Load replaces the in-memory store with the gob-encoded contents of path. A
// missing file is not an error — the store simply stays empty.
func (vs *VectorStore) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open store: %w", err)
	}
	defer f.Close()

	docs := make(map[string][]EmbeddedChunk)
	if err := gob.NewDecoder(f).Decode(&docs); err != nil {
		return fmt.Errorf("decode store: %w", err)
	}

	vs.mu.Lock()
	vs.docs = docs
	vs.mu.Unlock()
	return nil
}
