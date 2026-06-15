package service

import (
	"testing"

	"gorm.io/gorm"
)

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

func TestResolveHotCursorToken(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		cursor FeedCursor
		want   string
	}{
		{
			name: "uses wrapped hot token first",
			raw:  "raw-token",
			cursor: FeedCursor{
				HotToken: "wrapped-token",
			},
			want: "wrapped-token",
		},
		{
			name: "falls back to raw snapshot token",
			raw:  " raw-token ",
			cursor: FeedCursor{
				Mode: "hot",
			},
			want: "raw-token",
		},
		{
			name:   "empty token",
			raw:    " ",
			cursor: FeedCursor{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveHotCursorToken(tt.raw, tt.cursor)
			if got != tt.want {
				t.Fatalf("resolveHotCursorToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectHotVideoIDsPage(t *testing.T) {
	tests := []struct {
		name        string
		candidates  []uint
		visible     []uint
		offset      int
		limit       int
		total       int64
		wantIDs     []uint
		wantOffset  int
		wantHasMore bool
	}{
		{
			name:        "truncate to limit and advance by consumed candidates",
			candidates:  []uint{9, 8, 7, 6},
			visible:     []uint{9, 8, 7, 6},
			offset:      10,
			limit:       2,
			total:       20,
			wantIDs:     []uint{9, 8},
			wantOffset:  12,
			wantHasMore: true,
		},
		{
			name:        "advance past filtered items without repeating",
			candidates:  []uint{9, 8, 7, 6},
			visible:     []uint{7, 6},
			offset:      10,
			limit:       2,
			total:       20,
			wantIDs:     []uint{7, 6},
			wantOffset:  14,
			wantHasMore: true,
		},
		{
			name:        "empty visible page",
			candidates:  []uint{9, 8, 7},
			visible:     []uint{},
			offset:      10,
			limit:       2,
			total:       20,
			wantIDs:     []uint{},
			wantOffset:  13,
			wantHasMore: true,
		},
		{
			name:        "empty candidates",
			candidates:  []uint{},
			visible:     []uint{9},
			offset:      10,
			limit:       2,
			total:       20,
			wantIDs:     []uint{},
			wantOffset:  10,
			wantHasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIDs, gotOffset, gotHasMore := selectHotVideoIDsPage(tt.candidates, tt.visible, tt.offset, tt.limit, tt.total)
			if len(gotIDs) != len(tt.wantIDs) {
				t.Fatalf("selectHotVideoIDsPage() ids length = %d, want %d", len(gotIDs), len(tt.wantIDs))
			}
			for i := range tt.wantIDs {
				if gotIDs[i] != tt.wantIDs[i] {
					t.Fatalf("selectHotVideoIDsPage()[%d] = %d, want %d", i, gotIDs[i], tt.wantIDs[i])
				}
			}
			if gotOffset != tt.wantOffset {
				t.Fatalf("selectHotVideoIDsPage() offset = %d, want %d", gotOffset, tt.wantOffset)
			}
			if gotHasMore != tt.wantHasMore {
				t.Fatalf("selectHotVideoIDsPage() hasMore = %v, want %v", gotHasMore, tt.wantHasMore)
			}
		})
	}
}

func TestFanoutVideoToFollowersTxNil(t *testing.T) {
	var tx *gorm.DB
	err := fanoutVideoToFollowersTx(tx, 1, 2)
	if err == nil {
		t.Fatalf("fanoutVideoToFollowersTx() expected error for nil tx")
	}
}
