---
name: yaml-field-validation
project: ckjr-cli
created: 2026-03-28
tags: [校验, YAML, Go, JSON类型]
---

# YAML Field 约束校验

## 决策

| 决策点 | 选择 | 原因 |
|--------|------|------|
| 约束字段类型 | `*float64`/`*int` 指针 | 区分"未设置"和"零值"，omitempty 不影响 |
| 类型校验范围 | string/int/float/bool/array | 覆盖 JSON 原生类型，YAML 无需修改 |
| 约束与 type 耦合 | 宽松策略 | 约束与 type 不匹配时不报错，避免 YAML 作者被迫同步删除 |
| 校验入口 | ValidateAll 统一收集 | 与原有 validateRequired 风格一致 |
| pattern 编译 | 启动时预编译 | 配置错误应尽早暴露 |

## 坑点预警

- **JSON number 统一为 float64**：`json.Unmarshal` 将所有数字解析为 float64，int 类型校验需额外检查 `math.Trunc(v) == v`
- **yaml.v3 整数到 `*float64`**：`min: 1` 在 YAML 中是整数，但 yaml.v3 会自动转为 float64(1.0)，无需自定义 UnmarshalYAML
- **curlparse float 推断**：原来非整数 float64 归为 string，改为 float 后影响 yamlgen 生成结果

## 复用模式

```go
// 约束字段的指针类型模式
type Field struct {
    Min       *float64 `yaml:"min,omitempty"`
    Max       *float64 `yaml:"max,omitempty"`
    MinLength *int     `yaml:"minLength,omitempty"`
    MaxLength *int     `yaml:"maxLength,omitempty"`
    Pattern   string   `yaml:"pattern,omitempty"`
}

// JSON 原生类型校验
switch expectedType {
case "int":
    v, ok := value.(float64)
    if !ok || v != math.Trunc(v) { return error }
case "float":
    _, ok := value.(float64)
}
```
