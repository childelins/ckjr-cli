# Findings

## Go httptest Content-Type 自动检测

Go 的 httptest.NewServer 在 handler 中使用 `json.NewEncoder(w).Encode()` 时，如果未显式设置 `Content-Type` 头，ResponseWriter 会自动检测内容类型为 `text/plain; charset=utf-8`，而非 `application/json`。所有返回 JSON 的测试 handler 必须显式设置 `w.Header().Set("Content-Type", "application/json")`。要模拟完全无 Content-Type 的响应，需要使用 `w.Header()["Content-Type"] = nil` 后再调用 WriteHeader。
