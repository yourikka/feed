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
