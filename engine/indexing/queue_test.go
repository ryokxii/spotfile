package indexing

import (
	"sync"
	"testing"
)

// newTestIndexer builds an Indexer without starting the worker goroutine, so the
// queue-draining logic can be exercised without a real ONNX engine.
func newTestIndexer() *Indexer {
	ix := &Indexer{}
	ix.cond = sync.NewCond(&ix.mu)
	return ix
}

func TestIndexerDrainsHighestPriorityFirst(t *testing.T) {
	ix := newTestIndexer()
	ix.Enqueue([]PathPriority{{Path: "old.md", Priority: PriorityLow}}, Hooks{})
	ix.Enqueue([]PathPriority{{Path: "recent.md", Priority: PriorityHigh}}, Hooks{})
	ix.Enqueue([]PathPriority{{Path: "edited.md", Priority: PriorityUrgent}}, Hooks{})

	want := []string{"edited.md", "recent.md", "old.md"}
	for _, w := range want {
		batch, ok := ix.nextBatch()
		if !ok {
			t.Fatalf("nextBatch returned closed while expecting %q", w)
		}
		if len(batch) != 1 || batch[0].path != w {
			t.Fatalf("drained %#v, want %q first", batch, w)
		}
	}
}

func TestIndexerBatchDrainIsCapped(t *testing.T) {
	ix := newTestIndexer()
	items := make([]PathPriority, maxBatchDrain+10)
	for i := range items {
		items[i] = PathPriority{Path: "f", Priority: PriorityLow}
	}
	ix.Enqueue(items, Hooks{})

	first, ok := ix.nextBatch()
	if !ok || len(first) != maxBatchDrain {
		t.Fatalf("first batch len = %d, want %d", len(first), maxBatchDrain)
	}
	second, ok := ix.nextBatch()
	if !ok || len(second) != 10 {
		t.Fatalf("second batch len = %d, want 10", len(second))
	}
}

func TestIndexerEnqueueEmptyIsNoop(t *testing.T) {
	ix := newTestIndexer()
	ix.Enqueue(nil, Hooks{})
	ix.Enqueue([]PathPriority{}, Hooks{})

	ix.mu.Lock()
	defer ix.mu.Unlock()
	for p := range ix.queues {
		if len(ix.queues[p]) != 0 {
			t.Fatalf("queue %d not empty after empty enqueue", p)
		}
	}
}

func TestIndexerStopUnblocksWorker(t *testing.T) {
	ix := newTestIndexer()
	done := make(chan struct{})
	go func() {
		_, ok := ix.nextBatch() // blocks on empty queue
		if ok {
			t.Errorf("nextBatch returned ok after close")
		}
		close(done)
	}()

	ix.mu.Lock()
	ix.closed = true
	ix.cond.Broadcast()
	ix.mu.Unlock()

	<-done
}
