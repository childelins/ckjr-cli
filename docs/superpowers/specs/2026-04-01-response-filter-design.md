# 响应字段过滤设计文档

> Created: 2026-04-01
> Status: Draft

## 概述

当前 CLI 对 API 响应的 `data` 字段全量输出为 JSON。当 API 返回大量字段（包含冗余或敏感信息）时，输出过于冗长。需要在 route YAML 中声明响应字段过滤规则，在输出前按规则过滤 `result` 中的顶层字段。

已确认的设计决策：
1. 支持 `fields`（白名单）和 `exclude`（黑名单）两种过滤方式
2. 仅过滤顶层字段，不支持嵌套路径
3. 配置中声明但响应中不存在的字段，静默跳过

## 1. 当前状况

### 现有输出链路

```
client.DoCtx(ctx, method, path, input, &result)
  |
  apiResp.Data 反序列化到 result (interface{}, 即 map[string]interface{})
  |
  output.Print(os.Stdout, result, pretty)   -- 全量 JSON 输出，无过滤
```

关键代码位置：
- `internal/cmdgen/cmdgen.go:118-124`：请求执行和输出
- `internal/api/client.go:334-343`：data 解析
- `internal/output/output.go`：纯 JSON 序列化

### Route 结构

当前 Route 仅有 4 个字段：`method`, `path`, `description`, `template`。`template` 定义请求参数，没有响应字段定义。

## 2. YAML 格式变更

### 2.1 新增 response 字段

在 Route 级别新增 `response` 字段，包含 `fields` 和 `exclude` 两个可选列表：

```yaml
get:
    method: GET
    path: /admin/courses/{courseId}/edit
    description: 获取课程详情
    template:
        courseId:
            description: 课程ID
            required: true
            type: path
    response:
        fields:
            - courseId
            - name
            - courseType
            - status
```

黑名单模式示例：

```yaml
list:
    method: GET
    path: /admin/courses
    description: 获取课程列表
    template:
        # ...
    response:
        exclude:
            - detailInfo
            - internalFlag
```

### 2.2 语义规则

1. `response` 整体可选。未配置时，行为与当前一致，全量输出
2. `fields` 和 `exclude` 互斥。同时配置时，`fields` 优先，`exclude` 被忽略
3. `fields` 为空列表等同于未配置（全量输出）
4. `exclude` 为空列表等同于未配置（全量输出）
5. 过滤仅作用于顶层 key，嵌套结构整体保留或移除

## 3. 设计方案

### 3.1 Route 结构扩展

`internal/router/router.go` 中 Route 结构新增：

```go
type ResponseFilter struct {
    Fields  []string `yaml:"fields,omitempty"`
    Exclude []string `yaml:"exclude,omitempty"`
}

type Route struct {
    Method      string             `yaml:"method"`
    Path        string             `yaml:"path"`
    Description string             `yaml:"description"`
    Template    map[string]Field   `yaml:"template"`
    Response    *ResponseFilter    `yaml:"response,omitempty"`
}
```

`Response` 使用指针类型，nil 表示未配置。YAML 中不写 `response` 时，反序列化后为 nil，无内存开销。

### 3.2 过滤函数

新增文件 `internal/cmdgen/filter.go`：

```go
package cmdgen

import "github.com/childelins/ckjr-cli/internal/router"

// FilterResponse 根据 Route 的 response 配置过滤 result 的顶层字段
// 返回过滤后的新 map，不修改原始 result
func FilterResponse(result interface{}, respFilter *router.ResponseFilter) interface{} {
    if respFilter == nil {
        return result
    }

    m, ok := result.(map[string]interface{})
    if !ok {
        return result
    }

    // fields 白名单优先
    if len(respFilter.Fields) > 0 {
        return filterByFields(m, respFilter.Fields)
    }

    // exclude 黑名单
    if len(respFilter.Exclude) > 0 {
        return filterByExclude(m, respFilter.Exclude)
    }

    return result
}

// filterByFields 仅保留 fields 中列出的顶层 key
func filterByFields(m map[string]interface{}, fields []string) map[string]interface{} {
    allowed := make(map[string]bool, len(fields))
    for _, f := range fields {
        allowed[f] = true
    }

    filtered := make(map[string]interface{}, len(allowed))
    for k, v := range m {
        if allowed[k] {
            filtered[k] = v
        }
    }
    return filtered
}

// filterByExclude 移除 exclude 中列出的顶层 key
func filterByExclude(m map[string]interface{}, exclude []string) map[string]interface{} {
    excluded := make(map[string]bool, len(exclude))
    for _, e := range exclude {
        excluded[e] = true
    }

    filtered := make(map[string]interface{}, len(m))
    for k, v := range m {
        if !excluded[k] {
            filtered[k] = v
        }
    }
    return filtered
}
```

### 3.3 集成点

在 `internal/cmdgen/cmdgen.go` 的 `buildSubCommand` 中，输出前插入过滤调用：

```go
// 当前代码（第 118-124 行）：
var result interface{}
if err := client.DoCtx(ctx, route.Method, resolvedPath, input, &result); err != nil {
    handleAPIError(err, verbose)
    os.Exit(1)
}
output.Print(os.Stdout, result, pretty)

// 修改为：
var result interface{}
if err := client.DoCtx(ctx, route.Method, resolvedPath, input, &result); err != nil {
    handleAPIError(err, verbose)
    os.Exit(1)
}

// 响应字段过滤
result = FilterResponse(result, route.Response)

output.Print(os.Stdout, result, pretty)
```

改动极小：仅插入一行 `FilterResponse` 调用。

## 4. 修改点

### 4.1 修改文件：internal/router/router.go

新增 `ResponseFilter` 结构体，`Route` 中新增 `Response` 字段。见 3.1 节。

### 4.2 新增文件：internal/cmdgen/filter.go

包含 `FilterResponse`、`filterByFields`、`filterByExclude` 三个函数。见 3.2 节。

### 4.3 修改文件：internal/cmdgen/cmdgen.go

在 `output.Print` 之前插入过滤调用。见 3.3 节。

### 4.4 不修改的文件

- **internal/api/client.go**：`DoCtx` 逻辑不变，过滤在 cmdgen 层完成
- **internal/output/output.go**：接收的数据已是过滤后的，无需改动
- **internal/cmdgen/validate.go**：过滤与校验无关

## 5. 数据流

### 使用 fields 白名单

```
API 响应 data: {"courseId": 1, "name": "Go", "status": 1, "detailInfo": [...], "internalFlag": true}
          |
  route.Response.Fields = ["courseId", "name", "status"]
          |
  FilterResponse()
  - 遍历 data 的顶层 key
  - 仅保留 courseId, name, status
  - detailInfo, internalFlag 被排除
          |
  output: {"courseId": 1, "name": "Go", "status": 1}
```

### 使用 exclude 黑名单

```
API 响应 data: {"courseId": 1, "name": "Go", "detailInfo": [...], "internalFlag": true}
          |
  route.Response.Exclude = ["detailInfo", "internalFlag"]
          |
  FilterResponse()
  - 遍历 data 的顶层 key
  - 移除 detailInfo, internalFlag
          |
  output: {"courseId": 1, "name": "Go"}
```

### 未配置 response

```
API 响应 data: {"courseId": 1, "name": "Go", "detailInfo": [...]}
          |
  route.Response = nil
          |
  FilterResponse() -- 直接返回 result，无额外处理
          |
  output: {"courseId": 1, "name": "Go", "detailInfo": [...]}   (全量，与当前行为一致)
```

### fields 中声明但响应中不存在

```
API 响应 data: {"courseId": 1, "name": "Go"}
          |
  route.Response.Fields = ["courseId", "name", "createdAt"]
          |
  FilterResponse()
  - courseId: 存在 -> 保留
  - name: 存在 -> 保留
  - createdAt: 不存在 -> 静默跳过
          |
  output: {"courseId": 1, "name": "Go"}   (无警告)
```

## 6. 错误处理

过滤函数本身不产生错误，属于纯数据转换：

1. `route.Response` 为 nil -> 直接返回 result，不做任何处理
2. `result` 不是 `map[string]interface{}` -> 直接返回 result（如 result 为 nil、数组等异常情况）
3. `fields` 或 `exclude` 列表中的字段不存在于 result -> 静默跳过，不报错
4. `fields` 和 `exclude` 同时配置 -> `fields` 优先，`exclude` 被忽略（在 `FilterResponse` 中通过 if 分支保证）

## 7. 测试策略

### 单元测试：internal/cmdgen/filter_test.go（新建）

**FilterResponse 测试：**
- `TestFilterResponse_NilFilter`：response 为 nil 时原样返回
- `TestFilterResponse_NonMapResult`：result 不是 map 时原样返回（如 nil、数组）
- `TestFilterResponse_FieldsOnly`：仅配置 fields，白名单过滤正确
- `TestFilterResponse_ExcludeOnly`：仅配置 exclude，黑名单过滤正确
- `TestFilterResponse_FieldsAndExclude`：同时配置时 fields 优先
- `TestFilterResponse_EmptyFields`：fields 为空列表，等同于未配置
- `TestFilterResponse_EmptyExclude`：exclude 为空列表，等同于未配置
- `TestFilterResponse_FieldNotFound`：fields 中声明但 result 中不存在的字段，静默跳过
- `TestFilterResponse_ExcludeNotFound`：exclude 中声明但 result 中不存在的字段，无副作用
- `TestFilterResponse_EmptyResult`：result 为空 map，过滤后仍为空 map

**filterByFields 测试：**
- `TestFilterByFields_AllMatch`：所有 fields 都在 map 中
- `TestFilterByFields_PartialMatch`：部分 fields 不在 map 中
- `TestFilterByFields_NoneMatch`：所有 fields 都不在 map 中
- `TestFilterByFields_PreservesNested`：保留的顶层 key 的嵌套值完整

**filterByExclude 测试：**
- `TestFilterByExclude_AllMatch`：排除所有列出的 key
- `TestFilterByExclude_PartialMatch`：部分 exclude 不在 map 中
- `TestFilterByExclude_NoneMatch`：所有 exclude 都不在 map 中
- `TestFilterByExclude_EmptyExclude`：空列表不排除任何字段

### TDD 顺序

1. 先写 `filterByFields` 测试和实现
2. 再写 `filterByExclude` 测试和实现
3. 再写 `FilterResponse` 测试和实现（含边界情况）
4. 修改 `router.go`，添加 `ResponseFilter` 结构体
5. 修改 `cmdgen.go`，集成过滤调用
6. 更新 route YAML 文件（按需）

## 8. 实现注意事项

1. **不修改原始 result**：`filterByFields` 和 `filterByExclude` 创建新 map，确保 `api.Client.DoCtx` 的反序列化结果不被污染。

2. **互斥处理在 FilterResponse 层**：`fields` 和 `exclude` 的互斥逻辑集中在 `FilterResponse` 函数中，底层 `filterByFields`/`filterByExclude` 各自只做单一职责。

3. **Response 使用指针类型**：`Route.Response` 为 `*ResponseFilter`。YAML 未配置 `response` 时为 nil，无内存分配。这保持了向后兼容，已有的 route YAML 无需修改。

4. **过滤时机**：在 `client.DoCtx` 返回后、`output.Print` 之前。这是 cmdgen 层的职责，不侵入 API client 或 output 包。

5. **仅顶层过滤**：map 的遍历只处理第一层 key。嵌套对象/数组整体保留或移除，不做递归过滤。这保持实现简单，避免引入路径匹配的复杂性。

6. **不影响 --template 输出**：`printTemplate` 输出请求参数模板，与响应过滤无关。
