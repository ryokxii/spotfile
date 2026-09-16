package embedder

import "testing"

func encodingWithLen(n int) Encoding {
	mask := make([]int64, MaxSeqLen)
	for i := range n {
		mask[i] = 1
	}
	return Encoding{
		InputIDs:      make([]int64, MaxSeqLen),
		AttentionMask: mask,
		TokenTypeIDs:  make([]int64, MaxSeqLen),
	}
}

func TestLongestSequence(t *testing.T) {
	tests := []struct {
		name string
		lens []int
		want int
	}{
		{"single short", []int{5}, 5},
		{"mixed picks longest", []int{7, 120, 33}, 120},
		{"full length", []int{MaxSeqLen, 10}, MaxSeqLen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encs := make([]Encoding, len(tt.lens))
			for i, n := range tt.lens {
				encs[i] = encodingWithLen(n)
			}
			if got := longestSequence(encs); got != tt.want {
				t.Fatalf("longestSequence = %d, want %d", got, tt.want)
			}
		})
	}
}
