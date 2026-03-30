package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/childelins/ckjr-cli/internal/logging"
)

func TestClientDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("Missing Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Missing Content-Type header")
		}

		// 返回模拟响应
		w.Header().Set("Content-Type", "application/json")
		resp := Response{
			Data:       map[string]string{"id": "123"},
			Message:    "success",
			StatusCode: 200,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	var result map[string]string
	err := client.Do("POST", "/test", nil, &result)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("result = %v, want id=123", result)
	}
}

func TestClientUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		resp := Response{
			Message:    "Unauthorized",
			StatusCode: 401,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "invalid-key")

	var result interface{}
	err := client.Do("POST", "/test", nil, &result)
	if err == nil {
		t.Fatal("Do() should return error for 401")
	}

	if !IsUnauthorized(err) {
		t.Errorf("error should be ErrUnauthorized, got %v", err)
	}
}

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

func TestClientDo_EmptyContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 清除 Content-Type，返回合法 JSON
		w.Header()["Content-Type"] = nil
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"key":"value"},"msg":"ok","statusCode":200}`))
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
	tmpDir := t.TempDir()
	if err := logging.Init(false, tmpDir, logging.Development); err != nil {
		t.Fatalf("logging.Init: %v", err)
	}
	output := captureLog(func() {
		client.DoCtx(ctx, "POST", "/test", body, &result)
	})

	if !strings.Contains(output, "request_body") {
		t.Errorf("log should contain request_body field, got: %s", output)
	}
	if !strings.Contains(output, "name") || !strings.Contains(output, "test") {
		t.Errorf("log should contain request body content, got: %s", output)
	}
}

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
	tmpDir := t.TempDir()
	if err := logging.Init(false, tmpDir, logging.Development); err != nil {
		t.Fatalf("logging.Init: %v", err)
	}
	output := captureLog(func() {
		client.DoCtx(ctx, "GET", "/test", nil, &result)
	})

	if !strings.Contains(output, "response_body") {
		t.Errorf("log should contain response_body field, got: %s", output)
	}
	if !strings.Contains(output, "id") || !strings.Contains(output, "42") {
		t.Errorf("log should contain response body content, got: %s", output)
	}
}

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
	tmpDir := t.TempDir()
	if err := logging.Init(false, tmpDir, logging.Development); err != nil {
		t.Fatalf("logging.Init: %v", err)
	}
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

func TestDoCtx_NilBody_LogsEmptyRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: nil, Message: "ok", StatusCode: 200})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "nil-body-001")

	var result interface{}
	tmpDir := t.TempDir()
	if err := logging.Init(false, tmpDir, logging.Development); err != nil {
		t.Fatalf("logging.Init: %v", err)
	}
	output := captureLog(func() {
		client.DoCtx(ctx, "GET", "/test", nil, &result)
	})

	if !strings.Contains(output, "request_body") {
		t.Errorf("log should contain request_body field even for nil body, got: %s", output)
	}
}

func TestReadableJSON(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string // 期望包含的子串
	}{
		{
			name:  "unicode escapes decoded to UTF-8",
			input: []byte(`{"name":"\u5c0f\u5c0f\u753b\u624b"}`),
			want:  "小小画手",
		},
		{
			name:  "mixed unicode escapes and ASCII",
			input: []byte(`{"msg":"\u4f60\u597d world"}`),
			want:  "你好 world",
		},
		{
			name:  "already UTF-8 stays unchanged",
			input: []byte(`{"name":"小小画手"}`),
			want:  "小小画手",
		},
		{
			name:  "invalid JSON returns raw string",
			input: []byte(`not json`),
			want:  "not json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readableJSON(tt.input)
			if !strings.Contains(got, tt.want) {
				t.Errorf("readableJSON() = %s, want to contain %s", got, tt.want)
			}
		})
	}
}

func TestDoCtx_LogsChinese_Readable(t *testing.T) {
	// 模拟 PHP 服务端返回 Unicode 转义的 JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 手动写入包含 Unicode 转义的 JSON（模拟 PHP json_encode 行为）
		w.Write([]byte(`{"statusCode":200,"data":{"name":"\u5c0f\u5c0f\u753b\u624b"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	ctx := logging.WithRequestID(context.Background(), "chinese-001")

	var result map[string]string
	tmpDir := t.TempDir()
	if err := logging.Init(false, tmpDir, logging.Development); err != nil {
		t.Fatalf("logging.Init: %v", err)
	}
	output := captureLog(func() {
		client.DoCtx(ctx, "GET", "/test", nil, &result)
	})

	if !strings.Contains(output, "小小画手") {
		t.Errorf("log should contain readable Chinese, got: %s", output)
	}
	if strings.Contains(output, `\u5c0f`) {
		t.Errorf("log should NOT contain unicode escapes, got: %s", output)
	}
}

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

func TestResponse_UnmarshalJSON(t *testing.T) {
	input := `{"data":{"id":"1"},"msg":"ok","statusCode":200}`
	var resp Response
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if resp.Message != "ok" {
		t.Errorf("Message = %q, want %q", resp.Message, "ok")
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestClientDo_BusinessError(t *testing.T) {
	// HTTP 200 但 body 里业务码为 402
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ext":{"s":"abc"},"msg":"未找到该用户","statusCode":402}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	var result interface{}
	err := client.Do("POST", "/test", nil, &result)
	if err == nil {
		t.Fatal("Do() should return error for business status 402")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error should be APIError, got %T: %v", err, err)
	}
	if apiErr.ServerCode != 402 {
		t.Errorf("ServerCode = %d, want 402", apiErr.ServerCode)
	}
	if apiErr.Message != "未找到该用户" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "未找到该用户")
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
