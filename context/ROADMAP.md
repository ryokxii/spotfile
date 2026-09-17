# Spotfile — Project Roadmap

> Status legend: `[x]` complete · `[~]` in progress or partial (see notes) · `[ ]` not started
>
> Keep this file current in the same branch as the change (see `context/CLAUDE.md` §2).

## Phase 1: Project Scaffolding & Core Architecture

- [x] **Initialize Wails Project** — Wails v2 app with a Go backend and a Svelte + TypeScript frontend
- [x] **Local inference engine** — llama.cpp's `llama-server` runs as a child process (`engine/llamaserver`): started on demand, authenticated loopback API key passed in a private file, restarted after a crash, never outlives Spotfile. Replaced ONNX Runtime, which was removed (PR #5)
- [x] **Hardware acceleration** — `-ngl 99` puts all model layers on the GPU when present (Metal on macOS); CPU otherwise
- [x] **Define App Data Storage** — `~/.spotfile/` holds `models/` (GGUF files) and the persisted vector store `store.gob`, reloaded on launch

---

## Phase 2: High-Performance Indexing (Go Backend)

- [x] **Bounded read pool** — reader goroutines capped at the CPU count drain a shared path channel
- [x] **Streaming pipeline** — read → chunk → embed stages connected by bounded channels, so reading overlaps inference
- [x] **Embedding model** — bge-small-en-v1.5 as an f16 GGUF on `llama-server --embeddings` with CLS pooling and bge's query instruction for searches
- [x] **Token-accurate chunking** — 250-word windows (40-word overlap), split further with llama.cpp's tokenizer until each fits the 510-token limit; nothing is silently truncated
- [x] **Vector batching** — 16 chunks per embedding request, at most 2 concurrent requests (the server's parallel slots)
- [x] **Index migration** — `store.gob` records the embedding model ID; an old-format or different-model store is rebuilt on launch with progress

---

## Phase 3: Real-Time Sync & Local Inference

- [x] **"Silent Engine" watcher** — `fsnotify` re-indexes changed files and emits `watcher:reindexing` / `watcher:done`
- [x] **Debouncing** — ~2-second debounce per file prevents CPU spikes during active edits
- [x] **Priority queue** — urgent (live edits) > high (files modified in the last 7 days) > low (history backfill)
- [ ] **Local LLM answers** — `GenerateAnswer` and `engine/llm` are placeholders that return "model not loaded". Delivered by App piece 3, *Chat engine* (Gap 2)

---

## Phase 4: UI & User Experience

- [x] **Bridge Frontend & Backend** — Go methods bound via Wails; long work reports progress through events, each with a frontend listener
- [x] **PDF.js Viewer** — docked preview with page jumping and match highlighting; text and Markdown viewers alongside
- [x] **Progressive Status UI** — live indexing status from `index:*` and `watcher:*` events; `EngineStatus` lets a freshly loaded UI catch up
- [x] **Design system** — Nocturne and Daylight themes as CSS tokens (`context/DESIGN.md`, `styles/tokens.css`) with an automated contrast test; bundled Fraunces + Work Sans

---

## Phase 5: Deployment & Distribution

- [x] **Automate Multi-Platform Builds** — GitHub Actions builds a macOS universal `.app` and a Windows `.exe`, and publishes a release on `v*` tags; CI also runs lint, tests against a real llama-server, the peak-memory test and benchmarks
- [x] **"Quarantine" documentation** — `docs/INSTALLING.md` covers macOS Gatekeeper (`xattr`) and Windows SmartScreen
- [ ] **Generate App Icons** — `build/appicon.png` is still the Wails template icon

---

## Phase 6: App Experience (five pieces, in order)

- [x] **1. App shell** — sidebar navigation with routes, design tokens, stores for engine/search/theme, restyled Search view (PR #9)
- [~] **2. Settings** — theme (System / Nocturne / Daylight), reduce motion, one saved indexed folder with the watcher resumed on launch, storage figures, Clear index; saved in `~/.spotfile/settings.json`. Design: `docs/superpowers/specs/2026-09-17-settings-design.md` (PR #10). Plan: `docs/superpowers/plans/2026-09-17-settings.md`. Implementation not started
- [~] **3. Chat engine** — design in progress. Decided: engine plus a single unsaved chat page; Qwen3-1.7B (Q4_K_M) with thinking disabled; every message searches the index and answers only from your files with numbered citations; follow-ups search the new message blended with the previous one. Pending: streaming approach (proposed: Go streams tokens as Wails events, with Stop)
- [ ] **4. Chat sessions** — multiple conversations, saved and listed; clearing chat history
- [ ] **5. Profile**

---

## Follow-up Gaps (tracked)

- [x] **Gap 1 — Persist & de-duplicate the vector store.** `VectorStore` is keyed by `DocPath`; gob-persisted to `~/.spotfile/store.gob` (atomic temp+rename), loaded on launch; re-indexing a path replaces its chunks. (`engine/vectorstore/`)
- [~] **Gap 2 — Finish LLM integration.** Scheduled as App piece 3, *Chat engine*: a second llama-server running the chat model, replacing the `engine/llm` placeholder.
- [x] **Gap 3 — Emit watcher UI events.** `watcher:reindexing` (with path) and `watcher:done` drive the status line. (`engine/indexing/watcher.go`)
- [x] **Gap 4 — Real priority queue for indexing.** Per-level FIFO queues (Urgent/High/Low) drained by a single worker, at most 256 paths per level per iteration so new urgent work isn't starved. (`engine/indexing/queue.go`)
- [x] **Gap 5 — Bound file-reader concurrency.** Bounded reader pool instead of one goroutine per file. (`engine/indexing/pipeline.go`)
- [ ] **Gap 6 — Catch up on edits made while Spotfile was closed.** With Settings, the watcher resumes on the saved folder at launch, but files changed while the app was closed aren't re-indexed until they change again. Needs a startup scan comparing modification times with the index.
- [ ] **Gap 7 — Search scales linearly.** `VectorStore.Search` scores and sorts every chunk on each query. Fine at today's sizes; an approximate index or top-k heap is needed for very large libraries (see `PERFORMANCE.md` §7).
