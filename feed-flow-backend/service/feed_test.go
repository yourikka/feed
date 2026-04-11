package service

import "testing"

func TestNormalizeFeedLimit(t *testing.T) {
	tests := []struct {
		name   string
		input  int
		output int
	}{
		{name: "default when zero", input: 0, output: 10},
		{name: "default when negative", input: -1, output: 10},
		{name: "cap to max", input: 100, output: 30},
		{name: "keep normal value", input: 20, output: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeFeedLimit(tt.input)
			if got != tt.output {
				t.Fatalf("normalizeFeedLimit(%d) = %d, want %d", tt.input, got, tt.output)
			}
		})
	}
}
