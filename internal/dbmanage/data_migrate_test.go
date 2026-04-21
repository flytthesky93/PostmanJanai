package dbmanage

import (
	"context"
	"testing"

	"PostmanJanai/internal/testutil"
)

// TestMigrate_NoopWhenFromEqualsTo ensures the migration driver is idempotent for same-version DBs.
func TestMigrate_NoopWhenFromEqualsTo(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)
	if err := MigrateDataBetweenVersions(context.Background(), db, 6, 6); err != nil {
		t.Fatalf("noop should not error, got %v", err)
	}
}

// TestMigrate_4to5AddsSortOrderAndBackfill verifies the v4→v5 migration step:
// adds the column, then backfills sort_order per-parent alphabetically.
func TestMigrate_4to5AddsSortOrderAndBackfill(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)

	// Recreate the v4 shape of the `folders` table — no sort_order column yet.
	stmts := []string{
		`CREATE TABLE folders (
			id TEXT PRIMARY KEY,
			parent_id TEXT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// 2 roots (Zebra + Alpha) should be ordered Alpha=0, Zebra=1.
		`INSERT INTO folders (id, parent_id, name) VALUES ('r1', NULL, 'Zebra')`,
		`INSERT INTO folders (id, parent_id, name) VALUES ('r2', NULL, 'Alpha')`,
		// Children of r1: C, A, B → A=0, B=1, C=2.
		`INSERT INTO folders (id, parent_id, name) VALUES ('c1', 'r1', 'C')`,
		`INSERT INTO folders (id, parent_id, name) VALUES ('c2', 'r1', 'A')`,
		`INSERT INTO folders (id, parent_id, name) VALUES ('c3', 'r1', 'B')`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}

	if err := MigrateDataBetweenVersions(context.Background(), db, 4, 5); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	got := make(map[string]int)
	rows, err := db.Query(`SELECT id, sort_order FROM folders`)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var order int
		if err := rows.Scan(&id, &order); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[id] = order
	}

	want := map[string]int{
		"r2": 0, // Alpha (root)
		"r1": 1, // Zebra (root)
		"c2": 0, // A
		"c3": 1, // B
		"c1": 2, // C
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("sort_order(%s) = %d, want %d", id, got[id], w)
		}
	}
}

// TestMigrate_5to6AddsKindAndInsecureSkipVerify verifies v5→v6 adds additive columns only.
func TestMigrate_5to6AddsKindAndInsecureSkipVerify(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)

	stmts := []string{
		`CREATE TABLE environment_variables (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO environment_variables (id, environment_id, key, value) VALUES ('v1', 'e1', 'k', 'abc')`,
		`CREATE TABLE requests (
			id TEXT PRIMARY KEY,
			folder_id TEXT NOT NULL,
			name TEXT NOT NULL,
			method TEXT NOT NULL DEFAULT 'GET',
			url TEXT NOT NULL,
			body_mode TEXT NOT NULL,
			raw_body TEXT,
			auth_json TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO requests (id, folder_id, name, url, body_mode) VALUES ('r1', 'f1', 'x', 'https://example.com', 'none')`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}

	if err := MigrateDataBetweenVersions(context.Background(), db, 5, 6); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	row := db.QueryRow(`SELECT kind FROM environment_variables WHERE id = 'v1'`)
	var kind string
	if err := row.Scan(&kind); err != nil {
		t.Fatalf("scan kind: %v", err)
	}
	if kind != "plain" {
		t.Fatalf("kind = %q, want plain", kind)
	}

	row2 := db.QueryRow(`SELECT insecure_skip_verify FROM requests WHERE id = 'r1'`)
	var inv int
	if err := row2.Scan(&inv); err != nil {
		t.Fatalf("scan insecure_skip_verify: %v", err)
	}
	if inv != 0 {
		t.Fatalf("insecure_skip_verify = %d, want 0", inv)
	}
}

// TestMigrate_1to2DropsLegacyTables — exercises the drop-for-UUID-schema branch so we catch
// regressions in the statement list.
func TestMigrate_1to2DropsLegacyTables(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)

	// Seed with legacy table shapes we want to wipe.
	stmts := []string{
		`CREATE TABLE workspaces (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE histories (id INTEGER PRIMARY KEY, url TEXT)`,
		`CREATE TABLE folders (id TEXT PRIMARY KEY, name TEXT)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}

	if err := MigrateDataBetweenVersions(context.Background(), db, 1, 2); err != nil {
		t.Fatalf("migrate 1→2: %v", err)
	}

	// All three must be gone.
	for _, name := range []string{"workspaces", "histories", "folders"} {
		row := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name)
		var n int
		if err := row.Scan(&n); err != nil {
			t.Fatalf("probe %s: %v", name, err)
		}
		if n != 0 {
			t.Errorf("table %q still present after drop", name)
		}
	}
}

