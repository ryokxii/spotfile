package engine

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

const (
	MaxSeqLen     = 512
	maxChunkWords = 400 // leaves headroom for [CLS]/[SEP] plus subword expansion
	chunkOverlap  = 50
	defaultBatchSize = 32 // embed 32 chunks per ONNX call
	maxDocBuffer = 256 // limit document buffer to prevent memory bloat with 100K+ files
)

// inputNames matches the bge-small-en-v1.5 ONNX export order.
var inputNames = []string{"input_ids", "attention_mask", "token_type_ids"}
var outputNames = []string{"last_hidden_state"}

type Config struct {
	LibraryPath string
	ModelPath   string
	VocabPath   string
	Workers     int // 0 → runtime.NumCPU()
}

type Engine struct {
	cfg       Config
	tok       *BertTokenizer
	session   *ort.DynamicAdvancedSession
	jobs      chan embedJob
	closeOnce sync.Once
	closed    chan struct{}
	wg        sync.WaitGroup
}

type embedJob struct {
	text   string
	result chan<- embedResult
}

type embedResult struct {
	vec []float32
	err error
}

func New(cfg Config) (*Engine, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	ort.SetSharedLibraryPath(cfg.LibraryPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("ort init: %w", err)
	}

	tok, err := LoadTokenizer(cfg.VocabPath, MaxSeqLen)
	if err != nil {
		return nil, fmt.Errorf("tokenizer: %w", err)
	}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("session options: %w", err)
	}
	if err := configureExecutionProviders(opts); err != nil {
		opts.Destroy()
		return nil, fmt.Errorf("execution providers: %w", err)
	}

	session, err := ort.NewDynamicAdvancedSession(cfg.ModelPath, inputNames, outputNames, opts)
	opts.Destroy()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	e := &Engine{
		cfg:     cfg,
		tok:     tok,
		session: session,
		jobs:    make(chan embedJob, cfg.Workers*4),
		closed:  make(chan struct{}),
	}
	for range cfg.Workers {
		e.wg.Add(1)
		go e.worker()
	}
	return e, nil
}

// worker drains the jobs channel and runs ONNX inference on each item.
// ONNX Runtime is thread-safe; all workers share the single session.
func (e *Engine) worker() {
	defer e.wg.Done()
	for job := range e.jobs {
		vec, err := e.infer(job.text)
		job.result <- embedResult{vec: vec, err: err}
	}
}

// Embed tokenizes text, runs inference via the worker pool, and returns a
// normalized embedding vector. Safe to call from multiple goroutines.
func (e *Engine) Embed(ctx context.Context, text string) ([]float32, error) {
	result := make(chan embedResult, 1)
	select {
	case e.jobs <- embedJob{text: text, result: result}:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.closed:
		return nil, fmt.Errorf("engine closed")
	}
	select {
	case r := <-result:
		return r.vec, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IndexFiles runs the three-stage pipeline over the given file paths:
//  1. Fan-out file readers (I/O parallel)
//  2. Single chunker goroutine (serializes chunk ordering)
//  3. Fan-out embedders (compute parallel via worker pool)
//
// The returned channel is closed when all files have been processed.
func (e *Engine) IndexFiles(ctx context.Context, paths []string) <-chan EmbeddedChunk {
	out := make(chan EmbeddedChunk, 64)

	go func() {
		defer close(out)

		// Stage 1: read files in parallel → docs channel
		type doc struct {
			path string
			text string
		}
		docBufSize := len(paths)
		if docBufSize > maxDocBuffer {
			docBufSize = maxDocBuffer
		}
		docs := make(chan doc, docBufSize)
		var readWg sync.WaitGroup
		for _, p := range paths {
			readWg.Add(1)
			go func(path string) {
				defer readWg.Done()
				data, err := os.ReadFile(path)
				if err != nil {
					log.Printf("index: skip %s: %v", path, err)
					return
				}
				select {
				case docs <- doc{path: path, text: string(data)}:
				case <-ctx.Done():
				}
			}(p)
		}
		go func() {
			readWg.Wait()
			close(docs)
		}()

		// Stage 2: chunk documents → chunks channel (single goroutine preserves ordering)
		type chunk struct {
			docPath  string
			chunkIdx int
			text     string
		}
		chunks := make(chan chunk, defaultBatchSize*4)
		go func() {
			defer close(chunks)
			for d := range docs {
				for i, c := range Chunk(d.text, maxChunkWords, chunkOverlap) {
					select {
					case chunks <- chunk{docPath: d.path, chunkIdx: i, text: c}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()

		// Stage 3: embed chunks in batches, bounded to Workers goroutines
		var embedWg sync.WaitGroup
		for range e.cfg.Workers {
			embedWg.Add(1)
			go func() {
				defer embedWg.Done()
				batch := make([]chunk, 0, defaultBatchSize)
				for c := range chunks {
					batch = append(batch, c)
					if len(batch) < defaultBatchSize {
						continue
					}
					// Process batch
					texts := make([]string, len(batch))
					for i, ch := range batch {
						texts[i] = ch.text
					}
					vecs, err := e.BatchEmbed(texts)
					if err != nil {
						log.Printf("index: batch embed failed: %v", err)
						batch = batch[:0]
						continue
					}
					for i, ch := range batch {
						select {
						case out <- EmbeddedChunk{DocPath: ch.docPath, ChunkIdx: ch.chunkIdx, Text: ch.text, Embedding: vecs[i]}:
						case <-ctx.Done():
							return
						}
					}
					batch = batch[:0]
				}
				// Process remaining chunks in batch
				if len(batch) > 0 {
					texts := make([]string, len(batch))
					for i, ch := range batch {
						texts[i] = ch.text
					}
					vecs, err := e.BatchEmbed(texts)
					if err != nil {
						log.Printf("index: batch embed failed: %v", err)
						return
					}
					for i, ch := range batch {
						select {
						case out <- EmbeddedChunk{DocPath: ch.docPath, ChunkIdx: ch.chunkIdx, Text: ch.text, Embedding: vecs[i]}:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
		}
		embedWg.Wait()
	}()

	return out
}

// infer runs a single forward pass through the model for the given text.
func (e *Engine) infer(text string) ([]float32, error) {
	enc := e.tok.Encode(text)
	shape := ort.NewShape(1, int64(MaxSeqLen))

	inputIDs, err := ort.NewTensor(shape, enc.InputIDs)
	if err != nil {
		return nil, err
	}
	defer inputIDs.Destroy()

	attnMask, err := ort.NewTensor(shape, enc.AttentionMask)
	if err != nil {
		return nil, err
	}
	defer attnMask.Destroy()

	typeIDs, err := ort.NewTensor(shape, enc.TokenTypeIDs)
	if err != nil {
		return nil, err
	}
	defer typeIDs.Destroy()

	inputs := []ort.Value{inputIDs, attnMask, typeIDs}
	// Nil slot: Run auto-allocates the output tensor.
	outputs := make([]ort.Value, 1)
	if err := e.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("inference: %w", err)
	}
	defer outputs[0].Destroy()

	// last_hidden_state: [1, seq_len, hidden_size]
	hidden, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output type %T", outputs[0])
	}
	data := hidden.GetData()
	hiddenSize := len(data) / MaxSeqLen

	return l2Normalize(meanPool(data, enc.AttentionMask, MaxSeqLen, hiddenSize)), nil
}

// meanPool averages the token embeddings, excluding padding tokens.
func meanPool(data []float32, mask []int64, seqLen, hiddenSize int) []float32 {
	out := make([]float32, hiddenSize)
	var count float32
	for i := range seqLen {
		if mask[i] == 0 {
			continue
		}
		count++
		row := data[i*hiddenSize : (i+1)*hiddenSize]
		for j, v := range row {
			out[j] += v
		}
	}
	if count > 0 {
		for j := range out {
			out[j] /= count
		}
	}
	return out
}

func l2Normalize(v []float32) []float32 {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if norm == 0 {
		return v
	}
	norm = math.Sqrt(norm)
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out
}

func (e *Engine) Close() {
	e.closeOnce.Do(func() {
		close(e.closed)
		close(e.jobs)
		e.wg.Wait()
		e.session.Destroy()
		ort.DestroyEnvironment()
	})
}
