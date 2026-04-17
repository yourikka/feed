package ranking

import (
	"strings"
	"testing"
)

func TestGetHotWindowHours(t *testing.T) {
	t.Setenv("FEED_HOT_WINDOW_HOURS", "")
	if got := getHotWindowHours(); got != 24 {
		t.Fatalf("expected default hot window 24, got %d", got)
	}

	t.Setenv("FEED_HOT_WINDOW_HOURS", "48")
	if got := getHotWindowHours(); got != 48 {
		t.Fatalf("expected hot window 48, got %d", got)
	}

	t.Setenv("FEED_HOT_WINDOW_HOURS", "-1")
	if got := getHotWindowHours(); got != 24 {
		t.Fatalf("expected fallback hot window 24 for invalid value, got %d", got)
	}
}

func TestBuildAggKey(t *testing.T) {
	t.Setenv("FEED_HOT_WINDOW_HOURS", "24")
	key := buildAggKey()
	if !strings.HasPrefix(key, hotAggPrefix) {
		t.Fatalf("agg key %q should start with %q", key, hotAggPrefix)
	}
}
