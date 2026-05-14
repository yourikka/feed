package controller

import "testing"

func TestParseUintCSV(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		got, err := parseUintCSV("1, 2,3")
		if err != nil {
			t.Fatalf("parseUintCSV() error = %v", err)
		}
		want := []uint{1, 2, 3}
		if len(got) != len(want) {
			t.Fatalf("unexpected result length: got %d want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("parseUintCSV()[%d] = %d, want %d", i, got[i], want[i])
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := parseUintCSV("1,0,3"); err == nil {
			t.Fatalf("parseUintCSV() expected error for zero value")
		}
	})
}
