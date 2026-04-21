package dbmanage

import (
	"testing"

	"PostmanJanai/internal/testutil"
)

func TestUserVersionRoundTrip(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)

	v, err := UserVersion(db)
	if err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if v != 0 {
		t.Fatalf("fresh DB should have user_version 0, got %d", v)
	}

	if err := SetUserVersion(db, 7); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, _ = UserVersion(db)
	if v != 7 {
		t.Fatalf("want 7, got %d", v)
	}
}

func TestSetUserVersionRejectsNegative(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)
	if err := SetUserVersion(db, -1); err == nil {
		t.Fatal("negative user_version should fail")
	}
}
