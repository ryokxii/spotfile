# Spotfile

A local, privacy-first document search app. Point it at a folder and search across your files using semantic similarity — no cloud, no indexing service, everything runs on your machine.

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

Download the ONNX Runtime shared library from the [official releases](https://github.com/microsoft/onnxruntime/releases) and place `onnxruntime.dll` in the same directory as the Spotfile executable, or add it to your `PATH`.

---

### 2. Embedding model

Spotfile uses [bge-small-en-v1.5](https://huggingface.co/BAAI/bge-small-en-v1.5) (int8 ONNX export). Download the two required files and place them in `~/.spotfile/`:

```bash
mkdir -p ~/.spotfile

# model weights
curl -L "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/onnx/model_quantized.onnx" \
     -o ~/.spotfile/model.onnx

# vocabulary
curl -L "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/vocab.txt" \
     -o ~/.spotfile/vocab.txt
```

> If you have `git-lfs` installed you can also clone the repo:
> ```bash
> git clone https://huggingface.co/BAAI/bge-small-en-v1.5 /tmp/bge
> cp /tmp/bge/onnx/model_quantized.onnx ~/.spotfile/model.onnx
> cp /tmp/bge/vocab.txt ~/.spotfile/vocab.txt
> ```

---

### 3. Development tools (only needed to build from source)

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.24+ | [go.dev/dl](https://go.dev/dl) |
| Node | 18+ | [nodejs.org](https://nodejs.org) |
| Wails CLI | v2 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |

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

The output app is written to `build/bin/`. On macOS you will get a `.app` bundle; on Windows an `.exe`.

### macOS — Gatekeeper

Unsigned builds are quarantined by macOS. After moving the app to `/Applications`, run:

```bash
xattr -dr com.apple.quarantine /Applications/Spotfile.app
```

Or right-click → **Open** → **Open** on first launch.

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
| `no such file: model.onnx` | Download the model files (see Prerequisites §2) |
| `no supported files found` | The chosen folder contains no `.txt`, `.md`, or `.pdf` files |

---

## License

MIT
