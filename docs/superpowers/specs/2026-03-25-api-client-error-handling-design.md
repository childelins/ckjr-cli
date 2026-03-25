# API Client 错误处理改进设计文档

> Created: 2026-03-25
> Status: Draft

## 概述

`ckjr agent list` 等命令在 API 返回非 JSON 响应（HTML 错误页、登录重定向等）时报出不可读的错误信息 `解析响应失败: invalid character '<' looking for beginning of value`。根本原因是 `api.Client.Do()` 方法的处理顺序错误：先执行 JSON 解码，再检查 HTTP 状态码，导致非 JSON 响应直接在解码阶段失败，后续所有状态码判断均无法执行。

本设计改进 `Do()` 方法的响应处理流程，增加 Content-Type 校验、状态码前置检查和友好错误信息，同时支持 `--verbose` 模式输出调试信息。

## 用户决策

| 问题 | 决策 |
|------|------|
| 测试文件目录结构 | 保持 Go 惯例不变，_test.go 与源码同目录，不做重组 |
| 错误信息展示 | 默认简洁 + `--verbose` flag 显示完整调试信息 |
| 修复范围 | 完整方案：Content-Type 校验 + 非 2xx 状态码提前返回 + 友好错误信息 |

## 架构

改动集中在两个层面：

1. **传输层** (`internal/api/client.go`) - 响应处理流程重构
2. **表现层** (`internal/cmdgen/cmdgen.go`, `cmd/root.go`) - `--verbose` flag 透传与错误展示

```
用户 -> cobra Command -> cmdgen.Run -> api.Client.Do()
                |                            |
          --verbose flag              响应处理流程重构
                |                            |
          handleAPIError() <---- 新错误类型 ResponseError
```

## 组件

### 1. api.Client.Do() 响应处理流程重构

**当前流程（有缺陷）**：
```
resp := http.Do(req)
json.Decode(resp.Body) -> 失败则直接返回，无法继续
检查 StatusCode 401
检查 StatusCode 422
检查 StatusCode >= 400
解析 data 到 result
```

**改进后流程**：
```
resp := http.Do(req)
读取 resp.Body 到 []byte（便于多次使用）
检查 StatusCode 非 2xx -> 返回 ResponseError（含状态码、Content-Type、响应体摘要）
检查 Content-Type 是否包含 application/json -> 不是则返回 ResponseError
json.Unmarshal 响应体
检查业务状态码（401/422/其他）
解析 data 到 result
```

### 2. 新增 ResponseError 错误类型

```go
// ResponseError 非预期响应错误（非 JSON、非 2xx 等）
type ResponseError struct {
    StatusCode  int
    ContentType string
    Body        string // 响应体前 512 字符
    Message     string // 用户友好的错误信息
}
```

用途：
- 承载调试信息（状态码、Content-Type、响应体摘要）
- `Error()` 方法返回简洁的用户友好信息
- `--verbose` 模式下通过 `Detail()` 方法获取完整调试信息

### 3. --verbose 全局 Flag

在 `cmd/root.go` 的 `rootCmd.PersistentFlags()` 中添加 `--verbose` flag，与现有 `--pretty` 并列。

`cmdgen.buildSubCommand` 在调用 `handleAPIError` 时传入 verbose 参数，控制输出详细程度。

### 4. handleAPIError 增强

```go
func handleAPIError(err error, verbose bool) {
    // 现有逻辑保持不变（Unauthorized、Validation）

    var respErr *api.ResponseError
    if errors.As(err, &respErr) {
        output.PrintError(os.Stderr, respErr.Error())
        if verbose {
            fmt.Fprintf(os.Stderr, "  HTTP %d | Content-Type: %s\n", respErr.StatusCode, respErr.ContentType)
            fmt.Fprintf(os.Stderr, "  响应体: %s\n", respErr.Body)
        }
        return
    }

    output.PrintError(os.Stderr, err.Error())
}
```

## 数据流

### 正常请求流程（无变化）
```
Client.Do("GET", "/api/agents", input, &result)
  -> HTTP 200, Content-Type: application/json
  -> json.Unmarshal 成功
  -> 解析 data 到 result
  -> return nil
```

### 非 JSON 响应（改进后）
```
Client.Do("GET", "/api/agents", input, &result)
  -> HTTP 502, Content-Type: text/html
  -> StatusCode 502 >= 400，非 2xx
  -> 返回 ResponseError{StatusCode: 502, ContentType: "text/html", Body: "<html>...", Message: "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试"}
```

### 认证重定向（改进后）
```
Client.Do("GET", "/api/agents", input, &result)
  -> HTTP 302 -> 200, Content-Type: text/html（登录页）
  -> StatusCode 200，但 Content-Type 不含 application/json
  -> 返回 ResponseError{StatusCode: 200, ContentType: "text/html", Body: "<html>...", Message: "服务端返回非 JSON 响应，可能是 base_url 配置错误或需要重新认证"}
```

## 错误处理

### 错误分类与用户提示

| 场景 | 简洁信息 | verbose 附加信息 |
|------|---------|-----------------|
| 非 2xx + 非 JSON | 服务端返回异常 (HTTP {code})，请检查 base_url 配置或稍后重试 | 状态码、Content-Type、响应体前 512 字符 |
| 2xx + 非 JSON | 服务端返回非 JSON 响应，可能是 base_url 配置错误或需要重新认证 | 同上 |
| 非 2xx + JSON | 保持现有逻辑（读取 apiResp.Message） | 无额外信息（已由 API 错误信息覆盖） |
| 401 Unauthorized | 保持现有：api_key 已过期，请重新登录获取 | 无变化 |
| 422 Validation | 保持现有：参数校验失败 + 字段详情 | 无变化 |
| 网络错误 | 保持现有：请求失败: {error} | 无变化 |

### Content-Type 校验规则

检查 `Content-Type` header 是否包含 `application/json`（使用 `strings.Contains`，兼容 `application/json; charset=utf-8` 等变体）。如果 Content-Type 为空，仍尝试 JSON 解码（某些简单 API 可能不设置 Content-Type）。

## 测试策略

所有测试使用 `httptest.NewServer` 模拟服务端响应，保持包内测试（与源码同目录）。

### 新增测试用例

**`internal/api/client_test.go`**：

1. `TestClientDo_HTMLResponse` - 服务端返回 HTML（模拟网关错误页），验证返回 `ResponseError`
2. `TestClientDo_Non2xxWithJSON` - 服务端返回 500 + JSON 错误体，验证正确解析错误信息
3. `TestClientDo_Non2xxWithHTML` - 服务端返回 502 + HTML，验证返回 `ResponseError` 含状态码
4. `TestClientDo_EmptyContentType` - Content-Type 为空但响应体是合法 JSON，验证正常解析
5. `TestResponseError_Error` - 验证 `Error()` 返回用户友好信息
6. `TestResponseError_Detail` - 验证 `Detail()` 返回完整调试信息
7. `TestIsResponseError` - 验证 `errors.As` 类型断言

**`internal/cmdgen/cmdgen_test.go`**：

8. `TestHandleAPIError_ResponseError` - 验证非 verbose 模式输出简洁信息
9. `TestHandleAPIError_ResponseError_Verbose` - 验证 verbose 模式输出调试信息

### 现有测试影响

- `TestClientDo` - 无需修改（正常 JSON 响应流程不变）
- `TestClientUnauthorized` - 需微调：当前 mock 返回 401 + JSON，改进后仍正常命中 401 分支

## 实现注意事项

1. **resp.Body 只能读取一次** - 改用 `io.ReadAll` 读到 `[]byte`，然后用 `json.Unmarshal` 替代 `json.NewDecoder`。这对性能无影响（CLI 单次请求，响应体通常较小）。

2. **响应体截断** - `ResponseError.Body` 最多保留 512 字符，避免大 HTML 页面占用内存和输出。

3. **向后兼容** - 现有的 `ErrUnauthorized`、`ValidationError` 保持不变。`ResponseError` 是新增类型，不影响现有 `errors.Is` / `errors.As` 判断。

4. **--verbose flag 作用域** - 作为 `PersistentFlags` 注册在 rootCmd 上，所有子命令均可使用。需要将 verbose 值传递到 `handleAPIError`，可通过 `cmd.Flags().GetBool("verbose")` 获取。

5. **先检查状态码，再检查 Content-Type** - 因为非 2xx 的 HTML 错误页（如 502）比 2xx 的 HTML 响应更常见，先处理状态码能覆盖大部分情况。

6. **不实现重试机制** - 虽然用户选择了"完整方案"，但重试涉及幂等性判断、超时配置等复杂度，不在本次范围内。核心改进聚焦于错误处理流程和信息展示。如需重试可作为后续独立 spec。
