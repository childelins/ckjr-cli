# 简化 response 配置 + agent workflow upload-avatar 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 简化 route YAML 的 response 配置（去掉 fields/exclude 嵌套），并在 agent workflow 中添加 upload-avatar 步骤。

**Architecture:** response 从 `{fields: [...], exclude: [...]}` 对象改为直接数组 `[...]`。移除 exclude 全部代码。agent workflow 仿照 course workflow 添加 upload-avatar 前置步骤。

**Tech Stack:** Go, YAML (gopkg.in/yaml.v3), Cobra

---

## File Structure

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/router/router.go` | 修改 | ResponseFilter 结构简化，UnmarshalYAML 重写 |
| `internal/router/router_test.go` | 修改 | 新格式测试，移除 exclude 测试 |
| `internal/cmdgen/filter.go` | 修改 | 移除 exclude 相关函数，简化 FilterResponse |
| `internal/cmdgen/filter_test.go` | 修改 | 移除 exclude 测试，更新构造方式 |
| `internal/cmdgen/cmdgen_test.go` | 修改 | 移除 exclude 集成测试 |
| `cmd/ckjr-cli/routes/agent.yaml` | 修改 | response 格式更新 |
| `cmd/ckjr-cli/routes/course.yaml` | 修改 | response 格式更新 |
| `cmd/ckjr-cli/routes/common.yaml` | 修改 | response 格式更新 |
| `cmd/ckjr-cli/workflows/agent.yaml` | 修改 | 添加 upload-avatar 步骤 |
| `wiki/core-concepts.md` | 修改 | 更新 response 文档 |
| `wiki/extending.md` | 修改 | 更新 response 文档 |

---

### Task 1: 更新 router_test.go 测试为新 YAML 格式

**Files:**
- Modify: `internal/router/router_test.go:220-345`

- [ ] **Step 1: 更新 TestRoute_ResponseFilter_Unmarshal 测试为新格式**

将 YAML 从 `response.fields` 嵌套改为 `response` 直接数组：

```go
func TestRoute_ResponseFilter_Unmarshal(t *testing.T) {
	yamlData := `
method: GET
path: /admin/courses/{courseId}/edit
description: 获取课程详情
response:
    - courseId
    - name
    - status
`
	var route Route
	if err := yaml.Unmarshal([]byte(yamlData), &route); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if route.Response == nil {
		t.Fatal("Response should not be nil")
	}
	if len(route.Response.Fields) != 3 {
		t.Errorf("Fields count = %d, want 3", len(route.Response.Fields))
	}
}
```

- [ ] **Step 2: 删除 TestRoute_ResponseFilter_Exclude 测试**

整个 `TestRoute_ResponseFilter_Exclude` 函数（router_test.go:247-264），exclude 功能完全移除。

- [ ] **Step 3: 更新 TestResponseFilter_MixedFieldFormats 测试为新格式**

```go
func TestResponseFilter_MixedFieldFormats(t *testing.T) {
	yamlData := `
- data.courseId
- path: data.courseType
  description: "课程类型, 0-视频 1-音频 2-图文"
- path: data.status
  description: "上架状态, 1-已上架 2-已下架"
- data.name
`
	var rf ResponseFilter
	if err := yaml.Unmarshal([]byte(yamlData), &rf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ... 断言不变
```

注意：YAML 数据不再有 `fields:` 外层包裹，直接是数组。

- [ ] **Step 4: 更新 TestResponseFilter_BackwardCompat_PureStrings 测试**

```go
func TestResponseFilter_BackwardCompat_PureStrings(t *testing.T) {
	yamlData := `
- courseId
- name
- status
`
	var rf ResponseFilter
	// ... 其余不变
```

- [ ] **Step 5: 新增 TestResponseFilter_InvalidFormat 错误测试**

```go
func TestResponseFilter_InvalidFormat(t *testing.T) {
	yamlData := `
key: value
nested: true
`
	var rf ResponseFilter
	err := yaml.Unmarshal([]byte(yamlData), &rf)
	if err == nil {
		t.Error("expected error for non-sequence YAML")
	}
}
```

- [ ] **Step 6: 运行测试确认红灯**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/router/ -run "TestRoute_ResponseFilter|TestResponseFilter" -v`
Expected: FAIL（UnmarshalYAML 还是旧逻辑，无法解析新格式）

### Task 2: 重写 router.go 的 ResponseFilter

**Files:**
- Modify: `internal/router/router.go:33-71`

- [ ] **Step 1: 简化 ResponseFilter 结构体并重写 UnmarshalYAML**

```go
// ResponseFilter 定义响应字段过滤规则
type ResponseFilter struct {
	Fields []ResponseField
}

// FieldPaths 返回所有字段的路径列表（供过滤逻辑使用）
func (rf *ResponseFilter) FieldPaths() []string {
	paths := make([]string, len(rf.Fields))
	for i, f := range rf.Fields {
		paths[i] = f.Path
	}
	return paths
}

// UnmarshalYAML 自定义解析，支持纯字符串和 path+description 对象两种格式
func (rf *ResponseFilter) UnmarshalYAML(value *yaml.Node) error {
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

- [ ] **Step 2: 运行 router 测试确认绿灯**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/router/ -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add internal/router/router.go internal/router/router_test.go
git commit -m "refactor(router): simplify ResponseFilter to direct array, remove exclude"
```

### Task 3: 移除 filter.go 中 exclude 相关代码

**Files:**
- Modify: `internal/cmdgen/filter.go:60-213`
- Modify: `internal/cmdgen/filter_test.go`

- [ ] **Step 1: 更新 filter_test.go，移除所有 exclude 相关测试**

删除以下测试函数：
- `TestDeleteNestedPath`（filter_test.go:97-133）
- `TestDeleteNestedPath_ArrayTraversal`（filter_test.go:135-186）
- `TestDeepCopyMap`（filter_test.go:188-211）
- `TestDeepCopyMap_ArrayWithMaps`（filter_test.go:276-293）
- `TestFilterByExclude_NestedPath`（filter_test.go:378-404）
- `TestFilterByExclude_NestedPathPreservesOriginal`（filter_test.go:406-418）
- `TestFilterByExclude_ArrayTraversal`（filter_test.go:495-525）
- `TestFilterByExclude_BackwardCompatNoDot`（filter_test.go:544-554）
- `TestFilterByExclude_AllMatch`（filter_test.go:677-695）
- `TestFilterByExclude_PartialMatch`（filter_test.go:697-707）
- `TestFilterByExclude_NoneMatch`（filter_test.go:709-718）
- `TestFilterResponse_NestedExclude`（filter_test.go:587-608）
- `TestFilterResponse_ListWithExclude`（filter_test.go:773-800）
- `TestFilterResponse_ExcludeOnly`（filter_test.go:857-867）
- `TestFilterResponse_FieldsAndExclude`（filter_test.go:869-883）
- `TestFilterResponse_EmptyExclude`（filter_test.go:897-906）

更新 `TestFilterResponse_EmptyFields` 中的 `ResponseFilter` 构造（不再有 Exclude 字段）。

- [ ] **Step 2: 移除 filter.go 中 exclude 相关函数和简化 FilterResponse**

删除以下函数：
- `deleteNestedPath`（filter.go:60-64）
- `deleteNestedParts`（filter.go:66-95）
- `deepCopyValue`（filter.go:98-111）
- `deepCopyMap`（filter.go:114-120）
- `filterByExclude`（filter.go:179-189）

简化 `FilterResponse`：

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

- [ ] **Step 3: 运行 filter 测试确认绿灯**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run "TestFilter|TestGetNested|TestSetNested" -v`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go
git commit -m "refactor(cmdgen): remove exclude filtering, simplify FilterResponse"
```

### Task 4: 移除 cmdgen_test.go 中 exclude 集成测试

**Files:**
- Modify: `internal/cmdgen/cmdgen_test.go:737-803`

- [ ] **Step 1: 删除 TestBuildSubCommand_ResponseFilter_Exclude 测试函数**

删除整个 `TestBuildSubCommand_ResponseFilter_Exclude` 函数（cmdgen_test.go:737-803）。

- [ ] **Step 2: 运行全量 cmdgen 测试确认绿灯**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add internal/cmdgen/cmdgen_test.go
git commit -m "test(cmdgen): remove exclude integration test"
```

### Task 5: 更新 3 个 route YAML 文件

**Files:**
- Modify: `cmd/ckjr-cli/routes/agent.yaml`
- Modify: `cmd/ckjr-cli/routes/course.yaml`
- Modify: `cmd/ckjr-cli/routes/common.yaml`

- [ ] **Step 1: 更新 agent.yaml**

create 路由（注意当前有格式错误，`description` 是独立项而非 `path` 的子属性）：
```yaml
        response:
            - path: aikbId
              description: 智能体ID
```

get 路由：
```yaml
        response:
            - path: aikbId
              description: 智能体ID
            - path: name
              description: 智能体名称
            - path: avatar
              description: 智能体头像URL
            - path: desc
              description: 智能体描述
            - path: botType
              description: 智能体类型, 99-自营智能体, 100-扣子智能体
            - path: greeting
              description: 开场白文案
            - path: instructions
              description: 智能体提示词
            - path: price
              description: 价格
            - path: status
              description: 状态, 1-正常 3-下架
```

update 路由（同 create 有格式错误，一并修复）：
```yaml
        response:
            - path: aikbId
              description: 智能体ID
```

- [ ] **Step 2: 更新 course.yaml**

create 路由：
```yaml
        response:
            - path: courseId
              description: 课程ID
```

get 路由：
```yaml
        response:
            - data.courseId
            - data.name
            - path: data.courseType
              description: "课程类型, 0-视频 1-音频 2-图文"
            - path: data.status
              description: "上架状态, 1-已上架 2-已下架 3-定时上架"
            # ... 保持原有字段列表，去掉 fields 嵌套
```

list 路由同理。update 路由同理。

- [ ] **Step 3: 更新 common.yaml**

```yaml
        response:
            - path: url
              description: 访问链接
            - path: img
              description: 二维码
```

- [ ] **Step 4: 编译验证 YAML 解析**

Run: `cd /home/childelins/code/ckjr-cli && go build ./cmd/ckjr-cli/ && go test ./... 2>&1 | tail -20`
Expected: 编译成功，全量测试通过

- [ ] **Step 5: 提交**

```bash
git add cmd/ckjr-cli/routes/agent.yaml cmd/ckjr-cli/routes/course.yaml cmd/ckjr-cli/routes/common.yaml
git commit -m "refactor(routes): simplify response YAML format, remove fields nesting"
```

### Task 6: agent workflow 添加 upload-avatar 步骤

**Files:**
- Modify: `cmd/ckjr-cli/workflows/agent.yaml`

- [ ] **Step 1: 添加 upload-avatar 步骤和 asset allowed-route**

在 `create-agent` 工作流中：

1. `allowed-routes` 添加 `asset`
2. 在 `create` 步骤前插入 `upload-avatar` 步骤
3. 更新 `create` 和 `configure` 步骤的 avatar 引用

```yaml
    allowed-routes:
      - agent
      - common
      - asset
    # ...
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
          avatar: "{{steps.upload-avatar.imageUrl}}"
        output:
          aikbId: "response.aikbId"
      - id: configure
        description: 设置提示词和开场白
        command: agent update
        params:
          aikbId: "{{steps.create.aikbId}}"
          name: "{{inputs.name}}"
          desc: "{{inputs.desc}}"
          avatar: "{{steps.upload-avatar.imageUrl}}"
          instructions: "{{inputs.instructions}}"
          greeting: "{{inputs.greeting}}"
      # get-link 步骤不变
```

- [ ] **Step 2: 编译验证**

Run: `cd /home/childelins/code/ckjr-cli && go build ./cmd/ckjr-cli/`
Expected: 编译成功

- [ ] **Step 3: 提交**

```bash
git add cmd/ckjr-cli/workflows/agent.yaml
git commit -m "feat(workflow): add upload-avatar step to agent create-agent workflow"
```

### Task 7: 更新 wiki 文档

**Files:**
- Modify: `wiki/core-concepts.md:125-217`
- Modify: `wiki/extending.md:58-89`

- [ ] **Step 1: 更新 wiki/core-concepts.md**

替换 "响应字段过滤" 整个章节（第 125-217 行）：

1. 移除 "或使用黑名单排除特定字段" 示例（第 154-161 行的 exclude 示例）
2. 更新 YAML 配置示例为新格式（去掉 `fields:` 嵌套层级）
3. 更新 "点号路径" 说明中的 `fields 和 exclude` 为仅 `fields`
4. 更新 "语义规则" 表格，移除 exclude 相关行（第 212, 213, 215 行）

新 YAML 示例：
```yaml
        response:
            - data.courseId
            - path: data.courseType
              description: "课程类型, 0-视频 1-音频 2-图文"
            - path: data.status
              description: "上架状态, 1-已上架 2-已下架"
            - data.name
```

- [ ] **Step 2: 更新 wiki/extending.md**

替换 "response 字段（响应过滤）" 章节（第 58-89 行）：

1. 移除 exclude 相关注释和示例
2. 更新 YAML 示例为新格式
3. 移除属性表中 `exclude` 行
4. 移除互斥规则说明

新 YAML 示例：
```yaml
        response:
            - data.id
            - path: data.status
              description: "状态, 1-上架 2-下架"
            - data.name
```

新属性表只保留 `response` 为 list 类型，说明支持纯字符串和对象格式混合。

- [ ] **Step 3: 提交**

```bash
git add wiki/core-concepts.md wiki/extending.md
git commit -m "docs: update response filter documentation, remove exclude references"
```

### Task 8: 全量验证

- [ ] **Step 1: 全量测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... -v 2>&1 | tail -30`
Expected: ALL PASS

- [ ] **Step 2: 编译并手动验证 --template 输出**

Run: `cd /home/childelins/code/ckjr-cli && go build -o ckjr-cli ./cmd/ckjr-cli/ && ./ckjr-cli course get --template 2>/dev/null | head -20`
Expected: 输出包含 request 和 response 两部分，response 字段正确
