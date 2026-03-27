package route

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/curlparse"
	"github.com/childelins/ckjr-cli/internal/router"
	"github.com/childelins/ckjr-cli/internal/yamlgen"
)

// NewCommand 创建 route 命令及其子命令
func NewCommand() *cobra.Command {
	routeCmd := &cobra.Command{
		Use:    "route",
		Short:  "路由配置管理",
		Hidden: true,
	}

	routeImportCmd := &cobra.Command{
		Use:   "import",
		Short: "从 curl 命令导入路由配置",
		Long:  "解析 curl 命令并生成 YAML 路由配置。支持 stdin 管道输入或 --curl 参数。",
		RunE: func(cmd *cobra.Command, args []string) error {
			curlStr, _ := cmd.Flags().GetString("curl")
			file, _ := cmd.Flags().GetString("file")
			routeName, _ := cmd.Flags().GetString("name")
			nameDesc, _ := cmd.Flags().GetString("name-desc")

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

	routeImportCmd.Flags().String("curl", "", "curl 命令字符串")
	routeImportCmd.Flags().StringP("file", "f", "", "目标 YAML 文件路径")
	routeImportCmd.Flags().StringP("name", "n", "", "路由名称（默认从 URL 推导）")
	routeImportCmd.Flags().String("name-desc", "", "资源名称描述")

	routeCmd.AddCommand(routeImportCmd)
	return routeCmd
}

func runImport(curlStr, file, routeName, nameDesc string) error {
	result, err := curlparse.Parse(curlStr)
	if err != nil {
		return fmt.Errorf("curl 解析失败: %w", err)
	}

	if routeName == "" {
		routeName = router.InferRouteName(result.Path)
	}

	r := yamlgen.GenerateRoute(result)

	if _, err := os.Stat(file); err == nil {
		return yamlgen.AppendToFile(file, routeName, r)
	}

	name := router.InferNameFromPath(file)
	if nameDesc == "" {
		return fmt.Errorf("新建文件需要通过 --name-desc 指定资源描述")
	}
	return yamlgen.CreateFile(file, name, nameDesc, routeName, r)
}
