package logger

import (
	"PostmanJanai/internal/constant"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/natefinch/lumberjack"
)

var (
	once     sync.Once
	instance *slog.Logger
)

// Init khởi tạo logger duy nhất một lần.
// Bạn nên gọi hàm này trong main.go khi bắt đầu chạy app.
func Init(appPath string) {
	logPath := appPath + "/" + constant.LogPath
	once.Do(func() {
		lumberjackLogger := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    5, // megabytes
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		}

		// Ghi ra cả terminal và file
		multiWriter := io.MultiWriter(os.Stdout, lumberjackLogger)

		instance = slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
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
