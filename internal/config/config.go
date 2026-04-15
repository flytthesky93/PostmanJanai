package config

import (
	"PostmanJanai/internal/constant"
	"PostmanJanai/internal/pkg/logger"
	"os"
	"path/filepath"
	"sync"
)

var (
	once     sync.Once
	instance *Config
)

type Config struct {
	AppDir string
	DbPath string
}

const (
	envAppDir = "POSTMANJANAI_APP_DIR"
	envDBPath = "POSTMANJANAI_DB_PATH"
)

func resolvePath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	exePath, err := os.Executable()
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(exePath), path))
}

func LoadConfig() {
	once.Do(func() {
		appDir := resolvePath(os.Getenv(envAppDir))
		if appDir == "" {
			// Lấy đường dẫn thư mục AppData/Local (Windows) hoặc .local/share (Linux)
			userConfigDir, err := os.UserConfigDir()
			if err != nil {
				logger.L().Error("Loading Config failed", "error", err)
				panic("Loading Config failed: " + err.Error())
			}
			appDir = filepath.Join(userConfigDir, constant.AppName)
		}

		// Tạo thư mục nếu chưa tồn tại
		if err := os.MkdirAll(appDir, 0755); err != nil {
			logger.L().Error("Creating AppDir failed", "error", err)
			panic("Creating AppDir failed: " + err.Error())
		}

		dbPath := resolvePath(os.Getenv(envDBPath))
		if dbPath == "" {
			dbPath = filepath.Join(appDir, constant.AppDbName)
		}

		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			logger.L().Error("Creating DbDir failed", "error", err)
			panic("Creating DbDir failed: " + err.Error())
		}

		instance = &Config{
			AppDir: appDir,
			DbPath: dbPath,
		}
	})
}

func GetConfig() *Config {
	return instance
}
