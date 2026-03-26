# Log Environment Modes Implementation Plan

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 通过编译期注入的环境变量，使开发环境记录 DEBUG 级别日志和完整 request/response body，生产环境仅记录 INFO 级别日志并省略 body。

**Architecture:** 在 logging 包新增 Environment 类型和 IsDev() 函数，Init() 接收 env 参数决定日志级别，api.Client.DoCtx() 根据 IsDev() 条件记录 body。cmd/root.go 新增 ldflags 变量 Environment（默认 "production"）。

**Tech Stack:** Go 1.21, log/slog, go build -ldflags

---

## File Structure

| 操作 | 文件 | 职责 |
|------|------|------|
| Modify | `internal/logging/logging.go` | 新增 Environment 类型、ParseEnvironment()、IsDev()、currentEnv；Init 新增 env 参数 |
| Modify | `internal/logging/logging_test.go` | 新增环境相关测试，更新现有 Init 调用 |
| Modify | `internal/api/client.go` | DoCtx 中 request_body/response_body 改为条件记录 |
| Modify | `internal/api/client_test.go` | body 测试改为 Development 模式，新增 Production 省略 body 测试 |
| Modify | `cmd/root.go` | 新增 Environment 变量，initLogging 传递 env |
| Modify | `.github/workflows/release.yml` | release 构建注入 Environment=production |

---

### Task 1: logging 包新增 Environment 类型和辅助函数

**Files:**
- Modify: `internal/logging/logging.go`
- Test: `internal/logging/logging_test.go`

- [ ] **Step 1: 编写 ParseEnvironment 和 IsDev 的失败测试**

在 `internal/logging/logging_test.go` 末尾添加：

```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/logging/ -run "TestParseEnvironment|TestIsDev_DefaultProduction" -v`
Expected: FAIL，函数未定义

- [ ] **Step 3: 实现 Environment 类型和相关函数**

在 `internal/logging/logging.go` 中，`Init` 函数之前添加：

```go
// Environment 日志环境模式
type Environment int

const (
	Production  Environment = iota // 生产环境：INFO 级别，不记录 body
	Development                    // 开发环境：DEBUG 级别，记录完整 body
)

var currentEnv = Production // 默认生产环境

// ParseEnvironment 将字符串解析为 Environment
// 无效值默认为 Production
func ParseEnvironment(s string) Environment {
	if strings.EqualFold(s, "development") || strings.EqualFold(s, "dev") {
		return Development
	}
	return Production
}

// IsDev 返回当前是否为开发环境
func IsDev() bool {
	return currentEnv == Development
}
```

在 import 中添加 `"strings"`。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/logging/ -run "TestParseEnvironment|TestIsDev_DefaultProduction" -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/logging/logging.go internal/logging/logging_test.go
git commit -m "feat(logging): add Environment type, ParseEnvironment and IsDev"
```

---

### Task 2: 更新 Init 签名，支持环境感知的日志级别

**Files:**
- Modify: `internal/logging/logging.go`
- Modify: `internal/logging/logging_test.go`

- [ ] **Step 1: 编写 Init 环境级别相关测试**

在 `internal/logging/logging_test.go` 末尾添加：

```go
func TestInit_DevLogLevel(t *testing.T) {
	tmpDir := t.TempDir()
	err := Init(false, tmpDir, Development)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, "logs", today+".log")

	// DEBUG 级别日志应被记录
	slog.Debug("debug message")
	slog.Info("info message")

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "debug message") {
		t.Error("dev mode should log DEBUG messages")
	}
	if !strings.Contains(content, "info message") {
		t.Error("dev mode should log INFO messages")
	}
}

func TestInit_ProdLogLevel(t *testing.T) {
	tmpDir := t.TempDir()
	err := Init(false, tmpDir, Production)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	today := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, "logs", today+".log")

	// DEBUG 级别日志不应被记录
	slog.Debug("debug message")
	slog.Info("info message")

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "debug message") {
		t.Error("prod mode should NOT log DEBUG messages")
	}
	if !strings.Contains(content, "info message") {
		t.Error("prod mode should log INFO messages")
	}
}

func TestIsDev_AfterInit(t *testing.T) {
	tmpDir := t.TempDir()
	err := Init(false, tmpDir, Development)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !IsDev() {
		t.Error("IsDev() should return true after Init with Development")
	}
}
```

- [ ] **Step 2: 更新现有测试中 Init 的调用签名**

将 `internal/logging/logging_test.go` 中所有 `Init(false, tmpDir)` 和 `Init(true, tmpDir)` 调用加上第三个参数：

- `TestInit_CreatesLogDir`: `Init(false, tmpDir)` → `Init(false, tmpDir, Production)`
- `TestInit_CreatesLogFile`: `Init(false, tmpDir)` → `Init(false, tmpDir, Production)`
- `TestInit_VerboseMode`: `Init(true, tmpDir)` → `Init(true, tmpDir, Production)`

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/logging/ -v`
Expected: FAIL（Init 签名不匹配 + 新测试未通过）

- [ ] **Step 4: 修改 Init 函数签名和实现**

将 `internal/logging/logging.go:41` 的 Init 函数替换为：

```go
// Init 初始化日志系统
// baseDir 为日志根目录（生产环境传 ~/.ckjr，测试传 t.TempDir()）
// verbose=true 时额外输出到 stderr（不受 env 影响）
// env 控制日志级别：development=DEBUG，production=INFO
func Init(verbose bool, baseDir string, env Environment) error {
	currentEnv = env

	logDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	filename := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	level := slog.LevelInfo
	if env == Development {
		level = slog.LevelDebug
	}

	fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})

	if verbose {
		stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		slog.SetDefault(slog.New(newMultiHandler(fileHandler, stderrHandler)))
	} else {
		slog.SetDefault(slog.New(fileHandler))
	}

	return nil
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/logging/ -v`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add internal/logging/logging.go internal/logging/logging_test.go
git commit -m "feat(logging): Init accepts Environment, sets log level accordingly"
```

---

### Task 3: api.Client 条件记录 request/response body

**Files:**
- Modify: `internal/api/client.go`
- Modify: `internal/api/client_test.go`

- [ ] **Step 1: 编写 Production 模式省略 body 的测试**

在 `internal/api/client_test.go` 的 `TestDoCtx_NilBody_LogsEmptyRequestBody` 之后添加：

```go
func TestDoCtx_ProdOmitsBody(t *testing.T) {
	// 设置 Production 环境
	tmpDir := t.TempDir()
	if err := logging.Init(false, tmpDir, logging.Production); err != nil {
		t.Fatalf("logging.Init: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Data:       map[string]string{"id": "42"},
			Message:    "ok",
			StatusCode: 200,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "prod-omit-001")

	var result map[string]string
	output := captureLog(func() {
		client.DoCtx(ctx, "POST", "/test", map[string]string{"name": "test"}, &result)
	})

	if strings.Contains(output, "request_body") {
		t.Errorf("prod mode should NOT contain request_body, got: %s", output)
	}
	if strings.Contains(output, "response_body") {
		t.Errorf("prod mode should NOT contain response_body, got: %s", output)
	}
}
```

- [ ] **Step 2: 更新现有 body 测试，设置 Development 环境**

在以下 4 个测试函数中，在 `captureLog` 调用之前添加 `logging.Init` 调用：

**TestDoCtx_LogsRequestBody** (第 320 行附近)，在 `output := captureLog(...)` 之前添加：
```go
tmpDir := t.TempDir()
if err := logging.Init(false, tmpDir, logging.Development); err != nil {
	t.Fatalf("logging.Init: %v", err)
}
```

**TestDoCtx_LogsResponseBody** (第 344 行附近)，同样在 `captureLog` 之前添加：
```go
tmpDir := t.TempDir()
if err := logging.Init(false, tmpDir, logging.Development); err != nil {
	t.Fatalf("logging.Init: %v", err)
}
```

**TestDoCtx_LogsResponseBody_OnError** (第 371 行附近)，同样添加：
```go
tmpDir := t.TempDir()
if err := logging.Init(false, tmpDir, logging.Development); err != nil {
	t.Fatalf("logging.Init: %v", err)
}
```

**TestDoCtx_NilBody_LogsEmptyRequestBody** (第 399 行附近)，同样添加：
```go
tmpDir := t.TempDir()
if err := logging.Init(false, tmpDir, logging.Development); err != nil {
	t.Fatalf("logging.Init: %v", err)
}
```

**TestDoCtx_LogsChinese_Readable** (第 456 行附近)，同样添加：
```go
tmpDir := t.TempDir()
if err := logging.Init(false, tmpDir, logging.Development); err != nil {
	t.Fatalf("logging.Init: %v", err)
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/api/ -run "TestDoCtx_ProdOmitsBody" -v`
Expected: FAIL（request_body 和 response_body 仍然出现在日志中）

- [ ] **Step 4: 修改 DoCtx 中的日志记录逻辑**

在 `internal/api/client.go` 中：

1. 将第 112-117 行的 api_request 日志替换为条件记录：
```go
	attrs := []interface{}{
		"request_id", requestID,
		"method", method,
		"url", url,
	}
	if logging.IsDev() {
		attrs = append(attrs, "request_body", string(data))
	}
	slog.InfoContext(ctx, "api_request", attrs...)
```

2. 将所有包含 `"response_body", readableJSON(bodyBytes)` 的日志记录点（共 6 处：第 177、198、213、227、240、256、268 行），统一改为条件判断。每处将 `"response_body", readableJSON(bodyBytes),` 替换为：
```go
		// 注意：response_body 行前的缩进保持一致
```
并在 `)` 之前根据 IsDev() 条件追加 response_body。

具体方案——在每个包含 response_body 的 `slog.ErrorContext` / `slog.InfoContext` 调用前，收集 attrs 再统一传参。为保持改动最小，采用逐处修改策略：

**a) 第 170-179 行** (非 JSON 非 2xx 错误)：
```go
			errAttrs := []interface{}{
				"request_id", requestID,
				"method", method,
				"url", url,
				"status", resp.StatusCode,
				"duration_ms", duration.Milliseconds(),
				"error", respErr.Message,
			}
			if logging.IsDev() {
				errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
			}
			slog.ErrorContext(ctx, "api_response", errAttrs...)
```

**b) 第 191-201 行** (2xx 但非 JSON)：
```go
			errAttrs := []interface{}{
				"request_id", requestID,
				"method", method,
				"url", url,
				"status", resp.StatusCode,
				"duration_ms", duration.Milliseconds(),
				"error", respErr.Message,
			}
			if logging.IsDev() {
				errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
			}
			slog.ErrorContext(ctx, "api_response", errAttrs...)
```

**c) 第 206-216 行** (JSON 解析失败)：
```go
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
```

**d) 第 220-229 行** (401 Unauthorized)：
```go
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", "unauthorized",
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
```

**e) 第 233-246 行** (422 Unprocessable)：
```go
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
```

**f) 第 248-259 行** (>=400 其他错误)：
```go
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
```

**g) 第 262-269 行** (成功日志)：
```go
	respAttrs := []interface{}{
		"request_id", requestID,
		"method", method,
		"url", url,
		"status", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	}
	if logging.IsDev() {
		respAttrs = append(respAttrs, "response_body", readableJSON(bodyBytes))
	}
	slog.InfoContext(ctx, "api_response", respAttrs...)
```

- [ ] **Step 5: 运行 api 包测试**

Run: `go test ./internal/api/ -v`
Expected: 全部 PASS

- [ ] **Step 6: 运行全部测试确认无回归**

Run: `go test ./... -v`
Expected: 全部 PASS（logging 包测试在 Task 2 已通过签名更新）

- [ ] **Step 7: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat(api): conditionally log request/response body based on environment"
```

---

### Task 4: cmd/root.go 接入 Environment ldflags 变量

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: 修改 root.go**

在 `cmd/root.go:21-24` 的 var 块中新增 Environment 变量：

```go
var (
	// 版本信息，通过 ldflags 注入
	Version = "dev"
	// 环境模式，通过 ldflags 注入，可选值：development / production（默认）
	Environment = "production"
)
```

修改 `initLogging` 函数（第 63-74 行）：

```go
func initLogging() {
	verbose, _ := rootCmd.Flags().GetBool("verbose")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
		return
	}
	baseDir := filepath.Join(homeDir, ".ckjr")
	env := logging.ParseEnvironment(Environment)
	if err := logging.Init(verbose, baseDir, env); err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
	}
}
```

- [ ] **Step 2: 编译验证**

Run: `go build ./cmd/ckjr-cli`
Expected: 编译成功

- [ ] **Step 3: 运行全部测试**

Run: `go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/root.go
git commit -m "feat(cmd): add Environment ldflags variable, wire to logging.Init"
```

---

### Task 5: 更新 CI release 构建注入 Environment

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: 更新 release.yml 构建命令**

将第 46 行的 go build 命令：

```
go build -ldflags="-s -w -X main.Version=${VERSION}" -o dist/ckjr-cli_${VERSION}_${GOOS}_${GOARCH}/${BINARY_NAME} ./cmd/ckjr-cli
```

替换为：

```
go build -ldflags="-s -w -X main.Version=${VERSION} -X main.Environment=production" -o dist/ckjr-cli_${VERSION}_${GOOS}_${GOARCH}/${BINARY_NAME} ./cmd/ckjr-cli
```

- [ ] **Step 2: 运行全部测试**

Run: `go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 3: 提交**

```bash
git add .github/workflows/release.yml
git commit -m "ci: inject Environment=production in release builds"
```
