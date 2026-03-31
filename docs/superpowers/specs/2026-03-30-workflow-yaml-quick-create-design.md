# Workflow YAML 快速创建设计文档

> Created: 2026-03-30
> Status: Draft

## 概述

为 workflow YAML 提供骨架文件快速创建能力。只需提供模块名，即可在 `cmd/ckjr-cli/workflows/` 下生成一份结构完整的骨架 YAML 文件（含示例 workflow），供开发者参考和填充。作为 Go 包 API 暴露（不暴露到 CLI）。

## 背景

Route 已有快速创建能力（`internal/yamlgen` 包的 `GenerateRoute` + `AppendToFile` + `CreateFile`），但 workflow 当前只能通过手动编写 YAML 创建。

## 设计决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 输入 | 模块名（如 "asset"） | 最简输入，零学习成本 |
| 输出 | 含示例 workflow 骨架的 YAML 文件 | 包含完整字段结构，方便参考 |
| 包位置 | `internal/yamlgen` | 与 Route 生成逻辑同包 |
| 文件写入 | 直接写文件系统 | 编译时 go:embed 自动嵌入 |
| CLI 暴露 | 不暴露 | 仅作为 Go API |

## 函数签名

```go
// InitWorkflow 创建模块 workflow 骨架文件
// moduleName: 模块名（如 "asset"），用于生成文件名和 Config name
// 文件写入 cmd/ckjr-cli/workflows/<module>.yaml
func InitWorkflow(moduleName string) error
```

## 生成结果

输入 `InitWorkflow("asset")` 生成 `cmd/ckjr-cli/workflows/asset.yaml`：

```yaml
name: asset
description: asset
workflows:
  workflow-name:
    description: 工作流描述
    triggers: []
    inputs: {}
    steps: []
```

## 数据流

```
InitWorkflow("asset")
  |
  v
构造 workflow.Config{
    Name: "asset",
    Description: "asset",
    Workflows: {"workflow-name": {Description: "工作流描述", ...}},
}
  |
  v
yaml.Marshal(cfg)
  |
  v
os.WriteFile("cmd/ckjr-cli/workflows/asset.yaml", data, 0644)
```

## 错误处理

| 场景 | 处理 |
|------|------|
| 文件已存在 | 返回 `fmt.Errorf("workflow 文件已存在: %s", path)` |
| 写入失败 | 返回 `os.WriteFile` 的原始错误 |

## 文件结构

```
internal/yamlgen/
  generate.go      # 现有: route 生成
  workflow.go      # 新增: InitWorkflow
  generate_test.go # 现有: route 测试
  workflow_test.go # 新增: workflow 测试
```

## 测试策略

1. **TestInitWorkflow** -- 创建骨架文件，验证结构完整（name/description/workflows/workflow-name 字段存在）
2. **TestInitWorkflow_FileExists** -- 文件已存在时返回错误
3. **TestInitWorkflow_InvalidName** -- 空模块名时返回错误

使用 `t.TempDir()` 隔离文件操作，通过 `workflow.Parse()` 回读验证正确性。

## 实现注意事项

1. **yaml.v3 序列化顺序** -- `yaml.Marshal` 按结构体字段定义顺序输出，与 YAML 格式一致
2. **文件权限** -- 使用 `0644`，与 Route 模式一致
3. **循环依赖** -- `yamlgen` -> `workflow`（单向），不存在循环依赖
4. **embedFS 兼容** -- 文件直接写入文件系统，编译时 `go:embed` 自动嵌入（与 Route 模式一致）
