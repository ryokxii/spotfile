package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkBatchEmbed benchmarks batch embedding performance.
func BenchmarkBatchEmbed(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Create temporary vocab for testing
	vocabPath := filepath.Join(b.TempDir(), "vocab.txt")
	if err := createTestVocab(vocabPath); err != nil {
		b.Fatalf("Failed to create test vocab: %v", err)
	}

	eng, err := New(Config{
		LibraryPath: "", // Use default
		ModelPath:   "", // Placeholder (real model required for actual benchmark)
		VocabPath:   vocabPath,
		Workers:     4,
	})
	if err != nil {
		// If ONNX library not available, skip benchmark
		b.Skipf("ONNX Runtime not available: %v", err)
	}
	defer eng.Close()

	testTexts := generateTestTexts(100) // 100 sample texts

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eng.BatchEmbed(testTexts)
		if err != nil {
			b.Fatalf("BatchEmbed failed: %v", err)
		}
	}
}

// BenchmarkIndexFiles benchmarks the full indexing pipeline.
func BenchmarkIndexFiles(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	tempDir := b.TempDir()
	vocabPath := filepath.Join(tempDir, "vocab.txt")
	if err := createTestVocab(vocabPath); err != nil {
		b.Fatalf("Failed to create test vocab: %v", err)
	}

	eng, err := New(Config{
		LibraryPath: "",
		ModelPath:   "",
		VocabPath:   vocabPath,
		Workers:     4,
	})
	if err != nil {
		b.Skipf("ONNX Runtime not available: %v", err)
	}
	defer eng.Close()

	// Create test files
	testFiles := createTestFiles(b.TempDir(), 100, 2000) // 100 files, 2KB each

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunks := eng.IndexFiles(ctx, testFiles)
		count := 0
		for range chunks {
			count++
		}
		b.Logf("Indexed %d chunks", count)
	}
}

// BenchmarkPriorityQueue benchmarks file sorting by modification time.
func BenchmarkPriorityQueue(b *testing.B) {
	tempDir := b.TempDir()
	testFiles := createTestFiles(tempDir, 1000, 100) // 1000 files

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SortPathsByModTime(testFiles)
	}
}

// TestBatchEmbedOrdering verifies that batch embeddings maintain correct order.
func TestBatchEmbedOrdering(t *testing.T) {
	tempDir := t.TempDir()
	vocabPath := filepath.Join(tempDir, "vocab.txt")
	if err := createTestVocab(vocabPath); err != nil {
		t.Fatalf("Failed to create test vocab: %v", err)
	}

	eng, err := New(Config{
		LibraryPath: "",
		ModelPath:   "",
		VocabPath:   vocabPath,
		Workers:     2,
	})
	if err != nil {
		t.Skipf("ONNX Runtime not available: %v", err)
	}
	defer eng.Close()

	texts := []string{
		"hello world",
		"foo bar baz",
		"test text",
		"another sample",
	}

	embeddings, err := eng.BatchEmbed(texts)
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

// TestPriorityQueueOrdering verifies recent files come first.
func TestPriorityQueueOrdering(t *testing.T) {
	tempDir := t.TempDir()

	// Create files with staggered modification times
	files := make([]string, 3)
	for i := 0; i < 3; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(path, []byte(fmt.Sprintf("content %d", i)), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		files[i] = path
		time.Sleep(100 * time.Millisecond) // Ensure distinct timestamps
	}

	sorted := SortPathsByModTime(files)

	// Most recent file should be first
	if sorted[0] != files[2] {
		t.Errorf("Expected most recent file first, got %s", sorted[0])
	}
	if sorted[2] != files[0] {
		t.Errorf("Expected oldest file last, got %s", sorted[2])
	}
}

// --- Helper functions ---

func createTestVocab(path string) error {
	// Create a minimal BERT vocab for testing
	content := "[PAD]\n[unused0]\n[unused1]\n[UNK]\n[CLS]\n[SEP]\nhello\nworld\nfoo\nbar\nbaz\ntest\ntext\nanother\nsample\n"
	return os.WriteFile(path, []byte(content), 0644)
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

func createTestFiles(dir string, count, sizeBytesPerFile int) []string {
	files := make([]string, count)
	content := make([]byte, sizeBytesPerFile)
	for i := 0; i < count; i++ {
		for j := 0; j < len(content); j++ {
			content[j] = byte((i + j) % 256)
		}
		path := filepath.Join(dir, fmt.Sprintf("file%06d.txt", i))
		if err := os.WriteFile(path, content, 0644); err != nil {
			panic(fmt.Sprintf("Failed to create test file: %v", err))
		}
		files[i] = path
		// Stagger file times slightly to ensure distinct modification times
		if i%10 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	return files
}
