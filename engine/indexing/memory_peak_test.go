package indexing

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"spotfile/engine/embedder/embeddertest"
)

const (
	// peakRSSBudget is the ceiling for indexing one large document. Before the
	// fix, the app peaked at 12.7 GB (vmmap physical footprint) on a 60-page PDF.
	peakRSSBudget = 2 << 30 // 2 GiB

	memChildEnv     = "SPOTFILE_MEMTEST_CHILD"
	memDocPages     = 60
	memWordsPerPage = 500
)

var memVocabulary = strings.Fields(`
	index search vector semantic document page chapter model memory tensor
	runtime session batch worker channel pipeline query result ranking score
	history science biology chemistry physics economics language literature
	student course lecture exam assignment research method analysis theory
	system network protocol database storage cache latency throughput kernel
	river mountain city village market harbor forest weather season journey`)

// TestIndexPeakMemory indexes a ~60-page document through the real pipeline
// and asserts the process peak RSS stays within peakRSSBudget.
//
// ONNX Runtime allocates outside the Go heap, so runtime.MemStats cannot see
// it. The work runs in a child process instead, and the parent reads the
// child's kernel-reported max RSS — isolated from any other test in this run.
func TestIndexPeakMemory(t *testing.T) {
	if os.Getenv(memChildEnv) == "1" {
		runIndexForMemory(t)
		return
	}
	if testing.Short() {
		t.Skip("memory test indexes a large document; skipped in -short mode")
	}
	embeddertest.RealModel(t, 1) // skip early when assets are absent

	cmd := exec.Command(os.Args[0], "-test.run=^TestIndexPeakMemory$", "-test.v")
	cmd.Env = append(os.Environ(), memChildEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child indexing run failed: %v\n%s", err, out)
	}

	peak := maxRSSBytes(cmd.ProcessState)
	t.Logf("peak RSS indexing %d pages: %.2f GiB (budget %.2f GiB)",
		memDocPages, gib(peak), gib(peakRSSBudget))
	if peak > peakRSSBudget {
		t.Fatalf("peak RSS %.2f GiB exceeds budget %.2f GiB", gib(peak), gib(peakRSSBudget))
	}
}

func runIndexForMemory(t *testing.T) {
	// Match the app default: Workers → runtime.NumCPU().
	model := embeddertest.RealModel(t, 0)

	path := filepath.Join(t.TempDir(), "large.txt")
	if err := os.WriteFile(path, []byte(syntheticDocument()), 0o600); err != nil {
		t.Fatalf("write document: %v", err)
	}

	n := 0
	for range EmbedFiles(context.Background(), model, []string{path}) {
		n++
	}
	if n == 0 {
		t.Fatal("indexing produced no chunks")
	}
	fmt.Printf("indexed %d chunks\n", n)
}

// syntheticDocument returns a deterministic document of memDocPages pages.
// Text files and PDF pages share the same chunk → embed path, which is where
// the memory goes; a text file keeps the fixture free of binary assets.
func syntheticDocument() string {
	var b strings.Builder
	for i := range memDocPages * memWordsPerPage {
		b.WriteString(memVocabulary[(i*7+i/13)%len(memVocabulary)])
		b.WriteByte(' ')
	}
	return b.String()
}

func maxRSSBytes(ps *os.ProcessState) int64 {
	ru, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0
	}
	// ru_maxrss is bytes on darwin and kilobytes on linux.
	if runtime.GOOS == "darwin" {
		return ru.Maxrss
	}
	return ru.Maxrss * 1024
}

func gib(b int64) float64 { return float64(b) / (1 << 30) }
