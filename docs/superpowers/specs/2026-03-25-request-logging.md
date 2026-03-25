# Request Logging 设计文档

> Created: 2026-03-25
> Status: Draft

## 概述

`ckjr` CLI 当前缺少请求日志能力，无法追踪 API 请求的完整生命周期。本设计为每次命令调用生成唯一 requestId（UUID），通过 `log/slog` 结构化日志记录请求详情（method, url, status, duration, error），默认写入 `~/.ckjr/logs/` 日志文件（JSON 格式，按天轮转），`--verbose` 开启时同时输出到 stderr。

## 用户决策

| 问题 | 决策 |
|------|------|
| requestId 生成方式 | 客户端 `crypto/rand` 生成 UUID v4，每次命令调用生成一个，通过 `context.Context` 透传，仅本地使用不发服务端 |
| 日志输出目标 | 仅写日志文件 `~/.ckjr/logs/`（JSON 格式，按天轮转） |
| 触发机制 | 默认始终写日志文件，`--verbose` 控制是否同时输出到 stderr |
| 日志库 | Go 标准库 `log/slog` 结构化日志 |
| 日志记录内容 | requestId, method, url, status, duration, error(if any) |

## 架构

改动涉及四个层面：

1. **基础设施层** (`internal/logging/`) - 新增，日志初始化、requestId 生成
2. **传输层** (`internal/api/client.go`) - `Do()` 方法接收 context，记录请求日志
3. **编排层** (`internal/cmdgen/cmdgen.go`) - 生成 requestId，构建 context，传递给 Client
4. **入口层** (`cmd/root.go`) - 初始化日志系统

```
cmd/root.go init()
    |
    logging.Init() -> 创建 ~/.ckjr/logs/ 目录，配置 slog handler
    |
cobra Command.Run()
    |
    cmdgen.buildSubCommand.Run()
        |
        logging.NewRequestID() -> UUID v4
        logging.WithRequestID(ctx, id) -> context 注入
        |
        client.DoCtx(ctx, method, path, body, result)
            |
            slog.InfoContext(ctx, "request", "request_id", id, "method", method, "url", url)
            resp, err := http.Do(req)
            slog.InfoContext(ctx, "response", "request_id", id, "status", status, "duration", dur, "error", err)
```

## 组件

### 1. internal/logging/logging.go - 日志基础设施

```go
package logging

import (
    "context"
    "crypto/rand"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
    "time"
)

type ctxKey struct{}

// Init 初始化日志系统，创建日志目录和 handler
// verbose=true 时额外添加 stderr handler
func Init(verbose bool) error {
    logDir := filepath.Join(homeDir(), ".ckjr", "logs")
    if err := os.MkdirAll(logDir, 0755); err != nil {
        return fmt.Errorf("创建日志目录失败: %w", err)
    }

    filename := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
    file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("打开日志文件失败: %w", err)
    }

    fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})

    if verbose {
        stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
        slog.SetDefault(slog.New(multiHandler(fileHandler, stderrHandler)))
    } else {
        slog.SetDefault(slog.New(fileHandler))
    }

    return nil
}

// NewRequestID 生成 UUID v4
func NewRequestID() string {
    var uuid [16]byte
    _, _ = rand.Read(uuid[:])
    uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
    uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 1
    return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
        uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// WithRequestID 将 requestId 注入 context
func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, ctxKey{}, id)
}

// RequestIDFrom 从 context 提取 requestId
func RequestIDFrom(ctx context.Context) string {
    if id, ok := ctx.Value(ctxKey{}).(string); ok {
        return id
    }
    return ""
}
```

### 2. internal/logging/multi_handler.go - 多 handler 分发

`--verbose` 模式需要同时写文件和 stderr，需要一个简单的 multiHandler 将日志记录分发到多个 `slog.Handler`。

```go
// multiHandler 将日志分发到多个 handler
type multiHandlerImpl struct {
    handlers []slog.Handler
}

func multiHandler(handlers ...slog.Handler) slog.Handler {
    return &multiHandlerImpl{handlers: handlers}
}

func (h *multiHandlerImpl) Enabled(ctx context.Context, level slog.Level) bool { ... }
func (h *multiHandlerImpl) Handle(ctx context.Context, r slog.Record) error { ... }
func (h *multiHandlerImpl) WithAttrs(attrs []slog.Attr) slog.Handler { ... }
func (h *multiHandlerImpl) WithGroup(name string) slog.Handler { ... }
```

### 3. internal/api/client.go - 新增 DoCtx 方法

保留现有 `Do()` 签名不变（向后兼容），新增 `DoCtx()` 接收 `context.Context`。

```go
// DoCtx 执行 API 请求（带 context，支持日志追踪）
func (c *Client) DoCtx(ctx context.Context, method, path string, body interface{}, result interface{}) error {
    requestID := logging.RequestIDFrom(ctx)
    url := c.baseURL + path

    slog.InfoContext(ctx, "api_request",
        "request_id", requestID,
        "method", method,
        "url", url,
    )

    start := time.Now()
    err := c.do(method, path, body, result) // 提取现有逻辑到私有方法
    duration := time.Since(start)

    if err != nil {
        slog.ErrorContext(ctx, "api_response",
            "request_id", requestID,
            "method", method,
            "url", url,
            "duration_ms", duration.Milliseconds(),
            "error", err.Error(),
        )
    } else {
        slog.InfoContext(ctx, "api_response",
            "request_id", requestID,
            "method", method,
            "url", url,
            "status", "ok",
            "duration_ms", duration.Milliseconds(),
        )
    }

    return err
}

// Do 执行 API 请求（向后兼容，无日志追踪）
func (c *Client) Do(method, path string, body interface{}, result interface{}) error {
    return c.do(method, path, body, result)
}
```

重构策略：将现有 `Do()` 的核心逻辑提取到私有方法 `do()`，`Do()` 直接调用 `do()`，`DoCtx()` 在 `do()` 前后添加日志。这样现有 `Do()` 签名和行为完全不变，测试无需修改。

但 `do()` 内部的 HTTP 请求也需要传递 context 以支持 `http.NewRequestWithContext`，因此私有方法签名为：

```go
func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error
```

`Do()` 传入 `context.Background()`，`DoCtx()` 传入调用者的 ctx。

需要将 `http.NewRequest` 改为 `http.NewRequestWithContext(ctx, ...)`，这是唯一的行为变化，对现有测试无影响（`context.Background()` 行为等同于无 context）。

额外记录 HTTP status code：`do()` 方法目前不返回 status code。可以通过以下方式获取：
- 在 `DoCtx()` 中直接执行 HTTP 请求并记录（不提取 `do()`）
- 或让 `do()` 返回额外信息

考虑到简洁性，采用第一种方案：`DoCtx()` 包含完整逻辑（复制自 `Do()`），`Do()` 调用 `DoCtx(context.Background(), ...)`。这样 `DoCtx()` 可以直接访问 `resp.StatusCode`。

最终方案：

```go
// Do 向后兼容，内部调用 DoCtx
func (c *Client) Do(method, path string, body interface{}, result interface{}) error {
    return c.DoCtx(context.Background(), method, path, body, result)
}

// DoCtx 执行 API 请求（带 context 和日志）
func (c *Client) DoCtx(ctx context.Context, method, path string, body interface{}, result interface{}) error {
    requestID := logging.RequestIDFrom(ctx)
    url := c.baseURL + path

    slog.InfoContext(ctx, "api_request",
        "request_id", requestID,
        "method", method,
        "url", url,
    )

    start := time.Now()

    // ... 现有请求逻辑（用 http.NewRequestWithContext）...

    duration := time.Since(start)

    slog.InfoContext(ctx, "api_response",
        "request_id", requestID,
        "method", method,
        "url", url,
        "status", resp.StatusCode,
        "duration_ms", duration.Milliseconds(),
        "error", errMsg, // 如有
    )

    return err
}
```

### 4. internal/cmdgen/cmdgen.go - 生成 requestId 并注入 context

```go
// buildSubCommand.Run 内部变更
func(cmd *cobra.Command, args []string) {
    // ... 现有输入处理逻辑 ...

    // 生成 requestId，注入 context
    ctx := context.Background()
    requestID := logging.NewRequestID()
    ctx = logging.WithRequestID(ctx, requestID)

    // 调用 DoCtx 替代 Do
    if err := client.DoCtx(ctx, route.Method, route.Path, input, &result); err != nil {
        handleAPIError(err, verbose)
        os.Exit(1)
    }

    // ... 输出逻辑 ...
}
```

### 5. cmd/root.go - 初始化日志系统

```go
func init() {
    // 现有 flag 注册 ...

    // cobra.OnInitialize 在命令执行前初始化
    cobra.OnInitialize(initLogging)
}

func initLogging() {
    verbose, _ := rootCmd.Flags().GetBool("verbose")
    // 日志初始化失败不阻塞命令执行，仅输出警告
    if err := logging.Init(verbose); err != nil {
        fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
    }
}
```

## 数据流

### 正常请求
```
ckjr agent list
  -> logging.Init(verbose=false) -> 打开 ~/.ckjr/logs/2026-03-25.log
  -> requestID = "a1b2c3d4-..."
  -> ctx = WithRequestID(ctx, requestID)
  -> client.DoCtx(ctx, "GET", "/api/agents", ...)
     -> slog: {"level":"INFO","msg":"api_request","request_id":"a1b2c3d4-...","method":"GET","url":"https://..."}
     -> HTTP 200 OK
     -> slog: {"level":"INFO","msg":"api_response","request_id":"a1b2c3d4-...","status":200,"duration_ms":142}
  -> output.Print(result)
```

### 请求失败
```
ckjr agent list --verbose
  -> logging.Init(verbose=true) -> 打开日志文件 + stderr handler
  -> requestID = "e5f6g7h8-..."
  -> client.DoCtx(ctx, ...)
     -> slog(file+stderr): {"level":"INFO","msg":"api_request",...}
     -> HTTP 502
     -> slog(file+stderr): {"level":"ERROR","msg":"api_response","request_id":"e5f6g7h8-...","status":502,"duration_ms":89,"error":"服务端返回异常..."}
  -> handleAPIError(err, verbose=true)
```

### 日志文件查询
```
# 按 requestId 查询
grep "a1b2c3d4" ~/.ckjr/logs/2026-03-25.log

# 查所有错误
grep '"level":"ERROR"' ~/.ckjr/logs/2026-03-25.log
```

## 日志文件格式

路径：`~/.ckjr/logs/YYYY-MM-DD.log`

每行一条 JSON 记录（由 `slog.NewJSONHandler` 生成）：

```json
{"time":"2026-03-25T14:30:00.000+08:00","level":"INFO","msg":"api_request","request_id":"a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d","method":"GET","url":"https://api.example.com/api/agents"}
{"time":"2026-03-25T14:30:00.142+08:00","level":"INFO","msg":"api_response","request_id":"a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d","method":"GET","url":"https://api.example.com/api/agents","status":200,"duration_ms":142}
```

按天轮转：每天一个文件，文件名为日期。旧文件不自动清理（CLI 日志量极小，用户可手动删除）。

## 错误处理

| 场景 | 处理方式 |
|------|---------|
| 日志目录创建失败（权限不足等） | `logging.Init` 返回 error，`initLogging` 输出警告到 stderr，命令正常执行（日志降级为无输出） |
| 日志文件打开失败 | 同上 |
| `crypto/rand.Read` 失败 | 极端罕见，使用零值 UUID，不阻塞请求 |
| slog 写入失败（磁盘满等） | slog 内部静默处理，不影响主流程 |

设计原则：日志是辅助功能，任何日志相关错误不应阻塞命令的正常执行。

## 测试策略

### 新增测试

**`internal/logging/logging_test.go`**：

1. `TestNewRequestID_Format` - 验证生成的 ID 是合法 UUID v4 格式
2. `TestNewRequestID_Unique` - 生成多个 ID 验证互不重复
3. `TestWithRequestID_RoundTrip` - 写入 context 后能正确读取
4. `TestRequestIDFrom_Empty` - 无 requestId 的 context 返回空字符串
5. `TestInit_CreatesLogDir` - 验证初始化创建日志目录
6. `TestInit_CreatesLogFile` - 验证日志文件按天命名
7. `TestMultiHandler_WritesToAll` - 验证日志同时写入多个目标

**`internal/api/client_test.go`**：

8. `TestDoCtx_LogsRequest` - 验证 DoCtx 记录请求日志（通过自定义 handler 捕获）
9. `TestDoCtx_LogsError` - 验证错误请求记录 ERROR 级别日志
10. `TestDoCtx_RequestIDInLog` - 验证日志中包含 requestId
11. `TestDoCtx_Duration` - 验证日志中包含 duration_ms
12. `TestDo_BackwardCompatible` - 验证 Do() 行为不变

**`internal/cmdgen/cmdgen_test.go`**：

13. `TestBuildSubCommand_GeneratesRequestID` - 验证命令执行时生成 requestId

### 现有测试影响

- `internal/api/client_test.go` 中的现有测试全部通过 `Do()` 调用，`Do()` 内部调用 `DoCtx(context.Background(), ...)`，行为等价，无需修改
- `internal/cmdgen/cmdgen_test.go` 现有测试使用 mock clientFactory，不涉及日志，无需修改

## 实现注意事项

1. **零新依赖** - 全部使用 Go 标准库：`crypto/rand`（UUID）、`log/slog`（日志）、`context`（透传）。不引入第三方库。

2. **`Do()` 向后兼容** - `Do()` 签名不变，内部调用 `DoCtx(context.Background(), ...)`。所有现有调用方无需修改。新代码使用 `DoCtx()` 获取日志能力。

3. **`http.NewRequestWithContext`** - 将现有 `http.NewRequest` 改为 `http.NewRequestWithContext`，传入调用者的 context。这使得未来可以支持请求超时控制（通过 `context.WithTimeout`）。

4. **日志不阻塞主流程** - `logging.Init` 失败只输出警告，不影响命令执行。slog 写入异常由 slog 内部处理（静默丢弃）。

5. **不发送 requestId 到服务端** - requestId 仅用于本地日志关联，不添加到 HTTP header。未来如需服务端关联，可在 header 中添加 `X-Request-ID`，但不在本次范围内。

6. **日志文件不自动清理** - CLI 工具日志量极小（每次命令 2 条日志），长期累积也不会占用大量磁盘。暂不实现清理策略，用户可手动 `rm ~/.ckjr/logs/*.log`。

7. **multiHandler 实现** - `slog` 标准库不提供多 handler 分发，需自行实现。实现约 30 行，逻辑简单：遍历所有 handler 依次调用。

8. **实现顺序建议**：
   - Phase 1: `internal/logging/` 包（requestId + Init）+ 测试
   - Phase 2: `internal/api/client.go` 新增 `DoCtx()` + 测试
   - Phase 3: `internal/cmdgen/cmdgen.go` 集成 + 测试
   - Phase 4: `cmd/root.go` 初始化 + 端到端验证
