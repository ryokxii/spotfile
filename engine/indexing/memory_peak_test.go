//go:build !windows

package indexing

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"spotfile/engine/embedder/embeddertest"
)

const (
	// peakTreeRSSBudget caps total memory while indexing one large document.
	// Before the llama.cpp switch the app peaked at 12.7 GB (CoreML) and later
	// 1.6 GB (ONNX Runtime CPU); llama.cpp measured ~0.2 GB for the server.
	peakTreeRSSBudget = 512 << 20 // 512 MiB

	memChildEnv     = "SPOTFILE_MEMTEST_CHILD"
	memDocPages     = 60
	memWordsPerPage = 500
	memSampleEvery  = 50 * time.Millisecond
)

var memVocabulary = strings.Fields(`
	index search vector semantic document page chapter model memory tensor
	runtime session batch worker channel pipeline query result ranking score
	history science biology chemistry physics economics language literature
	student course lecture exam assignment research method analysis theory
	system network protocol database storage cache latency throughput kernel
	river mountain city village market harbor forest weather season journey`)

// TestIndexPeakMemory indexes a ~60-page document through the real pipeline
// and asserts the combined memory of the indexing process and its llama-server
// child stays within peakTreeRSSBudget.
//
// Embedding happens in a separate process, so the work runs in a child test
// process and the parent samples the resident memory of that child plus all of
// its descendants. RSS includes the memory-mapped model file.
func TestIndexPeakMemory(t *testing.T) {
	if os.Getenv(memChildEnv) == "1" {
		runIndexForMemory(t)
		return
	}
	if testing.Short() {
		t.Skip("memory test indexes a large document; skipped in -short mode")
	}
	embeddertest.RealModel(t) // skip early when llama.cpp or the model is absent

	cmd := exec.Command(os.Args[0], "-test.run=^TestIndexPeakMemory$", "-test.v")
	cmd.Env = append(os.Environ(), memChildEnv+"=1")
	var out strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var peak int64
	tick := time.NewTicker(memSampleEvery)
	defer tick.Stop()
	for waiting := true; waiting; {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("child indexing run failed: %v\n%s", err, out.String())
			}
			waiting = false
		case <-tick.C:
			if rss, err := treeRSS(cmd.Process.Pid); err == nil {
				peak = max(peak, rss)
			}
		}
	}

	t.Logf("peak RSS (indexer + llama-server) indexing %d pages: %.0f MiB (budget %.0f MiB)",
		memDocPages, mib(peak), mib(peakTreeRSSBudget))
	if peak == 0 {
		t.Fatal("no memory samples taken")
	}
	if peak > peakTreeRSSBudget {
		t.Fatalf("peak RSS %.0f MiB exceeds budget %.0f MiB", mib(peak), mib(peakTreeRSSBudget))
	}
}

func runIndexForMemory(t *testing.T) {
	model := embeddertest.RealModel(t)

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

// treeRSS sums the resident memory of pid and all of its descendants.
func treeRSS(root int) (int64, error) {
	out, err := exec.Command("ps", "-A", "-o", "pid=,ppid=,rss=").Output()
	if err != nil {
		return 0, err
	}
	children := map[int][]int{}
	rssKB := map[int]int64{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			continue
		}
		pid, _ := strconv.Atoi(f[0])
		ppid, _ := strconv.Atoi(f[1])
		rss, _ := strconv.ParseInt(f[2], 10, 64)
		children[ppid] = append(children[ppid], pid)
		rssKB[pid] = rss
	}
	var total int64
	for stack := []int{root}; len(stack) > 0; {
		pid := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		total += rssKB[pid]
		stack = append(stack, children[pid]...)
	}
	return total * 1024, nil
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

func mib(b int64) float64 { return float64(b) / (1 << 20) }
