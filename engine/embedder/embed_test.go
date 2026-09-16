package embedder_test

import (
	"context"
	"math"
	"strings"
	"testing"

	"spotfile/engine/embedder/embeddertest"
)

// TestEmbedSmoke runs a real end-to-end embedding against the installed
// bge-small-en-v1.5 assets in ~/.spotfile.
func TestEmbedSmoke(t *testing.T) {
	m := embeddertest.RealModel(t, 1)

	vec, err := m.Embed(context.Background(), "the quick brown fox")
	if err != nil {
		t.Fatalf("embed failed: %v", err)
	}
	// bge-small-en-v1.5 produces 384-dimensional embeddings.
	if len(vec) != 384 {
		t.Fatalf("unexpected embedding dimension: got %d, want 384", len(vec))
	}
}

// TestBatchEmbedMatchesFullPadding guards the dynamic-padding optimisation:
// trimming a batch to its longest sequence must not change the embeddings
// compared with the single-text path, which always pads to MaxSeqLen.
func TestBatchEmbedMatchesFullPadding(t *testing.T) {
	m := embeddertest.RealModel(t, 1)

	texts := []string{
		"the quick brown fox",
		strings.Repeat("semantic search over local documents ", 40),
		"a",
	}
	batch, err := m.BatchEmbed(texts)
	if err != nil {
		t.Fatalf("BatchEmbed: %v", err)
	}
	for i, text := range texts {
		single, err := m.Embed(context.Background(), text)
		if err != nil {
			t.Fatalf("Embed(%d): %v", i, err)
		}
		if sim := cosine(batch[i], single); sim < 0.9999 {
			t.Errorf("text %d: cosine(batch, single) = %.6f, want >= 0.9999", i, sim)
		}
	}
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
