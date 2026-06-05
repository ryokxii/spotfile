package engine

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

type VectorStore struct {
	mu     sync.RWMutex
	chunks []EmbeddedChunk
}

func (vs *VectorStore) Add(c EmbeddedChunk) {
	vs.mu.Lock()
	vs.chunks = append(vs.chunks, c)
	vs.mu.Unlock()
}

func (vs *VectorStore) Len() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	return len(vs.chunks)
}

// Search returns the topK nearest neighbors by cosine similarity.
// Assumes embeddings are already L2-normalized, so similarity == dot product.
func (vs *VectorStore) Search(query []float32, topK int) []SearchResult {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	type scored struct {
		c     EmbeddedChunk
		score float32
	}
	candidates := make([]scored, 0, len(vs.chunks))
	for _, c := range vs.chunks {
		candidates = append(candidates, scored{c, dot(query, c.Embedding)})
	}
	sort.Slice(candidates, func(i, j int) bool {
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
