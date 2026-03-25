# API Client 错误处理改进 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 重构 `api.Client.Do()` 的响应处理流程，修复非 JSON 响应导致的不可读错误，增加 Content-Type 校验和 `--verbose` 调试模式。

**Architecture:** 传输层（`api/client.go`）重构响应处理顺序：先状态码、再 Content-Type、再 JSON 解码。新增 `ResponseError` 类型承载调试信息。表现层（`cmdgen/cmdgen.go`、`cmd/root.go`）增加 `--verbose` flag 透传。

**Tech Stack:** Go, cobra, httptest

---

## 文件结构

| 文件 | 职责 | 操作 |
|------|------|------|
| `internal/api/client.go` | API 客户端，响应处理 | 修改：新增 `ResponseError` 类型，重构 `Do()` |
| `internal/api/client_test.go` | API 客户端测试 | 修改：新增 7 个测试用例 |
| `internal/cmdgen/cmdgen.go` | 命令生成，错误处理展示 | 修改：`handleAPIError` 增加 verbose 参数 |
| `internal/cmdgen/cmdgen_test.go` | 命令生成测试 | 修改：新增 2 个 handleAPIError 测试 |
| `cmd/root.go` | 根命令 | 修改：添加 `--verbose` PersistentFlag |

---

### Task 1: 新增 ResponseError 类型

**Files:**
- Modify: `internal/api/client.go`
- Modify: `internal/api/client_test.go`

- [ ] **Step 1: 写 ResponseError 类型的测试**

在 `internal/api/client_test.go` 末尾添加：

```go
func TestResponseError_Error(t *testing.T) {
	err := &ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试",
	}

	got := err.Error()
	want := "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestResponseError_Detail(t *testing.T) {
	err := &ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试",
	}

	detail := err.Detail()
	if detail == "" {
		t.Error("Detail() should not be empty")
	}
	// 验证包含关键调试信息
	if !containsAll(detail, "502", "text/html", "Bad Gateway") {
		t.Errorf("Detail() missing debug info: %s", detail)
	}
}

func TestIsResponseError(t *testing.T) {
	original := &ResponseError{
		StatusCode: 502,
		Message:    "test",
	}
	var wrapped error = fmt.Errorf("wrapped: %w", original)

	var respErr *ResponseError
	if !errors.As(wrapped, &respErr) {
		t.Error("errors.As should match ResponseError")
	}
	if respErr.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", respErr.StatusCode)
	}
}

// containsAll 检查 s 是否包含所有子串
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -run "TestResponseError|TestIsResponseError" -v`
Expected: 编译失败，ResponseError 未定义

- [ ] **Step 3: 实现 ResponseError 类型**

在 `internal/api/client.go` 中 `ValidationError` 后面添加：

```go
// ResponseError 非预期响应错误（非 JSON、非 2xx 等）
type ResponseError struct {
	StatusCode  int
	ContentType string
	Body        string // 响应体前 512 字符
	Message     string // 用户友好的错误信息
}

func (e *ResponseError) Error() string {
	return e.Message
}

// Detail 返回包含调试信息的详细错误描述
func (e *ResponseError) Detail() string {
	return fmt.Sprintf("HTTP %d | Content-Type: %s\n响应体: %s", e.StatusCode, e.ContentType, e.Body)
}
```

同时在 `client.go` 底部添加辅助函数：

```go
// IsResponseError 检查是否是非预期响应错误
func IsResponseError(err error) bool {
	var re *ResponseError
	return errors.As(err, &re)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -run "TestResponseError|TestIsResponseError" -v`
Expected: 3 个测试全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat(api): add ResponseError type for non-JSON response handling"
```

---

### Task 2: 重构 Do() 响应处理流程

**Files:**
- Modify: `internal/api/client.go`
- Modify: `internal/api/client_test.go`

- [ ] **Step 1: 写 HTML 响应测试**

在 `internal/api/client_test.go` 添加：

```go
func TestClientDo_HTMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Login Page</body></html>"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	var result interface{}
	err := client.Do("POST", "/test", nil, &result)
	if err == nil {
		t.Fatal("Do() should return error for HTML response")
	}

	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("error should be ResponseError, got %T: %v", err, err)
	}
	if respErr.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", respErr.StatusCode)
	}
	if respErr.ContentType != "text/html" {
		t.Errorf("ContentType = %s, want text/html", respErr.ContentType)
	}
}

func TestClientDo_Non2xxWithHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<html>Bad Gateway</html>"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	var result interface{}
	err := client.Do("POST", "/test", nil, &result)
	if err == nil {
		t.Fatal("Do() should return error for 502")
	}

	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("error should be ResponseError, got %T: %v", err, err)
	}
	if respErr.StatusCode != 502 {
		t.Errorf("StatusCode = %d, want 502", respErr.StatusCode)
	}
}

func TestClientDo_Non2xxWithJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		resp := Response{Message: "internal error", StatusCode: 500}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	var result interface{}
	err := client.Do("POST", "/test", nil, &result)
	if err == nil {
		t.Fatal("Do() should return error for 500")
	}
	// 500 + JSON 应走现有错误处理，不是 ResponseError
	if IsResponseError(err) {
		t.Error("500 with JSON should not be ResponseError")
	}
	if !strings.Contains(err.Error(), "internal error") {
		t.Errorf("error should contain API message, got: %v", err)
	}
}

func TestClientDo_EmptyContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 不设置 Content-Type，但返回合法 JSON
		w.WriteHeader(http.StatusOK)
		resp := Response{
			Data:       map[string]string{"key": "value"},
			Message:    "ok",
			StatusCode: 200,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	var result map[string]string
	err := client.Do("POST", "/test", nil, &result)
	if err != nil {
		t.Fatalf("Do() error = %v, empty Content-Type with valid JSON should succeed", err)
	}
	if result["key"] != "value" {
		t.Errorf("result = %v, want key=value", result)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -run "TestClientDo_HTMLResponse|TestClientDo_Non2xx|TestClientDo_EmptyContentType" -v`
Expected: HTMLResponse 和 Non2xxWithHTML 失败（当前代码 JSON 解码报错而非 ResponseError）

- [ ] **Step 3: 重构 Do() 方法**

将 `client.go` 中 `Do()` 方法的响应处理部分（`resp` 获取后的代码）替换为：

```go
	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")

	// 1. 非 2xx 状态码 + 非 JSON -> ResponseError
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if !isJSONContentType(contentType) {
			return &ResponseError{
				StatusCode:  resp.StatusCode,
				ContentType: contentType,
				Body:        truncate(string(bodyBytes), 512),
				Message:     fmt.Sprintf("服务端返回异常 (HTTP %d)，请检查 base_url 配置或稍后重试", resp.StatusCode),
			}
		}
	}

	// 2. 2xx 但 Content-Type 非 JSON 且非空 -> ResponseError
	if contentType != "" && !isJSONContentType(contentType) {
		return &ResponseError{
			StatusCode:  resp.StatusCode,
			ContentType: contentType,
			Body:        truncate(string(bodyBytes), 512),
			Message:     "服务端返回非 JSON 响应，可能是 base_url 配置错误或需要重新认证",
		}
	}

	// 3. JSON 解码
	var apiResp Response
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 4. 业务错误处理
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return &ValidationError{
			Message: apiResp.Message,
			Errors:  apiResp.Errors,
		}
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiResp.Message)
	}

	// 5. 解析 data 到 result
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
```

在 `client.go` 底部添加辅助函数：

```go
// isJSONContentType 检查 Content-Type 是否包含 application/json
func isJSONContentType(ct string) bool {
	return strings.Contains(ct, "application/json")
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
```

注意：需要在 import 中添加 `"strings"`。

- [ ] **Step 4: 运行全部 api 测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/api/ -v`
Expected: 所有测试（含原有 TestClientDo、TestClientUnauthorized）全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "refactor(api): reorder response handling - check status and content-type before JSON decode"
```

---

### Task 3: 添加 --verbose 全局 flag

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`

- [ ] **Step 1: 写 --verbose flag 测试**

在 `cmd/root_test.go` 添加：

```go
func TestVerboseFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("verbose")
	if f == nil {
		t.Fatal("--verbose flag 未注册")
	}
	if f.DefValue != "false" {
		t.Errorf("默认值 = %s, want false", f.DefValue)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/ -run "TestVerboseFlag" -v`
Expected: FAIL，verbose flag 不存在

- [ ] **Step 3: 在 root.go 中添加 --verbose flag**

在 `cmd/root.go` 的 `init()` 函数中，`--pretty` 后面添加：

```go
	rootCmd.PersistentFlags().Bool("verbose", false, "显示详细调试信息")
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/root.go cmd/root_test.go
git commit -m "feat(cmd): add --verbose persistent flag for debug output"
```

---

### Task 4: handleAPIError 增加 verbose 支持

**Files:**
- Modify: `internal/cmdgen/cmdgen.go`
- Modify: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 写 handleAPIError 处理 ResponseError 的测试**

在 `internal/cmdgen/cmdgen_test.go` 添加：

```go
func TestHandleAPIError_ResponseError(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试",
	}

	handleAPIErrorTo(&buf, respErr, false)

	got := buf.String()
	if !strings.Contains(got, "服务端返回异常") {
		t.Errorf("output should contain friendly message, got: %s", got)
	}
	if strings.Contains(got, "text/html") {
		t.Error("non-verbose should not contain Content-Type")
	}
}

func TestHandleAPIError_ResponseError_Verbose(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试",
	}

	handleAPIErrorTo(&buf, respErr, true)

	got := buf.String()
	if !strings.Contains(got, "服务端返回异常") {
		t.Errorf("output should contain friendly message, got: %s", got)
	}
	if !strings.Contains(got, "502") || !strings.Contains(got, "text/html") {
		t.Errorf("verbose should contain debug info, got: %s", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run "TestHandleAPIError" -v`
Expected: 编译失败，handleAPIErrorTo 未定义

- [ ] **Step 3: 重构 handleAPIError**

在 `internal/cmdgen/cmdgen.go` 中：

1. 将 `handleAPIError(err error)` 改为 `handleAPIError(err error, verbose bool)`，内部调用 `handleAPIErrorTo`。

2. 新增可测试的 `handleAPIErrorTo` 函数：

```go
func handleAPIError(err error, verbose bool) {
	handleAPIErrorTo(os.Stderr, err, verbose)
}

func handleAPIErrorTo(w io.Writer, err error, verbose bool) {
	if api.IsUnauthorized(err) {
		output.PrintError(w, "api_key 已过期，请重新登录获取")
		return
	}

	if api.IsValidationError(err) {
		errs := api.GetValidationErrors(err)
		output.PrintError(w, fmt.Sprintf("参数校验失败: %v", errs))
		return
	}

	var respErr *api.ResponseError
	if errors.As(err, &respErr) {
		output.PrintError(w, respErr.Error())
		if verbose {
			fmt.Fprintf(w, "  %s\n", respErr.Detail())
		}
		return
	}

	output.PrintError(w, err.Error())
}
```

注意：需要在 import 中添加 `"errors"`。

3. 更新 `buildSubCommand` 中的调用：

```go
			verbose, _ := cmd.Flags().GetBool("verbose")
			if err := client.Do(route.Method, route.Path, input, &result); err != nil {
				handleAPIError(err, verbose)
				os.Exit(1)
			}
```

- [ ] **Step 4: 运行全部测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... -v`
Expected: 所有测试全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/cmdgen_test.go
git commit -m "feat(cmdgen): enhance handleAPIError with verbose support for ResponseError"
```

---

### Task 5: 验收测试

- [ ] **Step 1: 运行全部测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... -v -count=1`
Expected: 所有测试 PASS

- [ ] **Step 2: 编译验证**

Run: `cd /home/childelins/code/ckjr-cli && go build -o ckjr .`
Expected: 编译成功

- [ ] **Step 3: 验证 --verbose flag 注册**

Run: `./ckjr --help`
Expected: 输出中包含 `--verbose` 说明

- [ ] **Step 4: 提交最终状态（如有调整）**

```bash
git add -A && git commit -m "chore: final cleanup for API error handling improvement"
```
