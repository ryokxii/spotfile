// Package embedder turns text into normalized embedding vectors by running the
// bge-small-en-v1.5 model through ONNX Runtime.
package embedder

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// MaxSeqLen is the model's maximum input length in tokens.
const MaxSeqLen = 512

// inputNames matches the bge-small-en-v1.5 ONNX export order.
var inputNames = []string{"input_ids", "attention_mask", "token_type_ids"}
var outputNames = []string{"last_hidden_state"}

type Config struct {
	LibraryPath string
	ModelPath   string
	VocabPath   string
	Workers     int // 0 → runtime.NumCPU()
}

// Model is a loaded embedding model with a worker pool for single-text requests.
type Model struct {
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

func New(cfg Config) (*Model, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	// The ORT environment is a process-global singleton. Initialize it once and
	// reuse it across Model instances — re-initializing returns an error, and a
	// previous partial init (env up, but a later New step failed) must not poison
	// subsequent New calls. SetSharedLibraryPath is only honoured before the
	// first init, so skip it once the env is already up.
	if !ort.IsInitialized() {
		ort.SetSharedLibraryPath(cfg.LibraryPath)
		if err := ort.InitializeEnvironment(); err != nil {
			return nil, fmt.Errorf("ort init: %w", err)
		}
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

	e := &Model{
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

// Workers returns the configured parallelism (NumCPU when unset).
func (e *Model) Workers() int {
	return e.cfg.Workers
}

// worker drains the jobs channel and runs ONNX inference on each item.
// ONNX Runtime is thread-safe; all workers share the single session.
func (e *Model) worker() {
	defer e.wg.Done()
	for job := range e.jobs {
		vec, err := e.infer(job.text)
		job.result <- embedResult{vec: vec, err: err}
	}
}

// Embed tokenizes text, runs inference via the worker pool, and returns a
// normalized embedding vector. Safe to call from multiple goroutines.
func (e *Model) Embed(ctx context.Context, text string) ([]float32, error) {
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

// infer runs a single forward pass through the model for the given text.
func (e *Model) infer(text string) ([]float32, error) {
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

// Close stops the worker pool and frees the ONNX session. It deliberately does
// NOT destroy the global ORT environment — that singleton is shared across
// Model instances and outlives any single Model. Tear it down once at process
// exit via ShutdownRuntime.
func (e *Model) Close() {
	e.closeOnce.Do(func() {
		close(e.closed)
		close(e.jobs)
		e.wg.Wait()
		e.session.Destroy()
	})
}

// ShutdownRuntime destroys the process-global ORT environment if it was
// initialized. Call once during application shutdown, after all engines are
// closed. Safe to call when the environment was never initialized.
func ShutdownRuntime() {
	if ort.IsInitialized() {
		_ = ort.DestroyEnvironment()
	}
}
