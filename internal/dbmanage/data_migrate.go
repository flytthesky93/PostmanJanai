package dbmanage

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

// MigrateDataBetweenVersions chạy trước ent.Schema.Create khi user_version DB < constant.DBSchemaUserVersion.
func MigrateDataBetweenVersions(ctx context.Context, db *sql.DB, from, to int) error {
	_ = ctx
	if from >= to {
		return nil
	}
	for v := from; v < to; v++ {
		if err := migrateOneStep(db, v, v+1); err != nil {
			return err
		}
	}
	return nil
}

func migrateOneStep(db *sql.DB, from, to int) error {
	switch from {
	case 0:
		// → 1: legacy; DB mới dùng Schema.Create
		return nil
	case 1:
		// → 2: schema Ent chuyển từ int PK sang UUID + bảng mới; DB cũ (chỉ workspaces/histories kiểu cũ) cần drop để Schema.Create tạo lại
		return dropLegacyTablesForUUIDSchema(db)
	case 2:
		// → 3: workspace + collection → nested folders + requests under folder_id; drop UUID schema để Ent tạo lại
		return dropLegacyTablesForUUIDSchema(db)
	case 3:
		// → 4: additive `requests.auth_json` (Ent Schema.Create)
		return nil
	case 4:
		// → 5: `folders.sort_order` for manual ordering in the sidebar tree
		if _, err := db.Exec(`ALTER TABLE folders ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		return backfillFolderSortOrder(db)
	case 5:
		// → 6: Phase 6 — Networking & Security (additive columns only; new tables come from Ent Schema.Create)
		if _, err := db.Exec(`ALTER TABLE environment_variables ADD COLUMN kind TEXT NOT NULL DEFAULT 'plain'`); err != nil {
			return err
		}
		if _, err := db.Exec(`ALTER TABLE requests ADD COLUMN insecure_skip_verify INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
		return nil
	case 6:
		// → 7: Phase 8 — Collection Runner & Chaining.
		// All deltas are additive new tables (request_captures, request_assertions,
		// runner_runs, runner_run_requests). Ent's Schema.Create handles their creation
		// after this migration step returns; no destructive SQL is required.
		return nil
	case 7:
		// → 8: Phase 8.1 — persist resolved request snapshot + response payload
		// per runner request so users can inspect what was actually sent and
		// received without re-running the request. Additive columns on
		// runner_run_requests; safe to run on existing databases.
		alters := []string{
			`ALTER TABLE runner_run_requests ADD COLUMN request_headers_json TEXT`,
			`ALTER TABLE runner_run_requests ADD COLUMN response_headers_json TEXT`,
			`ALTER TABLE runner_run_requests ADD COLUMN request_body TEXT`,
			`ALTER TABLE runner_run_requests ADD COLUMN response_body TEXT`,
			`ALTER TABLE runner_run_requests ADD COLUMN body_truncated INTEGER NOT NULL DEFAULT 0`,
		}
		for _, q := range alters {
			if _, err := db.Exec(q); err != nil {
				// SQLite doesn't ship `IF NOT EXISTS` for ADD COLUMN until 3.35; treat
				// "duplicate column" errors as a no-op so re-running the migration is safe.
				if !isDuplicateColumnErr(err) {
					return err
				}
			}
		}
		return nil
	default:
		return nil
	}
}

func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}

type folderSortRow struct {
	id       string
	parentID sql.NullString
	name     string
}

func backfillFolderSortOrder(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, parent_id, name FROM folders`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var list []folderSortRow
	for rows.Next() {
		var r folderSortRow
		if err := rows.Scan(&r.id, &r.parentID, &r.name); err != nil {
			return err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	byParent := make(map[string][]folderSortRow)
	for _, r := range list {
		key := ""
		if r.parentID.Valid {
			key = strings.TrimSpace(r.parentID.String)
		}
		byParent[key] = append(byParent[key], r)
	}
	for _, group := range byParent {
		sort.Slice(group, func(i, j int) bool {
			return strings.ToLower(group[i].name) < strings.ToLower(group[j].name)
		})
		for i, r := range group {
			if _, err := db.Exec(`UPDATE folders SET sort_order = ? WHERE id = ?`, i, r.id); err != nil {
				return err
			}
		}
	}
	return nil
}

func dropLegacyTablesForUUIDSchema(db *sql.DB) error {
	// Thứ tự: bảng phụ / FK trước (an toàn với IF EXISTS)
	stmts := []string{
		`DROP TABLE IF EXISTS settings`,
		`DROP TABLE IF EXISTS trusted_cas`,
		`DROP TABLE IF EXISTS environment_variables`,
		`DROP TABLE IF EXISTS environments`,
		`DROP TABLE IF EXISTS request_form_fields`,
		`DROP TABLE IF EXISTS request_query_params`,
		`DROP TABLE IF EXISTS request_headers`,
		`DROP TABLE IF EXISTS histories`,
		`DROP TABLE IF EXISTS requests`,
		`DROP TABLE IF EXISTS collections`,
		`DROP TABLE IF EXISTS workspaces`,
		`DROP TABLE IF EXISTS folders`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
