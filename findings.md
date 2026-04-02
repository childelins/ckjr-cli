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

## filterByFields get-then-set 模式对数组的局限

当 `getNestedValue` 穿透数组后返回的是扁平化的值列表（如 `[1, 2]`），再用 `setNestedValue` 设置到目标 map 时会丢失数组结构 -- 值被放入嵌套 map 而非保持在数组元素中。解决方案是将 `filterByFields` 重构为 `applyFieldPath` 递归构建模式，在遍历源结构时同步构建目标结构，遇到数组时对每个元素分别构建对应的目标 map。重构后不存在的嵌套路径需要额外处理：递归到空 sub map 后检查 `len(sub) > 0` 再写入 dst，避免产生空的中间 map 结构。

## mime.ExtensionsByType 扩展名优先级

Go 标准库 `mime.ExtensionsByType("image/jpeg")` 返回 `[.jpe .jpeg .jpg]`，第一个扩展名不是常见的 `.jpg`。在 `extFromContentType` 中需要优先查找 `.jpg`，然后回退到其他常见扩展名，最后才使用返回列表中的第一个。

## Workflow YAML 验证与手动注册命令

项目 `cmd/ckjr-cli/yaml_validate_test.go` 中的 `TestWorkflowCommandReferences` 会交叉验证 workflow 步骤中引用的命令是否在路由 YAML 配置中存在。对于通过代码手动注册的子命令（如 `asset upload-image`），需要在验证逻辑中维护一个白名单 `manualCommands` 来跳过验证，否则测试会误报错误。
