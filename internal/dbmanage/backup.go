package dbmanage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"PostmanJanai/internal/pkg/logger"
)

// BackupDatabaseIfNonEmpty copy file DB sang <appDir>/backups/postmanjanai_db_YYYYMMDD_HHMMSS.db khi file tồn tại và size > 0.
func BackupDatabaseIfNonEmpty(srcPath, appDir string) (backupPath string, err error) {
	st, err := os.Stat(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if !st.IsDir() && st.Size() == 0 {
		return "", nil
	}
	backupDir := filepath.Join(appDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	ts := time.Now().Format("20060102_150405")
	name := fmt.Sprintf("postmanjanai_db_%s.db", ts)
	dst := filepath.Join(backupDir, name)
	if err := copyFile(srcPath, dst); err != nil {
		return "", err
	}
	logger.L().Info("database backup created", "path", dst)
	return dst, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
