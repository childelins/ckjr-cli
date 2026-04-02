package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestParse_InvalidYAML(t *testing.T) {
	_, err := Parse([]byte(":::invalid"))
	if err == nil {
		t.Fatal("期望解析无效 YAML 时返回错误")
	}
}

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

func TestDescribe_AllowedRoutes(t *testing.T) {
	wf := Workflow{
		Description:   "限制路由的流程",
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
	if len(wf.Steps) != 4 {
		t.Errorf("Steps 长度 = %d, want 4", len(wf.Steps))
	}
	if len(wf.Inputs) != 5 {
		t.Errorf("Inputs 长度 = %d, want 5", len(wf.Inputs))
	}
	if len(wf.AllowedRoutes) != 3 {
		t.Errorf("AllowedRoutes 长度 = %d, want 3", len(wf.AllowedRoutes))
	}
	if wf.AllowedRoutes[0] != "agent" || wf.AllowedRoutes[1] != "common" || wf.AllowedRoutes[2] != "asset" {
		t.Errorf("AllowedRoutes = %v, want [agent, common, asset]", wf.AllowedRoutes)
	}
}
