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
	default:
		return nil
	}
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
