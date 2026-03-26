# Workflow YAML 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 ckjr-cli 添加 workflow 层，让 AI 通过 `workflow describe` 一次性获取多步骤任务的完整编排定义，替代逐步 --help/--template 发现模式。

**Architecture:** 新增独立的 `internal/workflow/` 包处理 YAML 解析和文本描述生成，`cmd/workflows/` 目录存放 workflow YAML 文件并通过 `//go:embed` 嵌入，`cmd/workflow.go` 注册 `workflow list/describe` 子命令。不修改现有 router/cmdgen 架构。

**Tech Stack:** Go 1.24, Cobra, gopkg.in/yaml.v3, embed

---

## 文件结构

| 操作 | 路径 | 职责 |
|------|------|------|
| Create | `internal/workflow/workflow.go` | 数据结构 + Parse() + Describe() |
| Create | `internal/workflow/workflow_test.go` | workflow 包单元测试 |
| Create | `cmd/workflows/agent.yaml` | 智能体工作流定义 |
| Create | `cmd/workflow.go` | workflow list/describe 命令 |
| Create | `cmd/workflow_test.go` | workflow 命令测试 |
| Modify | `cmd/root.go:40-58` | init() 中注册 workflow 命令 |
| Modify | `skills/ckjr-cli/SKILL.md` | 添加 workflow 优先策略 |

---

### Task 1: workflow 包 - 数据结构与 Parse

**Files:**
- Create: `internal/workflow/workflow.go`
- Create: `internal/workflow/workflow_test.go`

- [ ] **Step 1: 写 Parse 失败测试**

```go
// internal/workflow/workflow_test.go
package workflow

import "testing"

func TestParse_InvalidYAML(t *testing.T) {
	_, err := Parse([]byte(":::invalid"))
	if err == nil {
		t.Fatal("期望解析无效 YAML 时返回错误")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/workflow/ -run TestParse_InvalidYAML -v`
Expected: FAIL (package 不存在)

- [ ] **Step 3: 写数据结构和 Parse 最小实现**

```go
// internal/workflow/workflow.go
package workflow

import "gopkg.in/yaml.v3"

type Input struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Hint        string `yaml:"hint,omitempty"`
}

type Step struct {
	ID          string            `yaml:"id"`
	Description string            `yaml:"description"`
	Command     string            `yaml:"command"`
	Params      map[string]string `yaml:"params"`
	Output      map[string]string `yaml:"output,omitempty"`
}

type Workflow struct {
	Description string   `yaml:"description"`
	Triggers    []string `yaml:"triggers"`
	Inputs      []Input  `yaml:"inputs"`
	Steps       []Step   `yaml:"steps"`
	Summary     string   `yaml:"summary,omitempty"`
}

type Config struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Workflows   map[string]Workflow `yaml:"workflows"`
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/workflow/ -run TestParse_InvalidYAML -v`
Expected: PASS

- [ ] **Step 5: 写 Parse 成功测试**

```go
func TestParse_ValidWorkflow(t *testing.T) {
	yaml := `
name: test-workflows
description: 测试工作流
workflows:
  test-flow:
    description: 测试流程
    triggers:
      - 测试
    inputs:
      - name: title
        description: 标题
        required: true
      - name: tag
        description: 标签
        required: false
        hint: 可选标签
    steps:
      - id: step1
        description: 第一步
        command: mod create
        params:
          title: "{{inputs.title}}"
        output:
          itemId: "response.data.id"
      - id: step2
        description: 第二步
        command: mod update
        params:
          id: "{{steps.step1.itemId}}"
          tag: "{{inputs.tag}}"
    summary: |
      完成：{{inputs.title}}
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if cfg.Name != "test-workflows" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test-workflows")
	}
	wf, ok := cfg.Workflows["test-flow"]
	if !ok {
		t.Fatal("缺少 test-flow workflow")
	}
	if len(wf.Triggers) != 1 || wf.Triggers[0] != "测试" {
		t.Errorf("Triggers = %v, want [测试]", wf.Triggers)
	}
	if len(wf.Inputs) != 2 {
		t.Errorf("Inputs 长度 = %d, want 2", len(wf.Inputs))
	}
	if wf.Inputs[1].Hint != "可选标签" {
		t.Errorf("Inputs[1].Hint = %q, want %q", wf.Inputs[1].Hint, "可选标签")
	}
	if len(wf.Steps) != 2 {
		t.Errorf("Steps 长度 = %d, want 2", len(wf.Steps))
	}
	if wf.Steps[0].Params["title"] != "{{inputs.title}}" {
		t.Errorf("Step1 params title = %q", wf.Steps[0].Params["title"])
	}
	if wf.Steps[1].Params["id"] != "{{steps.step1.itemId}}" {
		t.Errorf("Step2 params id = %q", wf.Steps[1].Params["id"])
	}
}
```

- [ ] **Step 6: 运行测试确认通过**

Run: `go test ./internal/workflow/ -run TestParse_ValidWorkflow -v`
Expected: PASS

- [ ] **Step 7: 写空 workflows 测试**

```go
func TestParse_EmptyWorkflows(t *testing.T) {
	yaml := `
name: empty
description: 空
workflows: {}
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(cfg.Workflows) != 0 {
		t.Errorf("Workflows 长度 = %d, want 0", len(cfg.Workflows))
	}
}
```

- [ ] **Step 8: 运行全部 workflow 测试确认通过**

Run: `go test ./internal/workflow/ -v`
Expected: 全部 PASS

- [ ] **Step 9: 提交**

```bash
git add internal/workflow/workflow.go internal/workflow/workflow_test.go
git commit -m "feat(workflow): add data structures and Parse function"
```

---

### Task 2: Describe 函数

**Files:**
- Modify: `internal/workflow/workflow.go`
- Modify: `internal/workflow/workflow_test.go`

- [ ] **Step 1: 写 Describe 测试**

```go
func TestDescribe_Output(t *testing.T) {
	wf := Workflow{
		Description: "创建并配置智能体",
		Inputs: []Input{
			{Name: "name", Description: "名称", Required: true},
			{Name: "tag", Description: "标签", Required: false, Hint: "默认为空"},
		},
		Steps: []Step{
			{
				ID: "create", Description: "创建", Command: "agent create",
				Params: map[string]string{"name": "{{inputs.name}}"},
				Output: map[string]string{"id": "response.data.id"},
			},
			{
				ID: "update", Description: "更新", Command: "agent update",
				Params: map[string]string{"id": "{{steps.create.id}}", "tag": "{{inputs.tag}}"},
			},
		},
		Summary: "完成：{{inputs.name}}",
	}

	result := Describe(&wf, "test-flow")

	// 验证包含关键内容
	checks := []string{
		"Workflow: test-flow",
		"创建并配置智能体",
		"== 需要收集的信息 ==",
		"1. name (必填): 名称",
		"2. tag (可选): 标签",
		"提示: 默认为空",
		"== 执行步骤 ==",
		"Step 1: create - 创建",
		"命令: ckjr-cli agent create",
		"输出: id",
		"Step 2: update - 更新",
		"== 完成摘要 ==",
		"完成：{{inputs.name}}",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("输出缺少 %q\n实际输出:\n%s", check, result)
		}
	}
}
```

（在测试文件顶部添加 `"strings"` import）

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/workflow/ -run TestDescribe_Output -v`
Expected: FAIL (Describe 未定义)

- [ ] **Step 3: 实现 Describe**

在 `internal/workflow/workflow.go` 中添加：

```go
import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func Describe(wf *Workflow, name string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Workflow: %s\n", name)
	fmt.Fprintf(&b, "Description: %s\n", wf.Description)

	// Inputs
	b.WriteString("\n== 需要收集的信息 ==\n")
	for i, input := range wf.Inputs {
		req := "可选"
		if input.Required {
			req = "必填"
		}
		fmt.Fprintf(&b, "%d. %s (%s): %s\n", i+1, input.Name, req, input.Description)
		if input.Hint != "" {
			fmt.Fprintf(&b, "   提示: %s\n", input.Hint)
		}
	}

	// Steps
	b.WriteString("\n== 执行步骤 ==\n")
	for i, step := range wf.Steps {
		fmt.Fprintf(&b, "Step %d: %s - %s\n", i+1, step.ID, step.Description)
		fmt.Fprintf(&b, "  命令: ckjr-cli %s\n", step.Command)

		if len(step.Params) > 0 {
			params := make([]string, 0, len(step.Params))
			for k, v := range step.Params {
				params = append(params, k+"="+v)
			}
			fmt.Fprintf(&b, "  参数: %s\n", strings.Join(params, ", "))
		}

		if len(step.Output) > 0 {
			outputs := make([]string, 0, len(step.Output))
			for k := range step.Output {
				outputs = append(outputs, k)
			}
			fmt.Fprintf(&b, "  输出: %s\n", strings.Join(outputs, ", "))
		}
	}

	// Summary
	if wf.Summary != "" {
		b.WriteString("\n== 完成摘要 ==\n")
		b.WriteString(wf.Summary)
	}

	return b.String()
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/workflow/ -run TestDescribe_Output -v`
Expected: PASS

- [ ] **Step 5: 写无 inputs 的 Describe 测试**

```go
func TestDescribe_NoInputs(t *testing.T) {
	wf := Workflow{
		Description: "简单流程",
		Steps: []Step{
			{ID: "run", Description: "执行", Command: "mod run"},
		},
	}
	result := Describe(&wf, "simple")
	if !strings.Contains(result, "Workflow: simple") {
		t.Errorf("输出缺少 workflow 名称")
	}
	if !strings.Contains(result, "Step 1: run") {
		t.Errorf("输出缺少步骤")
	}
}
```

- [ ] **Step 6: 运行全部测试确认通过**

Run: `go test ./internal/workflow/ -v`
Expected: 全部 PASS

- [ ] **Step 7: 提交**

```bash
git add internal/workflow/workflow.go internal/workflow/workflow_test.go
git commit -m "feat(workflow): add Describe function for AI-readable output"
```

---

### Task 3: workflow YAML 文件

**Files:**
- Create: `cmd/workflows/agent.yaml`

- [ ] **Step 1: 创建 agent.yaml**

```yaml
name: agent-workflows
description: 智能体相关工作流

workflows:
  create-agent:
    description: 创建并配置一个完整的智能体
    triggers:
      - 创建智能体
      - 新建智能体
      - 创建一个AI助手
    inputs:
      - name: name
        description: 智能体名称
        required: true
      - name: desc
        description: 智能体描述/用途
        required: true
      - name: avatar
        description: 头像URL
        required: false
        hint: 如果用户未提供，询问用户或使用用户提供的素材链接
      - name: instructions
        description: 智能体提示词/角色设定
        required: true
        hint: 根据用户描述的用途，生成包含角色定位、能力、交流规则和响应方式的完整提示词
      - name: greeting
        description: 开场白文案
        required: false
        hint: 根据智能体角色生成一条友好的开场白
    steps:
      - id: create
        description: 创建智能体基本信息
        command: agent create
        params:
          name: "{{inputs.name}}"
          desc: "{{inputs.desc}}"
          avatar: "{{inputs.avatar}}"
        output:
          aikbId: "response.aikbId"
      - id: configure
        description: 设置提示词和开场白
        command: agent update
        params:
          aikbId: "{{steps.create.aikbId}}"
          name: "{{inputs.name}}"
          desc: "{{inputs.desc}}"
          avatar: "{{inputs.avatar}}"
          instructions: "{{inputs.instructions}}"
          greeting: "{{inputs.greeting}}"
      - id: get-link
        description: 获取公众号端访问链接和二维码
        command: common qrcodeImg
        params:
          prodId: "{{steps.create.aikbId}}"
          prodType: ai_service
        output:
          url: "response.url"
          qrcodeImg: "response.img"
    summary: |
      智能体创建完成：
      - 名称：{{inputs.name}}
      - ID：{{steps.create.aikbId}}
      - 访问链接：{{steps.get-link.url}}
      - 二维码：{{steps.get-link.qrcodeImg}}
```

- [ ] **Step 2: 验证 YAML 可被 Parse 解析**

写一个快速测试（可加到 workflow_test.go）：

```go
func TestParse_AgentWorkflowFile(t *testing.T) {
	data, err := os.ReadFile("../../cmd/workflows/agent.yaml")
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
}
```

（添加 `"os"` import）

- [ ] **Step 3: 运行测试确认通过**

Run: `go test ./internal/workflow/ -run TestParse_AgentWorkflowFile -v`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/workflows/agent.yaml internal/workflow/workflow_test.go
git commit -m "feat(workflow): add agent workflow YAML definition"
```

---

### Task 4: workflow 命令 (list + describe)

**Files:**
- Create: `cmd/workflow.go`
- Create: `cmd/workflow_test.go`
- Modify: `cmd/root.go:40-58` (注册命令)

- [ ] **Step 1: 写 workflow list 测试**

```go
// cmd/workflow_test.go
package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestWorkflowList(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"workflow", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "create-agent") {
		t.Errorf("输出缺少 create-agent, got: %s", output)
	}
	if !strings.Contains(output, "创建并配置一个完整的智能体") {
		t.Errorf("输出缺少 workflow 描述, got: %s", output)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./cmd/ -run TestWorkflowList -v`
Expected: FAIL (workflow 命令不存在)

- [ ] **Step 3: 实现 workflow.go**

```go
// cmd/workflow.go
package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/childelins/ckjr-cli/internal/workflow"
	"github.com/spf13/cobra"
)

//go:embed workflows
var workflowsFS embed.FS

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "工作流管理",
}

var workflowListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有可用的工作流",
	RunE: func(cmd *cobra.Command, args []string) error {
		configs, err := loadAllWorkflows()
		if err != nil {
			return err
		}

		type item struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Triggers    []string `json:"triggers"`
		}

		var items []item
		for _, cfg := range configs {
			for name, wf := range cfg.Workflows {
				items = append(items, item{
					Name:        name,
					Description: wf.Description,
					Triggers:    wf.Triggers,
				})
			}
		}

		data, err := json.Marshal(items)
		if err != nil {
			return err
		}

		pretty, _ := cmd.Flags().GetBool("pretty")
		if pretty {
			var indented bytes.Buffer
			json.Indent(&indented, data, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), indented.String())
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
		}
		return nil
	},
}

var workflowDescribeCmd = &cobra.Command{
	Use:   "describe <name>",
	Short: "输出工作流的完整描述",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		configs, err := loadAllWorkflows()
		if err != nil {
			return err
		}

		for _, cfg := range configs {
			if wf, ok := cfg.Workflows[name]; ok {
				fmt.Fprint(cmd.OutOrStdout(), workflow.Describe(&wf, name))
				return nil
			}
		}

		return fmt.Errorf("未找到工作流: %s", name)
	},
}

func loadAllWorkflows() ([]*workflow.Config, error) {
	entries, err := fs.ReadDir(workflowsFS, "workflows")
	if err != nil {
		return nil, err
	}

	var configs []*workflow.Config
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := workflowsFS.ReadFile("workflows/" + entry.Name())
		if err != nil {
			return nil, err
		}
		cfg, err := workflow.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %w", entry.Name(), err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

func init() {
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowDescribeCmd)
}
```

注意：需要添加 `"bytes"` import。

- [ ] **Step 4: 在 root.go 注册 workflow 命令**

在 `cmd/root.go` 的 `init()` 函数中，`registerRouteCommands()` 之前添加：

```go
rootCmd.AddCommand(workflowCmd)
```

- [ ] **Step 5: 运行 list 测试确认通过**

Run: `go test ./cmd/ -run TestWorkflowList -v`
Expected: PASS

- [ ] **Step 6: 写 workflow describe 测试**

```go
func TestWorkflowDescribe(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"workflow", "describe", "create-agent"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	output := buf.String()
	checks := []string{
		"Workflow: create-agent",
		"== 需要收集的信息 ==",
		"name (必填)",
		"instructions (必填)",
		"== 执行步骤 ==",
		"agent create",
		"agent update",
		"common qrcodeImg",
		"== 完成摘要 ==",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("输出缺少 %q", check)
		}
	}
}
```

- [ ] **Step 7: 运行 describe 测试确认通过**

Run: `go test ./cmd/ -run TestWorkflowDescribe -v`
Expected: PASS

- [ ] **Step 8: 写 describe 不存在的 workflow 测试**

```go
func TestWorkflowDescribe_NotFound(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"workflow", "describe", "nonexistent"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("期望返回错误")
	}
	if !strings.Contains(err.Error(), "未找到工作流") {
		t.Errorf("错误信息 = %q, 期望包含 '未找到工作流'", err.Error())
	}
}
```

- [ ] **Step 9: 运行全部 cmd 测试确认通过**

Run: `go test ./cmd/ -v`
Expected: 全部 PASS

- [ ] **Step 10: 运行全项目测试**

Run: `go test ./...`
Expected: 全部 PASS

- [ ] **Step 11: 提交**

```bash
git add cmd/workflow.go cmd/workflow_test.go cmd/root.go
git commit -m "feat(workflow): add workflow list and describe commands"
```

---

### Task 5: 更新 SKILL.md

**Files:**
- Modify: `skills/ckjr-cli/SKILL.md`

- [ ] **Step 1: 在 SKILL.md 的"使用规则"之前添加 workflow 策略**

在 `## 使用规则` 之前插入：

```markdown
## 任务执行策略

对于多步骤任务（如创建智能体、配置智能体等），优先使用 workflow：

1. **匹配工作流**: `ckjr-cli workflow list` 查看是否有匹配的工作流
2. **获取流程**: `ckjr-cli workflow describe <name>` 获取完整流程定义
3. **收集信息**: 根据 workflow 的 inputs 一次性向用户收集所需信息
4. **按步执行**: 按 steps 顺序逐步执行原子命令，注意步骤间的数据传递
5. **汇报结果**: 按 summary 模板汇报执行结果

对于简单的单步操作（如查看列表、删除），直接使用命令发现流程。
```

- [ ] **Step 2: 构建并验证 CLI**

Run: `go build -o /tmp/ckjr-cli . && /tmp/ckjr-cli workflow list --pretty`
Expected: 输出包含 create-agent 的 JSON

Run: `/tmp/ckjr-cli workflow describe create-agent`
Expected: 输出完整的 workflow 描述文本

- [ ] **Step 3: 提交**

```bash
git add skills/ckjr-cli/SKILL.md
git commit -m "docs(skill): add workflow-first strategy to SKILL.md"
```

---

### Task 6: 安装并端到端验证

- [ ] **Step 1: 安装更新后的 CLI**

Run: `go install ./cmd/ckjr-cli/`

- [ ] **Step 2: 验证 workflow list**

Run: `ckjr-cli workflow list --pretty`
Expected: JSON 数组包含 create-agent

- [ ] **Step 3: 验证 workflow describe**

Run: `ckjr-cli workflow describe create-agent`
Expected: 输出包含 inputs、steps、summary 的完整描述

- [ ] **Step 4: 验证 --help**

Run: `ckjr-cli workflow --help`
Expected: 显示 list 和 describe 子命令

- [ ] **Step 5: 验证原有命令未受影响**

Run: `ckjr-cli agent --help && ckjr-cli common --help`
Expected: 原有命令正常工作

- [ ] **Step 6: 全量测试**

Run: `go test ./... -v`
Expected: 全部 PASS
