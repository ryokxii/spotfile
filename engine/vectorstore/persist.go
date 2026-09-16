package vectorstore

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
)

// formatVersion is bumped whenever the on-disk layout changes.
const formatVersion = 2

// storeFile is the on-disk layout. Version 1 stores were a bare
// map[string][]EmbeddedChunk with no model information.
type storeFile struct {
	Version  int
	Embedder string
	Docs     map[string][]EmbeddedChunk
}

// StaleError reports a store on disk that cannot be used with the current
// embedding model — written by an older format or a different model. Its
// chunks are not loaded; Paths lists the documents it covered so they can be
// re-indexed.
type StaleError struct {
	Reason string
	Paths  []string
}

func (e *StaleError) Error() string {
	return fmt.Sprintf("index is stale (%s): %d documents need re-indexing", e.Reason, len(e.Paths))
}

// SetEmbedder records which embedding model produces this store's vectors.
// Save writes it, and Load refuses stores written with a different one.
func (vs *VectorStore) SetEmbedder(id string) {
	vs.mu.Lock()
	vs.embedder = id
	vs.mu.Unlock()
}

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
	embedder := vs.embedder
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

	file := storeFile{Version: formatVersion, Embedder: embedder, Docs: snapshot}
	if err := gob.NewEncoder(tmp).Encode(file); err != nil {
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

// Load replaces the in-memory store with the contents of path. A missing file
// is not an error — the store simply stays empty. A store written by an older
// format or a different embedder returns *StaleError and loads nothing.
func (vs *VectorStore) Load(path string) error {
	file, err := readStoreFile(path)
	if err != nil || file == nil {
		return err
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()
	if file.Embedder != vs.embedder {
		return &StaleError{
			Reason: fmt.Sprintf("built with %q, current model is %q", file.Embedder, vs.embedder),
			Paths:  docPaths(file.Docs),
		}
	}
	vs.docs = file.Docs
	return nil
}

// readStoreFile decodes path, returning nil for a missing file and
// *StaleError for a version-1 store.
func readStoreFile(path string) (*storeFile, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open store: %w", err)
	}
	defer f.Close()

	var file storeFile
	if err := gob.NewDecoder(f).Decode(&file); err == nil && file.Version == formatVersion {
		return &file, nil
	}

	// Not the current format: try version 1 before reporting corruption.
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("rewind store: %w", err)
	}
	var legacy map[string][]EmbeddedChunk
	if err := gob.NewDecoder(f).Decode(&legacy); err != nil {
		return nil, fmt.Errorf("decode store: %w", err)
	}
	return nil, &StaleError{Reason: "written by an older version of Spotfile", Paths: docPaths(legacy)}
}

func docPaths(docs map[string][]EmbeddedChunk) []string {
	paths := make([]string, 0, len(docs))
	for p := range docs {
		paths = append(paths, p)
	}
	return paths
}
