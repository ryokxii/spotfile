//go:build darwin

package embedder

import ort "github.com/yalue/onnxruntime_go"

// configureExecutionProviders uses the built-in CPU EP on macOS.
//
// The CoreML EP (AppendExecutionProviderCoreML(0), onnxruntime 1.26) was
// measured on bge-small at ~400-token chunks: it grew the process footprint by
// ~1 GB per text in a batch (17 GB for one batch of 16) and embedded fewer
// texts per second than the CPU EP, which plateaus under 1 GB at batch 16.
func configureExecutionProviders(_ *ort.SessionOptions) error {
	return nil
}
