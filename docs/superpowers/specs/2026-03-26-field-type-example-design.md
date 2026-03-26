# Field Type/Example 字段扩展设计文档

> Created: 2026-03-26
> Status: Draft

## 概述

为 agent.yaml 的参数定义（Field）增加 `type` 和 `example` 两个可选字段。`type` 标注参数数据类型，默认 "string"；`example` 用于额外补充示例值（如特殊格式、ID 示例等），不改动现有 description 内容。

## 变更范围

涉及 3 个文件修改 + 对应测试更新：

| 文件 | 变更 |
|------|------|
| `internal/router/router.go` | Field 结构体增加 Type/Example 字段 |
| `internal/cmdgen/cmdgen.go` | printTemplate 输出 type/example |
| `cmd/routes/agent.yaml` | 为枚举参数补充 type/example |

## 架构

无架构变更。本次是 Field 数据结构的字段扩展，沿用现有 YAML 解析 -> 结构体 -> 命令生成的数据流。

## 组件

### 1. Field 结构体扩展

文件：`internal/router/router.go`

```go
type Field struct {
    Description string      `yaml:"description"`
    Required    bool        `yaml:"required"`
    Default     interface{} `yaml:"default"`
    Type        string      `yaml:"type"`
    Example     string      `yaml:"example"`
}
```

- `Type`：参数类型，YAML 中省略时 Go 零值为 ""，在使用时视为 "string"
- `Example`：额外补充的示例值，可选，仅在有值时展示。不用于替代 description 中的说明

### 2. printTemplate 输出扩展

文件：`internal/cmdgen/cmdgen.go`

printTemplate 函数在构建输出 map 时增加 type 和 example：

```go
func printTemplate(template map[string]router.Field) {
    tmpl := make(map[string]interface{})
    for name, field := range template {
        entry := map[string]interface{}{
            "description": field.Description,
            "required":    field.Required,
        }
        if field.Default != nil {
            entry["default"] = field.Default
        }
        // type: 未设置时显示 "string"
        t := field.Type
        if t == "" {
            t = "string"
        }
        entry["type"] = t
        // example: 仅有值时输出
        if field.Example != "" {
            entry["example"] = field.Example
        }
        tmpl[name] = entry
    }
    output.Print(os.Stdout, tmpl, true)
}
```

### 3. agent.yaml 参数补充

为数值型参数补充 type 字段，description 保持不变。example 仅在需要额外补充示例值时添加（如 aikbId 给 ID 格式示例）。

涉及以下参数补充 type：

| 参数 | type | 说明 |
|------|------|------|
| page | int | 页码 |
| limit | int | 每页数量 |
| enablePagination | int | description 已含枚举说明，不额外加 example |
| platType | int | 同上 |
| botType | int | 同上 |
| isSaleOnly | int | 同上 |
| promptType | int | 同上 |

示例 YAML 变更：

```yaml
enablePagination:
  description: 是否分页返回, 1-是 0-否
  required: false
  default: 0
  type: int
```

注意：description 内容保持原样不拆分，type 标注数据类型，example 仅在需要额外补充时使用。

## 数据流

```
agent.yaml (含 type/example)
  -> yaml.Unmarshal -> Field{Type, Example}
  -> printTemplate -> JSON 输出含 type/example
  -> 用户通过 --template 查看参数信息
```

无新增数据流，只是现有流中携带了更多字段信息。

## 错误处理

- YAML 中 type/example 均为可选字段，缺失不报错（Go 零值处理）
- 不做 type 值的校验（不限制取值范围），type 仅作为展示用途
- example 为空字符串时不在 template 输出中显示，避免噪音

## 测试策略

### router_test.go

1. **TestParseRouteConfig_TypeAndExample**：验证 YAML 中含 type/example 的字段能正确解析
2. **TestParseRouteConfig_TypeDefault**：验证 YAML 中省略 type 时 Field.Type 为空字符串

### cmdgen_test.go

1. **TestPrintTemplate_WithTypeAndExample**：验证 printTemplate 输出中包含 type 和 example
2. **TestPrintTemplate_DefaultType**：验证未设置 type 时输出 "string"
3. **TestPrintTemplate_NoExample**：验证 example 为空时不出现在输出中

## 实现注意事项

1. **向后兼容**：type/example 均为可选字段，现有 YAML 无需强制修改即可继续工作
2. **Type 默认值策略**：在结构体层面不设默认值（保持 Go 零值），在展示层（printTemplate）处理默认值 "string"。这样 `Field.Type == ""` 可以明确表示"用户未指定"
3. **TDD 顺序**：先写测试，再改结构体，再改 printTemplate，最后更新 YAML
4. **不做类型校验**：type 字段纯粹用于展示，不影响参数解析或 API 调用逻辑。未来如需校验可另行扩展
