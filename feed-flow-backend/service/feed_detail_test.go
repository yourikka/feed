package service

import (
	"testing"

	"github.com/yourikka/feed-flow/model"
)

func TestNormalizeSortType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "hot", input: "hot", want: "hot"},
		{name: "follow", input: "follow", want: "follow"},
		{name: "latest", input: "latest", want: "latest"},
		{name: "fallback latest", input: "unknown", want: "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSortType(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeSortType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestModelVideosToIDs(t *testing.T) {
	input := make([]model.Video, 0, 3)
	for _, id := range []uint{3, 7, 11} {
		var item model.Video
		item.ID = id
		input = append(input, item)
	}

	got := modelVideosToIDs(input)
	want := []uint{3, 7, 11}
	if len(got) != len(want) {
		t.Fatalf("unexpected result length: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("modelVideosToIDs()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestArrangeFeedVideosByIDs(t *testing.T) {
	input := map[uint]FeedVideo{
		11: {ID: 11, Title: "a"},
		3:  {ID: 3, Title: "b"},
		7:  {ID: 7, Title: "c"},
	}
	got := arrangeFeedVideosByIDs([]uint{7, 100, 11, 3}, input)
	want := []uint{7, 11, 3}
	if len(got) != len(want) {
		t.Fatalf("arrange length got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("arrange[%d] id got %d want %d", i, got[i].ID, want[i])
		}
	}
}

func TestArrangeFeedVideosByIDsKeepOrderAndSkipMissing(t *testing.T) {
	items := map[uint]FeedVideo{
		2: {ID: 2},
		4: {ID: 4},
	}
	ordered := arrangeFeedVideosByIDs([]uint{4, 5, 2, 4}, items)
	want := []uint{4, 2, 4}
	if len(ordered) != len(want) {
		t.Fatalf("arrange length got %d want %d", len(ordered), len(want))
	}
	for i := range want {
		if ordered[i].ID != want[i] {
			t.Fatalf("arrange[%d] got %d want %d", i, ordered[i].ID, want[i])
		}
	}
}
