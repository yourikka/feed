package service

import "testing"

func TestParseInt64FromCacheValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		ok    bool
		want  int64
	}{
		{name: "int64", input: int64(12), ok: true, want: 12},
		{name: "int", input: int(23), ok: true, want: 23},
		{name: "string", input: "34", ok: true, want: 34},
		{name: "bytes", input: []byte("45"), ok: true, want: 45},
		{name: "nil", input: nil, ok: false, want: 0},
		{name: "invalid", input: "x", ok: false, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseInt64FromCacheValue(tt.input)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("parseInt64FromCacheValue(%v) = (%d, %v), want (%d, %v)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}
