# AI 友好错误处理设计文档

> Created: 2026-03-29
> Status: Draft

## 概述

审查当前 HTTP 客户端错误处理对 AI 的友好性。核心原则：**将服务端返回的原始 JSON response 结构原样透传到 stderr**，不重新包装。

## 现状分析

### 服务端 Response 结构

服务端返回统一 JSON 格式（`internal/api/client.go` 第 40-45 行）：

```go
type Response struct {
    Data       interface{}            `json:"data"`
    Message    string                 `json:"message"`
    StatusCode int                    `json:"status_code"`
    Errors     map[string]interface{} `json:"errors,omitempty"`
}
```

实际响应示例：
```json
// 成功
{"data": {...}, "message": "ok", "status_code": 200}

// 422 校验失败
{"data": null, "message": "参数校验失败", "status_code": 422, "errors": {"name": ["required"], "email": ["invalid format"]}}

// 401 认证失败
{"data": null, "message": "api_key 已过期", "status_code": 401}

// 403 权限不足
{"data": null, "message": "无权访问该资源", "status_code": 403}
```

### 当前错误处理链路

**API 层** (`internal/api/client.go`) 的错误分支：

| HTTP 状态 | 当前处理 | 输出到 cmdgen 的 error 类型 |
|-----------|---------|---------------------------|
| 401 | `return ErrUnauthorized` (静态哨兵错误) | `errors.New("api_key 已过期...")` |
| 422 | `return &ValidationError{Message, Errors}` | 保留了 message + errors map |
| 非 2xx + 非 JSON | `return &ResponseError{...}` | 保留了 status code + body |
| 其他 4xx/5xx (JSON) | `return fmt.Errorf("API 错误 (%d): %s", code, msg)` | **丢失 status_code 和 errors 字段** |

**命令层** (`internal/cmdgen/cmdgen.go`) 的 `handleAPIError`:

| 错误类型 | 当前处理 | 最终 stderr 输出 |
|---------|---------|----------------|
| ErrUnauthorized | `output.PrintError(w, "api_key 已过期...")` | `{"error":"api_key 已过期..."}` |
| ValidationError | `fmt.Sprintf("参数校验失败: %v", errs)` | `{"error":"参数校验失败: map[name:required]"}` |
| ResponseError | `respErr.Error()` + verbose 追加纯文本 | `{"error":"服务端返回异常..."}` + 非JSON文本 |
| fmt.wrapError (其他 API 错误) | `err.Error()` | `{"error":"API 错误 (403): 无权访问该资源"}` |

**输出层** (`internal/output/output.go`): 所有错误统一 `{"error":"msg"}`。

### 核心问题总结

1. **服务端 JSON 结构被丢弃**: `handleAPIError` 把服务端返回的 `{message, status_code, errors}` 结构重新包装为纯文本消息，再塞进 `{"error":"msg"}`。AI 看到的是扁平字符串，丢失了结构化的 `errors` 字段和 `status_code`。

2. **ValidationError 的 map 被 `%v` 序列化**: Go 的 `fmt.Sprintf("%v", map)` 输出 `map[name:required email:invalid]`，这不是 JSON，AI 无法可靠解析。

3. **通用 4xx/5xx JSON 错误被 `fmt.Errorf` 吞噬**: client.go 第 279 行 `return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiResp.Message)` 丢弃了 `apiResp.Errors` 和 `apiResp.StatusCode`，只保留 message 文本。

4. **verbose 破坏 JSON**: ResponseError 的 Detail() 直接 `fmt.Fprintf` 追加在 JSON 后面，导致 stderr 不是合法 JSON。

5. **config/root 的错误不走 JSON**: `fmt.Fprintf(os.Stderr, ...)` 输出纯文本。

## 改进方案

### 原则

- 不引入退出码机制，保留 exit 0/1
- **将服务端原始 response JSON 原样透传到 stderr**
- 客户端侧错误（网络、序列化、配置）用简单的 `{"error":"msg"}` 即可

### 方案 1: handleAPIError 透传服务端原始 JSON

**核心改动**: 在 `handleAPIErrorTo` 中，对每种 API 错误类型，构建一个保留服务端字段的结构输出到 stderr。

```go
func handleAPIErrorTo(w io.Writer, err error, verbose bool) {
    // 1. Unauthorized -- 构造类似服务端格式的 JSON
    if api.IsUnauthorized(err) {
        resp := map[string]interface{}{
            "message":    "api_key 已过期，请重新登录获取",
            "status_code": 401,
        }
        output.Print(w, resp, false)
        return
    }

    // 2. ValidationError -- 透传服务端原始结构
    if api.IsValidationError(err) {
        errs := api.GetValidationErrors(err)
        msg := api.GetValidationMessage(err)  // 需要新增 getter
        resp := map[string]interface{}{
            "message":    msg,
            "status_code": 422,
            "errors":     errs,
        }
        output.Print(w, resp, false)
        return
    }

    // 3. ResponseError (非 JSON 响应) -- 构造结构化输出
    var respErr *api.ResponseError
    if errors.As(err, &respErr) {
        detail := map[string]interface{}{
            "message":     respErr.Error(),
            "status_code": respErr.StatusCode,
            "content_type": respErr.ContentType,
        }
        if verbose {
            detail["body"] = respErr.Body
        }
        output.Print(w, detail, false)
        return
    }

    // 4. 通用 API 错误 (fmt.Errorf 包装的) -- 需要方案 2 的 APIError 改造
    output.PrintError(w, err.Error())
}
```

**注意**: 这里用 `output.Print` 而非 `output.PrintError`，因为 `Print` 会输出完整 JSON 对象，而 `PrintError` 只输出 `{"error":"msg"}`。

### 方案 2: 引入 APIError 类型，保留服务端完整 response

**问题**: client.go 第 279 行用 `fmt.Errorf` 包装通用 4xx/5xx 错误，丢失了服务端返回的 `status_code` 和 `errors` 字段。

**改动**: 新增 `APIError` 类型，携带服务端完整 response 数据：

```go
// APIError 服务端返回的业务错误（JSON 格式）
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

替换 client.go 第 279 行：
```go
// 改前
return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiResp.Message)

// 改后
return &APIError{
    StatusCode: resp.StatusCode,
    Message:    apiResp.Message,
    ServerCode: apiResp.StatusCode,
    Errors:     apiResp.Errors,
}
```

`handleAPIErrorTo` 新增 APIError 分支（在 ResponseError 之前）：
```go
// 通用 API 错误 -- 透传服务端原始结构
var apiErr *api.APIError
if errors.As(err, &apiErr) {
    resp := map[string]interface{}{
        "message":    apiErr.Message,
        "status_code": apiErr.ServerCode,
    }
    if len(apiErr.Errors) > 0 {
        resp["errors"] = apiErr.Errors
    }
    output.Print(w, resp, false)
    return
}
```

### 方案 3: 新增 ValidationError 的 Message getter

当前 `GetValidationErrors` 只返回 `Errors` map，没有暴露 `Message`。需要新增：

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

### 方案 4: config/root 命令统一 JSON 输出

将 `cmd/config/config.go` 和 `cmd/root.go` 中的 `fmt.Fprintf(os.Stderr, ...)` 替换为 `output.PrintError(os.Stderr, ...)`。

涉及文件和行：
- `cmd/config/config.go`: 第 63, 74, 88, 97 行
- `cmd/root.go`: 第 77, 83, 89, 95, 102 行

### 方案 5: verbose 模式嵌入 JSON

当前 ResponseError 的 verbose Detail() 追加在 JSON 后面，破坏结构。已在方案 1 中解决：verbose 信息嵌入 JSON 对象的 `body` 字段。

## 改动后的 AI 视角

### 改前

```
# 401 认证失败
stderr: {"error":"api_key 已过期，请重新登录获取"}

# 422 参数校验失败
stderr: {"error":"参数校验失败: map[name:required email:invalid format]"}

# 403 权限不足
stderr: {"error":"API 错误 (403): 无权访问该资源"}

# 502 网关错误 + verbose
stderr: {"error":"服务端返回异常 (HTTP 502)..."}
        HTTP 502 | Content-Type: text/html...
```

### 改后

```
# 401 认证失败
stderr: {"message":"api_key 已过期，请重新登录获取","status_code":401}

# 422 参数校验失败
stderr: {"message":"参数校验失败","status_code":422,"errors":{"name":["required"],"email":["invalid format"]}}

# 403 权限不足
stderr: {"message":"无权访问该资源","status_code":403}

# 502 网关错误
stderr: {"message":"服务端返回异常 (HTTP 502)","status_code":502,"content_type":"text/html"}
```

AI 可以直接 `JSON.parse(stderr)` 获取 `status_code` 和 `errors` 字段，无需 NLP 分析。

## 实现优先级

1. **P0 - APIError 类型** (方案 2): 补全类型体系，是后续方案的基础。改动范围仅 `client.go`，新增 `APIError` struct + 替换一行 `fmt.Errorf`。

2. **P0 - handleAPIError 透传** (方案 1 + 3): 修改 `cmdgen.go` 的 `handleAPIErrorTo`，改为 `output.Print` 输出结构化 JSON。新增 `GetValidationMessage` getter。

3. **P1 - config/root 统一输出** (方案 4): 将非 API 命令的错误也统一为 JSON 格式。低风险改动。

## 测试策略

### handleAPIErrorTo 测试

```go
func TestHandleAPIErrorTo(t *testing.T) {
    tests := []struct {
        name       string
        err        error
        verbose    bool
        wantFields map[string]interface{} // 期望 JSON 中包含的字段
    }{
        {
            name: "unauthorized",
            err:  api.ErrUnauthorized,
            wantFields: map[string]interface{}{
                "message":     "api_key 已过期，请重新登录获取",
                "status_code": float64(401),
            },
        },
        {
            name: "validation_error",
            err: &api.ValidationError{
                Message: "参数校验失败",
                Errors:  map[string]interface{}{"name": []interface{}{"required"}},
            },
            wantFields: map[string]interface{}{
                "message":     "参数校验失败",
                "status_code": float64(422),
                "errors":      map[string]interface{}{"name": []interface{}{"required"}},
            },
        },
        {
            name: "api_error_with_server_fields",
            err: &api.APIError{
                StatusCode: 403,
                Message:    "无权访问",
                ServerCode: 403,
            },
            wantFields: map[string]interface{}{
                "message":     "无权访问",
                "status_code": float64(403),
            },
        },
    }
    // 对每个用例：捕获 stderr 输出 -> json.Unmarshal -> 验证字段
}
```

### APIError 类型测试

```go
func TestAPIError(t *testing.T) {
    err := &APIError{
        StatusCode: 402,
        Message:    "余额不足",
        ServerCode: 402,
        Errors:     map[string]interface{}{"detail": "账户余额为0"},
    }
    // 验证 Error() 字符串
    // 验证 errors.As 匹配
}
```

### 集成测试

用 httptest 模拟服务端返回各种状态码的 JSON，验证 CLI 的 stderr 输出是合法 JSON 且包含服务端字段。

## 实现注意事项

- 透传到 stderr 的 JSON 格式应与服务端 response 的 `message`/`status_code`/`errors` 字段名保持一致，不引入新字段名
- `output.PrintError` 保留不变，用于客户端侧错误（网络、序列化、配置等非 API 错误）
- API 层的错误类型（ValidationError、APIError）只需携带服务端数据，不增加额外字段
- verbose 模式的额外调试信息嵌入 JSON 对象内部，确保 stderr 始终是合法 JSON
- 退出码保持 exit 0/1 不变
- `APIError` 需要相应的 `IsAPIError` 辅助函数
