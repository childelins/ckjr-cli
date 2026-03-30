package yamlgen

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/childelins/ckjr-cli/internal/workflow"
)

// InitWorkflowFile 创建模块 workflow 骨架文件
func InitWorkflowFile(path string, moduleName string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("文件已存在: %s", path)
	}

	cfg := &workflow.Config{
		Name:        moduleName,
		Description: moduleName,
		Workflows: map[string]workflow.Workflow{
			"workflow-name": {
				Description: "工作流描述",
				Triggers:    []string{},
				Inputs:      []workflow.Input{},
				Steps:       []workflow.Step{},
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("YAML 序列化失败: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
