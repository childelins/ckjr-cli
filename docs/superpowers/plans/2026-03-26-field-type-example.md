# Field Type/Example 字段扩展实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 为 agent.yaml 参数定义增加 type 和 example 字段，提升 --template 输出的信息完整度

**Architecture:** 扩展 Field 结构体增加两个可选 yaml 字段，printTemplate 展示层处理默认值和条件输出，agent.yaml 为数值型参数补充 type

**Tech Stack:** Go, yaml.v3, cobra

**Spec:** `docs/superpowers/specs/2026-03-26-field-type-example-design.md`

---

## 文件结构

| 文件 | 职责 | 操作 |
|------|------|------|
| `internal/router/router.go` | Field 结构体定义 | 修改 |
| `internal/router/router_test.go` | Field 解析测试 | 修改 |
| `internal/cmdgen/cmdgen.go` | printTemplate 输出逻辑 | 修改 |
| `internal/cmdgen/cmdgen_test.go` | printTemplate 测试 | 修改 |
| `cmd/routes/agent.yaml` | 路由参数定义 | 修改 |

---

### Task 1: Field 结构体增加 Type/Example 字段

**Files:**
- Modify: `internal/router/router.go:10-14`
- Modify: `internal/router/router_test.go`

- [ ] **Step 1: 写失败测试 — 验证 YAML 中 type/example 能正确解析**

在 `internal/router/router_test.go` 末尾添加：

```go
func TestParseRouteConfig_TypeAndExample(t *testing.T) {
	yamlContent := `
resource: test
description: 测试模块
routes:
  create:
    method: POST
    path: /create
    description: 创建
    template:
      count:
        description: 数量
        required: false
        default: 10
        type: int
        example: "10"
      name:
        description: 名称
        required: true
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	route, ok := cfg.Routes["create"]
	if !ok {
		t.Fatal("create route not found")
	}

	// 有 type/example 的字段
	countField := route.Template["count"]
	if countField.Type != "int" {
		t.Errorf("count.Type = %q, want \"int\"", countField.Type)
	}
	if countField.Example != "10" {
		t.Errorf("count.Example = %q, want \"10\"", countField.Example)
	}

	// 未设置 type/example 的字段，应为零值
	nameField := route.Template["name"]
	if nameField.Type != "" {
		t.Errorf("name.Type = %q, want \"\"", nameField.Type)
	}
	if nameField.Example != "" {
		t.Errorf("name.Example = %q, want \"\"", nameField.Example)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/router/ -run TestParseRouteConfig_TypeAndExample -v`
Expected: FAIL — Field 结构体无 Type/Example 字段，编译错误

- [ ] **Step 3: 实现 — Field 结构体增加字段**

修改 `internal/router/router.go` 的 Field 结构体：

```go
// Field 定义模板字段
type Field struct {
	Description string      `yaml:"description"`
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default"`
	Type        string      `yaml:"type"`
	Example     string      `yaml:"example"`
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/router/ -v`
Expected: ALL PASS

- [ ] **Step 5: 提交**

```bash
git add internal/router/router.go internal/router/router_test.go
git commit -m "feat(router): add Type and Example fields to Field struct"
```

---

### Task 2: printTemplate 输出 type 和 example

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:121-134`
- Modify: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 写失败测试 — 验证 printTemplate 输出包含 type 和 example**

在 `internal/cmdgen/cmdgen_test.go` 末尾添加：

```go
func TestPrintTemplate_TypeAndExample(t *testing.T) {
	template := map[string]router.Field{
		"count": {
			Description: "数量",
			Required:    false,
			Default:     10,
			Type:        "int",
			Example:     "10",
		},
		"name": {
			Description: "名称",
			Required:    true,
		},
	}

	var buf bytes.Buffer
	printTemplateTo(&buf, template)
	var result map[string]map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	// count: 有 type=int 和 example
	countEntry := result["count"]
	if countEntry["type"] != "int" {
		t.Errorf("count.type = %v, want \"int\"", countEntry["type"])
	}
	if countEntry["example"] != "10" {
		t.Errorf("count.example = %v, want \"10\"", countEntry["example"])
	}

	// name: 无 type 应默认 string，无 example 应不存在
	nameEntry := result["name"]
	if nameEntry["type"] != "string" {
		t.Errorf("name.type = %v, want \"string\"", nameEntry["type"])
	}
	if _, exists := nameEntry["example"]; exists {
		t.Error("name should not have example field")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestPrintTemplate_TypeAndExample -v`
Expected: FAIL — `printTemplateTo` 未定义

- [ ] **Step 3: 实现 — 重构 printTemplate 并增加 type/example 输出**

修改 `internal/cmdgen/cmdgen.go`：

1. 将 `printTemplate` 拆分为 `printTemplateTo`（接受 `io.Writer`，方便测试）和 `printTemplate`（调用前者写到 stdout）：

```go
func printTemplate(template map[string]router.Field) {
	printTemplateTo(os.Stdout, template)
}

func printTemplateTo(w io.Writer, template map[string]router.Field) {
	tmpl := make(map[string]interface{})
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
		if field.Example != "" {
			entry["example"] = field.Example
		}
		tmpl[name] = entry
	}
	output.PrintTo(w, tmpl, true)
}
```

2. 需要确认 `output.PrintTo` 是否存在，如果不存在需要添加。检查 `internal/output/` 包。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -v`
Expected: ALL PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/cmdgen_test.go
git commit -m "feat(cmdgen): show type and example in --template output"
```

---

### Task 3: 更新 agent.yaml 为数值型参数补充 type

**Files:**
- Modify: `cmd/routes/agent.yaml`

- [ ] **Step 1: 运行现有测试确保基线通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: 更新 agent.yaml**

为数值型参数补充 `type: int`，description 保持不变：

```yaml
resource: agent
description: AI智能体管理模块
routes:
  list:
    method: POST
    path: /admin/aiCreationCenter/listApp
    description: 获取智能体列表
    template:
      page:
        description: 页码
        required: false
        default: 1
        type: int
      limit:
        description: 每页数量
        required: false
        default: 10
        type: int
      name:
        description: 按名称搜索
        required: false
      enablePagination:
        description: 是否分页返回, 1-是 0-否
        required: false
        default: 0
        type: int
      platType:
        description: 智能体类型, 0-全部 1-自营智能体 2-Coze智能体
        required: false
        default: 0
        type: int
  get:
    method: POST
    path: /admin/aiCreationCenter/describeApp
    description: 获取智能体详情
    template:
      aikbId:
        description: 智能体ID
        required: true
  create:
    method: POST
    path: /admin/aiCreationCenter/createApp
    description: 创建智能体
    template:
      name:
        description: 智能体名称
        required: true
      avatar:
        description: 头像URL
        required: true
      desc:
        description: 描述
        required: true
      botType:
        description: 智能体类型, 99-自营智能体 100-Coze智能体
        required: false
        default: 99
        type: int
      isSaleOnly:
        description: 是否支持售卖, 1-交付型 0-客服型
        required: false
        default: 1
        type: int
      promptType:
        description: 提示词模板类型, 1-交付型 3-角色类/客服型
        required: false
        default: 3
        type: int
  update:
    method: POST
    path: /admin/aiCreationCenter/modifyApp
    description: 更新智能体
    template:
      aikbId:
        description: 智能体ID
        required: true
      name:
        description: 智能体名称
        required: true
      avatar:
        description: 头像URL
        required: true
      desc:
        description: 描述
        required: true
  delete:
    method: POST
    path: /admin/aiCreationCenter/deleteApp
    description: 删除智能体
    template:
      aikbId:
        description: 智能体ID
        required: true
```

- [ ] **Step 3: 运行全量测试确认无回归**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/routes/agent.yaml
git commit -m "feat(routes): add type field to numeric parameters in agent.yaml"
```
