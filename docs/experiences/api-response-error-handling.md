# 经验: API 响应错误处理

## 场景

CLI 工具调用后端 API，后端可能返回非 JSON 响应（HTML 错误页、502 网关、认证重定向等）。

## 根因

`client.Do()` 直接对响应体做 `json.Decode()`，未检查 Content-Type。当响应为 HTML 时，JSON 解码失败产生 `invalid character '<' looking for beginning of value` 这类不可读错误。

## 解决方案

### 1. 响应处理顺序

重构为：读取 body → 检查状态码 + Content-Type → JSON 解码 → 业务错误处理。

关键判断链：
- 非 2xx + 非 JSON → `ResponseError`（友好提示）
- 2xx + 非 JSON（且 CT 非空）→ `ResponseError`（配置错误提示）
- 空 Content-Type → 尝试 JSON 解码（兼容不规范的服务端）

### 2. ResponseError 类型设计

```go
type ResponseError struct {
    StatusCode  int
    ContentType string
    Body        string // 前 512 字符
    Message     string // 用户友好消息
}
```

- `Error()` 返回友好消息（默认展示）
- `Detail()` 返回调试信息（--verbose 时展示）
- 支持 `errors.As` 解包

### 3. 可测试的错误处理函数

将 `handleAPIError(err)` 拆为 `handleAPIError` + `handleAPIErrorTo(w io.Writer, err, verbose)`，注入 writer 方便测试输出内容。

## 踩坑记录

- 原有测试 handler 未设置 `Content-Type: application/json`，重构后需补上，否则会被新逻辑拦截为非 JSON 响应
- `truncate()` 用 `len(s)` 按字节截断，对中文可能截断到半个字符，但作为调试信息可接受

## 复用价值

任何 Go HTTP 客户端在做 JSON API 调用时，都应在解码前检查 Content-Type，避免用户看到底层解码错误。ResponseError 模式（友好消息 + 详情分离）适用于所有 CLI 工具。
