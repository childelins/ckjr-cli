package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	internalconfig "github.com/childelins/ckjr-cli/internal/config"
	"github.com/childelins/ckjr-cli/internal/output"
)

// NewCommand 创建 config 命令及其子命令
func NewCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "管理 CLI 配置",
	}

	configInitCmd := &cobra.Command{
		Use:   "init",
		Short: "交互式初始化配置",
		Run:   runConfigInit,
	}

	configSetCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "设置配置项",
		Args:  cobra.ExactArgs(2),
		Run:   runConfigSet,
	}

	configShowCmd := &cobra.Command{
		Use:   "show",
		Short: "查看当前配置",
		Run:   runConfigShow,
	}

	configCmd.AddCommand(configInitCmd, configSetCmd, configShowCmd)
	return configCmd
}

func runConfigInit(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入 API 地址 (base_url): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	fmt.Println("\n请按以下步骤获取 API Key:")
	fmt.Println("1. 访问公司 SaaS 平台并登录")
	fmt.Println("2. 进入个人设置 -> API 密钥")
	fmt.Println("3. 复制 API Key")
	fmt.Print("\n请粘贴 API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	cfg := &internalconfig.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
	if err := internalconfig.Save(cfg); err != nil {
		output.PrintError(os.Stderr, fmt.Sprintf("保存配置失败: %v", err))
		os.Exit(1)
	}
	fmt.Println("\n配置已保存到:", internalconfig.ConfigPath)
}

func runConfigSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]
	validKeys := map[string]bool{"base_url": true, "api_key": true}
	if !validKeys[key] {
		output.PrintError(os.Stderr, fmt.Sprintf("无效的配置项: %s。合法值: base_url, api_key", key))
		os.Exit(1)
	}
	cfg, err := internalconfig.Load()
	if err != nil {
		cfg = &internalconfig.Config{}
	}
	switch key {
	case "base_url":
		cfg.BaseURL = value
	case "api_key":
		cfg.APIKey = value
	}
	if err := internalconfig.Save(cfg); err != nil {
		output.PrintError(os.Stderr, fmt.Sprintf("保存配置失败: %v", err))
		os.Exit(1)
	}
	fmt.Printf("已设置 %s\n", key)
}

func runConfigShow(cmd *cobra.Command, args []string) {
	cfg, err := internalconfig.Load()
	if err != nil {
		output.PrintError(os.Stderr, fmt.Sprintf("读取配置失败: %v。请先执行 ckjr-cli config init", err))
		os.Exit(1)
	}
	pretty, _ := cmd.Flags().GetBool("pretty")
	result := map[string]string{
		"base_url": cfg.ResolveBaseURL(),
		"api_key":  cfg.MaskedAPIKey(),
	}
	output.Print(os.Stdout, result, pretty)
}
