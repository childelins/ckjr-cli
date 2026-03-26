---
name: log-environment-modes-with-ldflags
project: ckjr-cli
created: 2026-03-26
tags: [logging, slog, ldflags, environment, go-build]
---

# Go CLI 日志环境区分（编译期注入）

## 决策

| 决策点 | 选择 | 原因 |
|--------|------|------|
| 环境区分方式 | `go build -ldflags` 编译期注入 | CLI 工具不支持运行时配置切换，编译期注入最简且安全 |
| 区分维度 | 日志级别 + 内容详细度 | 开发需要 DEBUG+完整 body，生产只需 INFO+省略 body |
| 全局状态管理 | 包级变量 `currentEnv` | 最简方案，避免引入配置结构体；`Init` 在 `cobra.OnInitialize` 中调用，时序安全 |
| `IsDev()` 作为包级函数 | 供 api 包直接调用 | 不需要传递 env 参数跨多层调用 |

## 坑点预警

- **slog 键值对变长参数**: 当条件省略某些字段时，使用 `[]interface{}` 收集 attrs 再 `...` 展开，不要试图用固定参数列表 + 条件跳过（会导致键值不对齐）。
- **测试环境隔离**: `IsDev()` 依赖 `currentEnv` 包级变量，测试中必须在 `captureLog` 之前调用 `logging.Init` 设置环境，否则测试间状态泄漏。
- **CI 构建需显式注入**: `go build` 默认值 `"production"` 不会触发 ldflags 替换（因为默认值就在代码中），但 CI/release 脚本应显式注入 `-X main.Environment=production` 保持一致性。

## 复用模式

```go
// 编译期注入变量模式（与 Version 相同）
var (
    Version     = "dev"
    Environment = "production" // -ldflags "-X main.Environment=development"
)

// IsDev() 全局开关模式
var currentEnv = Production

func IsDev() bool {
    return currentEnv == Development
}

// slog 条件 attrs 模式
attrs := []interface{}{"key1", val1, "key2", val2}
if logging.IsDev() {
    attrs = append(attrs, "body", bodyContent)
}
slog.InfoContext(ctx, "message", attrs...)
```
