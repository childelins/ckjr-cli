package workflow

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	Steps         []Step   `yaml:"steps"`
	AllowedRoutes []string `yaml:"allowed-routes,omitempty"`
	Summary       string   `yaml:"summary,omitempty"`
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

func Describe(wf *Workflow, name string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Workflow: %s\n", name)
	fmt.Fprintf(&b, "Description: %s\n", wf.Description)

	// Allowed routes
	if len(wf.AllowedRoutes) > 0 {
		fmt.Fprintf(&b, "\n== 路由权限 ==\n")
		fmt.Fprintf(&b, "仅允许调用以下模块的路由: %s\n", strings.Join(wf.AllowedRoutes, ", "))
	}

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
