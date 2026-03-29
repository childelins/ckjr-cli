# AI 友好错误处理实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将服务端返回的原始 JSON response 结构原样透传到 stderr，使 AI 可靠解析错误。

**Architecture:** 在 API 层引入 `APIError` 类型保留服务端完整字段，在命令层 `handleAPIErrorTo` 中用 `output.Print` 输出结构化 JSON 替代 `output.PrintError` 的扁平 `{"error":"msg"}`。保留 exit 0/1 不变。

**Tech Stack:** Go, cobra, encoding/json

---

## Phase 1: API 层 - APIError 类型

### Task 1: 新增 APIError 类型

**Files:**
- Modify: `internal/api/client.go` (在 ResponseError 定义后添加 APIError)
- Test: `internal/api/client_test.go`

- [ ] **Step 1: 写失败测试**

在 `internal/api/client_test.go` 末尾添加：

```go
func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 403,
		Message:    "无权访问",
		ServerCode: 403,
		Errors:     map[string]interface{}{"detail": "权限不足"},
	}

	// 验证 Error() 字符串
	want := "API 错误 (403): 无权访问"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// 验证 errors.As 匹配
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Error("errors.As should match APIError")
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
	if apiErr.Message != "无权访问" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "无权访问")
	}
	if apiErr.ServerCode != 403 {
		t.Errorf("ServerCode = %d, want 403", apiErr.ServerCode)
	}
	if apiErr.Errors["detail"] != "权限不足" {
		t.Errorf("Errors = %v, want detail=权限不足", apiErr.Errors)
	}
}

func TestAPIError_NilErrors(t *testing.T) {
	err := &APIError{
		StatusCode: 500,
		Message:    "internal error",
		ServerCode: 500,
	}
	if err.Errors != nil {
		t.Errorf("Errors should be nil when not set, got %v", err.Errors)
	}
}

func TestIsAPIError(t *testing.T) {
	apiErr := &APIError{StatusCode: 402, Message: "余额不足", ServerCode: 402}
	if !IsAPIError(apiErr) {
		t.Error("IsAPIError should return true for APIError")
	}

	otherErr := fmt.Errorf("some error")
	if IsAPIError(otherErr) {
		t.Error("IsAPIError should return false for non-APIError")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/api/ -run "TestAPIError|TestIsAPIError" -v`
Expected: FAIL - APIError 和 IsAPIError 未定义

- [ ] **Step 3: 实现 APIError 类型和辅助函数**

在 `internal/api/client.go` 的 `ResponseError.Detail()` 方法后面（第 72 行后）添加：

```go
// APIError 服务端返回的业务错误（JSON 格式，如 402/403/500 等）
type APIError struct {
	StatusCode int                    // HTTP 状态码
	Message    string                 // 服务端 message 字段
	ServerCode int                    // 服务端 status_code 字段
	Errors     map[string]interface{} // 服务端 errors 字段
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API 错误 (%d): %s", e.StatusCode, e.Message)
}
```

在 `IsResponseError` 函数后面（第 333 行后）添加：

```go
// IsAPIError 检查是否是 API 业务错误
func IsAPIError(err error) bool {
	var ae *APIError
	return errors.As(err, &ae)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/api/ -run "TestAPIError|TestIsAPIError" -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat(api): add APIError type to preserve server response fields"
```

---

### Task 2: 替换 client.go 中的 fmt.Errorf 为 APIError

**Files:**
- Modify: `internal/api/client.go:279` (替换通用 4xx/5xx 错误处理)
- Test: `internal/api/client_test.go`

- [ ] **Step 1: 更新现有测试期望**

在 `internal/api/client_test.go` 的 `TestClientDo_Non2xxWithJSON` 中（第 177-199 行），更新测试以验证返回 APIError 类型：

```go
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

	// 500 + JSON 应返回 APIError
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error should be APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if apiErr.Message != "internal error" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "internal error")
	}
	if apiErr.ServerCode != 500 {
		t.Errorf("ServerCode = %d, want 500", apiErr.ServerCode)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/api/ -run TestClientDo_Non2xxWithJSON -v`
Expected: FAIL - error is not APIError (currently fmt.Errorf)

- [ ] **Step 3: 替换 client.go 第 279 行**

将 `internal/api/client.go` 第 279 行：

```go
return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiResp.Message)
```

替换为：

```go
return &APIError{
	StatusCode: resp.StatusCode,
	Message:    apiResp.Message,
	ServerCode: apiResp.StatusCode,
	Errors:     apiResp.Errors,
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/api/ -v`
Expected: ALL PASS

- [ ] **Step 5: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "refactor(api): replace fmt.Errorf with APIError for 4xx/5xx JSON responses"
```

---

### Task 3: 新增 GetValidationMessage getter

**Files:**
- Modify: `internal/api/client.go` (在 GetValidationErrors 后添加)
- Test: `internal/api/client_test.go`

- [ ] **Step 1: 写失败测试**

```go
func TestGetValidationMessage(t *testing.T) {
	err := &ValidationError{
		Message: "参数校验失败",
		Errors:  map[string]interface{}{"name": "required"},
	}
	if got := GetValidationMessage(err); got != "参数校验失败" {
		t.Errorf("GetValidationMessage() = %q, want %q", got, "参数校验失败")
	}

	// 非 ValidationError 返回空字符串
	otherErr := fmt.Errorf("other")
	if got := GetValidationMessage(otherErr); got != "" {
		t.Errorf("GetValidationMessage() = %q, want empty", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/api/ -run TestGetValidationMessage -v`
Expected: FAIL - GetValidationMessage 未定义

- [ ] **Step 3: 实现 GetValidationMessage**

在 `internal/api/client.go` 的 `GetValidationErrors` 函数后（第 327 行后）添加：

```go
// GetValidationMessage 获取验证错误的消息
func GetValidationMessage(err error) string {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve.Message
	}
	return ""
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/api/ -run TestGetValidationMessage -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/api/client.go internal/api/client_test.go
git commit -m "feat(api): add GetValidationMessage getter"
```

---

## Phase 2: 命令层 - handleAPIError 透传结构化 JSON

### Task 4: 重写 handleAPIErrorTo 输出结构化 JSON

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:199-221` (重写 handleAPIErrorTo)
- Test: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 写失败测试**

在 `internal/cmdgen/cmdgen_test.go` 末尾添加：

```go
func TestHandleAPIErrorTo_Unauthorized_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	handleAPIErrorTo(&buf, api.ErrUnauthorized, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["message"] != "api_key 已过期，请重新登录获取" {
		t.Errorf("message = %v, want api_key 已过期", result["message"])
	}
	if result["status_code"] != float64(401) {
		t.Errorf("status_code = %v, want 401", result["status_code"])
	}
}

func TestHandleAPIErrorTo_ValidationError_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	err := &api.ValidationError{
		Message: "参数校验失败",
		Errors:  map[string]interface{}{"name": []interface{}{"required"}},
	}
	handleAPIErrorTo(&buf, err, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["message"] != "参数校验失败" {
		t.Errorf("message = %v, want 参数校验失败", result["message"])
	}
	if result["status_code"] != float64(422) {
		t.Errorf("status_code = %v, want 422", result["status_code"])
	}
	// errors 字段应该是 map 而非字符串
	errorsMap, ok := result["errors"].(map[string]interface{})
	if !ok {
		t.Fatalf("errors should be a map, got %T: %v", result["errors"], result["errors"])
	}
	nameErrors, ok := errorsMap["name"].([]interface{})
	if !ok || len(nameErrors) == 0 || nameErrors[0] != "required" {
		t.Errorf("errors.name = %v, want [required]", errorsMap["name"])
	}
}

func TestHandleAPIErrorTo_APIError_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	err := &api.APIError{
		StatusCode: 403,
		Message:    "无权访问",
		ServerCode: 403,
	}
	handleAPIErrorTo(&buf, err, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["message"] != "无权访问" {
		t.Errorf("message = %v, want 无权访问", result["message"])
	}
	if result["status_code"] != float64(403) {
		t.Errorf("status_code = %v, want 403", result["status_code"])
	}
}

func TestHandleAPIErrorTo_APIError_WithErrors(t *testing.T) {
	var buf bytes.Buffer
	err := &api.APIError{
		StatusCode: 402,
		Message:    "余额不足",
		ServerCode: 402,
		Errors:     map[string]interface{}{"detail": "账户余额为0"},
	}
	handleAPIErrorTo(&buf, err, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["status_code"] != float64(402) {
		t.Errorf("status_code = %v, want 402", result["status_code"])
	}
	errorsMap, ok := result["errors"].(map[string]interface{})
	if !ok {
		t.Fatalf("errors should be a map, got %T", result["errors"])
	}
	if errorsMap["detail"] != "账户余额为0" {
		t.Errorf("errors.detail = %v", errorsMap["detail"])
	}
}

func TestHandleAPIErrorTo_ResponseError_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)",
	}
	handleAPIErrorTo(&buf, respErr, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["status_code"] != float64(502) {
		t.Errorf("status_code = %v, want 502", result["status_code"])
	}
	if result["content_type"] != "text/html" {
		t.Errorf("content_type = %v, want text/html", result["content_type"])
	}
	// 非verbose不应包含body
	if _, exists := result["body"]; exists {
		t.Error("non-verbose should not contain body field")
	}
}

func TestHandleAPIErrorTo_ResponseError_Verbose_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)",
	}
	handleAPIErrorTo(&buf, respErr, true)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["body"] != "<html>Bad Gateway</html>" {
		t.Errorf("verbose should contain body, got: %v", result["body"])
	}
}

func TestHandleAPIErrorTo_GenericError_FlatJSON(t *testing.T) {
	var buf bytes.Buffer
	err := fmt.Errorf("网络连接超时")
	handleAPIErrorTo(&buf, err, false)

	// 非 API 错误保持 {"error":"msg"} 格式
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["error"] != "网络连接超时" {
		t.Errorf("error = %v, want 网络连接超时", result["error"])
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cmdgen/ -run "TestHandleAPIErrorTo_" -v`
Expected: FAIL - Unauthorized 输出 `{"error":"..."}` 而非 `{"message":"...","status_code":401}`

- [ ] **Step 3: 重写 handleAPIErrorTo**

将 `internal/cmdgen/cmdgen.go` 的 `handleAPIErrorTo` 函数（第 199-221 行）替换为：

```go
func handleAPIErrorTo(w io.Writer, err error, verbose bool) {
	// 1. Unauthorized -- 构造类似服务端格式的 JSON
	if api.IsUnauthorized(err) {
		resp := map[string]interface{}{
			"message":     "api_key 已过期，请重新登录获取",
			"status_code": 401,
		}
		output.Print(w, resp, false)
		return
	}

	// 2. ValidationError -- 透传服务端原始结构
	if api.IsValidationError(err) {
		errs := api.GetValidationErrors(err)
		msg := api.GetValidationMessage(err)
		resp := map[string]interface{}{
			"message":     msg,
			"status_code": 422,
			"errors":      errs,
		}
		output.Print(w, resp, false)
		return
	}

	// 3. APIError -- 透传服务端原始结构
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		resp := map[string]interface{}{
			"message":     apiErr.Message,
			"status_code": apiErr.ServerCode,
		}
		if len(apiErr.Errors) > 0 {
			resp["errors"] = apiErr.Errors
		}
		output.Print(w, resp, false)
		return
	}

	// 4. ResponseError (非 JSON 响应) -- 构造结构化输出
	var respErr *api.ResponseError
	if errors.As(err, &respErr) {
		detail := map[string]interface{}{
			"message":      respErr.Error(),
			"status_code":  respErr.StatusCode,
			"content_type": respErr.ContentType,
		}
		if verbose {
			detail["body"] = respErr.Body
		}
		output.Print(w, detail, false)
		return
	}

	// 5. 客户端侧错误（网络、序列化等）-- 保持简单格式
	output.PrintError(w, err.Error())
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/cmdgen/ -v`
Expected: ALL PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/cmdgen_test.go
git commit -m "feat(cmdgen): output structured JSON preserving server response fields in error handling"
```

---

## Phase 3: 统一非 API 命令的错误输出

### Task 5: config 命令统一 JSON 输出

**Files:**
- Modify: `cmd/config/config.go` (第 63, 74, 88, 97 行)

- [ ] **Step 1: 替换 fmt.Fprintf 为 output.PrintError**

在 `cmd/config/config.go` 中，将所有 `fmt.Fprintf(os.Stderr, ...)` 替换为 `output.PrintError(os.Stderr, ...)`。

第 63 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
// 改后
output.PrintError(os.Stderr, fmt.Sprintf("保存配置失败: %v", err))
```

第 74 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "无效的配置项: %s\n合法值: base_url, api_key\n", key)
// 改后
output.PrintError(os.Stderr, fmt.Sprintf("无效的配置项: %s。合法值: base_url, api_key", key))
```

第 88 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
// 改后
output.PrintError(os.Stderr, fmt.Sprintf("保存配置失败: %v", err))
```

第 97 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "读取配置失败: %v\n请先执行 ckjr-cli config init\n", err)
// 改后
output.PrintError(os.Stderr, fmt.Sprintf("读取配置失败: %v。请先执行 ckjr-cli config init", err))
```

- [ ] **Step 2: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 3: 提交**

```bash
git add cmd/config/config.go
git commit -m "fix(config): unify error output to JSON format"
```

---

### Task 6: root 命令统一 JSON 输出

**Files:**
- Modify: `cmd/root.go` (第 77, 83, 89, 95, 102 行)

- [ ] **Step 1: 替换 fmt.Fprintf 为 output.PrintError**

需要在 `cmd/root.go` 中添加 `output` 包的 import，然后替换所有 `fmt.Fprintf(os.Stderr, ...)`。

添加 import：
```go
"github.com/childelins/ckjr-cli/internal/output"
```

第 77 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
// 改后
output.PrintError(os.Stderr, fmt.Sprintf("获取用户目录失败: %v", err))
```

第 83 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
// 改后
output.PrintError(os.Stderr, fmt.Sprintf("日志初始化失败: %v", err))
```

第 89 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "YAML 文件系统未初始化\n")
// 改后
output.PrintError(os.Stderr, "YAML 文件系统未初始化")
```

第 95 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "读取路由目录失败: %v\n", err)
// 改后
output.PrintError(os.Stderr, fmt.Sprintf("读取路由目录失败: %v", err))
```

第 102 行：
```go
// 改前
fmt.Fprintf(os.Stderr, "解析路由文件 %s 失败: %v\n", name, err)
// 改后
output.PrintError(os.Stderr, fmt.Sprintf("解析路由文件 %s 失败: %v", name, err))
```

检查 `fmt` 是否仍被使用（`filepath.Join` 等不依赖 `fmt`）。如果 `fmt` 不再被使用，从 import 中移除。注意 `registerRouteCommands` 中不再使用 `fmt`，但 `initLogging` 也不再使用。检查整个文件——`fmt` 可能不再需要。

- [ ] **Step 2: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 3: 提交**

```bash
git add cmd/root.go
git commit -m "fix(root): unify error output to JSON format"
```

---

## Phase 4: 验收测试

### Task 7: 集成测试 - 验证完整错误链路

**Files:**
- Test: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 写集成测试**

添加端到端测试，用 httptest 模拟服务端返回各种状态码，验证 CLI 的 stderr 输出：

```go
func TestHandleAPIErrorTo_Integration_ServerResponsePassthrough(t *testing.T) {
	tests := []struct {
		name       string
		setupError error
		verbose    bool
		wantFields map[string]interface{}
	}{
		{
			name:       "403_forbidden",
			setupError: &api.APIError{StatusCode: 403, Message: "无权访问", ServerCode: 403},
			wantFields: map[string]interface{}{
				"message":     "无权访问",
				"status_code": float64(403),
			},
		},
		{
			name:       "500_server_error",
			setupError: &api.APIError{StatusCode: 500, Message: "内部错误", ServerCode: 500},
			wantFields: map[string]interface{}{
				"message":     "内部错误",
				"status_code": float64(500),
			},
		},
		{
			name: "422_validation_with_field_errors",
			setupError: &api.ValidationError{
				Message: "参数校验失败",
				Errors:  map[string]interface{}{"name": "required"},
			},
			wantFields: map[string]interface{}{
				"message":     "参数校验失败",
				"status_code": float64(422),
			},
		},
		{
			name: "502_gateway_non_json_verbose",
			setupError: &api.ResponseError{
				StatusCode:  502,
				ContentType: "text/html",
				Body:        "Bad Gateway",
				Message:     "服务端返回异常 (HTTP 502)",
			},
			verbose: true,
			wantFields: map[string]interface{}{
				"message":      "服务端返回异常 (HTTP 502)",
				"status_code":  float64(502),
				"content_type": "text/html",
				"body":         "Bad Gateway",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handleAPIErrorTo(&buf, tt.setupError, tt.verbose)

			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("output should be valid JSON, got: %q", buf.String())
			}
			for key, wantVal := range tt.wantFields {
				gotVal, ok := result[key]
				if !ok {
					t.Errorf("missing field %q in output: %v", key, result)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("field %q = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}
```

- [ ] **Step 2: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 3: 运行 go vet**

Run: `go vet ./...`
Expected: 无警告

- [ ] **Step 4: 提交**

```bash
git add internal/cmdgen/cmdgen_test.go
git commit -m "test(cmdgen): add integration tests for structured error output"
```
