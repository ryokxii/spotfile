package engine

import (
	"context"
	"log"
	"sync"
)

// Priority orders indexing work. Higher values are drained first, so live edits
// jump ahead of a recent-files scan, which in turn jumps ahead of history.
type Priority int

const (
	PriorityLow    Priority = iota // historical backfill
	PriorityHigh                   // recently modified files on startup
	PriorityUrgent                 // live edits from the watcher
	numPriorities
)

// maxBatchDrain caps how many paths one worker iteration pulls from a single
// priority level, so a freshly-enqueued higher-priority item is picked up at the
// next iteration instead of waiting behind an entire history scan.
const maxBatchDrain = 256

// PathPriority pairs a file path with the priority it should be indexed at.
type PathPriority struct {
	Path     string
	Priority Priority
}

// IndexHooks are optional callbacks fired on the indexer's worker goroutine.
// OnStart fires once when a job's first path begins processing; OnChunk fires
// per stored chunk with the job's cumulative chunk count; OnDone fires once when
// every path in the job has been processed.
type IndexHooks struct {
	OnStart func()
	OnChunk func(chunk EmbeddedChunk, jobTotalChunks int)
	OnDone  func(jobTotalChunks int)
}

// indexJob tracks the completion of one Enqueue call across (possibly many)
// worker batches. All fields are touched only by the single worker goroutine
// except remaining, which is initialized before enqueue and published via the
// indexer mutex.
type indexJob struct {
	mu        sync.Mutex
	remaining int
	total     int
	started   bool
	onStart   func()
	onChunk   func(EmbeddedChunk, int)
	onDone    func(int)
}

type task struct {
	path string
	job  *indexJob
}

// Indexer owns a prioritized work queue drained by a single background worker.
// Batching within a priority tier is preserved by handing each drained batch to
// Engine.IndexFiles, which parallelizes reads and embedding internally.
type Indexer struct {
	ctx   context.Context
	eng   *Engine
	store *VectorStore

	mu     sync.Mutex
	cond   *sync.Cond
	queues [numPriorities][]task
	closed bool
	wg     sync.WaitGroup
}

// NewIndexer starts the background worker. Stop it with Stop.
func NewIndexer(ctx context.Context, eng *Engine, store *VectorStore) *Indexer {
	ix := &Indexer{ctx: ctx, eng: eng, store: store}
	ix.cond = sync.NewCond(&ix.mu)
	ix.wg.Add(1)
	go ix.run()
	return ix
}

// Enqueue schedules items as a single job. De-duplication (replacing a path's
// prior chunks) and persistence happen automatically when the job's paths are
// processed. A nil or empty items slice is a no-op.
func (ix *Indexer) Enqueue(items []PathPriority, hooks IndexHooks) {
	if len(items) == 0 {
		return
	}
	job := &indexJob{
		remaining: len(items),
		onStart:   hooks.OnStart,
		onChunk:   hooks.OnChunk,
		onDone:    hooks.OnDone,
	}
	ix.mu.Lock()
	for _, it := range items {
		p := it.Priority
		if p < 0 || p >= numPriorities {
			p = PriorityLow
		}
		ix.queues[p] = append(ix.queues[p], task{path: it.Path, job: job})
	}
	ix.cond.Signal()
	ix.mu.Unlock()
}

// Stop signals the worker to exit and waits for the in-flight batch to finish.
func (ix *Indexer) Stop() {
	ix.mu.Lock()
	ix.closed = true
	ix.cond.Broadcast()
	ix.mu.Unlock()
	ix.wg.Wait()
}

func (ix *Indexer) run() {
	defer ix.wg.Done()
	for {
		batch, ok := ix.nextBatch()
		if !ok {
			return
		}
		ix.process(batch)
	}
}

// nextBatch blocks until work is available, then returns up to maxBatchDrain
// tasks from the highest non-empty priority level. Returns ok=false once closed.
func (ix *Indexer) nextBatch() ([]task, bool) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	for {
		if ix.closed {
			return nil, false
		}
		for p := numPriorities - 1; p >= 0; p-- {
			q := ix.queues[p]
			if len(q) == 0 {
				continue
			}
			n := len(q)
			if n > maxBatchDrain {
				n = maxBatchDrain
			}
			batch := make([]task, n)
			copy(batch, q[:n])
			ix.queues[p] = q[n:]
			return batch, true
		}
		ix.cond.Wait()
	}
}

func (ix *Indexer) process(batch []task) {
	paths := make([]string, len(batch))
	jobOf := make(map[string]*indexJob, len(batch))
	for i, t := range batch {
		paths[i] = t.path
		jobOf[t.path] = t.job
		ix.fireStart(t.job)
	}

	// Replace prior chunks for these paths, then re-index.
	ix.store.RemoveDocs(paths)
	chunks := ix.eng.IndexFiles(ix.ctx, paths)
	for chunk := range chunks {
		ix.store.Add(chunk)
		job := jobOf[chunk.DocPath]
		job.mu.Lock()
		job.total++
		total := job.total
		onChunk := job.onChunk
		job.mu.Unlock()
		if onChunk != nil {
			onChunk(chunk, total)
		}
	}

	// One task processed per path (regardless of chunks produced). When a job's
	// last path completes, persist once and fire OnDone.
	for _, t := range batch {
		t.job.mu.Lock()
		t.job.remaining--
		done := t.job.remaining == 0
		total := t.job.total
		onDone := t.job.onDone
		t.job.mu.Unlock()
		if done {
			if err := ix.store.Persist(); err != nil {
				log.Printf("indexer: persist after job failed: %v", err)
			}
			if onDone != nil {
				onDone(total)
			}
		}
	}
}

// fireStart runs a job's OnStart exactly once, before its first path is indexed.
func (ix *Indexer) fireStart(job *indexJob) {
	job.mu.Lock()
	if job.started {
		job.mu.Unlock()
		return
	}
	job.started = true
	onStart := job.onStart
	job.mu.Unlock()
	if onStart != nil {
		onStart()
	}
}
