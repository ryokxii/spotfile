package indexing

import (
	"context"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"

	"spotfile/engine/embedder"
	"spotfile/engine/extract"
	"spotfile/engine/vectorstore"
)

const (
	// maxChunkWords keeps most prose chunks under the model's token window;
	// denser text (code, Markdown) is split further by token count.
	maxChunkWords    = 250
	chunkOverlap     = 40
	defaultBatchSize = 16 // chunks per embedding request
	// maxConcurrentBatches matches the embedding server's parallel slots;
	// additional concurrent requests would only queue.
	maxConcurrentBatches = embedder.ParallelRequests
	maxDocBuffer         = 256 // limit document buffer to prevent memory bloat with 100K+ files
)

// CountChunks estimates how many chunks EmbedFiles will create for paths, for
// an indexing percentage before inference begins. It counts word windows only;
// EmbedFiles may split a dense window further to fit the model's token limit.
func CountChunks(ctx context.Context, paths []string) (int, error) {
	count := 0
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		if strings.HasSuffix(strings.ToLower(path), ".pdf") {
			pages, err := extract.ReadPDFPages(path)
			if err != nil {
				log.Printf("index: skip pdf while counting %s: %v", path, err)
				continue
			}
			for _, page := range pages {
				count += len(extract.Chunk(page.Text, maxChunkWords, chunkOverlap))
			}
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("index: skip while counting %s: %v", path, err)
			continue
		}
		count += len(extract.Chunk(string(data), maxChunkWords, chunkOverlap))
	}
	return count, nil
}

// EmbedFiles runs the three-stage pipeline over the given file paths:
//  1. Fan-out file readers (I/O parallel)
//  2. Single chunker goroutine (serializes chunk ordering)
//  3. Fan-out embedders (compute parallel via worker pool)
//
// The returned channel is closed when all files have been processed.
func EmbedFiles(ctx context.Context, model *embedder.Model, paths []string) <-chan vectorstore.EmbeddedChunk {
	countTokens := func(text string) (int, error) { return model.CountTokens(ctx, text) }

	out := make(chan vectorstore.EmbeddedChunk, 64)

	go func() {
		defer close(out)

		// Stage 1: read files in parallel → pages channel.
		// PDFs are extracted page-by-page; other files are treated as one page (pageNum=0).
		type page struct {
			path    string
			pageNum int
			text    string
		}
		docBufSize := len(paths)
		if docBufSize > maxDocBuffer {
			docBufSize = maxDocBuffer
		}
		pages := make(chan page, docBufSize)

		// Feed paths through a channel so a bounded pool of readers can drain
		// them, rather than spawning one goroutine per file (which explodes on
		// 100K+ collections). A closer goroutine keeps the producer non-blocking.
		pathsCh := make(chan string, docBufSize)
		go func() {
			defer close(pathsCh)
			for _, p := range paths {
				select {
				case pathsCh <- p:
				case <-ctx.Done():
					return
				}
			}
		}()

		readOne := func(path string) {
			if strings.HasSuffix(strings.ToLower(path), ".pdf") {
				pdfPages, err := extract.ReadPDFPages(path)
				if err != nil {
					log.Printf("index: skip pdf %s: %v", path, err)
					return
				}
				for _, pp := range pdfPages {
					select {
					case pages <- page{path: path, pageNum: pp.PageNum, text: pp.Text}:
					case <-ctx.Done():
						return
					}
				}
				return
			}
			data, err := os.ReadFile(path)
			if err != nil {
				log.Printf("index: skip %s: %v", path, err)
				return
			}
			select {
			case pages <- page{path: path, pageNum: 0, text: string(data)}:
			case <-ctx.Done():
			}
		}

		// Bounded reader pool — I/O parallelism capped at the CPU count.
		var readWg sync.WaitGroup
		for range runtime.NumCPU() {
			readWg.Add(1)
			go func() {
				defer readWg.Done()
				for path := range pathsCh {
					readOne(path)
				}
			}()
		}
		go func() {
			readWg.Wait()
			close(pages)
		}()

		// Stage 2: chunk pages → chunks channel (single goroutine preserves ordering).
		// Word windows that exceed the model's token limit are split further.
		type chunk struct {
			docPath  string
			chunkIdx int
			pageNum  int
			text     string
		}
		chunks := make(chan chunk, defaultBatchSize*4)
		go func() {
			defer close(chunks)
			for pg := range pages {
				idx := 0
				for _, window := range extract.Chunk(pg.text, maxChunkWords, chunkOverlap) {
					pieces, err := fitToTokens(window, embedder.MaxTokens, countTokens)
					if err != nil {
						log.Printf("index: skip chunk in %s: %v", pg.path, err)
						continue
					}
					for _, text := range pieces {
						select {
						case chunks <- chunk{docPath: pg.path, chunkIdx: idx, pageNum: pg.pageNum, text: text}:
						case <-ctx.Done():
							return
						}
						idx++
					}
				}
			}
		}()

		// Stage 3: embed chunks in batches, bounded to maxConcurrentBatches goroutines
		var embedWg sync.WaitGroup
		for range maxConcurrentBatches {
			embedWg.Add(1)
			go func() {
				defer embedWg.Done()
				batch := make([]chunk, 0, defaultBatchSize)
				flush := func() {
					if len(batch) == 0 {
						return
					}
					texts := make([]string, len(batch))
					for i, ch := range batch {
						texts[i] = ch.text
					}
					vecs, err := model.BatchEmbed(ctx, texts)
					if err != nil {
						log.Printf("index: batch embed failed: %v", err)
						batch = batch[:0]
						return
					}
					for i, ch := range batch {
						select {
						case out <- vectorstore.EmbeddedChunk{
							DocPath:   ch.docPath,
							ChunkIdx:  ch.chunkIdx,
							PageNum:   ch.pageNum,
							Text:      ch.text,
							Embedding: vecs[i],
						}:
						case <-ctx.Done():
							return
						}
					}
					batch = batch[:0]
				}
				for c := range chunks {
					batch = append(batch, c)
					if len(batch) >= defaultBatchSize {
						flush()
					}
				}
				flush() // remaining
			}()
		}
		embedWg.Wait()
	}()

	return out
}
