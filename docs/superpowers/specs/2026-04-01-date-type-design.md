# date 类型参数设计文档

> Created: 2026-04-01
> Status: Draft

## 概述

为 route YAML 模板的类型系统新增 `date` 类型，支持日期时间格式校验。当前类型系统支持 string/int/float/bool/array/path 六种类型，`date` 是第七种。date 类型在 JSON 中以 string 传递，运行时通过 Go 的 `time.Parse` 进行格式和合法性双重校验。

已确认的设计决策：
1. 格式：仅支持 `YYYY-MM-DD HH:MM:SS` 单一严格格式
2. 错误提示：校验不通过时在错误信息中展示期望格式
3. 校验方式：使用 `time.Parse` 做完整校验（格式 + 日期合法性）

## 1. date 类型语义

### 在 JSON 中的表示

date 类型值在 JSON 中以 string 形式传递：

```json
{"delayTime": "2026-03-31 17:11:56"}
```

经过 `json.Unmarshal` 后，Go 中的值类型为 `string`。

### 格式规范

唯一接受的格式：`YYYY-MM-DD HH:MM:SS`

对应 Go 的 time layout：`"2006-01-02 15:04:05"`

合法示例：
- `"2026-03-31 17:11:56"`
- `"2026-01-01 00:00:00"`

非法示例：
- `"2026-03-31"`（缺少时间部分）
- `"2026/03/31 17:11:56"`（错误的分隔符）
- `"2026-02-30 12:00:00"`（2 月无 30 日，time.Parse 会拒绝）
- `"2026-13-01 00:00:00"`（月份超出范围）

## 2. 校验逻辑

### validateType 新增 date case

在 `internal/cmdgen/validate.go` 的 `validateType` 函数 switch 中新增：

```go
case "date":
    str, ok := value.(string)
    if !ok {
        return &FieldValidationError{
            Field:   fieldName,
            Message: fmt.Sprintf("类型应为 date（字符串格式），实际为 %T", value),
        }
    }
    const dateLayout = "2006-01-02 15:04:05"
    if _, err := time.Parse(dateLayout, str); err != nil {
        return &FieldValidationError{
            Field:   fieldName,
            Message: fmt.Sprintf("日期格式应为 YYYY-MM-DD HH:MM:SS，实际值 %q 无效: %s", str, err.Error()),
        }
    }
```

### 校验流程

1. 先检查值是否为 `string` 类型（JSON 中日期以字符串传递）
2. 再用 `time.Parse("2006-01-02 15:04:05", str)` 解析
3. `time.Parse` 同时验证格式和日期合法性（闰年、月份天数等）

### 约束校验

date 类型不参与 `validateConstraints` 中的任何约束分支（不需要 min/max/minLength/maxLength/pattern）。现有的 `validateConstraints` switch 无 `case "date"` 分支，自动跳过。

如果 YAML 中对 date 字段设置了约束（如 min），按照现有设计的宽松策略，约束会被静默忽略，不报错。

## 3. --template 输出

### date 类型展示

在 `printTemplateTo` 中，date 类型需要添加格式说明 note，与 path 类型的处理模式一致：

```go
if t == "date" {
    entry["note"] = "日期格式: YYYY-MM-DD HH:MM:SS"
}
```

--template 输出示例：

```json
{
  "request": {
    "delayTime": {
      "description": "定时上架时间，需大于当前时间",
      "required": false,
      "type": "date",
      "note": "日期格式: YYYY-MM-DD HH:MM:SS"
    }
  }
}
```

## 4. 修改点

### 4.1 internal/cmdgen/validate.go

- `validateType` 函数 switch 新增 `case "date"`
- 新增 `import "time"` 依赖
- dateLayout 常量定义在 case 内部（仅一处使用，无需包级常量）

### 4.2 internal/cmdgen/cmdgen.go

- `printTemplateTo` 函数中，在 path 类型的 note 判断后新增 date 类型的 note

### 4.3 wiki/core-concepts.md

- 类型校验表新增 `date` 行

### 4.4 wiki/extending.md

- template 字段完整属性表中 type 描述新增 date

### 不需要修改的文件

- `internal/router/router.go`：Field 结构体无需变更，date 只是 Type 字段的新合法值
- `internal/cmdgen/validate.go` 中的 `validateConstraints`：date 无约束需求，现有 switch 自动跳过
- `internal/cmdgen/pathparam.go`：IsPathParam 仅检查 `type == "path"`，不受影响
- `cmd/ckjr-cli/routes/course.yaml`：已声明 `type: date`，无需修改

## 5. 错误信息示例

类型不匹配（非字符串值）：
```
字段 "delayTime" 类型应为 date（字符串格式），实际为 float64
```

格式不正确：
```
字段 "delayTime" 日期格式应为 YYYY-MM-DD HH:MM:SS，实际值 "2026-03-31" 无效: parsing time "2026-03-31" as "2006-01-02 15:04:05": cannot parse "" as "15"
```

日期不合法：
```
字段 "delayTime" 日期格式应为 YYYY-MM-DD HH:MM:SS，实际值 "2026-02-30 12:00:00" 无效: parsing time "2026-02-30 12:00:00": day out of range
```

## 6. 向后兼容

1. **Field 结构无变更**：date 只是 Type 字段的一个新合法值，不引入新的结构字段
2. **现有 YAML 不受影响**：course.yaml 中 delayTime 已声明 `type: date`，当前会报"未知类型"错误，修改后将正确校验
3. **未声明 type 的字段**：行为不变，不做类型校验
4. **validateConstraints**：date 不匹配任何约束 case，无副作用

## 7. 测试策略

### 单元测试

**internal/cmdgen/validate_test.go** 新增：

- `TestValidateType_Date_Valid`：合法日期通过
- `TestValidateType_Date_InvalidFormat`：格式错误被拒绝（如缺少时间部分）
- `TestValidateType_Date_InvalidDate`：日期不合法被拒绝（如 2 月 30 日）
- `TestValidateType_Date_NonString`：非字符串值被拒绝（如 float64）
- `TestValidateType_Date_ErrorMessage`：错误信息包含期望格式提示

测试用例表：

| 输入值 | 期望结果 | 说明 |
|--------|---------|------|
| `"2026-03-31 17:11:56"` | 通过 | 标准格式 |
| `"2026-01-01 00:00:00"` | 通过 | 边界值 |
| `"2026-12-31 23:59:59"` | 通过 | 边界值 |
| `"2026-03-31"` | 拒绝 | 缺少时间部分 |
| `"2026/03/31 17:11:56"` | 拒绝 | 错误分隔符 |
| `"2026-02-30 12:00:00"` | 拒绝 | 日期不合法 |
| `"not-a-date"` | 拒绝 | 完全无效 |
| `float64(20260331)` | 拒绝 | 非字符串类型 |
| `nil` | 拒绝 | nil 值（在 validateType 入口统一处理） |

**internal/cmdgen/cmdgen_test.go** 新增：

- `TestPrintTemplate_DateFieldNote`：date 类型 --template 输出包含 note

### 测试顺序（TDD）

1. 先写 `TestValidateType_Date_*` 系列测试（此时运行会失败，因为 date 进入 default 分支）
2. 实现 `validateType` 的 `case "date"` 分支
3. 写 `TestPrintTemplate_DateFieldNote` 测试
4. 实现 `printTemplateTo` 的 date note 逻辑
5. 更新文档

## 8. 实现注意事项

1. **time.Parse 行为**：Go 的 `time.Parse` 使用参考时间 `Mon Jan 2 15:04:05 MST 2006` 作为 layout。`"2006-01-02 15:04:05"` 对应 `YYYY-MM-DD HH:MM:SS`。time.Parse 会自动验证日期合法性，不需要额外的正则校验。

2. **nil 值处理**：`validateType` 函数已有 nil 检查在 switch 之前（第 74-76 行），date case 不需要重复处理 nil。

3. **值不转换**：date 类型仅做校验，不将字符串转换为 `time.Time`。值原样传递给 API 服务端。

4. **改动量极小**：仅在 validateType 新增一个 case 分支（约 15 行），在 printTemplateTo 新增一个 if 判断（3 行），加上测试和文档更新。
