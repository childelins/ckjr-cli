package logging

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type ctxKey struct{}

// NewRequestID 生成 UUID v4
func NewRequestID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 1
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// WithRequestID 将 requestId 注入 context
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// RequestIDFrom 从 context 提取 requestId
func RequestIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKey{}).(string); ok {
		return id
	}
	return ""
}

// Init 初始化日志系统
// baseDir 为日志根目录（生产环境传 ~/.ckjr，测试传 t.TempDir()）
// verbose=true 时额外输出到 stderr
func Init(verbose bool, baseDir string) error {
	logDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	filename := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})

	if verbose {
		stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
		slog.SetDefault(slog.New(newMultiHandler(fileHandler, stderrHandler)))
	} else {
		slog.SetDefault(slog.New(fileHandler))
	}

	return nil
}
