package dbmanage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupDatabase_SkipsMissingFile(t *testing.T) {
	appDir := t.TempDir()
	src := filepath.Join(appDir, "does-not-exist.db")
	path, err := BackupDatabaseIfNonEmpty(src, appDir)
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if path != "" {
		t.Fatalf("no backup expected, got path %q", path)
	}
}

func TestBackupDatabase_SkipsEmptyFile(t *testing.T) {
	appDir := t.TempDir()
	src := filepath.Join(appDir, "empty.db")
	if err := os.WriteFile(src, nil, 0644); err != nil {
		t.Fatalf("prep empty: %v", err)
	}
	path, err := BackupDatabaseIfNonEmpty(src, appDir)
	if err != nil {
		t.Fatalf("empty file should not error, got %v", err)
	}
	if path != "" {
		t.Fatalf("no backup expected, got %q", path)
	}
}

func TestBackupDatabase_CopiesNonEmptyFile(t *testing.T) {
	appDir := t.TempDir()
	src := filepath.Join(appDir, "db.db")
	content := []byte("SQLite format 3\x00fake body")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("prep: %v", err)
	}

	path, err := BackupDatabaseIfNonEmpty(src, appDir)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty backup path")
	}
	if !strings.HasPrefix(path, filepath.Join(appDir, "backups")) {
		t.Fatalf("backup should live under <appDir>/backups, got %q", path)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("backup content mismatch")
	}
}
