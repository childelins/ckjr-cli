# Findings

## Go httptest Content-Type 自动检测

Go 的 httptest.NewServer 在 handler 中使用 `json.NewEncoder(w).Encode()` 时，如果未显式设置 `Content-Type` 头，ResponseWriter 会自动检测内容类型为 `text/plain; charset=utf-8`，而非 `application/json`。所有返回 JSON 的测试 handler 必须显式设置 `w.Header().Set("Content-Type", "application/json")`。要模拟完全无 Content-Type 的响应，需要使用 `w.Header()["Content-Type"] = nil` 后再调用 WriteHeader。

## Go //go:embed 路径限制

Go 的 `//go:embed` 只能引用当前 Go 源文件所在目录及其子目录，不支持 `..` 回溯。当 embed 声明需要与实际 main 包在不同目录时（如本项目 main 在 cmd/ckjr-cli/），需要将嵌入的资源放在 main 包目录下，或通过 fs.Sub() 调整路径前缀。

## Go init() 与 TestMain 执行顺序

Go 的 init() 在 TestMain 之前执行。如果 init() 依赖需要通过 TestMain 设置的变量（如 yamlFS），需要将 init() 中的逻辑延迟到 Execute() 等手动调用的函数中。

## fs.FS 接口 vs embed.FS

`fs.FS` 接口没有 `ReadFile` 方法。`embed.FS` 有 `ReadFile` 但它是 `embed.FS` 特有的。要使用通用的 `fs.FS` 接口，需要调用 `fs.ReadFile(f.fs, path)` 标准库函数。

## Go 同名包 import 冲突

当子包名与 internal 包名相同时（如 `cmd/config` 和 `internal/config` 都叫 `config`），需要使用 import alias 区分。常用命名规则：`internalconfig` 给 internal 包，`configcmd` 给 cmd 子包。

## cobra 命令延迟注册与测试

当命令在 `Execute()` 中动态注册（如 workflow 需要 yamlFS），TestMain 需要显式注册该命令，否则测试中 rootCmd.Execute() 会报 "unknown command"。
