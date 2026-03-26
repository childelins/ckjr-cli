package logging

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestNewRequestID_Format(t *testing.T) {
	id := NewRequestID()
	// UUID v4 格式: 8-4-4-4-12 hex
	pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`
	if !regexp.MustCompile(pattern).MatchString(id) {
		t.Errorf("NewRequestID() = %q, not valid UUID v4", id)
	}
}

func TestNewRequestID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewRequestID()
		if seen[id] {
			t.Fatalf("duplicate requestId: %s", id)
		}
		seen[id] = true
	}
}

func TestWithRequestID_RoundTrip(t *testing.T) {
	ctx := context.Background()
	id := "test-request-id-123"
	ctx = WithRequestID(ctx, id)

	got := RequestIDFrom(ctx)
	if got != id {
		t.Errorf("RequestIDFrom() = %q, want %q", got, id)
	}
}

func TestRequestIDFrom_Empty(t *testing.T) {
	ctx := context.Background()
	got := RequestIDFrom(ctx)
	if got != "" {
		t.Errorf("RequestIDFrom() = %q, want empty", got)
	}
}

func TestInit_CreatesLogDir(t *testing.T) {
	tmpDir := t.TempDir()
	err := Init(false, tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	logDir := filepath.Join(tmpDir, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Init() should create logs directory")
	}
}

func TestInit_CreatesLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	err := Init(false, tmpDir)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, "logs", today+".log")

	// 写一条日志触发文件创建
	slog.Info("test")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Init() should create log file %s", logFile)
	}
}

func TestParseEnvironment(t *testing.T) {
	tests := []struct {
		input string
		want  Environment
	}{
		{"development", Development},
		{"dev", Development},
		{"Development", Development},
		{"DEV", Development},
		{"production", Production},
		{"prod", Production},
		{"Production", Production},
		{"", Production},
		{"invalid", Production},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseEnvironment(tt.input)
			if got != tt.want {
				t.Errorf("ParseEnvironment(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsDev_DefaultProduction(t *testing.T) {
	// 未调用 Init 时 currentEnv 为默认值 Production
	if IsDev() {
		t.Error("IsDev() should return false by default")
	}
}

func TestInit_VerboseMode(t *testing.T) {
	tmpDir := t.TempDir()
	err := Init(true, tmpDir)
	if err != nil {
		t.Fatalf("Init(verbose=true) error = %v", err)
	}

	// 验证日志文件被创建
	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, "logs", today+".log")

	slog.Info("verbose test")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Init(verbose) should still create log file %s", logFile)
	}
}
