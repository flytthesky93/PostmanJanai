package dbmanage

import (
	"context"
	"database/sql"
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
	default:
		return nil
	}
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
