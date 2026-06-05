# Spotfile — Performance Architecture

A deep-dive into every performance strategy powering Spotfile's Go backend, combining insights from both technical briefs.

---

## 1. Parallelism with Goroutines & Worker Pools

Go is built for high-concurrency workloads. Instead of indexing files sequentially (one-by-one), Spotfile uses a **Worker Pool** pattern:

- A pool of goroutines listens on a shared channel of file paths
- Each worker independently handles chunking and embedding for its assigned file
- All available CPU cores are utilized — critical when converting thousands of document chunks into vectors

**Why it matters:** Vectorizing documents is computationally expensive. A single-threaded indexer wastes most of the machine. A worker pool turns all idle cores into throughput.

**Phase 2 Implementation:** Worker pool processes batches of 32 chunks per ONNX call instead of 1, reducing context-switching overhead and improving GPU saturation.

---

## 2. ONNX Runtime with Go Bindings

The primary inference engine for embedding generation is ONNX Runtime via Go bindings.

- The ONNX Runtime is **thread-safe**, meaning multiple goroutines can each hold a reference to the same ONNX session and call it concurrently without locking
- Workers pull document chunks from a shared channel and run embeddings in parallel
- No Python dependency — the entire inference stack runs natively in the Go process

**Hardware Acceleration:** Execution Providers are explicitly enabled at startup:
- **Metal** (macOS) — offloads math to the Apple GPU/Neural Engine
- **DirectML / CUDA** (Windows) — offloads to NVIDIA/AMD GPU

This moves the heavy linear algebra off the CPU entirely, freeing cores for I/O and coordination work.

---

## 3. Quantized Models for Consumer Hardware

To stay fast on everyday hardware, Spotfile uses heavily optimized model variants:

| Model | Use Case | Format | Speedup vs Full Precision |
|---|---|---|---|
| `bge-small-en-v1.5` | File embeddings | int8 quantized (ONNX) | ~4x smaller, significantly faster |
| Llama 3.2 (3B) or Gemma 3 (4B) | Generative responses | 4-bit Q4_K_M | Runs on CPU/GPU without VRAM pressure |

- **int8 quantization** (for the embedding model) halves memory bandwidth requirements and fits easily in L2/L3 cache
- **Q4_K_M quantization** (for the LLM) compresses weights to ~4 bits per parameter, enabling 3–4B parameter models to run in under 4 GB of RAM

---

## 4. Vector Streaming Pipeline (MPSC Channels)

Rather than waiting for a full file to be read before embedding begins, Spotfile **pipelines I/O and compute** using Go channels in an MPSC (multi-producer, single-consumer) pattern:

```
[File Reader Goroutines] → docs channel → [Chunker] → chunks channel → [Embedding Workers]
```

- **Stage 1 (Producers):** A set of goroutines reads files in parallel, fed to `docs` channel
- **Stage 2 (Single Chunker):** Serializes document text into chunks, preserving order
- **Stage 3 (Consumer Pool):** Embedding workers drain chunks and batch them for ONNX inference

**The benefit:** While the embedding model processes current batch, the next files are already being read and parsed. I/O and compute overlap instead of stacking — minimizing total "wait" time across the entire indexing run.

**Phase 2 Optimization:** 
- Capped `docs` channel buffer to prevent unbounded memory growth (maxDocBuffer = 256)
- Dynamically sized `chunks` buffer based on batch size (defaultBatchSize * 4 = 128)
- This maintains pipeline efficiency while protecting against memory bloat on large (100K+) file collections

---

## 5. Vector Batching (Phase 2 — Primary Optimization)

Individual model calls carry fixed overhead (context switching, kernel launches on GPU, session setup). Batching amortizes this cost:

- Workers **accumulate chunks** (16–32 at a time) before submitting to the embedding model
- The GPU/NPU processes all chunks in a **single pass**, leveraging its parallelism at the hardware level
- Fewer total model calls = less overhead = higher throughput

**Implementation:**
- `BatchEmbed(texts []string) ([][]float32, error)` batches tokenization, tensor creation, and ONNX inference
- Tokenizes all N texts, creates batch tensors [batch_size, seq_len], runs single inference
- Extracts and L2-normalizes embeddings for each text while maintaining order
- Expected speedup: **3–5x faster** than individual embeddings

**Rule of thumb:** Batch size of 16–32 is a practical sweet spot — large enough to saturate GPU lanes, small enough to keep latency per batch low.

---

## 6. Prioritized "Just-in-Time" Indexing (Phase 2)

Users shouldn't have to wait for a full library scan before they can search. Spotfile solves this with a **Priority Queue** in the indexer:

- **Recently modified files** are placed at the front of the queue via `SortPathsByModTime()`
- The user can search their current work **immediately** after launch
- Historical files continue scanning in the background as lower-priority work

**Implementation:**
- Uses `os.Stat()` to retrieve `ModTime` for each file
- Sorts files in descending order (most recent first)
- Gracefully handles missing/inaccessible files (logs and continues)

This separates *perceived* responsiveness from *actual* indexing completion, making the app feel instant even on large document libraries.

---

## 7. Real-Time Sync with Debounced File Watching

Spotfile's "Silent Engine" keeps the index fresh without manual re-runs:

- **`fsnotify`** watches the filesystem for create/modify events and triggers re-indexing automatically (Phase 3)
- A **~2-second debounce delay** is applied before acting on any event — this prevents CPU spikes when files are being actively written (e.g., autosave loops in editors)

**Result:** The index stays up-to-date in real time with near-zero idle overhead.

---

## 8. Local LLM Inference via llama.go

For generative responses (RAG answers), Spotfile runs a local LLM using `llama.go` (Phase 3):

- Provides idiomatic Go bindings for local model inference — no CGO complexity, no server process
- Integrates naturally with goroutines for handling concurrent user prompts or background tasks
- Runs **entirely on-device** — no network calls, no data leaving the machine

**Models supported:** Llama 3.2 (3B) and Gemma 3 (4B) at Q4_K_M quantization, balancing answer quality with consumer hardware constraints.

---

## Phase 2 Performance Gains Summary

| Optimization | Mechanism | Expected Speedup | Status |
|---|---|---|---|
| Vector Batching | Batch 32 chunks per ONNX call | 3–5x embedding faster | ✅ Complete |
| Priority Queue | Sort by mod time (recent first) | Instant search feedback | ✅ Complete |
| Buffer Tuning | Cap docs (256), adjust chunks (128) | Prevent memory bloat | ✅ Complete |
| Benchmarking | Test suite for perf validation | Measure improvements | ✅ Complete |

---

## Summary

| Strategy | Mechanism | Primary Benefit | Phase |
|---|---|---|---|
| Worker Pool | Goroutines + shared channel | Full CPU core utilization | 1 |
| ONNX + Go Bindings | Thread-safe inference sessions | Parallel embeddings, no Python | 1 |
| Hardware Acceleration | Metal / DirectML / CUDA | GPU offloading | 1 |
| Quantized Models | int8 / Q4_K_M | Speed + low memory on consumer hardware | 1 |
| MPSC Pipeline | Channels separating I/O, chunking, compute | Overlapping operations | 1 |
| Vector Batching | 32 chunks per inference | Reduced overhead, GPU saturation | 2 |
| Priority Queue | Recent files first | Immediate search on launch | 2 |
| Buffer Tuning | Capped/dynamic channel sizes | Memory safety at scale | 2 |
| Debounced fsnotify | 2s delay on file events | Real-time sync without CPU spikes | 3 |
| llama.go | Local 4-bit LLM | Private, offline generative responses | 3 |
