package workflow

import "testing"

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
