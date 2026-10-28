package core

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // 默认 Info 级别，生产环境只记录 Info/Warn/Error
	}
	if os.Getenv("APP_DEBUG") == "true" {
		opts.Level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

// SetLogLevel 设置日志级别
func SetLogLevel(level slog.Level) {
	// 动态调整（实际可通过重新创建 handler 或记录状态控制）
	opts := &slog.HandlerOptions{
		Level: level,
	}
	Logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(Logger)
}
