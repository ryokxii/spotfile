# Spotfile

A local, privacy-first document search app. Point it at a folder and search across your files using semantic similarity — no cloud, no indexing service, everything runs on your machine.

---

## Platform support

Spotfile is **cross-platform** — a single Go + Wails codebase builds a native app for each OS, and the embedding engine selects the right hardware accelerator automatically at startup:

| OS | Webview | ONNX Runtime acceleration |
|----|---------|---------------------------|
| **macOS** (Apple Silicon & Intel) | WKWebView | CPU (all cores) — CoreML EP disabled, see `engine/embedder/provider_darwin.go` |
| **Windows 10 / 11** (x64) | WebView2 | DirectML → GPU + NPU (incl. Copilot+ PCs) |
| **Linux** | WebKitGTK | CPU (all cores) |

Each OS needs its matching ONNX Runtime library — and, to build from source, a C toolchain — as described below. Because the ONNX Runtime binding uses **CGO, release binaries are built on the target OS**; cross-compiling from one OS to another is not supported.

---

## Prerequisites

### 1. ONNX Runtime

Spotfile uses ONNX Runtime to run the embedding model. Install it before launching the app.

**macOS (Homebrew)**
```bash
brew install onnxruntime
```

**Linux**
```bash
# Debian/Ubuntu
sudo apt install libonnxruntime-dev

# Arch
sudo pacman -S onnxruntime
```

**Windows**

Spotfile uses the **DirectML** execution provider on Windows, so download the *DirectML* build of ONNX Runtime — the `onnxruntime-win-x64-directml-*.zip` asset from the [official releases](https://github.com/microsoft/onnxruntime/releases). Place **both** `onnxruntime.dll` and `DirectML.dll` next to the Spotfile executable, or anywhere on your `PATH`.

> The plain CPU-only `onnxruntime.dll` will load but fail at startup, because the app requests the DirectML provider. Use the DirectML build.

---

### 2. Embedding model

Spotfile uses [bge-small-en-v1.5](https://huggingface.co/BAAI/bge-small-en-v1.5) (ONNX export). Download the two required files into Spotfile's data folder — `~/.spotfile/` on macOS/Linux, `%USERPROFILE%\.spotfile\` on Windows (the app resolves this from your home directory on every platform).

**macOS / Linux**
```bash
mkdir -p ~/.spotfile

# model weights (~127 MB)
curl -L "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/onnx/model.onnx" \
     -o ~/.spotfile/model.onnx

# vocabulary
curl -L "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/vocab.txt" \
     -o ~/.spotfile/vocab.txt
```

**Windows (PowerShell)**
```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\.spotfile" | Out-Null

# model weights (~127 MB)
curl.exe -L "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/onnx/model.onnx" -o "$env:USERPROFILE\.spotfile\model.onnx"

# vocabulary
curl.exe -L "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/vocab.txt" -o "$env:USERPROFILE\.spotfile\vocab.txt"
```

> If you have `git-lfs` installed you can also clone the repo:
> ```bash
> git clone https://huggingface.co/BAAI/bge-small-en-v1.5 /tmp/bge
> cp /tmp/bge/onnx/model.onnx ~/.spotfile/model.onnx
> cp /tmp/bge/vocab.txt ~/.spotfile/vocab.txt
> ```

---

### 3. Development tools (only needed to build from source)

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.24+ | [go.dev/dl](https://go.dev/dl) |
| Node | 18+ | [nodejs.org](https://nodejs.org) |
| Wails CLI | v2 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| C compiler (CGO) | any | see below |

The ONNX Runtime binding uses **CGO**, so a C toolchain is required to build:

- **macOS** — Xcode Command Line Tools: `xcode-select --install`
- **Windows** — a GCC toolchain such as [mingw-w64](https://www.mingw-w64.org/) (e.g. via [MSYS2](https://www.msys2.org/)), on your `PATH`
- **Linux** — `build-essential` (Debian/Ubuntu) or `base-devel` (Arch)

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

The output app is written to `build/bin/`. On macOS you get a `.app` bundle; on Windows a `.exe`. Build on the OS you are targeting — CGO does not cross-compile here.

### macOS — Gatekeeper

Unsigned builds are quarantined by macOS. After moving the app to `/Applications`, run:

```bash
xattr -dr com.apple.quarantine /Applications/Spotfile.app
```

Or right-click → **Open** → **Open** on first launch.

### Windows — SmartScreen & DLLs

Unsigned `.exe` files trigger a SmartScreen warning on first run: click **More info → Run anyway**. Keep `onnxruntime.dll` and `DirectML.dll` (from Prerequisites §1) in the same folder as `Spotfile.exe`, or on your `PATH`.

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
| `ONNX Runtime not found — run: brew install onnxruntime` | Install the library (see Prerequisites §1) |
| ONNX Runtime / DirectML fails to load on Windows | Use the **DirectML** build of ONNX Runtime and keep `onnxruntime.dll` + `DirectML.dll` next to `Spotfile.exe` (Prerequisites §1) — the CPU-only build lacks DirectML |
| `no such file: model.onnx` | Download the model files into `~/.spotfile` / `%USERPROFILE%\.spotfile` (see Prerequisites §2) |
| `no supported files found` | The chosen folder contains no `.txt`, `.md`, or `.pdf` files |

---

## License

MIT
