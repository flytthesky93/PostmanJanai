// Package testutil provides test-only helpers (e.g. throwaway ent.Client backed by SQLite).
// It is imported only from *_test.go files and has no place in production binaries.
package testutil

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"PostmanJanai/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	_ "github.com/glebarez/go-sqlite"
)

// NewEntClient returns a fresh ent.Client backed by a SQLite file under t.TempDir().
// Schema.Create is already applied. t.Cleanup closes the client automatically.
//
// Using a real file (not :memory:) keeps behaviour identical to production and avoids
// shared-cache surprises when several sub-tests run in parallel.
func NewEntClient(tb testing.TB) *ent.Client {
	tb.Helper()
	dbPath := filepath.Join(tb.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		tb.Fatalf("sql.Open: %v", err)
	}
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	if err := client.Schema.Create(context.Background()); err != nil {
		_ = client.Close()
		tb.Fatalf("schema create: %v", err)
	}
	tb.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

// NewSQLDB opens a raw *sql.DB on a throwaway file (used by dbmanage tests that don't
// need ent schema).
func NewSQLDB(tb testing.TB) (*sql.DB, string) {
	tb.Helper()
	dbPath := filepath.Join(tb.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		tb.Fatalf("sql.Open: %v", err)
	}
	tb.Cleanup(func() {
		_ = db.Close()
	})
	return db, dbPath
}
