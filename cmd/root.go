package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/cmdgen"
	internalconfig "github.com/childelins/ckjr-cli/internal/config"
	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
	"github.com/childelins/ckjr-cli/internal/logging"
	"github.com/childelins/ckjr-cli/internal/output"
	"github.com/childelins/ckjr-cli/internal/router"

	configcmd "github.com/childelins/ckjr-cli/cmd/config"
	routecmd "github.com/childelins/ckjr-cli/cmd/route"
	updatecmd "github.com/childelins/ckjr-cli/cmd/update"
	workflowcmd "github.com/childelins/ckjr-cli/cmd/workflow"
)

var yamlFS *configyaml.FS

// SetYAMLFS 设置 YAML 配置加载器，由 main 包调用
func SetYAMLFS(fs *configyaml.FS) {
	yamlFS = fs
}

var (
	version     = "dev"
	environment = "production"
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
	registerRouteCommands()
	rootCmd.AddCommand(workflowcmd.NewCommand(yamlFS))
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("pretty", false, "格式化 JSON 输出")
	rootCmd.PersistentFlags().Bool("verbose", false, "显示详细调试信息")
	cobra.OnInitialize(initLogging)

	rootCmd.AddCommand(configcmd.NewCommand())
	rootCmd.AddCommand(routecmd.NewCommand())
	updatecmd.SetVersion(version)
	rootCmd.AddCommand(updatecmd.NewCommand())
}

func initLogging() {
	verbose, _ := rootCmd.Flags().GetBool("verbose")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		output.PrintError(os.Stderr, fmt.Sprintf("获取用户目录失败: %v", err))
		return
	}
	baseDir := filepath.Join(homeDir, ".ckjr")
	env := logging.ParseEnvironment(environment)
	if err := logging.Init(verbose, baseDir, env); err != nil {
		output.PrintError(os.Stderr, fmt.Sprintf("日志初始化失败: %v", err))
	}
}

func registerRouteCommands() {
	if yamlFS == nil {
		output.PrintError(os.Stderr, "YAML 文件系统未初始化")
		return
	}
	files, err := yamlFS.LoadRoutes()
	if err != nil {
		output.PrintError(os.Stderr, fmt.Sprintf("读取路由目录失败: %v", err))
		return
	}
	for name, data := range files {
		cfg, err := router.Parse(data)
		if err != nil {
			output.PrintError(os.Stderr, fmt.Sprintf("解析路由文件 %s 失败: %v", name, err))
			continue
		}
		cmd := cmdgen.BuildCommand(cfg, createClient)
		rootCmd.AddCommand(cmd)

		// 为 asset 命令额外注册 upload-image 子命令
		if cfg.Name == "asset" {
			cmd.AddCommand(newUploadImageCmd(createClient))
		}
	}
}

// createClient 创建 API 客户端
func createClient() (*api.Client, error) {
	cfg, err := internalconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("未找到配置文件，请先执行 ckjr-cli config init")
	}
	return api.NewClient(cfg.BaseURL, cfg.APIKey), nil
}
