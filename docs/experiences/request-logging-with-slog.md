---
name: request-logging-with-slog
project: ckjr-cli
created: 2026-03-25
tags: [日志, slog, requestId, context, Go]
---

# Go CLI 请求日志：slog + requestId + context 透传

## 决策

| 决策点 | 选择 | 原因 |
|--------|------|------|
| 日志库 | 标准库 `log/slog` | Go 1.21+ 内置，零依赖，结构化日志 |
| requestId 生成 | `crypto/rand` UUID v4 | 标准库，无需第三方 UUID 包 |
| requestId 传递 | `context.Context` value | Go 惯例，与 `http.NewRequestWithContext` 天然集成 |
| 日志输出 | JSON 文件 `~/.ckjr/logs/YYYY-MM-DD.log` | 结构化便于 grep，按天自然轮转 |
| verbose 模式 | 自实现 multiHandler | slog 标准库不提供多 handler 分发 |
| 向后兼容 | `Do()` 委托 `DoCtx()` | 现有调用方零修改，新代码用 `DoCtx()` |

## 坑点预警

- **slog multiHandler 不是标准库功能**: 需自行实现 ~40 行。`Handle` 中必须用 `r.Clone()` 避免并发问题。`WithAttrs`/`WithGroup` 必须返回新的 multiHandler 实例。

- **Init 的 baseDir 参数化**: 不要硬编码 `~/.ckjr`，接收 `baseDir` 参数。生产传 `filepath.Join(homeDir, ".ckjr")`，测试传 `t.TempDir()`。否则测试会污染用户目录。

- **日志不阻塞主流程**: `logging.Init` 失败只输出 stderr 警告后 return，不 `os.Exit`。slog 写入失败由 slog 内部静默处理。CLI 工具的日志是辅助功能，不能因为日志失败导致命令不可用。

- **DoCtx 中日志记录分散**: 每个 error return 点前都要记一条 ERROR 日志，成功路径在最后记 INFO。容易遗漏某个分支。建议用 review 检查所有 return 路径。

## 复用模式

```go
// requestId 生成 + context 注入（在命令层）
ctx := context.Background()
requestID := logging.NewRequestID()
ctx = logging.WithRequestID(ctx, requestID)
client.DoCtx(ctx, method, path, body, &result)

// 日志初始化（在 cobra OnInitialize）
cobra.OnInitialize(func() {
    verbose, _ := rootCmd.Flags().GetBool("verbose")
    baseDir := filepath.Join(homeDir, ".ckjr")
    logging.Init(verbose, baseDir)
})

// 按 requestId 查本地日志
// grep "requestId值" ~/.ckjr/logs/2026-03-25.log
```
