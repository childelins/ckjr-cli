# 路由路径参数替换设计文档

> Created: 2026-03-31
> Status: Draft (v2 -- 显式声明方案)

## 概述

路由 YAML 的 `path` 字段支持 `{xxx}` 占位符（如 `/admin/courses/{courseId}`）。路径参数**必须**在 `template` 中显式声明，使用 `type: path` 标识。发送 HTTP 请求前，从 template 中筛选所有 `type: path` 的字段，用其值替换 path 中的对应占位符。`type: path` 的字段专用于路径替换，不参与 body/query 参数校验。

已确认的设计决策：
1. 路径参数在 template 中声明，type 设为 `path`
2. `type: path` 的字段用于路径替换，不进入 body/query
3. 缺少路径参数时立即报错，阻止请求发送

## 1. 当前状况

### 现有流程

```
YAML path: /admin/courses/{courseId}
          |
router.Parse() -> Route{Path: "/admin/courses/{courseId}"}
          |
cmdgen.buildSubCommand() -> client.DoCtx(ctx, method, route.Path, input, &result)
          |
api.Client.DoCtx() -> url = c.baseURL + path  (直接拼接，无替换)
```

`route.Path` 原样传递给 `DoCtx`，`{courseId}` 占位符未被替换，导致实际请求 URL 为 `https://api.example.com/admin/courses/{courseId}`，服务端返回 404。

### 影响范围

当前 `cmd/ckjr-cli/routes/course.yaml` 的 `update` 路由使用了路径参数，但 template 中**未声明** `courseId` 字段。设计完成后需同步修改 YAML 文件。

## 2. YAML 格式变更

### 2.1 变更前（当前 course.yaml update 路由）

```yaml
update:
    method: PUT
    path: /admin/courses/{courseId}
    template:
        courseType:
            description: 课程类型
            required: true
            type: int
        name:
            description: 课程名称
            required: true
            type: string
        # ... courseId 未声明
```

### 2.2 变更后

```yaml
update:
    method: PUT
    path: /admin/courses/{courseId}
    template:
        courseId:
            description: 课程ID
            required: true
            type: path
        courseType:
            description: 课程类型
            required: true
            type: int
        name:
            description: 课程名称
            required: true
            type: string
        # ...
```

关键点：`courseId` 字段的 `type` 为 `path`，标识这是一个路径参数。

### 2.3 多路径参数示例

```yaml
update-chapter:
    method: PUT
    path: /courses/{courseId}/chapters/{chapterId}
    template:
        courseId:
            description: 课程ID
            required: true
            type: path
        chapterId:
            description: 章节ID
            required: true
            type: path
        title:
            description: 章节标题
            required: true
            type: string
```

## 3. 设计方案

### 3.1 type: path 的语义

`type: path` 是一个特殊的字段类型，与 `string`/`int`/`float`/`bool`/`array` 并列。它表示该字段是路径参数，具有以下行为：

1. **参与路径替换**：值用于替换 path 中对应的 `{xxx}` 占位符
2. **不参与 body/query**：替换完成后，该字段从 input map 中移除，不发送给服务端
3. **不参与类型校验**：`validateType()` 跳过 `type: path` 字段
4. **不参与约束校验**：`validateConstraints()` 跳过 `type: path` 字段
5. **参与 required 校验**：如果标记 `required: true`，缺失时仍报错

### 3.2 路径参数提取与替换

从 template 中筛选所有 `type: path` 的字段，用于路径替换：

```go
// IsPathParam 判断字段是否为路径参数
func IsPathParam(field router.Field) bool {
    return field.Type == "path"
}

// CollectPathParams 从 template 中收集所有路径参数名
func CollectPathParams(template map[string]router.Field) []string {
    var params []string
    for name, field := range template {
        if IsPathParam(field) {
            params = append(params, name)
        }
    }
    return params
}
```

路径替换函数：

```go
// pathParamRe 匹配路径中的 {paramName} 占位符
var pathParamRe = regexp.MustCompile(`\{(\w+)\}`)

// ReplacePath 将 path 中的 {xxx} 占位符替换为 input 中的对应值
// 仅替换 template 中声明为 type: path 的字段
// 替换后从 input 中移除路径参数字段
// 返回替换后的 path 和错误（如有缺失或未声明的路径参数）
func ReplacePath(path string, input map[string]interface{}, template map[string]router.Field) (string, error)
```

### 3.3 校验规则：path 占位符与 template 声明的一致性

为防止配置错误，需要校验 path 中的占位符与 template 中 `type: path` 字段的一致性：

1. path 中存在 `{xxx}` 但 template 中没有对应的 `type: path` 字段 -> 报错
2. template 中声明了 `type: path` 但 path 中没有对应的 `{xxx}` -> 报错（可选，v1 可跳过）

v1 仅实现规则 1（确保路径替换不会遗漏），规则 2 作为后续优化。

### 3.4 缺失参数处理

当 `type: path` 字段标记 `required: true` 但 input 中缺少该字段时，通过现有的 required 校验报错。

当 input 中存在字段但值为 nil 时，`ReplacePath` 将其视为缺失并报错。

### 3.5 值类型转换

路径参数值使用 `fmt.Sprintf("%v", val)` 转为字符串：

| JSON 值 | Go 类型 | Sprintf 结果 | URL 路径 |
|---------|---------|-------------|---------|
| `123` | `float64` | `"123"` | `/courses/123` |
| `"abc"` | `string` | `"abc"` | `/courses/abc` |

注意：`float64` 的 `123` 会被格式化为 `"123"`（无小数点）。如果遇到大数值 ID 导致科学计数法问题，可引入 `formatPathValue` 函数：

```go
func formatPathValue(val interface{}) string {
    if v, ok := val.(float64); ok {
        if v == math.Trunc(v) {
            return strconv.FormatInt(int64(v), 10)
        }
    }
    return fmt.Sprintf("%v", val)
}
```

建议 v1 使用 `fmt.Sprintf("%v")`，如遇到问题再引入。

## 4. 修改点

### 4.1 新增文件：internal/cmdgen/pathparam.go

包含路径参数相关的所有逻辑：

```go
package cmdgen

import (
    "fmt"
    "net/url"
    "regexp"
    "strings"

    "github.com/childelins/ckjr-cli/internal/router"
)

var pathParamRe = regexp.MustCompile(`\{(\w+)\}`)

// PathParamError 路径参数错误
type PathParamError struct {
    Missing      []string // input 中缺失的路径参数
    Undeclared   []string // path 中有占位符但 template 未声明 type: path
}

func (e *PathParamError) Error() string {
    var parts []string
    if len(e.Undeclared) > 0 {
        parts = append(parts, fmt.Sprintf(
            "路径占位符 {%s} 未在 template 中声明为 type: path",
            strings.Join(e.Undeclared, "}, {")))
    }
    if len(e.Missing) > 0 {
        parts = append(parts, fmt.Sprintf(
            "缺少路径参数: %s", strings.Join(e.Missing, ", ")))
    }
    return strings.Join(parts, "; ")
}

// IsPathParam 判断字段是否为路径参数
func IsPathParam(field router.Field) bool {
    return field.Type == "path"
}

// ReplacePath 将 path 中的 {xxx} 占位符替换为 input 中的对应值
// 仅替换 template 中声明为 type: path 的字段
// 替换成功后从 input 中移除路径参数字段
func ReplacePath(path string, input map[string]interface{}, template map[string]router.Field) (string, error) {
    // 1. 从 path 中提取所有占位符名
    placeholders := extractPlaceholders(path)
    if len(placeholders) == 0 {
        return path, nil
    }

    // 2. 校验：path 中的占位符必须在 template 中声明为 type: path
    var undeclared []string
    for _, name := range placeholders {
        field, exists := template[name]
        if !exists || !IsPathParam(field) {
            undeclared = append(undeclared, name)
        }
    }
    if len(undeclared) > 0 {
        return "", &PathParamError{Undeclared: undeclared}
    }

    // 3. 校验：所有路径参数在 input 中存在且非 nil
    var missing []string
    for _, name := range placeholders {
        val, exists := input[name]
        if !exists || val == nil {
            missing = append(missing, name)
        }
    }
    if len(missing) > 0 {
        return "", &PathParamError{Missing: missing}
    }

    // 4. 执行替换
    result := pathParamRe.ReplaceAllStringFunc(path, func(match string) string {
        name := match[1 : len(match)-1]
        val := input[name]
        return url.PathEscape(fmt.Sprintf("%v", val))
    })

    // 5. 从 input 中移除路径参数字段
    for _, name := range placeholders {
        delete(input, name)
    }

    return result, nil
}

// extractPlaceholders 从 path 中提取所有 {xxx} 占位符名，去重保序
func extractPlaceholders(path string) []string {
    matches := pathParamRe.FindAllStringSubmatch(path, -1)
    if len(matches) == 0 {
        return nil
    }
    seen := make(map[string]bool)
    var params []string
    for _, m := range matches {
        name := m[1]
        if !seen[name] {
            seen[name] = true
            params = append(params, name)
        }
    }
    return params
}
```

### 4.2 修改文件：internal/cmdgen/validate.go

`validateType()` 和 `validateConstraints()` 需要跳过 `type: path` 字段：

```go
// validateTypes -- 修改：跳过 path 类型
func validateTypes(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
    var errs []*FieldValidationError
    for name, field := range template {
        if field.Type == "" || IsPathParam(field) {
            continue
        }
        val, exists := input[name]
        if !exists {
            continue
        }
        if err := validateType(name, val, field.Type); err != nil {
            errs = append(errs, err)
        }
    }
    return errs
}
```

注意：`validateConstraints()` 中的 switch 只处理 `int`/`float`/`string`，`path` 自然不会命中，无需额外修改。但为清晰起见，建议在循环开头加一行跳过：

```go
func validateConstraints(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
    // ...
    for name, field := range template {
        if IsPathParam(field) {
            continue
        }
        // ... 原有逻辑
    }
}
```

### 4.3 修改文件：internal/cmdgen/cmdgen.go

在 `buildSubCommand` 的 `Run` 函数中，在 `ValidateAll` 之前插入路径参数替换：

```go
// 当前代码（第 78-89 行）：
applyDefaults(input, route.Template)

if errs := ValidateAll(input, route.Template); len(errs) > 0 {
    // ...
}

// ...
if err := client.DoCtx(ctx, route.Method, route.Path, input, &result); err != nil {
```

修改为：

```go
applyDefaults(input, route.Template)

// 路径参数替换（在 template 校验之前执行）
// ReplacePath 会从 input 中移除 type: path 的字段
resolvedPath, err := ReplacePath(route.Path, input, route.Template)
if err != nil {
    output.PrintError(os.Stderr, err.Error())
    os.Exit(1)
}

// 校验剩余参数（路径参数已从 input 中移除，不参与校验）
if errs := ValidateAll(input, route.Template); len(errs) > 0 {
    // ...
}

// ...
if err := client.DoCtx(ctx, route.Method, resolvedPath, input, &result); err != nil {
```

关键变化：
1. `ReplacePath` 接收 `route.Template` 参数，用于判断哪些字段是路径参数
2. `ReplacePath` 替换路径后从 `input` 中**移除**路径参数字段
3. `ValidateAll` 在路径参数移除后执行，path 字段不会触发校验
4. `DoCtx` 的 path 参数从 `route.Path` 改为 `resolvedPath`
5. `input` 中不再包含路径参数字段，request body 干净

### 4.4 修改文件：internal/cmdgen/cmdgen.go -- printTemplateTo

`printTemplateTo` 需要正确处理 `type: path` 字段，让 AI 调用者知道这是路径参数：

当前代码无需修改。`type: path` 会自然输出 `"type": "path"`，AI 调用者看到后知道这是路径参数，需要在 JSON 中提供该字段值。

### 4.5 修改文件：cmd/ckjr-cli/routes/course.yaml

在 `update` 路由的 template 中添加 `courseId` 字段：

```yaml
update:
    method: PUT
    path: /admin/courses/{courseId}
    description: 更新课程
    template:
        courseId:
            description: 课程ID
            required: true
            type: path
        courseType:
            # ... 其余字段不变
```

### 4.6 不修改的文件

- **internal/router/router.go**：`Field.Type` 已经是 `string` 类型，`type: path` 通过 YAML 解析自动读入，无需任何改动
- **internal/api/client.go**：`DoCtx` 接收的 path 已是替换后的完整路径，无需改动

## 5. 数据流

### 完整流程

```
YAML 配置:
  path: /admin/courses/{courseId}
  template:
    courseId: { type: path, required: true }
    name:    { type: string, required: true }

用户输入: {"courseId": 123, "name": "Go入门"}
          |
   [1] applyDefaults -- 填充默认值（路径参数通常无默认值）
          |
   [2] ReplacePath(path, input, template)
       - 从 path 提取占位符: ["courseId"]
       - 校验 template 中 courseId 是 type: path -- 通过
       - 校验 input 中 courseId 存在 -- 通过
       - 替换: "/admin/courses/{courseId}" -> "/admin/courses/123"
       - 从 input 中移除 courseId
       - input 变为: {"name": "Go入门"}
          |
   [3] ValidateAll(input, template)
       - required 校验: name 存在 -- 通过
       - courseId 已从 input 移除，但 template 中仍有声明
         required 校验需跳过 type: path 字段（见第 6 节说明）
       - 类型校验: 跳过 type: path，校验 name 为 string -- 通过
          |
   [4] DoCtx(PUT, "/admin/courses/123", {"name": "Go入门"})
       - URL: https://api.example.com/admin/courses/123
       - Body: {"name": "Go入门"}  (不含 courseId)
```

### 错误场景：缺少路径参数

```
用户输入: {"name": "Go入门"}    (缺少 courseId)
          |
   [1] applyDefaults -- courseId 无默认值，不填充
          |
   [2] ReplacePath
       - 从 path 提取占位符: ["courseId"]
       - 校验 input 中 courseId -- 不存在
       - 返回错误: "缺少路径参数: courseId"
          |
   输出错误并退出，不发送请求
```

### 错误场景：占位符未声明

```
YAML 配置:
  path: /admin/courses/{courseId}
  template:
    name: { type: string }    (缺少 courseId 的 type: path 声明)

ReplacePath 校验:
  - path 中有 {courseId} 占位符
  - template 中无 courseId 或 courseId.type != "path"
  - 返回错误: "路径占位符 {courseId} 未在 template 中声明为 type: path"
```

## 6. ValidateAll 对 type: path 字段的处理

`ReplacePath` 在 `ValidateAll` 之前执行，会从 input 中移除路径参数字段。但 `ValidateAll` 遍历 template 时仍会遇到 `type: path` 的字段声明。需要确保：

1. **required 校验**：跳过 `type: path` 字段。因为路径参数已从 input 中移除，如果不跳过，required 校验会误报缺失。路径参数的缺失检查已由 `ReplacePath` 负责。

2. **类型校验**：跳过 `type: path` 字段（`validateType` 的 switch 会落入 default 分支报 "未知类型 path"）。

3. **约束校验**：`type: path` 不会命中任何 switch 分支，天然跳过。但建议显式跳过以提高可读性。

### 修改 validateRequiredErrors

```go
func validateRequiredErrors(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
    var errs []*FieldValidationError
    for name, field := range template {
        if IsPathParam(field) {
            continue  // 路径参数的必填检查由 ReplacePath 负责
        }
        if field.Required {
            if _, exists := input[name]; !exists {
                errs = append(errs, &FieldValidationError{Field: name, Message: "为必填字段"})
            }
        }
    }
    return errs
}
```

注意：这替代了原来调用 `validateRequired()` 的间接方式，直接在 `validateRequiredErrors` 中内联逻辑，使 `IsPathParam` 跳过更清晰。`validateRequired()` 函数如果无其他调用者可移除，否则保持不变。

## 7. 错误处理

### 7.1 路径参数缺失

```
用户输入: {"name": "Go入门"}
路由 path: /admin/courses/{courseId}

输出: 缺少路径参数: courseId
退出码: 1
```

### 7.2 多个路径参数部分缺失

```
用户输入: {"courseId": 123}
路由 path: /courses/{courseId}/chapters/{chapterId}

输出: 缺少路径参数: chapterId
退出码: 1
```

### 7.3 路径占位符未在 template 中声明

```
路由 path: /admin/courses/{courseId}
template 中无 courseId 或 courseId 的 type 不是 path

输出: 路径占位符 {courseId} 未在 template 中声明为 type: path
退出码: 1
```

### 7.4 无路径参数的路由

```
路由 path: /admin/courses
ReplacePath 检测无占位符，直接返回原始 path，无额外开销
```

### 7.5 错误优先级

执行顺序决定错误优先级：

1. JSON 解析错误（最先）
2. 默认值填充（`applyDefaults`）-- 不产生错误
3. 路径参数替换（`ReplacePath`）-- 占位符未声明/参数缺失
4. template 参数校验（`ValidateAll`）-- required/type/constraint
5. API 请求错误（最后）

## 8. 测试策略

### 单元测试：internal/cmdgen/pathparam_test.go（新建）

**extractPlaceholders 测试：**
- `TestExtractPlaceholders_None`：无占位符返回 nil
- `TestExtractPlaceholders_Single`：单个占位符
- `TestExtractPlaceholders_Multiple`：多个占位符，顺序保持
- `TestExtractPlaceholders_Duplicate`：重复占位符去重

**ReplacePath 测试：**
- `TestReplacePath_NoPlaceholders`：无占位符原样返回，input 不变
- `TestReplacePath_SingleParam`：单参数替换，input 中该字段被移除
- `TestReplacePath_MultipleParams`：多参数替换
- `TestReplacePath_MissingParam`：缺失参数返回 PathParamError（Missing 字段）
- `TestReplacePath_NilValue`：值为 nil 视为缺失
- `TestReplacePath_UndeclaredPlaceholder`：占位符未在 template 声明返回 PathParamError（Undeclared 字段）
- `TestReplacePath_TypeNotPath`：占位符在 template 中存在但 type 不是 path，报 Undeclared 错误
- `TestReplacePath_NumericValue`：float64 值正确转为字符串
- `TestReplacePath_SpecialChars`：特殊字符值正确 URL 编码
- `TestReplacePath_InputModified`：替换后 input map 中路径参数字段被移除

**IsPathParam 测试：**
- `TestIsPathParam_True`：type 为 "path" 返回 true
- `TestIsPathParam_False`：type 为其他值返回 false

**PathParamError 测试：**
- `TestPathParamError_Missing`：仅 Missing 的错误信息
- `TestPathParamError_Undeclared`：仅 Undeclared 的错误信息
- `TestPathParamError_Both`：同时有 Missing 和 Undeclared

### 单元测试：internal/cmdgen/validate_test.go（补充）

- `TestValidateAll_SkipsPathParam`：type: path 字段不触发类型校验错误
- `TestValidateRequiredErrors_SkipsPathParam`：type: path 字段不触发 required 校验

### 集成测试：internal/cmdgen/cmdgen_test.go（补充）

- 含路径参数的路由命令执行（mock client 验证 path 已替换且 body 不含路径参数）
- 路径参数缺失时的错误输出
- 占位符未声明时的错误输出

### TDD 顺序

1. 先写 `IsPathParam` 测试和实现
2. 再写 `extractPlaceholders` 测试和实现
3. 再写 `ReplacePath` 测试和实现
4. 补充 `validate.go` 中跳过 path 类型的测试和修改
5. 修改 `buildSubCommand`，补充集成测试
6. 更新 `course.yaml`

## 9. 实现注意事项

1. **正则预编译**：`pathParamRe` 作为包级变量预编译，避免每次调用时编译。

2. **URL 编码**：路径参数值使用 `url.PathEscape` 而非 `url.QueryEscape`。两者对空格的编码不同（PathEscape: `%20`，QueryEscape: `+`）。

3. **从 input 中移除路径参数**：`ReplacePath` 替换成功后 `delete(input, name)` 移除路径参数字段。这确保 request body 不包含路径参数，也避免 `ValidateAll` 对已移除字段的误校验。

4. **执行位置**：路径参数替换在 `applyDefaults` 之后、`ValidateAll` 之前。这样 `applyDefaults` 有机会为路径参数填充默认值（虽然实际场景极少），且路径参数替换和校验先于 template 校验执行。

5. **validateRequired 的兼容性**：`validateRequired()` 函数（在 `cmdgen.go` 中定义）目前被 `validateRequiredErrors()` 调用。修改 `validateRequiredErrors` 跳过 path 类型后，`validateRequired` 如无其他调用者可保留不动或标记为废弃。

6. **printTemplate 输出**：`type: path` 字段正常输出在模板中。AI 调用者看到 `"type": "path"` 后知道需要在 JSON 中提供该值用于路径替换。这对 AI 来说是必要的上下文信息。

7. **空值处理**：如果 input 中存在字段但值为 `nil`（JSON `null`），`ReplacePath` 将其视为缺失并报错。这比输出 `"<nil>"` 到 URL 中更安全。

8. **文件组织**：路径参数逻辑独立为 `pathparam.go` 文件，与 `validate.go` 平级，职责清晰。

## 10. 与旧方案的差异

| 维度 | 旧方案 (v1) | 新方案 (v2) |
|------|------------|------------|
| 路径参数来源 | 自动从 path 提取 `{xxx}` | template 中显式声明 `type: path` |
| YAML 变更 | 无需修改 | 需添加路径参数字段声明 |
| 参数进入 body | 保留在 body 中 | 从 input 中移除，不进入 body |
| 校验层修改 | 无需修改 validate.go | 需修改跳过 path 类型 |
| router 层修改 | 无需修改 | 无需修改（type 已是 string） |
| 配置一致性检查 | 无 | 校验 path 占位符与 template 声明一致 |
| 安全性 | 隐式依赖 path 格式 | 显式声明，配置错误可检测 |
