package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
