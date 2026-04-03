# 简化 response 配置 + agent workflow 补充 upload-avatar 设计文档

> Created: 2026-04-02
> Status: Draft

## 概述

本次变更包含两个子需求：

1. **简化 route YAML response 配置**：去掉 `response.fields` / `response.exclude` 嵌套层级，让 `response` 本身直接成为字段列表。完全废弃 `exclude` 功能。
2. **agent workflow 补充 upload-avatar 步骤**：在 `create-agent` 工作流中添加头像图片转存步骤，与 course workflow 保持一致。

## 子需求 1：简化 response 配置

### 当前格式 (Before)

```yaml
response:
    fields:
        - data.courseId
        - path: data.courseType
          description: "课程类型, 0-视频 1-音频 2-图文"
```

或：

```yaml
response:
    exclude:
        - detailInfo
        - internalFlag
```

### 目标格式 (After)

```yaml
response:
    - data.courseId
    - path: data.courseType
      description: "课程类型, 0-视频 1-音频 2-图文"
```

`response` 直接是字段数组，不再有 `fields` / `exclude` 嵌套。`exclude` 功能完全移除（当前没有任何 YAML 文件使用它）。

### 架构变更

#### 1. `internal/router/router.go` - ResponseFilter 结构简化

**Before:**

```go
type ResponseFilter struct {
    Fields  []ResponseField `yaml:"-"`
    Exclude []string        `yaml:"exclude,omitempty"`
}
```

**After:**

`ResponseFilter` 不再需要作为独立结构体。`Route.Response` 的类型从 `*ResponseFilter` 改为 `[]ResponseField`，配合自定义 `UnmarshalYAML` 直接解析数组。

具体方案：保留 `ResponseFilter` 类型名（避免修改面过大），但内部只保留 `Fields` 字段，移除 `Exclude`。

```go
type ResponseFilter struct {
    Fields []ResponseField
}
```

`UnmarshalYAML` 改为直接解析 YAML 数组节点（而非先解析为包含 `fields` key 的 map）。

#### 2. `internal/router/router.go` - UnmarshalYAML 重写

**Before:** 解析 `{fields: [...], exclude: [...]}` 结构。

**After:** 直接解析 `[...]` 数组，每个元素支持 scalar（纯字符串）和 mapping（path+description 对象）两种格式。

```go
func (rf *ResponseFilter) UnmarshalYAML(value *yaml.Node) error {
    // value 应该是 SequenceNode (数组)
    if value.Kind != yaml.SequenceNode {
        return fmt.Errorf("response should be a list, got %v", value.Kind)
    }
    for _, node := range value.Content {
        switch node.Kind {
        case yaml.ScalarNode:
            rf.Fields = append(rf.Fields, ResponseField{Path: node.Value})
        case yaml.MappingNode:
            var field ResponseField
            if err := node.Decode(&field); err != nil {
                return err
            }
            rf.Fields = append(rf.Fields, field)
        }
    }
    return nil
}
```

#### 3. `internal/cmdgen/filter.go` - 移除 exclude 相关代码

移除以下函数：
- `filterByExclude`
- `deleteNestedPath`
- `deleteNestedParts`

`FilterResponse` 简化为只处理 fields 逻辑：

```go
func FilterResponse(result interface{}, respFilter *router.ResponseFilter) interface{} {
    if respFilter == nil || len(respFilter.Fields) == 0 {
        return result
    }
    m, ok := result.(map[string]interface{})
    if !ok {
        return result
    }
    return filterByFields(m, respFilter.FieldPaths())
}
```

#### 4. `internal/cmdgen/cmdgen.go` - printTemplateTo 无需改动

`printTemplateTo` 已经通过 `response.Fields` 访问字段列表，逻辑不变。

#### 5. 更新 route YAML 文件

所有 4 个使用了 `response.fields` 的路由文件需要更新格式：

- `cmd/ckjr-cli/routes/agent.yaml` (create, get, update 路由)
- `cmd/ckjr-cli/routes/course.yaml` (create, get, list, update 路由)
- `cmd/ckjr-cli/routes/common.yaml` (link 路由)

示例变更（course.yaml get 路由）：

```yaml
# Before
response:
    fields:
        - data.courseId
        - data.name
        - path: data.courseType
          description: "课程类型, 0-视频 1-音频 2-图文"

# After
response:
    - data.courseId
    - data.name
    - path: data.courseType
      description: "课程类型, 0-视频 1-音频 2-图文"
```

#### 6. 更新测试文件

- `internal/router/router_test.go`：所有 response 相关测试用例更新 YAML 格式，移除 exclude 测试
- `internal/cmdgen/filter_test.go`：移除 `filterByExclude` 和 `deleteNestedPath` 相关测试，保留 `filterByFields` 和 `FilterResponse` 测试（更新 `ResponseFilter` 构造方式）
- `internal/cmdgen/cmdgen_test.go`：更新 `TestBuildSubCommand_ResponseFilter_Exclude` 测试（移除或改造）

#### 7. 更新文档

- `wiki/core-concepts.md`："响应字段过滤" 章节，移除 exclude 相关内容，更新 YAML 配置示例
- `wiki/extending.md`："response 字段" 章节，移除 exclude 配置和互斥规则说明

### 受影响的文件清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/router/router.go` | 修改 | ResponseFilter 结构体和 UnmarshalYAML |
| `internal/router/router_test.go` | 修改 | 更新 YAML 格式，移除 exclude 测试 |
| `internal/cmdgen/filter.go` | 修改 | 移除 exclude 相关函数，简化 FilterResponse |
| `internal/cmdgen/filter_test.go` | 修改 | 移除 exclude 测试，更新 ResponseFilter 构造 |
| `internal/cmdgen/cmdgen_test.go` | 修改 | 更新/移除 exclude 集成测试 |
| `cmd/ckjr-cli/routes/agent.yaml` | 修改 | response 格式更新 |
| `cmd/ckjr-cli/routes/course.yaml` | 修改 | response 格式更新 |
| `cmd/ckjr-cli/routes/common.yaml` | 修改 | response 格式更新 |
| `wiki/core-concepts.md` | 修改 | 移除 exclude 文档 |
| `wiki/extending.md` | 修改 | 移除 exclude 文档 |

## 子需求 2：agent workflow 补充 upload-avatar 步骤

### 当前状态

- course workflow 的 3 个工作流都已有 `upload-avatar` 步骤
- agent workflow 的 `create-agent` 有 `avatar` 输入但没有 `upload-avatar` 步骤
- `asset upload-image` 子命令已实现并可用

### 变更方案

在 `cmd/ckjr-cli/workflows/agent.yaml` 的 `create-agent` 工作流中，`create` 步骤之前插入 `upload-avatar` 步骤：

```yaml
steps:
  - id: upload-avatar
    description: 如果头像是外部图片URL，先转存到系统素材库
    command: asset upload-image
    params:
      url: "{{inputs.avatar}}"
    output:
      imageUrl: "response.imageUrl"
  - id: create
    description: 创建智能体基本信息
    command: agent create
    params:
      name: "{{inputs.name}}"
      desc: "{{inputs.desc}}"
      avatar: "{{steps.upload-avatar.imageUrl}}"   # 改为使用转存后的 URL
    output:
      aikbId: "response.aikbId"
  - id: configure
    description: 设置提示词和开场白
    command: agent update
    params:
      aikbId: "{{steps.create.aikbId}}"
      name: "{{inputs.name}}"
      desc: "{{inputs.desc}}"
      avatar: "{{steps.upload-avatar.imageUrl}}"   # 同步更新
      instructions: "{{inputs.instructions}}"
      greeting: "{{inputs.greeting}}"
  # ... get-link 步骤不变
```

同时需要在 `allowed-routes` 中添加 `asset`：

```yaml
allowed-routes:
  - agent
  - common
  - asset    # 新增
```

### 受影响的文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `cmd/ckjr-cli/workflows/agent.yaml` | 修改 | 添加 upload-avatar 步骤，更新 avatar 引用 |

## 数据流

### 子需求 1：response 解析流程

```
YAML 文件
  -> yaml.Unmarshal
  -> ResponseFilter.UnmarshalYAML (直接解析数组)
  -> ResponseFilter.Fields []ResponseField
  -> FilterResponse / printTemplateTo 使用
```

### 子需求 2：agent 创建头像转存流程

```
用户提供外部图片 URL
  -> upload-avatar 步骤: asset upload-image 转存到 OSS
  -> 获取系统内 imageUrl
  -> create 步骤: 使用系统 URL 创建智能体
  -> configure 步骤: 使用系统 URL 更新智能体配置
```

## 错误处理

### 子需求 1

- 如果 YAML 中 `response` 不是数组，`UnmarshalYAML` 返回格式错误
- 空的 `response: []` 等同于未配置，全量输出

### 子需求 2

- `upload-avatar` 步骤失败时（网络错误、URL 无效等），工作流在该步骤中止
- 与 course workflow 的 upload-avatar 行为一致，无额外错误处理

## 测试策略

### 子需求 1

1. **router_test.go 单元测试**：
   - 新格式 YAML 解析（纯字符串、对象格式、混合格式）
   - 空 response 数组
   - response 为 nil（未配置）
   - response 格式错误（非数组）

2. **filter_test.go 单元测试**：
   - 保留所有 `filterByFields` 和 `FilterResponse` 的 fields 相关测试
   - 移除所有 `filterByExclude` 和 `deleteNestedPath` 测试
   - 更新 `FilterResponse` 测试中 `ResponseFilter` 的构造方式（不再传 Exclude）

3. **cmdgen_test.go 集成测试**：
   - 保留 `TestBuildSubCommand_ResponseFilter` 测试（更新 ResponseFilter 构造）
   - 移除 `TestBuildSubCommand_ResponseFilter_Exclude` 测试
   - 保留 `TestBuildSubCommand_NoResponseFilter` 测试

4. **YAML 加载验证**：编译后对每个 route YAML 执行 `--template` 确认解析正常

### 子需求 2

- 编译后通过 `ckjr-cli workflow describe create-agent` 确认步骤结构正确
- 确认 `allowed-routes` 包含 `asset`

## 实现注意事项

1. **向后兼容**：这是破坏性变更。旧格式 `response.fields` 将无法解析。由于这是内部工具、YAML 文件内嵌在二进制中，所有 YAML 文件在同一次提交中更新，不存在兼容性问题。

2. **deepCopy 相关函数保留**：虽然移除了 `filterByExclude`（是 `deepCopyMap` 的唯一调用者），但 `deepCopyMap`/`deepCopyValue` 是通用工具函数，可保留供未来使用。也可选择一并移除（未使用代码），根据偏好决定。

3. **实现顺序建议**：
   - 先写测试（TDD）：更新 router_test.go 中的 YAML 格式 -> 红灯
   - 修改 router.go 实现 -> 绿灯
   - 移除 exclude 相关代码和测试
   - 更新所有 route YAML 文件
   - 更新 agent workflow
   - 更新 wiki 文档
   - 全量 `go test ./...` 验证
