package logger

import (
	"PostmanJanai/internal/constant"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/natefinch/lumberjack"
)

var (
	once          sync.Once
	instance      *slog.Logger
	debugInstance *slog.Logger
)

const (
	logMaxSizeMB    = 5
	logMaxBackups   = 5
	logMaxAgeDays   = 30
	logCompressFile = true
)

// Init khởi tạo logger duy nhất một lần.
// Bạn nên gọi hàm này trong main.go khi bắt đầu chạy app.
func Init(appPath string) {
	logPath := filepath.Join(appPath, constant.LogPath)
	debugLogPath := filepath.Join(appPath, constant.DebugLogPath)
	once.Do(func() {
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			panic("Creating log directory failed: " + err.Error())
		}

		appLogWriter := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    logMaxSizeMB,
			MaxBackups: logMaxBackups,
			MaxAge:     logMaxAgeDays,
			Compress:   logCompressFile,
		}

		debugLogWriter := &lumberjack.Logger{
			Filename:   debugLogPath,
			MaxSize:    logMaxSizeMB,
			MaxBackups: logMaxBackups,
			MaxAge:     logMaxAgeDays,
			Compress:   logCompressFile,
		}

		instance = slog.New(slog.NewJSONHandler(appLogWriter, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		debugInstance = slog.New(slog.NewJSONHandler(debugLogWriter, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		slog.SetDefault(instance) // Đặt làm logger mặc định của hệ thống
	})
}

// L lấy ra instance của logger
func L() *slog.Logger {
	if instance == nil {
		// Nếu chưa init mà đã gọi L(), trả về một logger mặc định để tránh crash
		return slog.Default()
	}
	return instance
}

// D lấy ra debug logger để ghi trace/diagnostic chi tiết.
func D() *slog.Logger {
	if debugInstance == nil {
		return slog.Default()
	}
	return debugInstance
}
