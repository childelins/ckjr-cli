package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/config"
	"github.com/childelins/ckjr-cli/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 CLI 配置",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "交互式初始化配置",
	Run:   runConfigInit,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置配置项",
	Args:  cobra.ExactArgs(2),
	Run:   runConfigSet,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "查看当前配置",
	Run:   runConfigShow,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	// 输入 base_url
	fmt.Print("请输入 API 地址 (base_url): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)

	// 引导获取 api_key
	fmt.Println("\n请按以下步骤获取 API Key:")
	fmt.Println("1. 访问公司 SaaS 平台并登录")
	fmt.Println("2. 进入个人设置 -> API 密钥")
	fmt.Println("3. 复制 API Key")
	fmt.Print("\n请粘贴 API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	cfg := &config.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n配置已保存到:", config.ConfigPath)
}

func runConfigSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]

	// 验证 key
	validKeys := map[string]bool{"base_url": true, "api_key": true}
	if !validKeys[key] {
		fmt.Fprintf(os.Stderr, "无效的配置项: %s\n合法值: base_url, api_key\n", key)
		os.Exit(1)
	}

	// 加载现有配置或创建新配置
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	switch key {
	case "base_url":
		cfg.BaseURL = value
	case "api_key":
		cfg.APIKey = value
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已设置 %s\n", key)
}

func runConfigShow(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n请先执行 ckjr config init\n", err)
		os.Exit(1)
	}

	pretty, _ := cmd.Flags().GetBool("pretty")
	result := map[string]string{
		"base_url": cfg.BaseURL,
		"api_key":  cfg.MaskedAPIKey(),
	}
	output.Print(os.Stdout, result, pretty)
}
