// Package vectorstore holds embedded chunks in memory, answers nearest-neighbour
// queries, and persists the index to disk.
package vectorstore

import (
	"sort"
	"sync"
)

type EmbeddedChunk struct {
	DocPath   string
	ChunkIdx  int
	PageNum   int // 1-based page number for PDFs; 0 for non-PDF files
	Text      string
	Embedding []float32
}

type SearchResult struct {
	DocPath  string  `json:"docPath"`
	ChunkIdx int     `json:"chunkIdx"`
	PageNum  int     `json:"pageNum"`
	Text     string  `json:"text"`
	Score    float32 `json:"score"`
}

// VectorStore holds embedded chunks keyed by document path. Keying by DocPath
// (rather than a flat slice) lets a re-index of a single file replace exactly
// that file's chunks in O(1) instead of accumulating duplicates.
type VectorStore struct {
	mu          sync.RWMutex
	docs        map[string][]EmbeddedChunk
	persistPath string // where Persist() writes; empty disables autosave
}

// Add appends a chunk to its document's bucket.
func (vs *VectorStore) Add(c EmbeddedChunk) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	if vs.docs == nil {
		vs.docs = make(map[string][]EmbeddedChunk)
	}
	vs.docs[c.DocPath] = append(vs.docs[c.DocPath], c)
}

// Len returns the total number of stored chunks across all documents.
func (vs *VectorStore) Len() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	n := 0
	for _, chunks := range vs.docs {
		n += len(chunks)
	}
	return n
}

// Search returns the topK nearest chunks by cosine similarity. Embeddings are
// assumed L2-normalized, so similarity == dot product. Returns an empty slice
// for an empty query or non-positive topK. Candidates whose embedding
// dimensionality differs from the query are skipped rather than mis-scored.
func (vs *VectorStore) Search(query []float32, topK int) []SearchResult {
	if len(query) == 0 || topK <= 0 {
		return []SearchResult{}
	}

	vs.mu.RLock()
	defer vs.mu.RUnlock()

	type scored struct {
		c     EmbeddedChunk
		score float32
	}
	var candidates []scored
	for _, chunks := range vs.docs {
		for _, c := range chunks {
			if len(c.Embedding) != len(query) {
				continue // dimensionality mismatch — not comparable
			}
			candidates = append(candidates, scored{c, dot(query, c.Embedding)})
		}
	}

	// SliceStable keeps ranking deterministic when scores tie (map iteration
	// order is otherwise random).
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if topK > len(candidates) {
		topK = len(candidates)
	}
	results := make([]SearchResult, topK)
	for i := range topK {
		results[i] = SearchResult{
			DocPath:  candidates[i].c.DocPath,
			ChunkIdx: candidates[i].c.ChunkIdx,
			PageNum:  candidates[i].c.PageNum,
			Text:     candidates[i].c.Text,
			Score:    candidates[i].score,
		}
	}
	return results
}

func dot(a, b []float32) float32 {
	var s float32
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

// RemoveDoc deletes all chunks belonging to docPath. Call before re-adding a
// changed file so a re-index replaces its chunks instead of duplicating them.
func (vs *VectorStore) RemoveDoc(docPath string) {
	vs.mu.Lock()
	delete(vs.docs, docPath)
	vs.mu.Unlock()
}

// RemoveDocs deletes all chunks for each of the given document paths.
func (vs *VectorStore) RemoveDocs(paths []string) {
	vs.mu.Lock()
	for _, p := range paths {
		delete(vs.docs, p)
	}
	vs.mu.Unlock()
}
