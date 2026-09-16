package indexing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"spotfile/engine/embedder"
	"spotfile/engine/embedder/embeddertest"
)

// BenchmarkEmbedFiles benchmarks the full indexing pipeline.
func BenchmarkEmbedFiles(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	m, err := embedder.New(embedder.Config{
		LibraryPath: "",
		ModelPath:   "",
		VocabPath:   embeddertest.WriteVocab(b),
		Workers:     4,
	})
	if err != nil {
		b.Skipf("ONNX Runtime not available: %v", err)
	}
	defer m.Close()

	// Create test files
	testFiles := createTestFiles(b.TempDir(), 100, 2000) // 100 files, 2KB each

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunks := EmbedFiles(ctx, m, testFiles)
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
