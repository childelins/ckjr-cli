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
		// 清除 Content-Type，返回合法 JSON
		w.Header()["Content-Type"] = nil
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"key":"value"},"message":"ok","status_code":200}`))
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

// containsAll 检查 s 是否包含所有子串
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
