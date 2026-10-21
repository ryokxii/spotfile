package vectorstore

import "sort"

// DocStat is one indexed file's share of the index.
type DocStat struct {
	Path   string
	Chunks int
	Bytes  int64 // estimate: 4 bytes per vector component plus the chunk text
}

// DocStats returns every document with chunks, sorted by path.
func (vs *VectorStore) DocStats() []DocStat {
	vs.mu.RLock()
	out := make([]DocStat, 0, len(vs.docs))
	for p, chunks := range vs.docs {
		var b int64
		for _, c := range chunks {
			b += int64(len(c.Embedding))*4 + int64(len(c.Text))
		}
		out = append(out, DocStat{Path: p, Chunks: len(chunks), Bytes: b})
	}
	vs.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
