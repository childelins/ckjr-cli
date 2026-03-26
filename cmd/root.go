package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/cmdgen"
	"github.com/childelins/ckjr-cli/internal/config"
	"github.com/childelins/ckjr-cli/internal/logging"
	"github.com/childelins/ckjr-cli/internal/router"
)

//go:embed routes
var routesFS embed.FS

var (
	version      = "dev"
	environment  = "production"
)

// SetVersion 由 main 包调用，通过 ldflags 注入版本号
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// SetEnvironment 由 main 包调用，通过 ldflags 注入环境模式
func SetEnvironment(e string) {
	environment = e
}

var rootCmd = &cobra.Command{
	Use:               "ckjr-cli",
	Short:             "创客匠人 CLI - 知识付费 SaaS 系统的命令行工具",
	Version:           version,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// 添加 --pretty 全局 flag
	rootCmd.PersistentFlags().Bool("pretty", false, "格式化 JSON 输出")

	// 添加 --verbose 全局 flag
	rootCmd.PersistentFlags().Bool("verbose", false, "显示详细调试信息")

	// 初始化日志系统
	cobra.OnInitialize(initLogging)

	// 注册 config 命令
	rootCmd.AddCommand(configCmd)

	// 注册 route 命令
	rootCmd.AddCommand(routeCmd)

	// 注册 workflow 命令
	rootCmd.AddCommand(workflowCmd)

	// 注册动态生成的命令
	registerRouteCommands()
}

func initLogging() {
	verbose, _ := rootCmd.Flags().GetBool("verbose")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
		return
	}
	baseDir := filepath.Join(homeDir, ".ckjr")
	env := logging.ParseEnvironment(environment)
	if err := logging.Init(verbose, baseDir, env); err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
	}
}

func registerRouteCommands() {
	// 读取 embed 的路由文件
	entries, err := routesFS.ReadDir("routes")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取路由目录失败: %v\n", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理 .yaml 文件
		name := entry.Name()
		if len(name) < 5 || name[len(name)-5:] != ".yaml" {
			continue
		}

		// 读取并解析路由配置
		data, err := routesFS.ReadFile("routes/" + name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取路由文件 %s 失败: %v\n", name, err)
			continue
		}

		cfg, err := router.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "解析路由文件 %s 失败: %v\n", name, err)
			continue
		}

		// 生成命令并注册
		cmd := cmdgen.BuildCommand(cfg, createClient)
		rootCmd.AddCommand(cmd)
	}
}

// createClient 创建 API 客户端
func createClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("未找到配置文件，请先执行 ckjr-cli config init")
	}

	return api.NewClient(cfg.BaseURL, cfg.APIKey), nil
}
