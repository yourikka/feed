package service

import (
	"errors"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

func TestIsDuplicateEntry(t *testing.T) {
	if !isDuplicateEntry(&mysqlDriver.MySQLError{Number: 1062}) {
		t.Fatalf("expected duplicate for mysql 1062")
	}

	if isDuplicateEntry(&mysqlDriver.MySQLError{Number: 1048}) {
		t.Fatalf("did not expect duplicate for mysql 1048")
	}

	if !isDuplicateEntry(errors.New("Duplicate entry")) {
		t.Fatalf("expected duplicate for message match")
	}

	if isDuplicateEntry(nil) {
		t.Fatalf("did not expect duplicate for nil")
	}
}

func TestParseInteractionCacheValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		raw         string
		wantState   bool
		wantVersion int64
		wantOK      bool
	}{
		{name: "active state", raw: "1:123", wantState: true, wantVersion: 123, wantOK: true},
		{name: "inactive state", raw: "0:456", wantState: false, wantVersion: 456, wantOK: true},
		{name: "invalid format", raw: "broken", wantOK: false},
		{name: "invalid version", raw: "1:notnum", wantOK: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotState, gotVersion, gotOK := parseInteractionCacheValue(tt.raw)
			if gotOK != tt.wantOK {
				t.Fatalf("parseInteractionCacheValue() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotState != tt.wantState {
				t.Fatalf("parseInteractionCacheValue() state = %v, want %v", gotState, tt.wantState)
			}
			if gotVersion != tt.wantVersion {
				t.Fatalf("parseInteractionCacheValue() version = %d, want %d", gotVersion, tt.wantVersion)
			}
		})
	}
}

func TestFormatInteractionCacheValue(t *testing.T) {
	t.Parallel()

	if got := formatInteractionCacheValue(true, 12); got != "1:12" {
		t.Fatalf("formatInteractionCacheValue(true, 12) = %q", got)
	}
	if got := formatInteractionCacheValue(false, 0); got != "0:0" {
		t.Fatalf("formatInteractionCacheValue(false, 0) = %q", got)
	}
}
