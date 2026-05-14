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
