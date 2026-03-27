package workflow

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
)

func setupTestYAMLFS(t *testing.T) *configyaml.FS {
	t.Helper()
	memFS := fstest.MapFS{
		"workflows/agent.yaml": {Data: []byte(`name: agent-workflows
description: 智能体相关工作流
workflows:
  create-agent:
    description: 创建并配置一个完整的智能体
    triggers:
      - 用户请求创建智能体
    inputs:
      - name: name
        description: 智能体名称
        required: true
      - name: instructions
        description: 提示词
        required: true
    steps:
      - id: create
        description: 创建智能体
        command: agent create
        params:
          name: "{{inputs.name}}"
      - id: configure
        description: 设置提示词
        command: agent update
        params:
          aikbId: "{{steps.create.aikbId}}"
          instructions: "{{inputs.instructions}}"
      - id: get-link
        description: 获取访问链接
        command: common getLink
        params:
          prodId: "{{steps.create.aikbId}}"
        output:
          url: "response.url"
          qrcodeImg: "response.img"
    summary: |
      智能体配置完成
`)},
	}
	return configyaml.New(memFS)
}

func TestWorkflowList(t *testing.T) {
	yamlFS := setupTestYAMLFS(t)
	cmd := NewCommand(yamlFS)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
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

func TestWorkflowDescribe(t *testing.T) {
	yamlFS := setupTestYAMLFS(t)
	cmd := NewCommand(yamlFS)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"describe", "create-agent"})

	err := cmd.Execute()
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
		"common getLink",
		"== 完成摘要 ==",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("输出缺少 %q", check)
		}
	}
}

func TestWorkflowDescribe_NotFound(t *testing.T) {
	yamlFS := setupTestYAMLFS(t)
	cmd := NewCommand(yamlFS)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"describe", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("期望返回错误")
	}
	if !strings.Contains(err.Error(), "未找到工作流") {
		t.Errorf("错误信息 = %q, 期望包含 '未找到工作流'", err.Error())
	}
}
