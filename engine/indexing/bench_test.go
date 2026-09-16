package indexing

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"spotfile/engine/embedder/embeddertest"
)

const (
	benchFiles        = 100
	benchWordsPerFile = 450 // ~2 chunks per file at maxChunkWords=250 with overlap
	benchSeed         = 42
)

// BenchmarkEmbedFiles measures the full read → chunk → embed pipeline on
// realistic documents. The llama-server is started before timing, so results
// reflect indexing throughput rather than model load time.
//
//   - prose:    plain English paragraphs, typical of notes and PDFs
//   - markdown: headings, lists, inline code and code blocks, which tokenize
//     densely and exercise token-based chunk splitting
func BenchmarkEmbedFiles(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}
	m := embeddertest.RealModel(b)
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		b.Fatalf("start embedding server: %v", err)
	}

	for _, kind := range []struct {
		name string
		gen  func(r *rand.Rand, words int) string
	}{
		{"prose", proseDocument},
		{"markdown", markdownDocument},
	} {
		b.Run(kind.name, func(b *testing.B) {
			paths, totalBytes := writeCorpus(b, kind.gen, benchFiles, benchWordsPerFile)
			b.SetBytes(totalBytes)

			chunks := 0
			for b.Loop() {
				chunks = 0
				for range EmbedFiles(ctx, m, paths) {
					chunks++
				}
			}
			if chunks == 0 {
				b.Fatal("indexing produced no chunks")
			}
			b.ReportMetric(float64(chunks), "chunks/op")
			b.ReportMetric(float64(chunks)*float64(b.N)/b.Elapsed().Seconds(), "chunks/s")
		})
	}
}

// BenchmarkPriorityQueue benchmarks file sorting by modification time.
func BenchmarkPriorityQueue(b *testing.B) {
	paths, _ := writeCorpus(b, proseDocument, 1000, 20)
	for b.Loop() {
		_ = SortPathsByModTime(paths)
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

// --- Benchmark corpus --------------------------------------------------------

// writeCorpus writes count deterministic documents and returns their paths and
// total size. Modification times are spaced one minute apart, newest last.
func writeCorpus(tb testing.TB, gen func(*rand.Rand, int) string, count, words int) ([]string, int64) {
	tb.Helper()
	dir := tb.TempDir()
	r := rand.New(rand.NewPCG(benchSeed, uint64(count)))
	base := time.Now().Add(-time.Duration(count) * time.Minute)

	paths := make([]string, count)
	var total int64
	for i := range paths {
		text := gen(r, words)
		path := filepath.Join(dir, fmt.Sprintf("doc%04d.md", i))
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			tb.Fatalf("write corpus: %v", err)
		}
		mtime := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			tb.Fatalf("set mtime: %v", err)
		}
		paths[i] = path
		total += int64(len(text))
	}
	return paths, total
}

var benchVocabulary = strings.Fields(`
	the a an of to in for on with as by from at into about over after before
	student course lecture exam assignment deadline syllabus grade chapter
	research method analysis theory result evidence sample survey experiment
	history economy policy government market trade population city region
	energy climate water system network storage memory process signal model
	report meeting project budget schedule team client proposal review draft
	important recent early final main common different similar several major
	describe explain compare measure improve reduce increase include require
	because although however therefore while during between within across`)

func sentence(r *rand.Rand, words int) string {
	w := make([]string, words)
	for i := range w {
		w[i] = benchVocabulary[r.IntN(len(benchVocabulary))]
	}
	w[0] = strings.ToUpper(w[0][:1]) + w[0][1:]
	if words > 8 && r.IntN(3) == 0 {
		w[words/2] += ","
	}
	return strings.Join(w, " ") + "."
}

// proseDocument returns paragraphs of 3–6 sentences of 8–20 words.
func proseDocument(r *rand.Rand, words int) string {
	var b strings.Builder
	written := 0
	for written < words {
		for range 3 + r.IntN(4) {
			n := 8 + r.IntN(13)
			b.WriteString(sentence(r, n))
			b.WriteByte(' ')
			written += n
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

var benchCode = []string{
	"func (s *Store) Search(query []float32, topK int) []Result {",
	"\tif len(query) == 0 || topK <= 0 { return nil }",
	"\tresults := make([]Result, 0, topK)",
	"const response = await fetch(`/api/v1/items?limit=${limit}`);",
	"for (let i = 0; i < items.length; i++) { total += items[i].price * 1.2; }",
	"SELECT id, name, created_at FROM users WHERE email LIKE '%@example.com';",
	"export PATH=\"$HOME/.local/bin:$PATH\" && make build -j8",
}

// markdownDocument returns sections with headings, bullet lists, inline code
// and fenced code blocks.
func markdownDocument(r *rand.Rand, words int) string {
	var b strings.Builder
	written := 0
	for section := 1; written < words; section++ {
		fmt.Fprintf(&b, "## %d. %s\n\n", section, strings.TrimSuffix(sentence(r, 3+r.IntN(3)), "."))
		n := 10 + r.IntN(10)
		fmt.Fprintf(&b, "%s Use `%s` with `--%s=%d`.\n\n", sentence(r, n),
			benchVocabulary[r.IntN(len(benchVocabulary))], benchVocabulary[r.IntN(len(benchVocabulary))], r.IntN(100))
		written += n + 3
		for range 2 + r.IntN(3) {
			m := 5 + r.IntN(8)
			fmt.Fprintf(&b, "- %s\n", sentence(r, m))
			written += m
		}
		b.WriteString("\n```\n")
		for range 3 + r.IntN(4) {
			line := benchCode[r.IntN(len(benchCode))]
			b.WriteString(line + "\n")
			written += len(strings.Fields(line))
		}
		b.WriteString("```\n\n")
	}
	return b.String()
}
