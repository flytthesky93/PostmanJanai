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

func LoadConfig() {
	once.Do(func() {
		// Lấy đường dẫn thư mục AppData/Local (Windows) hoặc .local/share (Linux)
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			logger.L().Error("Loading Config failed: ", err)
			panic("Loading Config failed: " + err.Error())
		}

		appDir := filepath.Join(userConfigDir, constant.AppName)

		// Tạo thư mục nếu chưa tồn tại
		if _, err = os.Stat(appDir); os.IsNotExist(err) {
			err = os.MkdirAll(appDir, 0755)
			if err != nil {
				logger.L().Error("Creating AppDir failed: ", err)
				panic("Creating AppDir failed: " + err.Error())
			}
		}

		instance = &Config{
			AppDir: appDir,
			DbPath: filepath.Join(appDir, constant.AppDbName),
		}
	})
}

func GetConfig() *Config {
	return instance
}
