# 生产环境静默 HTTP 请求日志设计文档

> Created: 2026-03-29
> Status: Draft

## 概述

当前生产环境日志级别为 `slog.LevelInfo`，会记录 `api_request`（INFO）和成功的 `api_response`（INFO）。用户希望生产环境不记录任何 HTTP 请求日志，只保留 error 级别的错误日志，避免客户端 CLI 暴露过多信息。

方案：将生产环境的日志文件级别从 `slog.LevelInfo` 提升为 `slog.LevelError`，同时 `--verbose` 的 stderr handler 使用独立的 INFO 级别（不受 env 影响），确保 AI 通过 `--verbose` 仍能获取请求/响应状态反馈。

## 分析

### 当前日志分布

| 日志消息 | 级别 | 触发条件 |
|---------|------|---------|
| `api_request` | INFO | 每次 API 请求 |
| `api_response` | INFO | 请求成功 (2xx) |
| `api_response` | ERROR | 请求失败 (网络错误/非2xx/JSON解析失败等) |

### 关键发现

生产代码中所有 `slog.Info` 调用仅存在于 `internal/api/client.go` 的两处（`api_request` 和成功 `api_response`）。没有其他业务逻辑使用 INFO 级别日志。因此将生产环境文件级别改为 `slog.LevelError` 不会产生任何副作用。

### 方案对比

| 方案 | 做法 | 改动量 | 风险 |
|------|------|-------|------|
| A. 提升日志级别到 ERROR | 改 `logging.go` 两行 | 极小 | 无 |
| B. 按日志消息过滤 | 自定义 handler 判断 msg 字段 | 较大 | 过滤规则硬编码 |
| C. 新增环境枚举 | 增加 Silent 模式 | 中等 | 增加复杂度 |

选择方案 A：最简洁，完全复用现有机制。

## 环境行为对比

| 维度 | development | production (改后) |
|------|------------|------------------|
| 日志文件级别 | `slog.LevelDebug` | `slog.LevelError` (原 INFO) |
| `api_request` 日志文件 | 记录（含 body） | 不记录 |
| `api_response` (成功) 日志文件 | 记录（含 body） | 不记录 |
| `api_response` (失败) 日志文件 | 记录（含 body） | 记录（不含 body） |
| `--verbose` stderr 级别 | `slog.LevelInfo` | `slog.LevelInfo`（统一） |
| `--verbose` stderr 输出 | 请求/响应（不含 body） | 请求/响应（不含 body） |

## 架构

改动涉及一个文件：

```
internal/logging/logging.go
  |
  Init() 中:
    file handler:
      env == production -> slog.LevelError  (原 slog.LevelInfo)
      env == development -> slog.LevelDebug (不变)
    verbose stderr handler:
      始终使用 slog.LevelInfo（开发环境为 slog.LevelDebug）
      不受 env 的 file level 影响
```

无需修改 `api/client.go`、`cmd/root.go` 或任何其他文件。

## 组件

### internal/logging/logging.go - 修改日志级别

```go
func Init(verbose bool, baseDir string, env Environment) error {
    currentEnv = env
    // ...
    level := slog.LevelError       // 生产环境文件只记录 ERROR
    if env == Development {
        level = slog.LevelDebug    // 开发环境记录所有级别
    }

    fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: level})

    if verbose {
        verboseLevel := slog.LevelInfo // verbose stderr 统一 INFO，不受 env 影响
        stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: verboseLevel})
        slog.SetDefault(slog.New(newMultiHandler(fileHandler, stderrHandler)))
    } else {
        slog.SetDefault(slog.New(fileHandler))
    }
    return nil
}
```

## 数据流

### 生产环境 - 正常请求（改后，无 --verbose）

```
# 无日志输出
```

### 生产环境 - 正常请求（改后，--verbose）

```
time=... level=INFO msg=api_request request_id=a1b2... method=GET url=https://...
time=... level=INFO msg=api_response request_id=a1b2... method=GET url=https://... status=200 duration_ms=142
```

### 生产环境 - 请求失败（改后）

```
# 日志文件
{"time":"...","level":"ERROR","msg":"api_response","request_id":"a1b2...","method":"POST","url":"https://...","status":502,"duration_ms":89,"error":"服务端返回异常 (HTTP 502)..."}

# --verbose stderr 同样输出 ERROR
```

### 开发环境 - 正常请求（改后，--verbose）

```
time=... level=INFO msg=api_request request_id=a1b2... method=GET url=https://...
time=... level=INFO msg=api_response request_id=a1b2... method=GET url=https://... status=200 duration_ms=142
```

### 开发环境 - 日志文件（不变）

```
{"time":"...","level":"DEBUG","msg":"api_request","request_id":"a1b2...","method":"GET","url":"https://...","request_body":"{...}"}
{"time":"...","level":"INFO","msg":"api_response","request_id":"a1b2...","method":"GET","url":"https://...","status":200,"duration_ms":142,"response_body":"{...}"}
```

## 文件变更清单

| 操作 | 文件 | 变更内容 |
|------|------|---------|
| Modify | `internal/logging/logging.go` | Production 日志级别从 `slog.LevelInfo` 改为 `slog.LevelError`；verbose stderr 使用独立级别 |
| Modify | `internal/logging/logging_test.go` | `TestInit_ProdLogLevel` 断言调整：INFO 日志不再被记录到文件；verbose 测试更新 |

## 错误处理

无新增错误场景。这是纯配置调整。

## 测试策略

### 修改现有测试

**`internal/logging/logging_test.go`**：

1. `TestInit_ProdLogLevel` - 调整断言：Production 模式下 INFO 级别日志不再被记录到文件，ERROR 级别日志仍被记录
2. Verbose 相关测试 - 验证 Production verbose stderr 输出 INFO 级别日志

### 不需要修改的测试

**`internal/api/client_test.go`** - 所有现有测试在 Development 模式下运行，行为不受影响。

## 实现注意事项

1. **verbose handler 统一 INFO** - `--verbose` 的 stderr handler 固定使用 `slog.LevelInfo`，不受 env 影响。开发和生产环境的 verbose 输出内容一致（请求/响应状态，不含 body）。

2. **日志文件几乎为空** - 正常使用时生产环境日志文件将只有 ERROR 日志。日志文件仍会被创建，但内容极少。

3. **body 不在 verbose 中** - 生产环境 verbose stderr 使用 INFO 级别，不包含 request_body 和 response_body（这些在 DEBUG 级别），因为 body 由 AI 自己构造，无需反馈。
