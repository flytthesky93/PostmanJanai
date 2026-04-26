package dbmanage

import (
	"context"
	"testing"

	"PostmanJanai/internal/testutil"
)

// TestMigrate_NoopWhenFromEqualsTo ensures the migration driver is idempotent for same-version DBs.
func TestMigrate_NoopWhenFromEqualsTo(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)
	if err := MigrateDataBetweenVersions(context.Background(), db, 8, 8); err != nil {
		t.Fatalf("noop should not error, got %v", err)
	}
}

// TestMigrate_6to7IsAdditive verifies v6→v7 (Phase 8) is a no-op for legacy data:
// the new runner / capture / assertion tables are created later by ent.Schema.Create,
// not in this migration step.
func TestMigrate_6to7IsAdditive(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)
	if _, err := db.Exec(`CREATE TABLE requests (id TEXT PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO requests (id, name) VALUES ('r1', 'x')`); err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	if err := MigrateDataBetweenVersions(context.Background(), db, 6, 7); err != nil {
		t.Fatalf("migrate 6→7: %v", err)
	}
	row := db.QueryRow(`SELECT COUNT(*) FROM requests`)
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("count requests: %v", err)
	}
	if n != 1 {
		t.Fatalf("requests row count = %d, want 1 (additive migration must not touch existing tables)", n)
	}
}

// TestMigrate_7to8AddsRunnerRequestSnapshots verifies v7→v8 (Phase 8.1) adds
// the request/response snapshot columns to runner_run_requests as additive,
// idempotent ALTER TABLE statements.
func TestMigrate_7to8AddsRunnerRequestSnapshots(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)
	stmts := []string{
		`CREATE TABLE runner_run_requests (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			request_id TEXT NULL,
			request_name TEXT NOT NULL DEFAULT '',
			method TEXT NOT NULL DEFAULT 'GET',
			url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'passed',
			status_code INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			response_size_bytes INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			assertions_json TEXT NOT NULL DEFAULT '',
			captures_json TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO runner_run_requests (id, run_id, request_name) VALUES ('r1', 'run1', 'seed')`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}

	if err := MigrateDataBetweenVersions(context.Background(), db, 7, 8); err != nil {
		t.Fatalf("migrate 7→8: %v", err)
	}

	// Re-running must not error (idempotent ALTERs).
	if err := MigrateDataBetweenVersions(context.Background(), db, 7, 8); err != nil {
		t.Fatalf("migrate 7→8 (rerun): %v", err)
	}

	cols := map[string]bool{}
	rows, err := db.Query(`PRAGMA table_info(runner_run_requests)`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid                int
			name, ctype        string
			notnull, pk        int
			dflt               *string
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		cols[name] = true
	}
	for _, c := range []string{"request_headers_json", "response_headers_json", "request_body", "response_body", "body_truncated"} {
		if !cols[c] {
			t.Errorf("expected column %q after migration", c)
		}
	}

	row := db.QueryRow(`SELECT COUNT(*) FROM runner_run_requests`)
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("rows = %d, want 1 (additive migration must keep rows)", n)
	}
}

// TestMigrate_6to8Chain runs the full v6 → v7 → v8 chain on a v6-shape DB
// (Phase 8 left some tables to ent.Schema.Create, so we recreate the v7 shape
// of `runner_run_requests` here). This is a smoke test to catch regressions
// where a future step accidentally depends on side-effects of an earlier one.
func TestMigrate_6to8Chain(t *testing.T) {
	db, _ := testutil.NewSQLDB(t)

	// Seed v6-era tables we still touch + v7 shape of runner_run_requests so
	// the v7→v8 ALTERs find their target. Phase 8 didn't ALTER any table in
	// the v6→v7 step, but downstream code (incl. the v7→v8 step) needs the
	// runner table present.
	stmts := []string{
		`CREATE TABLE folders (
			id TEXT PRIMARY KEY,
			parent_id TEXT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO folders (id, name) VALUES ('f1', 'root')`,
		`CREATE TABLE runner_run_requests (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			request_name TEXT NOT NULL DEFAULT '',
			method TEXT NOT NULL DEFAULT 'GET',
			url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'passed',
			status_code INTEGER NOT NULL DEFAULT 0,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			response_size_bytes INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			assertions_json TEXT NOT NULL DEFAULT '',
			captures_json TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO runner_run_requests (id, run_id) VALUES ('rr1', 'run1')`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}

	if err := MigrateDataBetweenVersions(context.Background(), db, 6, 8); err != nil {
		t.Fatalf("migrate 6→8: %v", err)
	}

	// Pre-existing data must survive both steps.
	if row, n := db.QueryRow(`SELECT COUNT(*) FROM folders`), 0; row.Scan(&n) == nil && n != 1 {
		t.Errorf("folders rows = %d, want 1", n)
	}
	row := db.QueryRow(`SELECT COUNT(*) FROM runner_run_requests`)
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("runner_run_requests rows = %d, want 1", n)
	}

	// v8 columns must exist.
	cols := map[string]bool{}
	rows, err := db.Query(`PRAGMA table_info(runner_run_requests)`)
	if err != nil {
		t.Fatalf("pragma: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid         int
			name, ctype string
			notnull, pk int
			dflt        *string
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		cols[name] = true
	}
	for _, c := range []string{"request_headers_json", "response_headers_json", "request_body", "response_body", "body_truncated"} {
		if !cols[c] {
			t.Errorf("expected column %q after v6→v8 chain", c)
		}
	}

	// Re-running the chain should be a no-op (idempotent).
	if err := MigrateDataBetweenVersions(context.Background(), db, 6, 8); err != nil {
		t.Fatalf("migrate 6→8 (rerun): %v", err)
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

