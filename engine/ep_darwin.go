//go:build darwin

package engine

import ort "github.com/yalue/onnxruntime_go"

// configureExecutionProviders enables the CoreML EP, which routes work to
// Apple's Metal GPU and/or Neural Engine when available.
func configureExecutionProviders(opts *ort.SessionOptions) error {
	return opts.AppendExecutionProviderCoreML(0)
}
