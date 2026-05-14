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

func TestGetFollowPushThreshold(t *testing.T) {
	t.Setenv("FEED_FOLLOW_PUSH_MAX_FANS", "")
	if got := getFollowPushThreshold(); got != 2000 {
		t.Fatalf("default threshold = %d, want 2000", got)
	}

	t.Setenv("FEED_FOLLOW_PUSH_MAX_FANS", "5000")
	if got := getFollowPushThreshold(); got != 5000 {
		t.Fatalf("threshold = %d, want 5000", got)
	}

	t.Setenv("FEED_FOLLOW_PUSH_MAX_FANS", "-1")
	if got := getFollowPushThreshold(); got != 2000 {
		t.Fatalf("invalid threshold fallback = %d, want 2000", got)
	}
}

func TestDedupeVideoIDs(t *testing.T) {
	got := dedupeVideoIDs([]uint{9, 7, 9, 3, 7, 1})
	want := []uint{9, 7, 3, 1}
	if len(got) != len(want) {
		t.Fatalf("dedupe length got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dedupe[%d] got %d want %d", i, got[i], want[i])
		}
	}
}
