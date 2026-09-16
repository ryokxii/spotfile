# Spotfile

A local, privacy-first document search app. Point it at a folder and search across your files using semantic similarity — no cloud, no indexing service, everything runs on your machine.

---

## Platform support

Spotfile is **cross-platform** — a single Go + Wails codebase builds a native app for each OS. Embeddings are computed by [llama.cpp](https://github.com/ggml-org/llama.cpp)'s `llama-server`, which Spotfile runs as a background child process on `127.0.0.1` and stops when the app quits (or crashes).

| OS | Webview | Embedding acceleration (llama.cpp) |
|----|---------|------------------------------------|
| **macOS** (Apple Silicon) | WKWebView | Metal GPU |
| **Windows 10 / 11** (x64) | WebView2 | Vulkan GPU build, or CPU build |
| **Linux** | WebKitGTK | Vulkan/CUDA builds, or CPU |

---

## Prerequisites

### 1. llama.cpp

Spotfile looks for `llama-server` in this order: the `SPOTFILE_LLAMA_SERVER` environment variable, next to the Spotfile executable, then your `PATH`.

**macOS / Linux (Homebrew)**
```bash
brew install llama.cpp
```

**Windows**

Download a Windows build from the [llama.cpp releases](https://github.com/ggml-org/llama.cpp/releases) — the `win-vulkan-x64` zip for GPU acceleration on NVIDIA/AMD/Intel, or `win-cpu-x64` otherwise. Place `llama-server.exe` **and the DLLs from the zip** next to `Spotfile.exe`, or set `SPOTFILE_LLAMA_SERVER` to the full path of `llama-server.exe`.

---

### 2. Embedding model

Spotfile uses [bge-small-en-v1.5](https://huggingface.co/BAAI/bge-small-en-v1.5) as a 16-bit GGUF file (64 MB). Download it into Spotfile's models folder — `~/.spotfile/models/` on macOS/Linux, `%USERPROFILE%\.spotfile\models\` on Windows.

**macOS / Linux**
```bash
mkdir -p ~/.spotfile/models
curl -L "https://huggingface.co/unsloth/bge-small-en-v1.5-GGUF/resolve/main/bge-small-en-v1.5-f16.gguf" \
     -o ~/.spotfile/models/bge-small-en-v1.5-f16.gguf
```

**Windows (PowerShell)**
```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\.spotfile\models" | Out-Null
curl.exe -L "https://huggingface.co/unsloth/bge-small-en-v1.5-GGUF/resolve/main/bge-small-en-v1.5-f16.gguf" -o "$env:USERPROFILE\.spotfile\models\bge-small-en-v1.5-f16.gguf"
```

SHA-256: `c5d2302edc429f679642433b9f96dd217799727a06e675abef3b40a79f5e1589`

> **Upgrading from the ONNX Runtime version:** the first launch detects the old index and rebuilds it automatically (progress shows in the status bar). The old `~/.spotfile/model.onnx` and `vocab.txt` are no longer used and can be deleted, and ONNX Runtime can be uninstalled.

---

### 3. Development tools (only needed to build from source)

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.24+ | [go.dev/dl](https://go.dev/dl) |
| Node | 18+ | [nodejs.org](https://nodejs.org) |
| Wails CLI | v2 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |

Wails needs the platform webview toolchain:

- **macOS** — Xcode Command Line Tools: `xcode-select --install`
- **Windows** — nothing extra (WebView2 ships with Windows 10/11)
- **Linux** — `build-essential` and `libwebkit2gtk-4.0-dev` (Debian/Ubuntu), or the equivalents for your distro

---

## Running from source

```bash
# Clone
git clone https://github.com/spotfile-app/spotfile
cd spotfile

# Install frontend dependencies
cd frontend && npm install && cd ..

# Start in development mode (hot reload)
wails dev
```

---

## Building a release binary

```bash
wails build
```

The output app is written to `build/bin/`. On macOS you get a `.app` bundle; on Windows a `.exe`. Build on the OS you are targeting.

### macOS — Gatekeeper

Unsigned builds are quarantined by macOS. After moving the app to `/Applications`, run:

```bash
xattr -dr com.apple.quarantine /Applications/Spotfile.app
```

Or right-click → **Open** → **Open** on first launch.

### Windows — SmartScreen & DLLs

Unsigned `.exe` files trigger a SmartScreen warning on first run: click **More info → Run anyway**. Keep `llama-server.exe` and its DLLs (from Prerequisites §1) in the same folder as `Spotfile.exe`.

---

## Usage

1. Launch Spotfile.
2. Click the folder icon in the search bar and pick a directory.
3. Spotfile recursively indexes all `.txt`, `.md`, and `.pdf` files.
4. Type a question or phrase and press **Enter**.
5. Click any PDF result to open it at the matching page with the relevant text highlighted.

Spotfile watches the indexed folder for changes and re-indexes modified files automatically.

---

## Troubleshooting

| Error | Fix |
|-------|-----|
| `llama.cpp not found — install it …` | Install llama.cpp or place `llama-server` next to Spotfile (Prerequisites §1) |
| `missing bge-small-en-v1.5-f16.gguf` | Download the model into `~/.spotfile/models` / `%USERPROFILE%\.spotfile\models` (Prerequisites §2) |
| `start embedding server: llama-server exited during startup …` | The message includes llama-server's own log. Usually a corrupted model download (check the SHA-256) or a llama.cpp build too old for the model — update llama.cpp |
| `no supported files found` | The chosen folder contains no supported text or PDF files |

---

## License

MIT
