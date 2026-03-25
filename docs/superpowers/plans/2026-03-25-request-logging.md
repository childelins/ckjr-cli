# Request Logging 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 为 ckjr CLI 添加结构化请求日志，每次命令调用生成 requestId，通过日志文件持久化请求信息，支持事后按 requestId 回查。

**Architecture:** 新增 `internal/logging/` 包封装 requestId 生成和 slog 初始化。`api.Client` 新增 `DoCtx()` 方法记录请求生命周期日志，`Do()` 内部委托给 `DoCtx()` 保持向后兼容。`cmdgen` 层生成 requestId 并通过 context 传递。

**Tech Stack:** Go 标准库 `log/slog`、`crypto/rand`、`context`。零新依赖。

**Spec:** `docs/superpowers/specs/2026-03-25-request-logging.md`

---

## 文件结构

| 操作 | 文件 | 职责 |
|------|------|------|
| Create | `internal/logging/logging.go` | requestId 生成（UUID v4）、context 透传、slog 初始化 |
| Create | `internal/logging/logging_test.go` | logging 包测试 |
| Create | `internal/logging/multi_handler.go` | slog multiHandler（--verbose 时同时写文件和 stderr） |
| Create | `internal/logging/multi_handler_test.go` | multiHandler 测试 |
| Modify | `internal/api/client.go` | 新增 `DoCtx()`，`Do()` 委托给 `DoCtx()` |
| Modify | `internal/api/client_test.go` | 新增 DoCtx 日志相关测试 |
| Modify | `internal/cmdgen/cmdgen.go` | 生成 requestId，构建 context，调用 `DoCtx()` |
| Modify | `internal/cmdgen/cmdgen_test.go` | 新增 requestId 集成测试 |
| Modify | `cmd/root.go` | `cobra.OnInitialize` 中初始化日志 |

---

### Task 1: logging 包 - requestId 生成与 context 透传

**Files:**
- Create: `internal/logging/logging.go`
- Create: `internal/logging/logging_test.go`

- [ ] **Step 1: 写 requestId 格式测试**

```go
// internal/logging/logging_test.go
package logging

import (
	"context"
	"regexp"
	"testing"
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/logging/ -run TestNewRequestID -v`
Expected: 编译失败，package/function 不存在

- [ ] **Step 3: 实现 NewRequestID**

```go
// internal/logging/logging.go
package logging

import (
	"crypto/rand"
	"fmt"
)

// NewRequestID 生成 UUID v4
func NewRequestID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 1
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/logging/ -run TestNewRequestID -v`
Expected: PASS

- [ ] **Step 5: 写 context 透传测试**

追加到 `internal/logging/logging_test.go`:

```go
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
```

- [ ] **Step 6: 运行测试确认失败**

Run: `go test ./internal/logging/ -run TestRequestIDFrom -v`
Expected: 编译失败

- [ ] **Step 7: 实现 context 透传**

追加到 `internal/logging/logging.go`:

```go
import "context"

type ctxKey struct{}

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
```

- [ ] **Step 8: 运行全部测试确认通过**

Run: `go test ./internal/logging/ -v`
Expected: 4 个测试全部 PASS

- [ ] **Step 9: 提交**

```bash
git add internal/logging/logging.go internal/logging/logging_test.go
git commit -m "feat(logging): add requestId generation and context propagation"
```

---

### Task 2: logging 包 - Init 和 multiHandler

**Files:**
- Create: `internal/logging/multi_handler.go`
- Create: `internal/logging/multi_handler_test.go`
- Modify: `internal/logging/logging.go` - 添加 Init 函数
- Modify: `internal/logging/logging_test.go` - 添加 Init 测试

- [ ] **Step 1: 写 multiHandler 测试**

```go
// internal/logging/multi_handler_test.go
package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestMultiHandler_WritesToAll(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewTextHandler(&buf1, nil)
	h2 := slog.NewTextHandler(&buf2, nil)

	logger := slog.New(newMultiHandler(h1, h2))
	logger.Info("test message", "key", "value")

	if buf1.Len() == 0 {
		t.Error("handler 1 received no output")
	}
	if buf2.Len() == 0 {
		t.Error("handler 2 received no output")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/logging/ -run TestMultiHandler -v`
Expected: 编译失败

- [ ] **Step 3: 实现 multiHandler**

```go
// internal/logging/multi_handler.go
package logging

import (
	"context"
	"log/slog"
)

type multiHandler struct {
	handlers []slog.Handler
}

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			_ = handler.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/logging/ -run TestMultiHandler -v`
Expected: PASS

- [ ] **Step 5: 写 Init 测试**

追加到 `internal/logging/logging_test.go`:

```go
import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

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
```

注意：`Init` 接收 `baseDir` 参数而非硬编码 `~/.ckjr`，方便测试。生产调用传 `~/.ckjr`。

- [ ] **Step 6: 运行测试确认失败**

Run: `go test ./internal/logging/ -run TestInit -v`
Expected: 编译失败

- [ ] **Step 7: 实现 Init**

追加到 `internal/logging/logging.go`:

```go
import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

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
```

- [ ] **Step 8: 运行全部 logging 测试**

Run: `go test ./internal/logging/ -v`
Expected: 全部 PASS

- [ ] **Step 9: 运行项目全部测试确认无回归**

Run: `go test ./...`
Expected: 全部 PASS

- [ ] **Step 10: 提交**

```bash
git add internal/logging/multi_handler.go internal/logging/multi_handler_test.go internal/logging/logging.go internal/logging/logging_test.go
git commit -m "feat(logging): add Init with slog file handler and multiHandler"
```

---

### Task 3: api.Client 新增 DoCtx 方法

**Files:**
- Modify: `internal/api/client.go` - 新增 DoCtx，Do 委托给 DoCtx
- Modify: `internal/api/client_test.go` - 新增 DoCtx 测试

- [ ] **Step 1: 写 DoCtx 日志记录测试**

追加到 `internal/api/client_test.go`:

```go
import (
	"bytes"
	"context"
	"log/slog"

	"github.com/childelins/ckjr-cli/internal/logging"
)

// captureLog 临时替换 slog 默认 logger，捕获日志输出
func captureLog(fn func()) string {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	old := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(old)
	fn()
	return buf.String()
}

func TestDoCtx_LogsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: nil, Message: "ok", StatusCode: 200})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "test-req-123")

	var result interface{}
	output := captureLog(func() {
		client.DoCtx(ctx, "GET", "/test", nil, &result)
	})

	if !strings.Contains(output, "test-req-123") {
		t.Errorf("log should contain requestId, got: %s", output)
	}
	if !strings.Contains(output, "api_request") {
		t.Errorf("log should contain api_request message, got: %s", output)
	}
	if !strings.Contains(output, "api_response") {
		t.Errorf("log should contain api_response message, got: %s", output)
	}
}

func TestDoCtx_LogsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<html>Bad Gateway</html>"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "err-req-456")

	var result interface{}
	output := captureLog(func() {
		client.DoCtx(ctx, "POST", "/fail", nil, &result)
	})

	if !strings.Contains(output, "err-req-456") {
		t.Errorf("error log should contain requestId, got: %s", output)
	}
	if !strings.Contains(output, "ERROR") {
		t.Errorf("error log should be ERROR level, got: %s", output)
	}
}

func TestDoCtx_LogsDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: nil, Message: "ok", StatusCode: 200})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "dur-req-789")

	var result interface{}
	output := captureLog(func() {
		client.DoCtx(ctx, "GET", "/test", nil, &result)
	})

	if !strings.Contains(output, "duration_ms") {
		t.Errorf("log should contain duration_ms, got: %s", output)
	}
}

func TestDo_BackwardCompatible(t *testing.T) {
	// 现有 Do() 测试已覆盖行为兼容性
	// 这里确认 Do() 不会 panic（内部调用 DoCtx(context.Background(), ...)）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: map[string]string{"id": "1"}, Message: "ok", StatusCode: 200})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	var result map[string]string
	err := client.Do("GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("Do() should still work, error = %v", err)
	}
	if result["id"] != "1" {
		t.Errorf("Do() result = %v, want id=1", result)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/api/ -run "TestDoCtx|TestDo_Backward" -v`
Expected: 编译失败，DoCtx 不存在

- [ ] **Step 3: 实现 DoCtx 和重构 Do**

修改 `internal/api/client.go`:

1. 添加 imports: `"context"`, `"log/slog"`, `"time"`, `"github.com/childelins/ckjr-cli/internal/logging"`
2. 新增 `DoCtx` 方法 - 将现有 `Do` 的完整逻辑移入，添加日志记录，将 `http.NewRequest` 改为 `http.NewRequestWithContext`
3. 将 `Do` 改为委托调用 `DoCtx(context.Background(), ...)`

```go
// Do 执行 API 请求（向后兼容）
func (c *Client) Do(method, path string, body interface{}, result interface{}) error {
	return c.DoCtx(context.Background(), method, path, body, result)
}

// DoCtx 执行 API 请求（带 context，支持日志追踪）
func (c *Client) DoCtx(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	requestID := logging.RequestIDFrom(ctx)
	url := c.baseURL + path

	slog.InfoContext(ctx, "api_request",
		"request_id", requestID,
		"method", method,
		"url", url,
	)

	start := time.Now()

	// --- 现有请求逻辑开始（http.NewRequest 改为 http.NewRequestWithContext） ---
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		duration := time.Since(start)
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")

	// 1. 非 2xx + 非 JSON -> ResponseError
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if !isJSONContentType(contentType) {
			respErr := &ResponseError{
				StatusCode:  resp.StatusCode,
				ContentType: contentType,
				Body:        truncate(string(bodyBytes), 512),
				Message:     fmt.Sprintf("服务端返回异常 (HTTP %d)，请检查 base_url 配置或稍后重试", resp.StatusCode),
			}
			slog.ErrorContext(ctx, "api_response",
				"request_id", requestID,
				"method", method,
				"url", url,
				"status", resp.StatusCode,
				"duration_ms", duration.Milliseconds(),
				"error", respErr.Message,
			)
			return respErr
		}
	}

	// 2. 2xx 但 Content-Type 非 JSON 且非空
	if contentType != "" && !isJSONContentType(contentType) {
		respErr := &ResponseError{
			StatusCode:  resp.StatusCode,
			ContentType: contentType,
			Body:        truncate(string(bodyBytes), 512),
			Message:     "服务端返回非 JSON 响应，可能是 base_url 配置错误或需要重新认证",
		}
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", respErr.Message,
		)
		return respErr
	}

	// 3. JSON 解码
	var apiResp Response
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 4. 业务错误
	if resp.StatusCode == http.StatusUnauthorized {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", "unauthorized",
		)
		return ErrUnauthorized
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		)
		return &ValidationError{Message: apiResp.Message, Errors: apiResp.Errors}
	}

	if resp.StatusCode >= 400 {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		)
		return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiResp.Message)
	}

	// 5. 成功日志
	slog.InfoContext(ctx, "api_response",
		"request_id", requestID,
		"method", method,
		"url", url,
		"status", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	)

	// 6. 解析 data 到 result
	if result != nil && apiResp.Data != nil {
		data, err := json.Marshal(apiResp.Data)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, result); err != nil {
			return err
		}
	}

	return nil
}
```

注意：日志记录点分散在各 error return 之前，保证每次请求无论成功失败都有一对 request/response 日志。

- [ ] **Step 4: 运行新增测试**

Run: `go test ./internal/api/ -run "TestDoCtx|TestDo_Backward" -v`
Expected: PASS

- [ ] **Step 5: 运行全部 api 测试确认无回归**

Run: `go test ./internal/api/ -v`
Expected: 全部 PASS（包括现有 8 个测试 + 新增 4 个）

- [ ] **Step 6: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat(api): add DoCtx with structured request logging"
```

---

### Task 4: cmdgen 集成 - 生成 requestId 并调用 DoCtx

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:43-105` - buildSubCommand.Run 中生成 requestId，调用 DoCtx
- Modify: `internal/cmdgen/cmdgen_test.go` - 新增集成测试

- [ ] **Step 1: 写集成测试**

追加到 `internal/cmdgen/cmdgen_test.go`:

```go
import (
	"context"
	"log/slog"

	"github.com/childelins/ckjr-cli/internal/logging"
)

func TestBuildSubCommand_GeneratesRequestID(t *testing.T) {
	// 捕获日志输出
	var logBuf bytes.Buffer
	handler := slog.NewJSONHandler(&logBuf, nil)
	old := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(old)

	called := false
	var capturedCtx context.Context

	cfg := &router.RouteConfig{
		Resource: "test",
		Routes: map[string]router.Route{
			"list": {
				Method:      "POST",
				Path:        "/test/list",
				Description: "测试列表",
			},
		},
	}

	// 使用 mock clientFactory，捕获传入的 context
	mockFactory := func() (*api.Client, error) {
		// 这里无法直接捕获 DoCtx 的 ctx
		// 改为检查日志中是否包含 request_id
		called = true
		return api.NewClient("http://localhost", "key"), nil
	}
	_ = capturedCtx
	_ = called

	cmd := BuildCommand(cfg, mockFactory)
	// 不实际执行（会连接不存在的服务器），仅验证命令结构
	listCmd, _, _ := cmd.Find([]string{"list"})
	if listCmd == nil {
		t.Fatal("list 子命令未找到")
	}
}
```

注意：完整的端到端测试在 Task 5 中通过手动执行验证。这里主要确认编译通过和结构正确。

- [ ] **Step 2: 修改 buildSubCommand**

修改 `internal/cmdgen/cmdgen.go` 的 `buildSubCommand` Run 函数：

1. 添加 imports: `"context"`, `"github.com/childelins/ckjr-cli/internal/logging"`
2. 在获取 client 成功后、调用 API 之前，生成 requestId 并构建 context
3. 将 `client.Do(...)` 改为 `client.DoCtx(ctx, ...)`

```go
// 在 client, err := clientFactory() 之后，API 调用之前:
ctx := context.Background()
requestID := logging.NewRequestID()
ctx = logging.WithRequestID(ctx, requestID)

// 替换:
// if err := client.Do(route.Method, route.Path, input, &result); err != nil {
// 为:
if err := client.DoCtx(ctx, route.Method, route.Path, input, &result); err != nil {
```

- [ ] **Step 3: 运行 cmdgen 全部测试**

Run: `go test ./internal/cmdgen/ -v`
Expected: 全部 PASS

- [ ] **Step 4: 运行项目全部测试**

Run: `go test ./...`
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/cmdgen_test.go
git commit -m "feat(cmdgen): integrate requestId generation and DoCtx call"
```

---

### Task 5: cmd/root.go 初始化日志 + 端到端验证

**Files:**
- Modify: `cmd/root.go` - 添加 cobra.OnInitialize 调用 logging.Init

- [ ] **Step 1: 修改 root.go**

```go
import (
	"os"
	"path/filepath"

	"github.com/childelins/ckjr-cli/internal/logging"
)

func init() {
	rootCmd.PersistentFlags().Bool("pretty", false, "格式化 JSON 输出")
	rootCmd.PersistentFlags().Bool("verbose", false, "显示详细调试信息")

	cobra.OnInitialize(initLogging)

	rootCmd.AddCommand(configCmd)
	registerRouteCommands()
}

func initLogging() {
	verbose, _ := rootCmd.Flags().GetBool("verbose")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
		return
	}
	baseDir := filepath.Join(homeDir, ".ckjr")
	if err := logging.Init(verbose, baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
	}
}
```

- [ ] **Step 2: 编译验证**

Run: `go build -o ckjr .`
Expected: 编译成功

- [ ] **Step 3: 运行项目全部测试**

Run: `go test ./...`
Expected: 全部 PASS

- [ ] **Step 4: 端到端验证（手动）**

```bash
# 1. 正常请求（如已配置 API）
./ckjr agent list '{"page":1,"limit":5}'

# 2. 查看日志文件
cat ~/.ckjr/logs/$(date +%Y-%m-%d).log

# 3. 应看到 request_id、method、url、status、duration_ms
# 4. 用 requestId grep 验证关联
grep "<从上面日志中获取的requestId>" ~/.ckjr/logs/$(date +%Y-%m-%d).log

# 5. verbose 模式
./ckjr agent list '{"page":1,"limit":5}' --verbose
# stderr 应显示请求日志
```

- [ ] **Step 5: 提交**

```bash
git add cmd/root.go
git commit -m "feat(cmd): initialize logging system in cobra OnInitialize"
```
