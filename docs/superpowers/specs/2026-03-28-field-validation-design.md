# Field 类型与约束校验设计文档

> Created: 2026-03-28
> Status: Draft

## 概述

为 route YAML 模板的 Field 结构增加类型校验和约束校验能力。当前仅有 `validateRequired`（key 存在性检查），`type` 字段仅用于 `printTemplate` 展示。本次扩展使 type 具有运行时校验语义，并新增 min/max/minLength/maxLength/pattern 五种约束。

已确认的设计决策：
1. 约束范围：min/max + minLength/maxLength + pattern（正则）
2. 类型扩展：新增 float/bool/array，完整覆盖 JSON 原生类型
3. 错误输出：收集所有错误一次性输出

## 1. Field 结构扩展

### 当前结构

```go
type Field struct {
    Description string      `yaml:"description"`
    Required    bool        `yaml:"required"`
    Default     interface{} `yaml:"default,omitempty"`
    Type        string      `yaml:"type,omitempty"`
    Example     string      `yaml:"example,omitempty"`
}
```

### 新增字段

```go
type Field struct {
    Description string      `yaml:"description"`
    Required    bool        `yaml:"required"`
    Default     interface{} `yaml:"default,omitempty"`
    Type        string      `yaml:"type,omitempty"`
    Example     string      `yaml:"example,omitempty"`

    // 数值约束（适用于 type: int, type: float）
    Min         *float64    `yaml:"min,omitempty"`
    Max         *float64    `yaml:"max,omitempty"`

    // 字符串约束（适用于 type: string）
    MinLength   *int        `yaml:"minLength,omitempty"`
    MaxLength   *int        `yaml:"maxLength,omitempty"`
    Pattern     string      `yaml:"pattern,omitempty"`
}
```

### 设计说明

- 所有新增字段使用 `omitempty`，现有 YAML 文件无需修改
- Min/Max 使用 `*float64` 指针类型，可以区分"未设置"和"设置了零值"
- MinLength/MaxLength 使用 `*int` 指针类型，同理
- Pattern 为 string，空值即表示不启用
- YAML tag 使用 camelCase（`minLength`、`maxLength`），与 JSON schema 惯例一致

### YAML 示例

```yaml
template:
    page:
        description: 页码
        required: false
        default: 1
        type: int
        min: 1
        max: 1000
    keyword:
        description: 搜索关键词
        required: false
        type: string
        minLength: 1
        maxLength: 100
    email:
        description: 邮箱地址
        required: true
        type: string
        pattern: "^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$"
    tags:
        description: 标签列表
        required: false
        type: array
    score:
        description: 评分
        required: false
        type: float
        min: 0.0
        max: 10.0
```

## 2. 类型校验逻辑

### type 与 JSON 原生类型映射

用户输入经过 `json.Unmarshal` 解析为 `map[string]interface{}`，值的 Go 类型遵循 JSON 标准映射：

| YAML type | 期望的 Go 类型 | JSON 原始类型 |
|-----------|--------------|-------------|
| `string`  | `string`     | JSON string |
| `int`     | `float64`（整数值）| JSON number |
| `float`   | `float64`    | JSON number |
| `bool`    | `bool`       | JSON boolean |
| `array`   | `[]interface{}` | JSON array |
| (空)      | 不做类型校验   | 任意 |

### 校验规则

**type 为空时**：不校验类型，保持向后兼容。

**string**：检查 `val.(string)` 是否成功。

**int**：检查 `val.(float64)` 成功且 `math.Trunc(v) == v`（整数值）。JSON 的 number 统一解析为 float64，int 类型要求值没有小数部分。

**float**：检查 `val.(float64)` 成功。接受任何 number，包括整数值（如 `10` 等价于 `10.0`）。

**bool**：检查 `val.(bool)` 成功。

**array**：检查 `val.([]interface{})` 成功。注意 JSON object（`map[string]interface{}`）不是 array。

### 未知 type

如果 type 不在上述五种之内，`validateType` 返回错误，防止用户写错 type 名而静默跳过校验。

## 3. 约束校验逻辑

### min/max（数值约束）

适用条件：type 为 `int` 或 `float`，且字段值已通过类型校验。

```
if field.Min != nil && value < *field.Min → 错误
if field.Max != nil && value > *field.Max → 错误
```

注意事项：
- min/max 在 YAML 中写为 `min: 1` 或 `min: 0.5`，YAML 解析器自动处理为 float64
- 约束与 type 无耦合：同一个 `validateConstraints` 函数统一处理，只是 min/max 仅对数值类型生效

### minLength/maxLength（字符串约束）

适用条件：type 为 `string`，且字段值已通过类型校验。

```
if field.MinLength != nil && len(str) < *field.MinLength → 错误
if field.MaxLength != nil && len(str) > *field.MaxLength → 错误
```

使用 `len()` 按 byte 计数。如果未来需要按 rune 计数，可在此处调整。

### pattern（正则约束）

适用条件：type 为 `string`，且字段值已通过类型校验。

```
if field.Pattern != "" {
    re = regexp.MustCompile(field.Pattern) // 预编译或运行时编译
    if !re.MatchString(str) → 错误
}
```

错误信息示例：`字段 "email" 的值 "abc" 不匹配正则 "^[\\w.-]+@...$"`

### 约束不适用时的处理

如果某个约束与 type 不匹配（如对 string 字段设置 min/max），校验函数不做任何检查也不报错。这是宽松策略：YAML 作者可以在未来改变 type 时保留约束字段，无需同步删除。

## 4. 校验函数设计

### 错误结构

```go
// ValidationError 单个字段的校验错误
type ValidationError struct {
    Field   string // 字段名
    Message string // 错误描述
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("字段 %q %s", e.Field, e.Message)
}
```

### 函数签名

在 `internal/cmdgen/` 中新增 `validate.go`：

```go
// ValidateAll 校验输入数据，返回所有校验错误
// 执行顺序：先 validateRequired，再 validateTypes，最后 validateConstraints
func ValidateAll(input map[string]interface{}, template map[string]router.Field) []*ValidationError

// validateTypes 校验所有非空 type 字段的类型
func validateTypes(input map[string]interface{}, template map[string]router.Field) []*ValidationError

// validateType 校验单个字段的类型
func validateType(fieldName string, value interface{}, expectedType string) *ValidationError

// validateConstraints 校验所有约束（min/max/minLength/maxLength/pattern）
func validateConstraints(input map[string]interface{}, template map[string]router.Field) []*ValidationError
```

### 与现有 validateRequired 的关系

`validateRequired` 保持不变，仍然是 `[]string` 返回类型。`ValidateAll` 内部调用它，将 missing 字段名转换为 `ValidationError`。这样保持了函数单一职责，也避免了修改 `validateRequired` 的调用方。

`ValidateAll` 整体流程：

```
1. validateRequired → 收集缺失的必填字段 → 转为 ValidationError
2. validateTypes → 对每个有 type 声明且存在于 input 中的字段校验类型
3. validateConstraints → 对每个有约束声明且通过类型校验的字段校验约束值
4. 返回全部错误（如不为空）
```

### cmdgen 调用方修改

`buildSubCommand` 中的调用从：

```go
if missing := validateRequired(input, route.Template); len(missing) > 0 {
    output.PrintError(os.Stderr, fmt.Sprintf("缺少必填字段: %v", missing))
    os.Exit(1)
}
```

改为：

```go
if errs := ValidateAll(input, route.Template); len(errs) > 0 {
    var msgs []string
    for _, e := range errs {
        msgs = append(msgs, e.Error())
    }
    output.PrintError(os.Stderr, fmt.Sprintf("参数校验失败:\n  %s", strings.Join(msgs, "\n  ")))
    os.Exit(1)
}
```

## 5. 联动修改点

### 5.1 router/router.go

修改 `Field` 结构体，新增 Min/Max/MinLength/MaxLength/Pattern 五个字段。

### 5.2 cmdgen/validate.go（新建）

新增校验逻辑文件，包含 `ValidateAll`、`validateTypes`、`validateType`、`validateConstraints` 和 `ValidationError`。

### 5.3 cmdgen/cmdgen.go

- `buildSubCommand`：将 `validateRequired` 调用替换为 `ValidateAll`
- `printTemplateTo`：在输出中展示约束信息

`printTemplateTo` 扩展输出格式：

```go
// 有约束时追加 constraints 子对象
if hasConstraints(field) {
    constraints := map[string]interface{}{}
    if field.Min != nil { constraints["min"] = *field.Min }
    if field.Max != nil { constraints["max"] = *field.Max }
    if field.MinLength != nil { constraints["minLength"] = *field.MinLength }
    if field.MaxLength != nil { constraints["maxLength"] = *field.MaxLength }
    if field.Pattern != "" { constraints["pattern"] = field.Pattern }
    entry["constraints"] = constraints
}
```

### 5.4 curlparse/parse.go

`inferField` 函数当前将非整数的 float64 归为 string。修改为：

```go
case float64:
    if v == math.Trunc(v) {
        return Field{Type: "int", Example: int(v)}, true
    }
    return Field{Type: "float", Example: v}, true  // 改为 float
```

`inferQueryParam` 目前只能推断 string/int/bool，无需修改（query param 中没有 float/array 的自然来源）。

### 5.5 yamlgen/generate.go

`GenerateRoute` 已正确传递 type，无需修改。curlparse 新增 float type 会被自动传递。

### 5.6 已有 YAML 文件

现有 `agent.yaml` 和 `common.yaml` 中的字段不需要修改：
- 新字段全部 `omitempty`，未设置时为零值
- 现有 type 值（int/string）不变
- 未来可按需为字段添加约束

## 6. 向后兼容

1. **Field 结构**：新增字段全部 `omitempty`，YAML 反序列化时未设置的字段为零值，不影响已有解析逻辑
2. **validateRequired**：保留原有签名 `func validateRequired(...) []string`，不修改其行为
3. **type=""**：当 type 为空时，`validateType` 直接返回 nil（不校验），保持现有行为
4. **pattern 编译失败**：在 `ValidateAll` 启动阶段预编译所有 pattern，编译失败直接 panic（属于配置错误，应在开发阶段发现）
5. **已有 YAML 无需修改**：所有新约束字段都是可选的

## 7. 测试策略

### 单元测试

**internal/cmdgen/validate_test.go**（新建）：

- `TestValidateType_String`：string 值通过/失败
- `TestValidateType_Int`：整数值通过/浮点值失败
- `TestValidateType_Float`：浮点值通过
- `TestValidateType_Bool`：bool 值通过/其他类型失败
- `TestValidateType_Array`：数组通过/object 失败
- `TestValidateType_Empty`：type 为空不校验
- `TestValidateType_Unknown`：未知 type 返回错误
- `TestValidateConstraints_MinMax`：边界值测试
- `TestValidateConstraints_MinLengthMaxLength`：长度边界
- `TestValidateConstraints_Pattern`：正则匹配/不匹配
- `TestValidateConstraints_Irrelevant`：约束与 type 不匹配时不报错
- `TestValidateAll_Required`：必填字段缺失
- `TestValidateAll_MultipleErrors`：多错误收集
- `TestValidateAll_Pass`：全部通过返回空

**internal/router/router_test.go**：

- `TestParseRouteConfig_Constraints`：解析含约束字段的 YAML

**internal/curlparse/parse_test.go**：

- 更新已有 float 值推断的期望结果（string → float）

### 测试顺序（TDD）

1. 先写 `validateType` 的测试和实现
2. 再写 `validateConstraints` 的测试和实现
3. 然后 `ValidateAll` 集成
4. 最后修改 cmdgen 调用方和 printTemplate

## 8. 实现注意事项

1. **Pattern 预编译**：在 `ValidateAll` 入口处，先遍历 template 收集所有 pattern 并 `regexp.Compile`，失败则作为配置错误返回。避免每次校验都编译正则。

2. **Min/Max YAML 解析**：`min: 1` 在 YAML 中是整数，但 Go 的 `yaml.v3` 会将 `*float64` 正确解析。测试中需确认这一点。如果 YAML v3 将整数解析为 `int` 而非 `float64`，需要在 `Field` 上实现自定义 `UnmarshalYAML` 或改用 `interface{}`+运行时转换。根据 yaml.v3 行为，`*float64` tag 配合 `min: 1` 会自动转为 `float64(1.0)`，无需额外处理。

3. **nil 值处理**：JSON 中 `null` 解析为 Go 的 `nil`。`validateType` 应将 `nil` 视为类型不匹配（因为 `required` 检查已先执行）。

4. **array 元素校验**：本设计不包含 array 元素的类型校验（即不定义 `items`），遵循 YAGNI 原则。

5. **文件组织**：校验逻辑放在 `internal/cmdgen/validate.go`，因为它与 cmdgen 的命令执行流程紧密相关，且依赖 `router.Field`。
