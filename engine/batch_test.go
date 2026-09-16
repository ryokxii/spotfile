package engine

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func encodingWithLen(n int) Encoding {
	mask := make([]int64, MaxSeqLen)
	for i := range n {
		mask[i] = 1
	}
	return Encoding{
		InputIDs:      make([]int64, MaxSeqLen),
		AttentionMask: mask,
		TokenTypeIDs:  make([]int64, MaxSeqLen),
	}
}

func TestLongestSequence(t *testing.T) {
	tests := []struct {
		name string
		lens []int
		want int
	}{
		{"single short", []int{5}, 5},
		{"mixed picks longest", []int{7, 120, 33}, 120},
		{"full length", []int{MaxSeqLen, 10}, MaxSeqLen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encs := make([]Encoding, len(tt.lens))
			for i, n := range tt.lens {
				encs[i] = encodingWithLen(n)
			}
			if got := longestSequence(encs); got != tt.want {
				t.Fatalf("longestSequence = %d, want %d", got, tt.want)
			}
		})
	}
}

// realAssetEngine builds an Engine on the installed ~/.spotfile assets, or
// skips the test when they are absent.
func realAssetEngine(t *testing.T, workers int) *Engine {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	dir := filepath.Join(home, ".spotfile")
	modelPath := filepath.Join(dir, "model.onnx")
	vocabPath := filepath.Join(dir, "vocab.txt")
	for _, p := range []string{modelPath, vocabPath} {
		if _, statErr := os.Stat(p); statErr != nil {
			t.Skipf("asset missing: %s", p)
		}
	}
	eng, err := New(Config{
		LibraryPath: testLibraryPath(t),
		ModelPath:   modelPath,
		VocabPath:   vocabPath,
		Workers:     workers,
	})
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}
	t.Cleanup(eng.Close)
	return eng
}

// TestBatchEmbedMatchesFullPadding guards the dynamic-padding optimisation:
// trimming a batch to its longest sequence must not change the embeddings
// compared with the single-text path, which always pads to MaxSeqLen.
func TestBatchEmbedMatchesFullPadding(t *testing.T) {
	eng := realAssetEngine(t, 1)

	texts := []string{
		"the quick brown fox",
		strings.Repeat("semantic search over local documents ", 40),
		"a",
	}
	batch, err := eng.BatchEmbed(texts)
	if err != nil {
		t.Fatalf("BatchEmbed: %v", err)
	}
	for i, text := range texts {
		single, err := eng.Embed(context.Background(), text)
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
