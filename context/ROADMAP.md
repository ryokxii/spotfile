# Spotfile — Project Roadmap

## Phase 1: Project Scaffolding & Core Architecture

- [ ] **Initialize Wails Project** — Scaffold the app using `wails init` with Go backend + preferred frontend (React/Vue)
- [ ] **Configure ONNX Runtime** — Integrate Go bindings for ONNX Runtime for local AI inference without a Python dependency
- [ ] **Setup Hardware Acceleration** — Enable Execution Providers: Metal (macOS), DirectML/CUDA (Windows) for GPU/NPU offloading
- [ ] **Define App Data Storage** — Locate and use standard OS Application Data folders for persistent vector storage

---

## Phase 2: High-Performance Indexing (Go Backend)

- [ ] **Implement Parallel Indexer** — Use Goroutines + Worker Pools to process files simultaneously across all CPU cores
- [ ] **Build Vector Streaming Pipeline** — MPSC channels to separate document parsing from model inference (read next file while current is being embedded)
- [ ] **Configure bge-small-en-v1.5** — Use the int8 quantized BGE model within ONNX for fast, memory-efficient vectorization
- [ ] **Implement Vector Batching** — Group text chunks (16–32 at a time) before calling inference to reduce overhead

---

## Phase 3: Real-Time Sync & Local Inference

- [ ] **Setup "Silent Engine" Watcher** — Real-time file watcher via `fsnotify` to trigger instant re-indexing on file create/modify
- [ ] **Apply Debouncing Logic** — ~2-second delay on watcher to prevent CPU spikes during active edits
- [ ] **Integrate Local LLM Inference** — Use `llama.go` to run a 4-bit quantized (Q4_K_M) Llama 3.2 (3B) or Gemma 3 (4B) for private generative responses
- [ ] **Prioritize Recent Files** — Priority Queue in the indexer so recently modified files are searchable immediately while history scans in background

---

## Phase 4: UI & User Experience

- [ ] **Bridge Frontend & Backend** — Bind Go indexing/search methods to the frontend via Wails' native binding system
- [ ] **Integrate PDF.js Viewer** — Embed PDF viewer with page-level jumping so the app opens documents at the exact location of the AI's answer
- [ ] **Add Progressive Status UI** — Non-intrusive live indexing indicator (e.g., "45/200 files indexed")

---

## Phase 5: Deployment & Distribution

- [ ] **Automate Multi-Platform Builds** — GitHub Actions CI/CD to build native binaries for macOS (`.app`/`.dmg`) and Windows (`.exe`)
- [ ] **Prepare "Quarantine" Documentation** — Guide for bypassing macOS Gatekeeper (`xattr`) and Windows SmartScreen for unsigned releases
- [ ] **Generate App Icons** — Use Wails tools to produce `.icns` and `.ico` formats for a professional native look
