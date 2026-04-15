package dbmanage

import (
	"database/sql"
	"fmt"
)

// UserVersion đọc SQLite PRAGMA user_version (0 nếu DB mới).
func UserVersion(db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}

// SetUserVersion ghi PRAGMA user_version sau khi Ent schema và dữ liệu đã khớp mục tiêu.
func SetUserVersion(db *sql.DB, v int) error {
	if v < 0 {
		return fmt.Errorf("invalid user_version %d", v)
	}
	_, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", v))
	return err
}
