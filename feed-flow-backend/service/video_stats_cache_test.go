package service

import (
	"strings"
	"testing"
)

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

func TestNormalizeVideoIDs(t *testing.T) {
	got := normalizeVideoIDs([]uint{4, 2, 4, 0, 3, 2, 1})
	want := []uint{1, 2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("normalizeVideoIDs length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizeVideoIDs[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestBuildVideoStatsBatchLoadKey(t *testing.T) {
	got := buildVideoStatsBatchLoadKey([]uint{9, 3, 9, 1})
	if got != "1,3,9" {
		t.Fatalf("buildVideoStatsBatchLoadKey() = %s, want 1,3,9", got)
	}
}

func TestGetVideoStatsCacheTTL(t *testing.T) {
	for i := 0; i < 50; i++ {
		got := getVideoStatsCacheTTL()
		if got < videoStatsCacheTTL {
			t.Fatalf("ttl %s is smaller than base ttl %s", got, videoStatsCacheTTL)
		}
		if got > videoStatsCacheTTL+videoStatsCacheTTLJitter {
			t.Fatalf("ttl %s exceeds max ttl %s", got, videoStatsCacheTTL+videoStatsCacheTTLJitter)
		}
	}
}

func TestBuildVideoStatsBatchLoadKeyEmpty(t *testing.T) {
	got := buildVideoStatsBatchLoadKey([]uint{0, 0})
	if strings.TrimSpace(got) != "" {
		t.Fatalf("expected empty key, got %q", got)
	}
}
