# Request Body 日志 & Route 命令隐藏 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 HTTP 请求日志中增加 request body 和 response body 字段，并将 route 命令从 --help 中隐藏。

**Architecture:** Q3 修改 `internal/api/client.go` 的 `DoCtx` 方法，调整 body 序列化顺序使 `data` 在日志前可用，在所有日志点增加 body 字段。Q4 在 `cmd/route.go` 的 `routeCmd` 中设置 `Hidden: true`。

**Tech Stack:** Go 1.24.3, log/slog, Cobra

---

### Task 1: DoCtx 增加 request_body 日志

**Files:**
- Modify: `internal/api/client.go:81-100`
- Test: `internal/api/client_test.go`

**Spec 参考:** `docs/superpowers/specs/2026-03-26-request-body-logging-and-route-hidden-design.md` Q3 请求日志部分

**改动说明:** 将 `json.Marshal(body)` 提前到 `api_request` 日志之前执行，在日志中增加 `request_body` 字段。

- [ ] **Step 1: 写失败测试 TestDoCtx_LogsRequestBody**

在 `internal/api/client_test.go` 中新增：

```go
func TestDoCtx_LogsRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: nil, Message: "ok", StatusCode: 200})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "body-req-001")

	var result interface{}
	body := map[string]interface{}{"name": "test", "page": 1}
	output := captureLog(func() {
		client.DoCtx(ctx, "POST", "/test", body, &result)
	})

	if !strings.Contains(output, "request_body") {
		t.Errorf("log should contain request_body field, got: %s", output)
	}
	if !strings.Contains(output, `"name"`) {
		t.Errorf("log should contain request body content, got: %s", output)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -run TestDoCtx_LogsRequestBody -v`
Expected: FAIL — 日志中不含 `request_body` 字段

- [ ] **Step 3: 写失败测试 TestDoCtx_NilBody_LogsEmptyRequestBody**

```go
func TestDoCtx_NilBody_LogsEmptyRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: nil, Message: "ok", StatusCode: 200})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "nil-body-001")

	var result interface{}
	output := captureLog(func() {
		client.DoCtx(ctx, "GET", "/test", nil, &result)
	})

	if !strings.Contains(output, "request_body") {
		t.Errorf("log should contain request_body field even for nil body, got: %s", output)
	}
}
```

- [ ] **Step 4: 实现 request_body 日志**

修改 `internal/api/client.go` 的 `DoCtx` 方法（L81-100），将 body 序列化移到日志之前：

```go
func (c *Client) DoCtx(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	requestID := logging.RequestIDFrom(ctx)
	url := c.baseURL + path

	// body 序列化提前，使 data 在日志时可用
	var data []byte
	var reqBody io.Reader
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	slog.InfoContext(ctx, "api_request",
		"request_id", requestID,
		"method", method,
		"url", url,
		"request_body", string(data),
	)

	start := time.Now()
	// ... 后续代码不变（删除原 L93-100 的 body 序列化块）
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -run "TestDoCtx_Logs(RequestBody|NilBody)" -v`
Expected: PASS

- [ ] **Step 6: 运行全部已有测试确认无回归**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -v`
Expected: 全部 PASS

- [ ] **Step 7: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat(api): add request_body field to api_request log"
```

---

### Task 2: DoCtx 增加 response_body 日志

**Files:**
- Modify: `internal/api/client.go:126-243`
- Test: `internal/api/client_test.go`

**Spec 参考:** Q3 响应日志部分

**改动说明:** 在所有 `api_response` 日志点增加 `response_body` 字段。网络错误（无响应体）和读取响应体失败的日志点不添加。

- [ ] **Step 1: 写失败测试 TestDoCtx_LogsResponseBody**

```go
func TestDoCtx_LogsResponseBody(t *testing.T) {
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
	ctx := logging.WithRequestID(context.Background(), "resp-body-001")

	var result map[string]string
	output := captureLog(func() {
		client.DoCtx(ctx, "GET", "/test", nil, &result)
	})

	if !strings.Contains(output, "response_body") {
		t.Errorf("log should contain response_body field, got: %s", output)
	}
	if !strings.Contains(output, `"id"`) {
		t.Errorf("log should contain response body content, got: %s", output)
	}
}
```

- [ ] **Step 2: 写失败测试 TestDoCtx_LogsResponseBody_OnError**

```go
func TestDoCtx_LogsResponseBody_OnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(Response{
			Message:    "validation failed",
			StatusCode: 422,
			Errors:     map[string]interface{}{"name": "required"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "err-resp-001")

	var result interface{}
	output := captureLog(func() {
		client.DoCtx(ctx, "POST", "/test", map[string]string{"x": "y"}, &result)
	})

	if !strings.Contains(output, "response_body") {
		t.Errorf("error log should contain response_body field, got: %s", output)
	}
	if !strings.Contains(output, "validation failed") {
		t.Errorf("error log should contain response body content, got: %s", output)
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -run "TestDoCtx_LogsResponseBody" -v`
Expected: FAIL

- [ ] **Step 4: 实现 response_body 日志**

在 `internal/api/client.go` 的所有 `api_response` 日志点增加 `"response_body", string(bodyBytes)` 字段。需要修改的日志点（共 7 处，排除网络错误 L113 和读取失败 L129）：

**L151-158** (非 JSON 非 2xx):
```go
slog.ErrorContext(ctx, "api_response",
    "request_id", requestID,
    "method", method,
    "url", url,
    "status", resp.StatusCode,
    "duration_ms", duration.Milliseconds(),
    "error", respErr.Message,
    "response_body", string(bodyBytes),
)
```

**L171-178** (2xx 但非 JSON):
同上模式添加 `"response_body", string(bodyBytes)`

**L185-192** (JSON 解码失败):
同上

**L198-206** (401):
同上

**L210-217** (422):
同上

**L225-232** (其他 4xx/5xx):
同上

**L237-243** (成功):
```go
slog.InfoContext(ctx, "api_response",
    "request_id", requestID,
    "method", method,
    "url", url,
    "status", resp.StatusCode,
    "duration_ms", duration.Milliseconds(),
    "response_body", string(bodyBytes),
)
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -v`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat(api): add response_body field to api_response log"
```

---

### Task 3: 隐藏 route 命令

**Files:**
- Modify: `cmd/route.go:14-17`
- Test: `cmd/route_test.go`

**Spec 参考:** Q4 部分

- [ ] **Step 1: 写失败测试 TestRouteCmd_IsHidden**

在 `cmd/route_test.go` 中新增：

```go
func TestRouteCmd_IsHidden(t *testing.T) {
	if !routeCmd.Hidden {
		t.Error("routeCmd should be hidden")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/ -run TestRouteCmd_IsHidden -v`
Expected: FAIL — routeCmd.Hidden 为 false

- [ ] **Step 3: 设置 Hidden: true**

修改 `cmd/route.go` L14-17：

```go
var routeCmd = &cobra.Command{
	Use:    "route",
	Short:  "路由配置管理",
	Hidden: true,
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/ -run TestRouteCmd_IsHidden -v`
Expected: PASS

- [ ] **Step 5: 运行全部测试确认无回归**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... 2>&1 | tail -20`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add cmd/route.go cmd/route_test.go
git commit -m "feat(cmd): hide route command from --help output"
```
