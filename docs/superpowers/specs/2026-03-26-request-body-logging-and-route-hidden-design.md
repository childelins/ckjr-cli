# Request Body 日志 & Route 命令隐藏 设计文档

> Created: 2026-03-26
> Status: Draft

## 概述

本文档包含两个独立改进：

1. **Q3 - Request/Response Body 日志**: 在 `DoCtx` 的日志中增加 request body 和 response body，始终记录到日志文件，`--verbose` 时同时输出到 stderr。原样记录，不截断不脱敏。
2. **Q4 - Route 命令隐藏**: 将 `route` 命令标记为 Hidden，使其不出现在 `--help` 输出中，防止 AI 通过自发现机制调用内部开发工具。

## Q3: Request/Response Body 日志

### 架构

不引入新组件。仅修改 `internal/api/client.go` 的 `DoCtx` 方法，在现有日志点增加 body 字段。

### 修改点

#### `DoCtx` 方法 (`internal/api/client.go`)

**请求日志** (L85-89): 在 `api_request` 日志中增加 `request_body` 字段。

```go
// 序列化 body 提前到日志之前
var data []byte
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
    "request_body", string(data), // 新增：nil body 时 data 为 nil，输出空字符串
)
```

注意：需要调整代码顺序，将 `json.Marshal(body)` 移到日志语句之前，使 `data` 在日志时可用。

**响应日志** (所有 `api_response` 日志点): 在每个记录响应的日志调用中增加 `response_body` 字段。

涉及的日志点：
- L113-119: 网络请求失败 (无 response body，不添加)
- L129-137: 读取响应体失败 (无 response body，不添加)
- L151-158: 非 JSON 非 2xx 响应 -> 增加 `"response_body", string(bodyBytes)`
- L166-178: 2xx 但非 JSON 响应 -> 增加 `"response_body", string(bodyBytes)`
- L184-192: JSON 解码失败 -> 增加 `"response_body", string(bodyBytes)`
- L198-206: 401 未授权 -> 增加 `"response_body", string(bodyBytes)`
- L210-222: 422 验证失败 -> 增加 `"response_body", string(bodyBytes)`
- L225-233: 其他 4xx/5xx -> 增加 `"response_body", string(bodyBytes)`
- L237-243: 成功响应 -> 增加 `"response_body", string(bodyBytes)`

### 数据流

```
用户调用命令
  -> DoCtx 序列化 body 为 data []byte
  -> slog.Info "api_request" {request_body: string(data)}
  -> http.Do 发送请求
  -> io.ReadAll 读取 bodyBytes
  -> slog.Info/Error "api_response" {response_body: string(bodyBytes)}
  -> 日志写入 ~/.ckjr/logs/YYYY-MM-DD.log (JSON 格式)
  -> --verbose 时同时输出到 stderr (text 格式)
```

### 日志格式示例

日志文件 (JSON):
```json
{"time":"2026-03-26T10:00:00Z","level":"INFO","msg":"api_request","request_id":"abc-123","method":"POST","url":"https://api.example.com/v1/users","request_body":"{\"name\":\"test\"}"}
{"time":"2026-03-26T10:00:01Z","level":"INFO","msg":"api_response","request_id":"abc-123","method":"POST","url":"https://api.example.com/v1/users","status":200,"duration_ms":150,"response_body":"{\"data\":{\"id\":1},\"message\":\"success\",\"status_code\":200}"}
```

### 错误处理

- `body == nil` 时，`data` 为 `nil`，`string(nil)` 输出空字符串，无需特殊处理
- `bodyBytes` 已在读取后可用，所有使用 `bodyBytes` 的日志点均在 `io.ReadAll` 成功之后
- 日志写入失败不影响主流程（slog 默认行为）

## Q4: Route 命令隐藏

### 架构

仅修改 `cmd/route.go`，设置 Cobra 命令的 `Hidden` 属性。

### 修改点

#### `routeCmd` 定义 (`cmd/route.go` L14-17)

```go
var routeCmd = &cobra.Command{
    Use:    "route",
    Short:  "路由配置管理",
    Hidden: true, // 新增：对 --help 不可见，防止 AI 自发现
}
```

### 行为变化

| 场景 | 修改前 | 修改后 |
|------|--------|--------|
| `ckjr-cli --help` | route 出现在 Available Commands | route 不出现 |
| `ckjr-cli route --help` | 正常显示 | 正常显示（仍可使用） |
| `ckjr-cli route import ...` | 正常执行 | 正常执行（仍可使用） |
| AI 通过 `--help` 自发现 | 会发现 route 命令 | 不会发现 route 命令 |

### 不需要修改的部分

- `cmd/root.go` 的 `rootCmd.AddCommand(routeCmd)` 保持不变
- `SKILL.md` 不需要修改（AI 已无法发现该命令）
- `route import` 子命令不需要单独设置 Hidden（父命令隐藏后子命令自然不可见）

## 测试策略

### Q3 Body 日志测试

在 `internal/api/client_test.go` 中新增测试：

1. **TestDoCtx_LogsRequestBody**: 验证 POST 请求的 request body 出现在日志中
2. **TestDoCtx_LogsResponseBody**: 验证成功响应的 response body 出现在日志中
3. **TestDoCtx_LogsResponseBody_OnError**: 验证错误响应的 response body 也出现在日志中
4. **TestDoCtx_NilBody_LogsEmpty**: 验证 body 为 nil 时日志中 request_body 为空字符串

测试方法：使用自定义 slog.Handler 捕获日志记录，验证字段存在及内容。

### Q4 Route Hidden 测试

1. **TestRouteCmd_IsHidden**: 验证 `routeCmd.Hidden == true`
2. 手动验证：`ckjr-cli --help` 输出不含 route

## 实现注意事项

1. **代码顺序调整 (Q3)**: `json.Marshal(body)` 需要在 `api_request` 日志之前执行，需要重新组织 L93-100 和 L85-89 的顺序
2. **GET 请求 (Q3)**: GET 请求通常 `body == nil`，`request_body` 字段会输出空字符串，这是预期行为
3. **大响应体 (Q3)**: 用户已确认不截断，日志文件按天轮转且存储在用户本地，体积增长可接受
4. **一行改动 (Q4)**: Route 隐藏只需增加 `Hidden: true` 一行，改动极小
