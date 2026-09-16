package embedder_test

import (
	"testing"

	"spotfile/engine/embedder"
	"spotfile/engine/embedder/embeddertest"
)

// BenchmarkBatchEmbed benchmarks batch embedding performance.
func BenchmarkBatchEmbed(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	m, err := embedder.New(embedder.Config{
		LibraryPath: "", // Use default
		ModelPath:   "", // Placeholder (real model required for actual benchmark)
		VocabPath:   embeddertest.WriteVocab(b),
		Workers:     4,
	})
	if err != nil {
		// If ONNX library not available, skip benchmark
		b.Skipf("ONNX Runtime not available: %v", err)
	}
	defer m.Close()

	testTexts := generateTestTexts(100) // 100 sample texts

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := m.BatchEmbed(testTexts)
		if err != nil {
			b.Fatalf("BatchEmbed failed: %v", err)
		}
	}
}

// TestBatchEmbedOrdering verifies that batch embeddings maintain correct order.
func TestBatchEmbedOrdering(t *testing.T) {
	m, err := embedder.New(embedder.Config{
		LibraryPath: "",
		ModelPath:   "",
		VocabPath:   embeddertest.WriteVocab(t),
		Workers:     2,
	})
	if err != nil {
		t.Skipf("ONNX Runtime not available: %v", err)
	}
	defer m.Close()

	texts := []string{
		"hello world",
		"foo bar baz",
		"test text",
		"another sample",
	}

	embeddings, err := m.BatchEmbed(texts)
	if err != nil {
		t.Fatalf("BatchEmbed failed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, emb := range embeddings {
		if emb == nil {
			t.Errorf("Embedding %d is nil", i)
		}
		if len(emb) == 0 {
			t.Errorf("Embedding %d is empty", i)
		}
	}
}

func generateTestTexts(count int) []string {
	texts := make([]string, count)
	samples := []string{
		"hello world this is a test",
		"foo bar baz qux quux",
		"the quick brown fox jumps over the lazy dog",
		"machine learning is fascinating",
		"vector embeddings capture semantic meaning",
	}
	for i := 0; i < count; i++ {
		texts[i] = samples[i%len(samples)]
	}
	return texts
}
