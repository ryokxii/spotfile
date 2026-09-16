package llamaserver

import (
	"bytes"
	"strings"
	"sync"
)

const maxLogLineBytes = 512

// tailBuffer is an io.Writer that keeps only the last few lines of output, so a
// failed start can report llama-server's own error without unbounded memory.
type tailBuffer struct {
	mu      sync.Mutex
	max     int
	lines   []string
	partial []byte
}

func newTailBuffer(maxLines int) *tailBuffer {
	return &tailBuffer{max: maxLines}
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	data := append(b.partial, p...)
	for {
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}
		b.push(string(data[:i]))
		data = data[i+1:]
	}
	if len(data) > maxLogLineBytes {
		data = data[len(data)-maxLogLineBytes:]
	}
	b.partial = append([]byte(nil), data...)
	return len(p), nil
}

func (b *tailBuffer) push(line string) {
	line = strings.TrimRight(line, "\r")
	if len(line) > maxLogLineBytes {
		line = line[:maxLogLineBytes] + "…"
	}
	b.lines = append(b.lines, line)
	if len(b.lines) > b.max {
		b.lines = b.lines[len(b.lines)-b.max:]
	}
}

// String returns the retained lines, including any unterminated final line.
func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := append([]string(nil), b.lines...)
	if len(b.partial) > 0 {
		out = append(out, string(b.partial))
	}
	return strings.Join(out, "\n")
}
