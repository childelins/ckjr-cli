# 日志环境区分设计文档

> Created: 2026-03-26
> Status: Draft

## 概述

当前 `ckjr-cli` 日志系统在所有环境下行为一致：固定 `slog.LevelInfo`，始终记录完整的 `request_body` 和 `response_body`。本设计引入编译期环境模式（development / production），通过 `go build -ldflags` 注入，使开发和生产环境在日志级别和内容详细程度上有不同表现。

## 用户决策

| 问题 | 决策 |
|------|------|
| 区分维度 | 日志级别 + 内容详细程度：开发=DEBUG+完整 body，生产=INFO+省略 body |
| 配置方式 | `go build -ldflags` 注入变量，不支持运行时 CLI 更改 |
| `--verbose` 行为 | 保持现有行为（同时输出到 stderr），不受环境模式影响 |

## 环境行为对比

| 维度 | development | production |
|------|------------|------------|
| 日志级别 | `slog.LevelDebug` | `slog.LevelInfo` |
| `request_body` | 记录完整内容 | 不记录 |
| `response_body` | 记录完整内容 | 不记录 |
| 日志文件格式 | JSON（不变） | JSON（不变） |
| `--verbose` | 正常工作（stderr 输出） | 正常工作（stderr 输出） |
| 默认值 | `false`（即 production） | - |

## 架构

改动涉及两个层面：

1. **基础设施层** (`internal/logging/logging.go`) - 新增 `Environment` 类型和 `Init` 签名变更
2. **传输层** (`internal/api/client.go`) - 根据 `Environment` 决定是否记录 body
3. **入口层** (`cmd/root.go`) - 新增 ldflags 变量，传递给 `logging.Init`

```
cmd/root.go
  |
  var Environment = "production"   // -ldflags "-X main.Environment=development" 覆盖
  |
  logging.Init(verbose, baseDir, logging.ParseEnvironment(Environment))
      |
      env == development -> slog.LevelDebug
      env == production  -> slog.LevelInfo
      |
      logging.Environment() -> "development" | "production"
      |
cmdgen.buildSubCommand.Run()
  |
  client.DoCtx(ctx, ...)
      |
      logging.IsDevelopment() -> true/false
      if dev: log request_body, response_body
      if prod: omit body fields
```

## 组件

### 1. internal/logging/logging.go - 新增 Environment 类型

```go
// Environment 日志环境模式
type Environment int

const (
    Production  Environment = iota // 生产环境：INFO 级别，不记录 body
    Development                    // 开发环境：DEBUG 级别，记录完整 body
)

// ParseEnvironment 将字符串解析为 Environment
// 无效值默认为 Production
func ParseEnvironment(s string) Environment {
    if strings.EqualFold(s, "development") || strings.EqualFold(s, "dev") {
        return Development
    }
    return Production
}

// IsDev 返回当前是否为开发环境
// 供 api.Client 等包使用，决定日志详细程度
func IsDev() bool {
    return currentEnv == Development
}

var currentEnv = Production // 默认生产环境
```

`Init` 签名变更：

```go
// Init 初始化日志系统
// env 控制日志级别：development=DEBUG，production=INFO
// verbose=true 时额外输出到 stderr（不受 env 影响）
func Init(verbose bool, baseDir string, env Environment) error {
    currentEnv = env

    level := slog.LevelInfo
    if env == Development {
        level = slog.LevelDebug
    }

    // ... 现有文件创建逻辑 ...

    fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})

    if verbose {
        stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
        slog.SetDefault(slog.New(newMultiHandler(fileHandler, stderrHandler)))
    } else {
        slog.SetDefault(slog.New(fileHandler))
    }

    return nil
}
```

### 2. internal/api/client.go - 条件记录 body

`DoCtx` 方法中 `request_body` 和 `response_body` 的记录改为条件判断：

```go
// api_request 日志
attrs := []interface{}{
    "request_id", requestID,
    "method", method,
    "url", url,
}
if logging.IsDev() {
    attrs = append(attrs, "request_body", string(data))
}
slog.InfoContext(ctx, "api_request", attrs...)

// api_response 日志（成功和失败各一处）
attrs = []interface{}{
    "request_id", requestID,
    "method", method,
    "url", url,
    "status", resp.StatusCode,
    "duration_ms", duration.Milliseconds(),
}
if logging.IsDev() {
    attrs = append(attrs, "response_body", readableJSON(bodyBytes))
}
slog.InfoContext(ctx, "api_response", attrs...)
```

为保持代码简洁，提取辅助函数：

```go
// logAttrs 构建 slog 属性键值对，开发环境附加 body 字段
func logAttrs(requestID, method, url string, status int, durationMs int64, body []byte, errMsg string) []interface{} {
    attrs := []interface{}{
        "request_id", requestID,
        "method", method,
        "url", url,
        "status", status,
        "duration_ms", durationMs,
    }
    if errMsg != "" {
        attrs = append(attrs, "error", errMsg)
    }
    if logging.IsDev() && body != nil {
        attrs = append(attrs, "response_body", readableJSON(body))
    }
    return attrs
}
```

### 3. cmd/root.go - 新增 ldflags 变量

```go
var (
    // 版本信息，通过 ldflags 注入
    Version = "dev"
    // 环境模式，通过 ldflags 注入，可选值：development / production（默认）
    Environment = "production"
)

func initLogging() {
    verbose, _ := rootCmd.Flags().GetBool("verbose")
    homeDir, err := os.UserHomeDir()
    if err != nil {
        fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
        return
    }
    baseDir := filepath.Join(homeDir, ".ckjr")
    env := logging.ParseEnvironment(Environment)
    if err := logging.Init(verbose, baseDir, env); err != nil {
        fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
    }
}
```

构建命令示例：

```bash
# 开发构建
go build -ldflags="-X main.Environment=development" -o ckjr-cli ./cmd/ckjr-cli

# 生产构建（默认，可省略）
go build -ldflags="-X main.Environment=production" -o ckjr-cli ./cmd/ckjr-cli
go build -o ckjr-cli ./cmd/ckjr-cli

# CI/release 构建时同时注入版本和环境
go build -ldflags="-s -w -X main.Version=${VERSION} -X main.Environment=production" -o dist/ckjr-cli ./cmd/ckjr-cli
```

## 数据流

### 开发环境 - 正常请求

```json
{"time":"...","level":"DEBUG","msg":"api_request","request_id":"a1b2...","method":"GET","url":"https://...","request_body":"{\"page\":1,\"limit\":5}"}
{"time":"...","level":"INFO","msg":"api_response","request_id":"a1b2...","method":"GET","url":"https://...","status":200,"duration_ms":142,"response_body":"{\"data\":{...},\"message\":\"ok\"}"}
```

### 生产环境 - 正常请求

```json
{"time":"...","level":"INFO","msg":"api_request","request_id":"a1b2...","method":"GET","url":"https://..."}
{"time":"...","level":"INFO","msg":"api_response","request_id":"a1b2...","method":"GET","url":"https://...","status":200,"duration_ms":142}
```

### 生产环境 - 请求失败（ERROR 日志同样不包含 body）

```json
{"time":"...","level":"INFO","msg":"api_request","request_id":"a1b2...","method":"POST","url":"https://..."}
{"time":"...","level":"ERROR","msg":"api_response","request_id":"a1b2...","method":"POST","url":"https://...","status":502,"duration_ms":89,"error":"服务端返回异常 (HTTP 502)..."}
```

## 文件变更清单

| 操作 | 文件 | 变更内容 |
|------|------|---------|
| Modify | `internal/logging/logging.go` | 新增 `Environment` 类型、`ParseEnvironment()`、`IsDev()`、`currentEnv` 变量；`Init` 新增 `env` 参数，根据 env 设置日志级别 |
| Modify | `internal/logging/logging_test.go` | 新增 `ParseEnvironment`、`IsDev`、`Init` 环境级别相关测试 |
| Modify | `internal/api/client.go` | `DoCtx` 中 `request_body`/`response_body` 改为条件记录，提取 `logAttrs` 辅助函数 |
| Modify | `internal/api/client_test.go` | 更新现有 body 相关测试仅在开发模式下断言；新增生产模式省略 body 测试 |
| Modify | `cmd/root.go` | 新增 `Environment` 变量（ldflags 注入），`initLogging` 传递 env 参数 |
| Modify | `.github/workflows/release.yml` | release 构建中添加 `-X main.Environment=production` |

## 错误处理

无新增错误场景。`ParseEnvironment` 对无效值静默降级为 `Production`，与现有日志错误处理策略一致：日志相关错误不阻塞命令执行。

## 测试策略

### 新增测试

**`internal/logging/logging_test.go`**：

1. `TestParseEnvironment_Development` - "development" / "dev" / "Development" 均返回 Development
2. `TestParseEnvironment_Production` - "production" / "prod" / 空字符串 / 任意值均返回 Production
3. `TestIsDev_AfterInit` - Init 传入 Development 后 IsDev() 返回 true
4. `TestIsDev_DefaultProduction` - 未调用 Init 或传入 Production 时 IsDev() 返回 false
5. `TestInit_DevLogLevel` - Development 模式下 DEBUG 级别日志被记录到文件
6. `TestInit_ProdLogLevel` - Production 模式下 DEBUG 级别日志不被记录

### 修改现有测试

**`internal/api/client_test.go`**：

- `TestDoCtx_LogsRequestBody` / `TestDoCtx_LogsResponseBody` / `TestDoCtx_ErrorLogsResponseBody` / `TestDoCtx_NilBody`：改为在 Development 模式下断言 body 存在
- 新增 `TestDoCtx_ProdOmitsBody`：Production 模式下断言日志不含 body 字段

**`internal/cmdgen/cmdgen_test.go`**：

- 现有测试调用 `logging.Init` 的地方需补充 env 参数（传 `logging.Production`）

## 实现注意事项

1. **遵循现有 ldflags 模式** - `Version` 变量已在 `cmd/root.go` 中通过 ldflags 注入，`Environment` 采用完全相同的模式，保持一致性。

2. **`currentEnv` 包级变量** - 使用包级变量 `currentEnv` 存储环境状态，供 `IsDev()` 读取。这是最简方案，避免引入全局配置结构体或单例。`Init` 在 `cobra.OnInitialize` 中调用，早于任何 API 请求，时序安全。

3. **slog 键值对展开** - slog 的 `InfoContext(ctx, msg, args...)` 接受交替的 key-value 对。当 body 字段被条件省略时，直接在 `args` 切片中不添加即可，slog 正常处理可变数量参数。

4. **测试中显式设置环境** - 测试通过 `logging.Init(verbose, tmpDir, logging.Development)` 或 `logging.Production` 显式控制，不依赖全局状态。测试前后注意重置 `currentEnv` 或使用子测试隔离。

5. **Production 构建的 CI/CD** - `.github/workflows/release.yml` 需更新构建命令，添加 `-X main.Environment=production`。本地 `go build` 默认即 production，无需额外配置。
