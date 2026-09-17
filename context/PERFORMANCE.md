# Spotfile — Performance Architecture

How Spotfile keeps indexing, search and inference fast and memory-bounded on
consumer hardware, and what has been measured.

> Keep this file current in the same branch as any change to indexing, search,
> inference, memory or startup (see `context/CLAUDE.md` §2). Numbers are
> measurements with the machine named; anything not yet measured is marked
> **expected**.

---

## 1. Inference runs in llama.cpp, out of process

Embeddings (and, from App piece 3, chat) run in llama.cpp's `llama-server`, a
child process managed by `engine/llamaserver`:

- **One server per role.** Each `llamaserver.Server` owns one process with its own model and flags. The embedder's server starts at launch, so a missing binary or bad model is reported immediately rather than on the first search.
- **GPU offload.** `-ngl 99` places every layer on the GPU when one exists (Metal on macOS). There is no per-OS execution-provider code to maintain.
- **Isolation.** Model memory lives in the child. A crash restarts the server instead of taking the app down, and the server never outlives Spotfile.
- **Lifecycle.** `Config.IdleTimeout` can stop a server with no active leases. The embedder doesn't set it (it serves search and the watcher). The chat server is **expected** to use it to release its RAM when chat is idle.

**Why it replaced ONNX Runtime** (PR #5, measured on an Apple M2, indexing a 60-page document):

| Engine | Peak memory (Spotfile + inference) | Indexing speed |
|---|---|---|
| ONNX Runtime, CoreML EP | 12.7 GB | — |
| ONNX Runtime, CPU EP | ~1.6 GB | baseline |
| llama.cpp `llama-server` (Metal) | **281 MiB** | **~3× faster** |

---

## 2. Models

| Model | Role | Format | Notes |
|---|---|---|---|
| `bge-small-en-v1.5` | Embeddings | f16 GGUF (~65 MB) | 384 dimensions, CLS pooling, 510-token input limit, query instruction prefix on searches |
| `Qwen3-1.7B` | Chat (App piece 3) | Q4_K_M GGUF (~1.1 GB) | Chosen for speed on 16 GB machines; thinking disabled per request. Throughput and memory not yet measured |

`embedder.ID` (`bge-small-en-v1.5-f16/cls`) is stored in `store.gob`. Changing
the model, quantization or pooling changes the ID, which triggers a rebuild
instead of mixing incompatible vectors.

---

## 3. Streaming indexing pipeline

`engine/indexing/pipeline.go`:

```
paths ─▶ [reader pool: NumCPU goroutines] ─▶ pages ─▶ [chunker: 1 goroutine] ─▶ chunks ─▶ [batcher: ≤2 concurrent requests] ─▶ out
```

- **Readers** extract text (PDF pages or whole files) in parallel. The pool is bounded by the CPU count, not one goroutine per file.
- **The single chunker** keeps chunk order stable per document.
- **The batcher** sends 16 chunks per `/v1/embeddings` request, with at most `embedder.ParallelRequests` (2) in flight. That matches the server's parallel slots, so extra concurrency would only queue.
- **Bounded buffers:** the pages and paths channels are capped at 256, chunks at 64 (4 × batch size), output at 64. Memory stays flat on 100K-file libraries.
- Reading overlaps inference: the next files are parsed while the current batch embeds.

### Token-accurate chunking

Word windows of 250 words with a 40-word overlap keep most prose under the
model's window. Any window over 510 tokens is halved by words, using
llama.cpp's own `/tokenize`, until every piece fits. When the old 400-word
chunker ran on the repo's docs, 7 of 10 chunks exceeded the window and their
tails were silently dropped. That no longer happens.

---

## 4. Prioritized indexing

`engine/indexing/queue.go`: one background worker drains three FIFO levels.

| Priority | Source | Effect |
|---|---|---|
| Urgent | Watcher: a file just edited | Re-indexed ahead of everything |
| High | Files modified in the last 7 days | Searchable first after choosing a folder |
| Low | Older files | Backfilled in the background |

The worker takes at most 256 paths from a level per iteration, so an urgent
edit waits for at most one batch, not an entire history scan. A job's chunks
replace any earlier chunks for the same paths (`RemoveDocs`, then add).

---

## 5. Watcher debounce

`fsnotify` events are debounced per file for 2 seconds. An editor's autosave
loop triggers one re-index after the file settles, not one per write.

---

## 6. Persistence

- `store.gob` is gob-encoded and written through a temp file and atomic rename, so a crash never leaves a truncated index.
- It is written **once per completed indexing job**, not per chunk. Writes snapshot the map under a read lock and encode outside it.
- It is loaded once at launch. Settings (App piece 2, planned) will add `DiskSize()`, a single `os.Stat` per `EngineStatus` call, and `settings.json`, a small JSON file written only when a preference changes. Neither is **expected** to affect indexing or search.

---

## 7. Search

`VectorStore.Search` is exact nearest-neighbour search. It computes the dot
product (vectors are L2-normalized, so dot product = cosine) against every
chunk, then sorts all candidates. The query itself costs one embedding request.

- **Cost:** linear in chunk count, plus an O(n log n) sort. Fine at current library sizes; not measured at scale.
- **Planned improvement (ROADMAP Gap 7):** a bounded top-k heap removes the full sort; an approximate index is needed for very large libraries.

---

## 8. Measured results

All on an Apple M2 (16 GB) unless noted. CI runs the benchmarks on `macos-latest` and posts results in the job summary.

| What | Test | Result | Budget / note |
|---|---|---|---|
| Peak memory indexing a 60-page document (Spotfile + llama-server) | `TestIndexPeakMemory` | 281 MiB | Budget 512 MiB; fails CI above it |
| Embedding throughput, prose corpus (100 files × 450 words) | `BenchmarkEmbedFiles/prose` | 86 chunks/s (283 chunks in 3.27 s) | Server started before timing |
| Embedding throughput, Markdown + code corpus | `BenchmarkEmbedFiles/markdown` | 48 chunks/s (309 chunks in 6.46 s) | Denser tokens, more splitting |
| Legacy index rebuild in the real app | manual | 17 files, 819 chunks in 5 s | PR #5 |

Benchmarks use deterministic, realistic prose and Markdown corpora
(`engine/indexing/bench_test.go`), never synthetic byte sequences.

---

## Summary

| Strategy | Mechanism | Benefit |
|---|---|---|
| Out-of-process llama.cpp | `llama-server` child per role, Metal offload | 281 MiB peak, crash isolation, ~3× faster than ONNX |
| Streaming pipeline | Bounded reader pool → chunker → batcher | Reading overlaps inference; flat memory |
| Batching | 16 chunks/request, 2 concurrent | Amortized request overhead, matches server slots |
| Token-accurate chunking | 250/40 word windows, split by real token count | No silently dropped text |
| Priority queue | Urgent > recent 7 days > history, 256 per drain | Current work searchable first |
| Debounced watcher | 2 s per file | Live sync without CPU spikes |
| Atomic, per-job persistence | temp + rename, once per job | Crash-safe, few writes |
| Model-ID'd index | `embedder.ID` in `store.gob` | Safe automatic rebuild on model change |
