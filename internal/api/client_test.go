package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

// containsAll 检查 s 是否包含所有子串
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
