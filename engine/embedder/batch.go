package embedder

import (
	"fmt"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

type batchResult struct {
	vec []float32
	err error
}

// BatchEmbed tokenizes and embeds multiple texts in a single ONNX inference call.
// This is ~3-5x faster than embedding texts individually.
func (e *Model) BatchEmbed(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Tokenize all texts
	encodings := make([]Encoding, len(texts))
	for i, text := range texts {
		encodings[i] = e.tok.Encode(text)
	}

	// Pad only to the longest sequence in this batch, not MaxSeqLen. Attention
	// memory grows with seqLen², so short chunks padded to 512 cost as much as
	// full ones.
	batchSize := int64(len(texts))
	seqLen := longestSequence(encodings)
	shape := ort.NewShape(batchSize, int64(seqLen))

	// Flatten all input_ids, attention_masks, and token_type_ids into single arrays
	flatInputIDs := make([]int64, batchSize*int64(seqLen))
	flatAttentionMask := make([]int64, batchSize*int64(seqLen))
	flatTokenTypeIDs := make([]int64, batchSize*int64(seqLen))

	for i, enc := range encodings {
		offset := i * seqLen
		copy(flatInputIDs[offset:offset+seqLen], enc.InputIDs[:seqLen])
		copy(flatAttentionMask[offset:offset+seqLen], enc.AttentionMask[:seqLen])
		copy(flatTokenTypeIDs[offset:offset+seqLen], enc.TokenTypeIDs[:seqLen])
	}

	// Create ONNX tensors
	inputIDs, err := ort.NewTensor(shape, flatInputIDs)
	if err != nil {
		return nil, fmt.Errorf("create input_ids tensor: %w", err)
	}
	defer inputIDs.Destroy()

	attnMask, err := ort.NewTensor(shape, flatAttentionMask)
	if err != nil {
		return nil, fmt.Errorf("create attention_mask tensor: %w", err)
	}
	defer attnMask.Destroy()

	typeIDs, err := ort.NewTensor(shape, flatTokenTypeIDs)
	if err != nil {
		return nil, fmt.Errorf("create token_type_ids tensor: %w", err)
	}
	defer typeIDs.Destroy()

	// Run inference
	inputs := []ort.Value{inputIDs, attnMask, typeIDs}
	outputs := make([]ort.Value, 1)
	if err := e.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("batch inference: %w", err)
	}
	defer outputs[0].Destroy()

	// Extract output: [batch_size, seq_len, hidden_size]
	hidden, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output type %T", outputs[0])
	}
	data := hidden.GetData()
	hiddenSize := len(data) / int(batchSize) / seqLen

	// Extract and normalize each embedding
	results := make([][]float32, len(texts))
	for i := 0; i < len(texts); i++ {
		// Extract the sequence [seqLen, hidden_size] for this batch item
		offset := i * seqLen * hiddenSize
		seqData := data[offset : offset+seqLen*hiddenSize]

		// Mean pool using this text's attention mask
		pooled := meanPool(seqData, encodings[i].AttentionMask, seqLen, hiddenSize)
		results[i] = l2Normalize(pooled)
	}

	return results, nil
}

// longestSequence returns the longest unpadded length in encodings. The
// tokenizer pads at the tail, so the unpadded length is the mask's 1-count.
func longestSequence(encodings []Encoding) int {
	longest := 0
	for _, enc := range encodings {
		n := 0
		for _, m := range enc.AttentionMask {
			n += int(m)
		}
		longest = max(longest, n)
	}
	return longest
}

// batchWorker consumes batches from the jobs channel and processes them.
type batchJob struct {
	texts   []string
	results chan [][]float32
	err     chan error
}

// BatchEmbedBatches is a convenience method that wraps BatchEmbed for concurrent access.
// Safe to call from multiple goroutines via a worker pool.
func (e *Model) BatchEmbedBatches(jobs <-chan batchJob, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		results, err := e.BatchEmbed(job.texts)
		if err != nil {
			job.err <- err
		} else {
			job.results <- results
		}
	}
}
