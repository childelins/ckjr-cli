package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/curlparse"
	"github.com/childelins/ckjr-cli/internal/yamlgen"
)

var routeCmd = &cobra.Command{
	Use:    "route",
	Short:  "路由配置管理",
	Hidden: true,
}

var routeImportCmd = &cobra.Command{
	Use:   "import",
	Short: "从 curl 命令导入路由配置",
	Long:  "解析 curl 命令并生成 YAML 路由配置。支持 stdin 管道输入或 --curl 参数。",
	RunE: func(cmd *cobra.Command, args []string) error {
		curlStr, _ := cmd.Flags().GetString("curl")
		file, _ := cmd.Flags().GetString("file")
		routeName, _ := cmd.Flags().GetString("name")
		nameDesc, _ := cmd.Flags().GetString("name-desc")

		// 从 stdin 读取
		if curlStr == "" {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("读取 stdin 失败: %w", err)
				}
				curlStr = string(data)
			}
		}

		if curlStr == "" {
			return fmt.Errorf("请通过 --curl 参数或 stdin 管道提供 curl 命令")
		}
		if file == "" {
			return fmt.Errorf("请通过 --file 参数指定目标 YAML 文件路径")
		}

		if err := runImport(curlStr, file, routeName, nameDesc); err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "已添加路由 %s 到 %s\n", routeName, file)
		return nil
	},
}

// runImport 核心逻辑，方便测试
func runImport(curlStr, file, routeName, nameDesc string) error {
	result, err := curlparse.Parse(curlStr)
	if err != nil {
		return fmt.Errorf("curl 解析失败: %w", err)
	}

	// 自动推导 route name
	if routeName == "" {
		routeName = inferRouteName(result.Path)
	}

	route := yamlgen.GenerateRoute(result)

	// 判断追加还是新建
	if _, err := os.Stat(file); err == nil {
		return yamlgen.AppendToFile(file, routeName, route)
	}

	if nameDesc == "" {
		return fmt.Errorf("新建文件需要通过 --name-desc 指定资源描述")
	}
	return yamlgen.CreateFile(file, routeName, nameDesc, routeName, route)
}

// inferRouteName 从 URL path 末段推导 route name
func inferRouteName(path string) string {
	// 取最后一个路径段
	parts := splitPath(path)
	if len(parts) == 0 {
		return "unknown"
	}
	last := parts[len(parts)-1]

	// 常见前缀映射
	prefixes := map[string]string{
		"modify": "update",
		"edit":   "update",
		"remove": "delete",
		"add":    "create",
		"create": "create",
		"query":  "list",
	}
	lower := toLower(last)
	for prefix, mapped := range prefixes {
		if len(lower) >= len(prefix) && lower[:len(prefix)] == prefix {
			return mapped
		}
	}

	// describe*/get*Info -> get
	if len(lower) >= 8 && lower[:8] == "describe" {
		return "get"
	}

	return last
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range split(path, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func split(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func init() {
	routeImportCmd.Flags().String("curl", "", "curl 命令字符串")
	routeImportCmd.Flags().StringP("file", "f", "", "目标 YAML 文件路径")
	routeImportCmd.Flags().StringP("name", "n", "", "路由名称（默认从 URL 推导）")
	routeImportCmd.Flags().String("name-desc", "", "资源名称描述")

	routeCmd.AddCommand(routeImportCmd)
}
