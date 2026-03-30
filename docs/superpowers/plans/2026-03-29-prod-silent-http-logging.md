# 生产环境静默 HTTP 请求日志实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 生产环境日志文件只记录 ERROR 级别，`--verbose` stderr 统一输出 INFO 级别（不受 env 影响）

**Architecture:** 将 `logging.go` 中生产环境的 file handler 级别从 `slog.LevelInfo` 改为 `slog.LevelError`；verbose stderr handler 使用独立的 `slog.LevelInfo`，不再共享 file handler 的 level 变量。

**Tech Stack:** Go, log/slog

---

### Task 1: 修改生产环境日志级别并分离 verbose handler

**Files:**
- Modify: `internal/logging/logging.go:42-43,65-66,81,89`
- Test: `internal/logging/logging_test.go:161-186`

- [ ] **Step 1: 更新 TestInit_ProdLogLevel 测试 — 生产环境文件只记录 ERROR**

修改 `internal/logging/logging_test.go` 中的 `TestInit_ProdLogLevel`：

```go
func TestInit_ProdLogLevel(t *testing.T) {
	tmpDir := t.TempDir()
	err := Init(false, tmpDir, Production)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, "logs", today+".log")

	// 生产环境文件只记录 ERROR
	slog.Debug("debug message")
	slog.Info("info message")
	slog.Error("error message")

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "debug message") {
		t.Error("prod mode should NOT log DEBUG messages")
	}
	if strings.Contains(content, "info message") {
		t.Error("prod mode should NOT log INFO messages")
	}
	if !strings.Contains(content, "error message") {
		t.Error("prod mode should log ERROR messages")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/logging/ -run TestInit_ProdLogLevel -v`
Expected: FAIL — "prod mode should NOT log INFO messages"

- [ ] **Step 3: 修改 logging.go — 生产环境级别改为 ERROR + verbose 独立级别**

修改 `internal/logging/logging.go`：

1. 更新常量注释（第42-43行）：
```go
const (
	Production  Environment = iota // 生产环境：ERROR 级别，不记录 body
	Development                    // 开发环境：DEBUG 级别，记录完整 body
)
```

2. 更新 Init 函数注释（第65-66行）：
```go
// env 控制日志文件级别：development=DEBUG，production=ERROR
```

3. 修改 level 赋值（第81行）：
```go
level := slog.LevelError
```

4. 修改 verbose stderr handler（第89行），使用独立级别：
```go
if verbose {
	stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(newMultiHandler(fileHandler, stderrHandler)))
} else {
	slog.SetDefault(slog.New(fileHandler))
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/logging/ -run TestInit_ProdLogLevel -v`
Expected: PASS

- [ ] **Step 5: 运行全部 logging 测试确认无回归**

Run: `go test ./internal/logging/ -v`
Expected: 全部 PASS

- [ ] **Step 6: 运行全部项目测试确认无回归**

Run: `go test ./...`
Expected: 全部 PASS

- [ ] **Step 7: Commit**

```bash
git add internal/logging/logging.go internal/logging/logging_test.go
git commit -m "feat(logging): 生产环境日志级别提升至ERROR，verbose stderr独立INFO级别"
```
