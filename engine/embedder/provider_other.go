//go:build !darwin && !windows

package embedder

import ort "github.com/yalue/onnxruntime_go"

// configureExecutionProviders uses the built-in CPU EP on Linux and other
// platforms. ONNX Runtime will still use all available CPU threads.
func configureExecutionProviders(_ *ort.SessionOptions) error {
	return nil
}
