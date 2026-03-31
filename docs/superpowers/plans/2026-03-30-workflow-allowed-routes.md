# Workflow allowed-routes 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 Workflow YAML 中增加 `allowed-routes` 字段，通过 `Describe()` 输出限制 AI 只能调用指定模块的路由。

**Architecture:** 在 `Workflow` struct 中新增 `AllowedRoutes []string` 字段（YAML tag: `allowed-routes`），`Describe()` 函数在非空时在描述和 Inputs 之间插入路由权限段落。软约束机制，通过文本告知 AI。

**Tech Stack:** Go, gopkg.in/yaml.v3

---

### Task 1: 添加 AllowedRoutes 字段 + 解析测试

**Files:**
- Modify: `internal/workflow/workflow.go:25-31`
- Modify: `internal/workflow/workflow_test.go`

- [ ] **Step 1: 添加解析 allowed-routes 的失败测试**

在 `workflow_test.go` 中添加测试，验证包含 `allowed-routes` 的 YAML 能正确解析：

```go
func TestParse_AllowedRoutes(t *testing.T) {
	yaml := `
name: test-workflows
description: 测试工作流
workflows:
  restricted-flow:
    description: 限制路由的流程
    triggers:
      - 测试
    allowed-routes:
      - agent
      - common
    inputs:
      - name: title
        description: 标题
        required: true
    steps:
      - id: step1
        description: 第一步
        command: agent create
        params:
          title: "{{inputs.title}}"
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	wf, ok := cfg.Workflows["restricted-flow"]
	if !ok {
		t.Fatal("缺少 restricted-flow workflow")
	}
	if len(wf.AllowedRoutes) != 2 {
		t.Errorf("AllowedRoutes 长度 = %d, want 2", len(wf.AllowedRoutes))
	}
	if wf.AllowedRoutes[0] != "agent" || wf.AllowedRoutes[1] != "common" {
		t.Errorf("AllowedRoutes = %v, want [agent, common]", wf.AllowedRoutes)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/workflow/ -run TestParse_AllowedRoutes -v`
Expected: FAIL — `AllowedRoutes` 字段不存在或长度为 0

- [ ] **Step 3: 在 Workflow struct 中添加 AllowedRoutes 字段**

在 `internal/workflow/workflow.go` 的 `Workflow` struct 中，`Steps` 和 `Summary` 之间添加：

```go
type Workflow struct {
	Description   string   `yaml:"description"`
	Triggers      []string `yaml:"triggers"`
	Inputs        []Input  `yaml:"inputs"`
	Steps         []Step   `yaml:"steps"`
	AllowedRoutes []string `yaml:"allowed-routes,omitempty"`
	Summary       string   `yaml:"summary,omitempty"`
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/workflow/ -run TestParse_AllowedRoutes -v`
Expected: PASS

- [ ] **Step 5: 运行全部 workflow 测试确保无回归**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/workflow/ -v`
Expected: 全部 PASS

---

### Task 2: Describe() 输出路由权限

**Files:**
- Modify: `internal/workflow/workflow.go:47-96`
- Modify: `internal/workflow/workflow_test.go`

- [ ] **Step 1: 添加 Describe() 路由权限输出测试**

在 `workflow_test.go` 中添加测试：

```go
func TestDescribe_AllowedRoutes(t *testing.T) {
	wf := Workflow{
		Description: "限制路由的流程",
		AllowedRoutes: []string{"agent", "common"},
		Inputs: []Input{
			{Name: "name", Description: "名称", Required: true},
		},
		Steps: []Step{
			{ID: "run", Description: "执行", Command: "agent create"},
		},
	}

	result := Describe(&wf, "restricted")

	// 应包含路由权限段落
	if !strings.Contains(result, "== 路由权限 ==") {
		t.Errorf("输出缺少路由权限标题\n实际输出:\n%s", result)
	}
	if !strings.Contains(result, "仅允许调用以下模块的路由: agent, common") {
		t.Errorf("输出缺少路由权限说明\n实际输出:\n%s", result)
	}

	// 路由权限应在描述之后、输入信息之前
	descIdx := strings.Index(result, "Description:")
	routesIdx := strings.Index(result, "== 路由权限 ==")
	inputsIdx := strings.Index(result, "== 需要收集的信息 ==")
	if descIdx >= routesIdx {
		t.Error("路由权限应在描述之后")
	}
	if routesIdx >= inputsIdx {
		t.Error("路由权限应在输入信息之前")
	}
}

func TestDescribe_NoAllowedRoutes(t *testing.T) {
	wf := Workflow{
		Description: "无限制流程",
		Inputs: []Input{
			{Name: "name", Description: "名称", Required: true},
		},
		Steps: []Step{
			{ID: "run", Description: "执行", Command: "agent create"},
		},
	}

	result := Describe(&wf, "unrestricted")

	if strings.Contains(result, "== 路由权限 ==") {
		t.Errorf("无 allowed-routes 时不应输出路由权限\n实际输出:\n%s", result)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/workflow/ -run TestDescribe_AllowedRoutes -v`
Expected: FAIL — 输出缺少路由权限段落

- [ ] **Step 3: 修改 Describe() 函数**

在 `internal/workflow/workflow.go` 的 `Describe()` 函数中，在 `Description` 输出之后、`// Inputs` 注释之前插入：

```go
	// Allowed routes
	if len(wf.AllowedRoutes) > 0 {
		fmt.Fprintf(&b, "\n== 路由权限 ==\n")
		fmt.Fprintf(&b, "仅允许调用以下模块的路由: %s\n", strings.Join(wf.AllowedRoutes, ", "))
	}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/workflow/ -v`
Expected: 全部 PASS

---

### Task 3: 更新 agent.yaml 并验证集成测试

**Files:**
- Modify: `cmd/ckjr-cli/workflows/agent.yaml:11`
- Modify: `internal/workflow/workflow_test.go:157-176`

- [ ] **Step 1: 更新集成测试以验证 allowed-routes**

在 `TestParse_AgentWorkflowFile` 中添加 `AllowedRoutes` 断言：

```go
func TestParse_AgentWorkflowFile(t *testing.T) {
	data, err := os.ReadFile("../../cmd/ckjr-cli/workflows/agent.yaml")
	if err != nil {
		t.Fatalf("读取 agent.yaml: %v", err)
	}
	cfg, err := Parse(data)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	wf, ok := cfg.Workflows["create-agent"]
	if !ok {
		t.Fatal("缺少 create-agent workflow")
	}
	if len(wf.Steps) != 3 {
		t.Errorf("Steps 长度 = %d, want 3", len(wf.Steps))
	}
	if len(wf.Inputs) != 5 {
		t.Errorf("Inputs 长度 = %d, want 5", len(wf.Inputs))
	}
	if len(wf.AllowedRoutes) != 2 {
		t.Errorf("AllowedRoutes 长度 = %d, want 2", len(wf.AllowedRoutes))
	}
	if wf.AllowedRoutes[0] != "agent" || wf.AllowedRoutes[1] != "common" {
		t.Errorf("AllowedRoutes = %v, want [agent, common]", wf.AllowedRoutes)
	}
}
```

- [ ] **Step 2: 运行测试验证失败（agent.yaml 尚未添加字段）**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/workflow/ -run TestParse_AgentWorkflowFile -v`
Expected: FAIL — AllowedRoutes 长度 != 2

- [ ] **Step 3: 在 agent.yaml 中添加 allowed-routes**

在 `cmd/ckjr-cli/workflows/agent.yaml` 的 `create-agent` workflow 中，`triggers` 之后添加：

```yaml
    allowed-routes:
      - agent
      - common
```

- [ ] **Step 4: 运行全部测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/workflow/ -v`
Expected: 全部 PASS

- [ ] **Step 5: 编译验证**

Run: `cd /home/childelins/code/ckjr-cli && go build ./...`
Expected: 无错误

- [ ] **Step 6: Commit**

```bash
git add internal/workflow/workflow.go internal/workflow/workflow_test.go cmd/ckjr-cli/workflows/agent.yaml
git commit -m "feat(workflow): add allowed-routes field to restrict AI route access"
```
