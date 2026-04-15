package dbmanage

import (
	"context"
	"database/sql"
)

// MigrateDataBetweenVersions chạy trước ent.Schema.Create khi user_version DB < constant.DBSchemaUserVersion.
// Thêm case khi bump DBSchemaUserVersion: export/transform SQL/chép bảng tạm giữa DB cũ → file mới (tùy kịch bản phá vỡ).
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
		// → 1: không cần chuyển dữ liệu (schema Ent + Schema.Create xử lý additive)
		return nil
	default:
		return nil
	}
}
