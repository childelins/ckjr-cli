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
