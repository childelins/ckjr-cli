# 响应过滤自动数组穿透设计文档

> Created: 2026-04-01
> Status: Draft
> Depends on: 2026-04-01-response-filter-design.md

## 概述

当前 response filter 的 `getNestedValue`/`setNestedValue`/`deleteNestedPath` 仅支持 `map[string]interface{}` 逐层遍历。当路径中间遇到 `[]interface{}`（数组）时返回 false/panic。

实际场景中，list 路由返回分页数据结构 `{list: {data: [{courseId: ..., name: ...}, ...], total: 1}}`，需要 `fields: [list.data.courseId]` 自动穿透 `data` 数组，对每个元素提取 `courseId`。

方案：自动数组穿透 -- 遍历路径时遇到 `[]interface{}`，自动对数组内每个元素应用剩余路径段，不引入新的配置语法。

## 1. 当前状况

### 现有三个核心函数

```
getNestedValue(m, "list.data.courseId")
  → list -> map ✓
  → data -> []interface{} ✗ (当前返回 nil, false)

setNestedValue(m, "list.data.courseId", val)
  → list -> map ✓
  → data -> []interface{} ✗ (当前 panic: interface conversion)

deleteNestedPath(m, "list.data.courseId")
  → list -> map ✓
  → data -> []interface{} ✗ (当前返回 false)
```

### 已确认的需求

- 配置语法不变，仍用 dot notation
- 无层数限制，for 循环逐层遍历
- 方案 B：自动穿透，不引入 `[]` 显式语法
- 覆盖 `filterByFields` 和 `filterByExclude` 两条路径

### 实际 API 响应结构（DoCtx 提取 Data 后）

get 路由：
```json
{
  "data": {"courseId": 15427611, "name": "Go 入门", ...}
}
```
fields 配置 `data.courseId` -- 纯 map 路径，已有逻辑可处理。

list 路由：
```json
{
  "list": {
    "current_page": 1,
    "data": [
      {"courseId": 15427611, "name": "Go 入门"},
      {"courseId": 15427612, "name": "Go 进阶"}
    ],
    "total": 2
  }
}
```
fields 配置 `list.data.courseId` -- `data` 是数组，需穿透。

## 2. 设计方案

### 2.1 getNestedValue 改造

当前逻辑：逐层 `current.(map[string]interface{})`，失败即返回。

改造后：逐层遍历时，若 `current` 是 `[]interface{}`，对数组内每个元素递归调用，将结果收集为 `[]interface{}` 返回。

```go
func getNestedValue(m map[string]interface{}, path string) (interface{}, bool) {
    parts := strings.Split(path, ".")
    var current interface{} = m
    for i, part := range parts {
        switch v := current.(type) {
        case map[string]interface{}:
            val, ok := v[part]
            if !ok {
                return nil, false
            }
            current = val
        case []interface{}:
            // 对数组内每个元素应用剩余路径
            remaining := strings.Join(parts[i:], ".")
            var results []interface{}
            for _, elem := range v {
                em, ok := elem.(map[string]interface{})
                if !ok {
                    continue // 跳过非 map 元素
                }
                if val, ok := getNestedValue(em, remaining); ok {
                    results = append(results, val)
                }
            }
            if len(results) == 0 {
                return nil, false
            }
            return results, true
        default:
            return nil, false
        }
    }
    return current, true
}
```

语义变化：
- `getNestedValue(m, "list.data.courseId")` 遇到 `data` 是数组 -> 对每个元素取 `courseId` -> 返回 `[]interface{}{15427611, 15427612}`
- 纯 map 路径行为不变，向后兼容

### 2.2 setNestedValue 改造

当前逻辑：逐层取 `map[string]interface{}`，最后一段赋值。

改造后：遇到 `[]interface{}` 时，对数组内每个 map 元素递归调用 `setNestedValue`。

```go
func setNestedValue(m map[string]interface{}, path string, value interface{}) {
    parts := strings.Split(path, ".")
    setNestedParts(m, parts, value)
}

func setNestedParts(m map[string]interface{}, parts []string, value interface{}) {
    for i := 0; i < len(parts)-1; i++ {
        next, ok := m[parts[i]]
        if !ok {
            next = make(map[string]interface{})
            m[parts[i]] = next
        }
        switch v := next.(type) {
        case map[string]interface{}:
            m = v
        case []interface{}:
            remaining := parts[i+1:]
            for _, elem := range v {
                if em, ok := elem.(map[string]interface{}); ok {
                    setNestedParts(em, remaining, value)
                }
            }
            return
        default:
            return
        }
    }
    m[parts[len(parts)-1]] = value
}
```

但 `setNestedValue` 在 `filterByFields` 中的用法需要特殊考虑。当 `getNestedValue` 返回 `[]interface{}` 时，`setNestedValue` 需要在目标 map 中重建包含数组的结构。

实际场景分析：
```
fields: ["list.data.courseId", "list.data.name", "list.total"]
```

对于 `list.data.courseId`：
- `getNestedValue` 返回 `[15427611, 15427612]`（穿透数组后收集的值列表）
- 但我们不能简单地 `setNestedValue(filtered, "list.data.courseId", [15427611, 15427612])`
- 因为这会把 `list.data` 设为 map 而非保持数组结构

**核心问题**：`filterByFields` 的 get-then-set 模式在数组穿透下不适用。需要改为在原结构上直接操作。

### 2.3 filterByFields 重构

改用递归 map 构建方式，遇到数组时对每个元素分别构建过滤后的 map：

```go
func filterByFields(m map[string]interface{}, fields []string) map[string]interface{} {
    filtered := make(map[string]interface{})
    for _, f := range fields {
        applyFieldPath(m, filtered, strings.Split(f, "."))
    }
    return filtered
}

func applyFieldPath(src, dst map[string]interface{}, parts []string) {
    if len(parts) == 0 {
        return
    }
    key := parts[0]
    val, ok := src[key]
    if !ok {
        return
    }
    if len(parts) == 1 {
        // 叶节点，直接复制
        dst[key] = val
        return
    }
    // 多段路径，递归
    remaining := parts[1:]
    switch v := val.(type) {
    case map[string]interface{}:
        sub, ok := dst[key].(map[string]interface{})
        if !ok {
            sub = make(map[string]interface{})
            dst[key] = sub
        }
        applyFieldPath(v, sub, remaining)
    case []interface{}:
        // 数组穿透：对每个元素应用剩余路径
        existingArr, _ := dst[key].([]interface{})
        if existingArr == nil {
            existingArr = make([]interface{}, len(v))
            for i := range existingArr {
                existingArr[i] = make(map[string]interface{})
            }
            dst[key] = existingArr
        }
        for i, elem := range v {
            em, ok := elem.(map[string]interface{})
            if !ok {
                continue
            }
            dm, ok := existingArr[i].(map[string]interface{})
            if !ok {
                dm = make(map[string]interface{})
                existingArr[i] = dm
            }
            applyFieldPath(em, dm, remaining)
        }
    }
}
```

这样 `fields: ["list.data.courseId", "list.data.name"]` 的处理过程：
1. 第一个 field `list.data.courseId`：构建 `{list: {data: [{courseId: 15427611}, {courseId: 15427612}]}}`
2. 第二个 field `list.data.name`：追加到同一结构 `{list: {data: [{courseId: 15427611, name: "Go 入门"}, {courseId: 15427612, name: "Go 进阶"}]}}`

### 2.4 deleteNestedPath 改造

同理，遇到数组时对每个元素递归删除：

```go
func deleteNestedPath(m map[string]interface{}, path string) bool {
    parts := strings.Split(path, ".")
    return deleteNestedParts(m, parts)
}

func deleteNestedParts(m map[string]interface{}, parts []string) bool {
    if len(parts) == 0 {
        return false
    }
    if len(parts) == 1 {
        _, exists := m[parts[0]]
        if !exists {
            return false
        }
        delete(m, parts[0])
        return true
    }
    val, ok := m[parts[0]]
    if !ok {
        return false
    }
    switch v := val.(type) {
    case map[string]interface{}:
        return deleteNestedParts(v, parts[1:])
    case []interface{}:
        deleted := false
        for _, elem := range v {
            if em, ok := elem.(map[string]interface{}); ok {
                if deleteNestedParts(em, parts[1:]) {
                    deleted = true
                }
            }
        }
        return deleted
    default:
        return false
    }
}
```

### 2.5 deepCopyMap 改造

当前 `deepCopyMap` 只复制 `map[string]interface{}`，不处理数组。`filterByExclude` 依赖 deepCopy，数组内的 map 不会被深拷贝，导致 `deleteNestedPath` 穿透数组删除时会修改原始数据。

```go
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
    cp := make(map[string]interface{}, len(m))
    for k, v := range m {
        cp[k] = deepCopyValue(v)
    }
    return cp
}

func deepCopyValue(v interface{}) interface{} {
    switch val := v.(type) {
    case map[string]interface{}:
        return deepCopyMap(val)
    case []interface{}:
        cp := make([]interface{}, len(val))
        for i, elem := range val {
            cp[i] = deepCopyValue(elem)
        }
        return cp
    default:
        return v
    }
}
```

### 2.6 getNestedValue 的角色变化

重构后 `filterByFields` 不再使用 get-then-set 模式，改为 `applyFieldPath` 递归构建。`getNestedValue` 仍保留并增加数组穿透能力，但其主要用途变为独立的值查询（如未来可能的条件过滤等场景）。

如果当前没有其他调用者，可考虑暂不改造 `getNestedValue`，保持原样。但为保持三个函数的一致性和可测试性，建议一并改造。

## 3. 修改点

### 3.1 修改文件：internal/cmdgen/filter.go

| 函数 | 变更 |
|------|------|
| `getNestedValue` | type switch 添加 `[]interface{}` 分支，递归穿透 |
| `setNestedValue` | 保留原签名，内部改用 `setNestedParts` 递归 |
| `filterByFields` | 改用 `applyFieldPath` 递归构建，替换 get-then-set |
| `applyFieldPath` | 新增，递归处理 map/array/leaf |
| `deleteNestedPath` | 保留原签名，内部改用 `deleteNestedParts` 递归 |
| `deleteNestedParts` | 新增，递归处理 map/array |
| `deepCopyMap` | 改用 `deepCopyValue` 处理数组 |
| `deepCopyValue` | 新增，递归复制 map/array/原始值 |
| `filterByExclude` | 无变更（依赖 deepCopyMap 和 deleteNestedPath） |
| `FilterResponse` | 无变更 |

### 3.2 不修改的文件

- `internal/router/router.go` -- 结构无变化
- `internal/cmdgen/cmdgen.go` -- 集成点不变，仍是 `FilterResponse(result, route.Response)`
- `internal/api/client.go` -- 不涉及
- `internal/output/output.go` -- 不涉及

## 4. 数据流

### fields 穿透数组

```
API 响应:
{
  "list": {
    "current_page": 1,
    "data": [
      {"courseId": 15427611, "name": "Go 入门", "secret": "x"},
      {"courseId": 15427612, "name": "Go 进阶", "secret": "y"}
    ],
    "total": 2
  }
}

fields: ["list.data.courseId", "list.data.name", "list.total"]

applyFieldPath 处理 "list.data.courseId":
  list -> map -> 递归
    data -> []interface{} -> 穿透
      元素0: {courseId: 15427611}
      元素1: {courseId: 15427612}

applyFieldPath 处理 "list.data.name":
  list -> 已有 map -> 递归
    data -> 已有数组 -> 穿透追加
      元素0: {courseId: 15427611, name: "Go 入门"}
      元素1: {courseId: 15427612, name: "Go 进阶"}

applyFieldPath 处理 "list.total":
  list -> 已有 map
    total -> 叶节点 -> 直接复制

结果:
{
  "list": {
    "data": [
      {"courseId": 15427611, "name": "Go 入门"},
      {"courseId": 15427612, "name": "Go 进阶"}
    ],
    "total": 2
  }
}
```

### exclude 穿透数组

```
API 响应: (同上)

exclude: ["list.data.secret"]

deepCopyMap -> 深拷贝（含数组内 map）
deleteNestedPath "list.data.secret":
  list -> map -> 递归
    data -> []interface{} -> 穿透
      元素0: delete "secret" ✓
      元素1: delete "secret" ✓

结果:
{
  "list": {
    "current_page": 1,
    "data": [
      {"courseId": 15427611, "name": "Go 入门"},
      {"courseId": 15427612, "name": "Go 进阶"}
    ],
    "total": 2
  }
}
```

### 纯 map 路径（向后兼容）

```
fields: ["data.courseId", "data.name"]

处理方式与改造前完全一致，不经过数组分支。
```

## 5. 边界情况

1. **空数组**：`data: []` -> 穿透后结果为空数组 `[]`，不报错
2. **数组内非 map 元素**：`[1, 2, "str"]` -> 跳过非 map 元素
3. **多层数组嵌套**：`a.b.c` 中 b 和 c 都是数组 -> 递归穿透，每层都展开
4. **数组内混合类型**：`[{id: 1}, "str", {id: 2}]` -> 仅处理 map 元素，跳过其余
5. **路径不存在**：`list.nonexistent.field` -> 静默跳过，返回空/false

## 6. 错误处理

保持现有策略 -- 过滤函数不产生错误，属于纯数据转换：

1. 路径中间遇到非 map/非 array 类型 -> 静默跳过
2. 数组为空 -> fields 返回空数组，exclude 保持空数组
3. 数组内元素缺少目标字段 -> 该元素跳过

## 7. 测试策略

### 新增测试用例

**getNestedValue 数组穿透：**
- `TestGetNestedValue_ArrayTraversal`：`list.data.courseId` 穿透数组取值
- `TestGetNestedValue_ArrayTraversal_EmptyArray`：空数组返回 nil, false
- `TestGetNestedValue_ArrayTraversal_NonMapElements`：数组内非 map 元素被跳过
- `TestGetNestedValue_ArrayTraversal_MixedElements`：数组内混合类型

**setNestedValue 数组穿透：**
- `TestSetNestedValue_ArrayTraversal`：穿透数组在每个元素中设值
- `TestSetNestedValue_ArrayTraversal_EmptyArray`：空数组无操作

**deleteNestedPath 数组穿透：**
- `TestDeleteNestedPath_ArrayTraversal`：穿透数组删除每个元素中的字段
- `TestDeleteNestedPath_ArrayTraversal_EmptyArray`：空数组返回 false

**deepCopyMap 数组处理：**
- `TestDeepCopyMap_WithArray`：深拷贝包含数组的 map
- `TestDeepCopyMap_WithNestedArrayMap`：数组内的 map 应独立于原始

**filterByFields 数组穿透：**
- `TestFilterByFields_ArrayTraversal`：完整的分页列表 fields 过滤
- `TestFilterByFields_ArrayTraversal_MultipleFields`：多个穿透同一数组的 fields
- `TestFilterByFields_ArrayTraversal_MixedPathTypes`：混合普通路径和数组穿透路径

**filterByExclude 数组穿透：**
- `TestFilterByExclude_ArrayTraversal`：穿透数组排除字段
- `TestFilterByExclude_ArrayTraversal_PreservesOriginal`：深拷贝不影响原始数据

**FilterResponse 集成：**
- `TestFilterResponse_ListWithFields`：list 路由 fields 场景端到端
- `TestFilterResponse_ListWithExclude`：list 路由 exclude 场景端到端

**向后兼容：**
- 所有现有测试必须继续通过

### TDD 顺序

1. `deepCopyMap` 数组支持 -- 测试 + 实现
2. `getNestedValue` 数组穿透 -- 测试 + 实现
3. `deleteNestedPath` 数组穿透 -- 测试 + 实现
4. `filterByExclude` 数组穿透 -- 测试验证（依赖 2、3）
5. `filterByFields` 重构为 `applyFieldPath` -- 测试 + 实现
6. FilterResponse 集成测试
7. 运行全量测试确认向后兼容

## 8. 实现注意事项

1. **`filterByFields` 必须重构**：get-then-set 模式无法保持数组结构。改用 `applyFieldPath` 递归构建是关键设计变更。

2. **`filterByExclude` 无需重构**：只需 `deepCopyMap` 正确复制数组、`deleteNestedPath` 正确穿透数组即可。现有的 deepCopy -> delete 模式天然适配。

3. **`deepCopyMap` 必须先改**：是 `filterByExclude` 正确性的前提。如果不深拷贝数组内的 map，穿透删除会修改原始数据。

4. **函数签名不变**：`getNestedValue`、`setNestedValue`、`deleteNestedPath` 的外部签名保持不变，内部通过 type switch 扩展行为。新增的 helper 函数（`applyFieldPath`、`setNestedParts`、`deleteNestedParts`、`deepCopyValue`）均为包内私有。

5. **递归深度**：实际 API 响应中数组嵌套不会超过 2-3 层，递归深度可控，无需设置上限。

6. **`setNestedValue` 保留但简化**：由于 `filterByFields` 不再使用它，`setNestedValue` 可以仅保留纯 map 路径功能（如无其他调用者），也可以一并加上数组穿透以保持一致性。建议一并改造。
