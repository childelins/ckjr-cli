# Response Field Descriptions 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 让 response fields 支持描述信息，--template 输出区分 request/response 层级

**Architecture:** 将 `ResponseFilter.Fields` 从 `[]string` 改为 `[]ResponseField`（含 Path+Description），通过自定义 `UnmarshalYAML` 兼容纯字符串和对象两种 YAML 格式。`printTemplateTo` 输出结构从扁平改为 `{ "request": {...}, "response": {...} }`。过滤逻辑通过 `FieldPaths()` 方法提取路径，保持不变。

**Tech Stack:** Go, gopkg.in/yaml.v3, cobra

---

## File Structure

- Modify: `internal/router/router.go` — 添加 ResponseField 类型、自定义 UnmarshalYAML、FieldPaths 方法
- Modify: `internal/router/router_test.go` — 新增混合格式 YAML 解析测试
- Modify: `internal/cmdgen/filter.go` — FilterResponse 改用 FieldPaths()
- Modify: `internal/cmdgen/filter_test.go` — 更新 ResponseFilter 构造方式
- Modify: `internal/cmdgen/cmdgen.go` — printTemplateTo 输出 request/response 结构
- Modify: `internal/cmdgen/cmdgen_test.go` — 更新 template 输出测试

---

### Task 1: ResponseField 类型 + 自定义 UnmarshalYAML

**Files:**
- Modify: `internal/router/router.go:27-31`
- Test: `internal/router/router_test.go`

- [ ] **Step 1: 写失败测试 — 混合格式 YAML 解析**

```go
func TestResponseFilter_MixedFieldFormats(t *testing.T) {
	yamlData := `
fields:
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

	if len(rf.Fields) != 4 {
		t.Fatalf("Fields count = %d, want 4", len(rf.Fields))
	}

	// 纯字符串格式
	if rf.Fields[0].Path != "data.courseId" {
		t.Errorf("Fields[0].Path = %q, want data.courseId", rf.Fields[0].Path)
	}
	if rf.Fields[0].Description != "" {
		t.Errorf("Fields[0].Description = %q, want empty", rf.Fields[0].Description)
	}

	// 对象格式
	if rf.Fields[1].Path != "data.courseType" {
		t.Errorf("Fields[1].Path = %q, want data.courseType", rf.Fields[1].Path)
	}
	if rf.Fields[1].Description != "课程类型, 0-视频 1-音频 2-图文" {
		t.Errorf("Fields[1].Description = %q", rf.Fields[1].Description)
	}

	// FieldPaths 返回纯路径列表
	paths := rf.FieldPaths()
	want := []string{"data.courseId", "data.courseType", "data.status", "data.name"}
	for i, p := range paths {
		if p != want[i] {
			t.Errorf("FieldPaths()[%d] = %q, want %q", i, p, want[i])
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/router/ -run TestResponseFilter_MixedFieldFormats -v`
Expected: FAIL — ResponseField 未定义

- [ ] **Step 3: 实现 ResponseField + UnmarshalYAML + FieldPaths**

在 `internal/router/router.go` 中：

```go
// ResponseField 定义响应字段（路径 + 可选描述）
type ResponseField struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description,omitempty"`
}

// ResponseFilter 定义响应字段过滤规则
type ResponseFilter struct {
	Fields  []ResponseField `yaml:"-"`
	Exclude []string        `yaml:"exclude,omitempty"`
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
	var raw struct {
		Fields  []yaml.Node `yaml:"fields"`
		Exclude []string    `yaml:"exclude"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	rf.Exclude = raw.Exclude
	for _, node := range raw.Fields {
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

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/router/ -run TestResponseFilter_MixedFieldFormats -v`
Expected: PASS

- [ ] **Step 5: 写纯字符串向后兼容测试**

```go
func TestResponseFilter_BackwardCompat_PureStrings(t *testing.T) {
	yamlData := `
fields:
  - courseId
  - name
  - status
`
	var rf ResponseFilter
	if err := yaml.Unmarshal([]byte(yamlData), &rf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rf.Fields) != 3 {
		t.Fatalf("Fields count = %d, want 3", len(rf.Fields))
	}
	paths := rf.FieldPaths()
	if paths[0] != "courseId" || paths[1] != "name" || paths[2] != "status" {
		t.Errorf("FieldPaths() = %v", paths)
	}
}
```

- [ ] **Step 6: 运行所有 router 测试确认无回归**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/router/ -v`
Expected: ALL PASS（包含已有的 TestRoute_ResponseFilter_Unmarshal 等）

- [ ] **Step 7: 提交**

```bash
git add internal/router/router.go internal/router/router_test.go
git commit -m "feat(router): add ResponseField type with mixed YAML format support"
```

---

### Task 2: 迁移 FilterResponse 使用 FieldPaths

**Files:**
- Modify: `internal/cmdgen/filter.go:193-212`
- Modify: `internal/cmdgen/filter_test.go`

- [ ] **Step 1: 修改 FilterResponse 使用 FieldPaths()**

在 `internal/cmdgen/filter.go` 中将：

```go
if len(respFilter.Fields) > 0 {
    return filterByFields(m, respFilter.Fields)
}
```

改为：

```go
paths := respFilter.FieldPaths()
if len(paths) > 0 {
    return filterByFields(m, paths)
}
```

- [ ] **Step 2: 更新 filter_test.go 中所有 ResponseFilter 构造**

将所有 `&router.ResponseFilter{Fields: []string{...}}` 改为使用 `ResponseField` 切片：

```go
// 旧
filter := &router.ResponseFilter{Fields: []string{"a", "c"}}

// 新
filter := &router.ResponseFilter{Fields: []router.ResponseField{
    {Path: "a"},
    {Path: "c"},
}}
```

需要更新的测试函数：
- `TestFilterResponse_FieldsOnly`
- `TestFilterResponse_NestedFields`
- `TestFilterResponse_FieldsAndExclude`
- `TestFilterResponse_EmptyFields`
- `TestFilterResponse_FieldNotFound`
- `TestFilterResponse_EmptyResult`
- `TestFilterResponse_NonMapResult`
- `TestFilterResponse_SliceResult`
- `TestFilterResponse_ListWithFields`

- [ ] **Step 3: 运行全部 filter 测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestFilter -v`
Expected: ALL PASS

- [ ] **Step 4: 更新 cmdgen_test.go 中 ResponseFilter 构造**

`TestBuildSubCommand_ResponseFilter` 中的 `Fields: []string{...}` 也要改为 `[]router.ResponseField{...}`。

- [ ] **Step 5: 运行全部 cmdgen 测试确认无回归**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -v`
Expected: ALL PASS

- [ ] **Step 6: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go internal/cmdgen/cmdgen_test.go
git commit -m "refactor(filter): migrate FilterResponse to use FieldPaths"
```

---

### Task 3: --template 输出 request/response 结构

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:136-186`
- Test: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 写失败测试 — 有 response 时输出 request/response 结构**

```go
func TestPrintTemplate_WithResponse(t *testing.T) {
	template := map[string]router.Field{
		"courseType": {
			Description: "课程类型",
			Required:    false,
			Type:        "int",
		},
	}
	response := &router.ResponseFilter{
		Fields: []router.ResponseField{
			{Path: "list.data.courseId"},
			{Path: "list.data.courseType", Description: "课程类型, 0-视频 1-音频 2-图文"},
			{Path: "list.data.status", Description: "上架状态, 1-已上架 2-已下架"},
			{Path: "list.data.name"},
		},
	}

	var buf bytes.Buffer
	printTemplateTo(&buf, template, response)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v, output: %s", err, buf.String())
	}

	// 应有 request 和 response 两个顶层 key
	reqSection, ok := result["request"].(map[string]interface{})
	if !ok {
		t.Fatal("missing or invalid 'request' section")
	}
	if _, exists := reqSection["courseType"]; !exists {
		t.Error("request should contain courseType")
	}

	respSection, ok := result["response"].(map[string]interface{})
	if !ok {
		t.Fatal("missing or invalid 'response' section")
	}

	// 有描述的字段
	if respSection["list.data.courseType"] != "课程类型, 0-视频 1-音频 2-图文" {
		t.Errorf("response[list.data.courseType] = %v", respSection["list.data.courseType"])
	}
	// 无描述的字段值为空字符串
	if respSection["list.data.courseId"] != "" {
		t.Errorf("response[list.data.courseId] = %v, want empty string", respSection["list.data.courseId"])
	}
}
```

- [ ] **Step 2: 写失败测试 — 无 response 时输出不变**

```go
func TestPrintTemplate_WithoutResponse(t *testing.T) {
	template := map[string]router.Field{
		"name": {
			Description: "名称",
			Required:    true,
		},
	}

	var buf bytes.Buffer
	printTemplateTo(&buf, template, nil)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v, output: %s", err, buf.String())
	}

	// 应有 request 但无 response
	if _, ok := result["request"]; !ok {
		t.Error("should have 'request' section")
	}
	if _, ok := result["response"]; ok {
		t.Error("should not have 'response' section when no response filter")
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestPrintTemplate_With -v`
Expected: FAIL — printTemplateTo 签名不匹配

- [ ] **Step 4: 实现 printTemplateTo 结构化输出**

修改 `internal/cmdgen/cmdgen.go`：

```go
func printTemplate(template map[string]router.Field, response *router.ResponseFilter) {
	printTemplateTo(os.Stdout, template, response)
}

func printTemplateTo(w io.Writer, template map[string]router.Field, response *router.ResponseFilter) {
	result := make(map[string]interface{})

	// request 部分
	request := make(map[string]interface{})
	for name, field := range template {
		entry := map[string]interface{}{
			"description": field.Description,
			"required":    field.Required,
		}
		if field.Default != nil {
			entry["default"] = field.Default
		}
		t := field.Type
		if t == "" {
			t = "string"
		}
		entry["type"] = t
		if t == "path" {
			entry["note"] = "路径参数，必须包含在 JSON 中，将自动替换 URL 中的占位符"
		}
		if field.Example != "" {
			entry["example"] = field.Example
		}

		constraints := map[string]interface{}{}
		if field.Min != nil {
			constraints["min"] = *field.Min
		}
		if field.Max != nil {
			constraints["max"] = *field.Max
		}
		if field.MinLength != nil {
			constraints["minLength"] = *field.MinLength
		}
		if field.MaxLength != nil {
			constraints["maxLength"] = *field.MaxLength
		}
		if field.Pattern != "" {
			constraints["pattern"] = field.Pattern
		}
		if len(constraints) > 0 {
			entry["constraints"] = constraints
		}

		request[name] = entry
	}
	result["request"] = request

	// response 部分（仅在配置了 fields 时输出）
	if response != nil && len(response.Fields) > 0 {
		respFields := make(map[string]interface{})
		for _, f := range response.Fields {
			respFields[f.Path] = f.Description
		}
		result["response"] = respFields
	}

	output.Print(w, result, true)
}
```

- [ ] **Step 5: 更新调用方 — buildSubCommand 传递 response**

```go
// --template 模式：输出模板并退出
if showTemplate {
    printTemplate(route.Template, route.Response)
    return
}
```

- [ ] **Step 6: 更新已有 printTemplateTo 测试的调用签名**

所有调用 `printTemplateTo(&buf, template)` 改为 `printTemplateTo(&buf, template, nil)`：
- `TestPrintTemplate_TypeAndExample`
- `TestPrintTemplate_Constraints`
- `TestPrintTemplate_PathFieldNote`

对于这些已有测试，解析结果从 `result["fieldName"]` 改为 `result["request"].(map[string]interface{})["fieldName"]`。

- [ ] **Step 7: 运行全部 cmdgen 测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -v`
Expected: ALL PASS

- [ ] **Step 8: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/cmdgen_test.go
git commit -m "feat(template): restructure --template output with request/response sections"
```

---

### Task 4: 为 course.yaml 添加响应字段描述

**Files:**
- Modify: `cmd/ckjr-cli/routes/course.yaml`

- [ ] **Step 1: 更新 course.yaml list 路由的 response fields**

```yaml
    response:
        fields:
            - list.data.courseId
            - path: list.data.name
              description: 课程名称
            - path: list.data.courseType
              description: "课程类型, 0-视频 1-音频 2-图文"
            - path: list.data.status
              description: "上架状态, 1-已上架 2-已下架"
            - path: list.data.isSaleOnly
              description: "是否单独售卖, 1-是 0-否"
            - list.data.price
            - path: list.data.payType
              description: "售卖类型, 1-免费 2-付费 3-加密"
            - list.data.courseAvatar
            - list.data.createdAt
            - path: list.data.contentAuditStatus
              description: 内容审核状态
            - list.data.adminUser
            - list.from
            - list.to
            - list.total
            - list.current_page
            - list.last_page
            - list.per_page
```

- [ ] **Step 2: 更新 course.yaml get 路由的 response fields**

```yaml
    response:
        fields:
            - data.courseId
            - data.name
            - path: data.courseType
              description: "课程类型, 0-视频 1-音频 2-图文"
            - path: data.status
              description: "上架状态, 1-已上架 2-已下架"
            - path: data.isSaleOnly
              description: "是否单独售卖, 1-是 0-否"
            - data.price
            - path: data.payType
              description: "售卖类型, 1-免费 2-付费 3-加密"
            - data.courseAvatar
            - data.detailInfo
            - path: data.playMode
              description: "播放模式, 1-横屏 2-竖屏（仅视频课程）"
            - path: data.articleType
              description: "图文模式, 1-图文 2-文章 3-多图（仅图文课程）"
            - data.articleImgs
            - data.delayTime
            - data.createdAt
            - data.updatedAt
```

- [ ] **Step 3: 验证 YAML 解析**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/router/ -v`
Expected: ALL PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/ckjr-cli/routes/course.yaml
git commit -m "feat(course): add response field descriptions to list and get routes"
```
