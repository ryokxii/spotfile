//go:build windows

package embedder

import ort "github.com/yalue/onnxruntime_go"

// configureExecutionProviders enables DirectML (device 0), which covers
// both NVIDIA/AMD GPUs and the NPU on Copilot+ PCs.
func configureExecutionProviders(opts *ort.SessionOptions) error {
	return opts.AppendExecutionProviderDirectML(0)
}
