package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/childelins/ckjr-cli/internal/workflow"
	"github.com/spf13/cobra"
)

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
	if yamlFS == nil {
		return nil, fmt.Errorf("YAML 文件系统未初始化")
	}

	files, err := yamlFS.LoadWorkflows()
	if err != nil {
		return nil, err
	}

	var configs []*workflow.Config
	for name, data := range files {
		cfg, err := workflow.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %w", name, err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}

func init() {
	workflowCmd.AddCommand(workflowListCmd)
	workflowCmd.AddCommand(workflowDescribeCmd)
}
