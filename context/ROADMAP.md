# Spotfile — Project Roadmap

> Status legend: `[x]` complete · `[~]` partial (see **Follow-up Gaps**) · `[ ]` not started

## Phase 1: Project Scaffolding & Core Architecture

- [x] **Initialize Wails Project** — Scaffold the app using `wails init` with Go backend + Svelte/TypeScript frontend
- [x] **Configure ONNX Runtime** — Integrate Go bindings for ONNX Runtime for local AI inference without a Python dependency
- [x] **Setup Hardware Acceleration** — Enable Execution Providers: Metal (macOS), DirectML/CUDA (Windows) for GPU/NPU offloading (`engine/embedder/provider_*.go`)
- [x] **Define App Data Storage** — `~/.spotfile` used for model/vocab assets and the persisted vector store (`store.gob`), gob-encoded and reloaded on launch (Gap 1)

---

## Phase 2: High-Performance Indexing (Go Backend)

- [x] **Implement Parallel Indexer** — Use Goroutines + Worker Pools to process files simultaneously across all CPU cores
- [x] **Build Vector Streaming Pipeline** — MPSC channels to separate document parsing from model inference (read next file while current is being embedded)
- [x] **Configure bge-small-en-v1.5** — Use the int8 quantized BGE model within ONNX for fast, memory-efficient vectorization
- [x] **Implement Vector Batching** — Group text chunks (16–32 at a time) before calling inference to reduce overhead (`BatchEmbed`, batch size 32)

---

## Phase 3: Real-Time Sync & Local Inference

- [x] **Setup "Silent Engine" Watcher** — `fsnotify` watcher re-indexes changed files and emits `watcher:reindexing` / `watcher:done` (Gap 3)
- [x] **Apply Debouncing Logic** — ~2-second debounce on watcher events prevents CPU spikes during active edits
- [ ] **Integrate Local LLM Inference** — Wiring is in place (`GenerateAnswer` → `LLM.Generate`), but generation returns a graceful "model not loaded" message; real GGUF inference **deferred** (Gap 2)
- [x] **Prioritize Recent Files** — Real two-tier priority queue (`engine/indexing/queue.go`): recent files index at high priority, history backfills in the background, watcher edits jump the queue at urgent priority (Gap 4)

---

## Phase 4: UI & User Experience

- [x] **Bridge Frontend & Backend** — Go indexing/search methods bound to the frontend via Wails' native binding system
- [x] **Integrate PDF.js Viewer** — PDF viewer with page-level jumping and answer highlighting (`ReadFileAsBase64`, `PageNum` on results)
- [x] **Add Progressive Status UI** — Live indexing indicator via `index:start` / `index:prepared` / `index:chunk` / `index:done` events

---

## Phase 5: Deployment & Distribution

- [ ] **Automate Multi-Platform Builds** — GitHub Actions CI/CD to build native binaries for macOS (`.app`/`.dmg`) and Windows (`.exe`)
- [ ] **Prepare "Quarantine" Documentation** — Guide for bypassing macOS Gatekeeper (`xattr`) and Windows SmartScreen for unsigned releases
- [ ] **Generate App Icons** — Use Wails tools to produce `.icns` and `.ico` formats for a professional native look

---

## Follow-up Gaps (tracked)

- [x] **Gap 1 — Persist & de-duplicate the vector store.** `VectorStore` is now keyed by `DocPath`; gob-persisted to `~/.spotfile/store.gob` (atomic temp+rename), loaded on launch, and re-indexing a path replaces its chunks via `RemoveDoc`/`RemoveDocs` instead of accumulating duplicates. (`engine/vectorstore/`)
- [ ] **Gap 2 — Finish LLM integration.** *Deferred by decision.* `GenerateAnswer` searches and formats context correctly, but `LLM.Generate` returns a graceful "model not loaded" message. Loading a real 4-bit GGUF model (cgo llama binding or Ollama HTTP) is tracked as its own follow-up.
- [x] **Gap 3 — Emit watcher UI events.** The watcher's urgent re-index job emits `watcher:reindexing` (with path) on start and `watcher:done` on completion, driving the StatusBar. (`engine/indexing/watcher.go`)
- [x] **Gap 4 — Real priority queue for indexing.** New `Indexer` (`engine/indexing/queue.go`) with per-level FIFO queues (Urgent/High/Low) drained by a single background worker; recent files enqueue High, history Low, watcher edits Urgent. Preserves batching by handing each drained batch to `indexing.EmbedFiles`.
- [x] **Gap 5 — Bound file-reader concurrency.** The read stage now uses a bounded pool of `Workers` reader goroutines draining a shared channel, instead of one goroutine per file. (`engine/indexing/pipeline.go`)
