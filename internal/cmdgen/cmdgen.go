package cmdgen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/logging"
	"github.com/childelins/ckjr-cli/internal/output"
	"github.com/childelins/ckjr-cli/internal/router"
)

// APIClientFactory 创建 API 客户端的工厂函数
type APIClientFactory func() (*api.Client, error)

// BuildCommand 从路由配置构建 cobra 命令
func BuildCommand(cfg *router.RouteConfig, clientFactory APIClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   cfg.Name,
		Short: cfg.Description,
	}

	for name, route := range cfg.Routes {
		subCmd := buildSubCommand(cfg.Name, name, route, clientFactory)
		cmd.AddCommand(subCmd)
	}

	return cmd
}

func buildSubCommand(resource, name string, route router.Route, clientFactory APIClientFactory) *cobra.Command {
	var showTemplate bool
	var inputJSON string

	cmd := &cobra.Command{
		Use:   name + " [json]",
		Short: route.Description,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// --template 模式：输出模板并退出
			if showTemplate {
				printTemplate(route.Template)
				return
			}

			// 获取输入 JSON
			var input map[string]interface{}
			if len(args) > 0 {
				if args[0] == "-" {
					// 从 stdin 读取
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						output.PrintError(os.Stderr, "读取 stdin 失败: "+err.Error())
						os.Exit(1)
					}
					inputJSON = string(data)
				} else {
					inputJSON = args[0]
				}
			}

			if inputJSON != "" {
				if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
					output.PrintError(os.Stderr, "JSON 解析失败: "+err.Error())
					os.Exit(1)
				}
			} else {
				input = make(map[string]interface{})
			}

			// 应用默认值
			applyDefaults(input, route.Template)

			// 校验必填字段
			if missing := validateRequired(input, route.Template); len(missing) > 0 {
				output.PrintError(os.Stderr, fmt.Sprintf("缺少必填字段: %v", missing))
				os.Exit(1)
			}

			// 执行 API 请求
			if clientFactory == nil {
				output.PrintError(os.Stderr, "API 客户端未配置")
				os.Exit(1)
			}

			client, err := clientFactory()
			if err != nil {
				output.PrintError(os.Stderr, err.Error())
				os.Exit(1)
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			verbose, _ := cmd.Flags().GetBool("verbose")

			// 生成 requestId 并构建 context
			ctx := context.Background()
			requestID := logging.NewRequestID()
			ctx = logging.WithRequestID(ctx, requestID)

			var result interface{}
			if err := client.DoCtx(ctx, route.Method, route.Path, input, &result); err != nil {
				handleAPIError(err, verbose)
				os.Exit(1)
			}

			output.Print(os.Stdout, result, pretty)
		},
	}

	cmd.Flags().BoolVar(&showTemplate, "template", false, "显示参数模板")

	return cmd
}

func printTemplate(template map[string]router.Field) {
	printTemplateTo(os.Stdout, template)
}

func printTemplateTo(w io.Writer, template map[string]router.Field) {
	tmpl := make(map[string]interface{})
	for name, field := range template {
		entry := map[string]interface{}{
			"description": field.Description,
			"required":    field.Required,
		}
		if field.Default != nil {
			entry["default"] = field.Default
		}
		t := field.Type
		if t == "" {
			t = "string"
		}
		entry["type"] = t
		if field.Example != "" {
			entry["example"] = field.Example
		}
		tmpl[name] = entry
	}
	output.Print(w, tmpl, true)
}

func applyDefaults(input map[string]interface{}, template map[string]router.Field) {
	for name, field := range template {
		if _, exists := input[name]; !exists && field.Default != nil {
			input[name] = field.Default
		}
	}
}

func validateRequired(input map[string]interface{}, template map[string]router.Field) []string {
	var missing []string
	for name, field := range template {
		if field.Required {
			if _, exists := input[name]; !exists {
				missing = append(missing, name)
			}
		}
	}
	return missing
}

func handleAPIError(err error, verbose bool) {
	handleAPIErrorTo(os.Stderr, err, verbose)
}

func handleAPIErrorTo(w io.Writer, err error, verbose bool) {
	if api.IsUnauthorized(err) {
		output.PrintError(w, "api_key 已过期，请重新登录获取")
		return
	}

	if api.IsValidationError(err) {
		errs := api.GetValidationErrors(err)
		output.PrintError(w, fmt.Sprintf("参数校验失败: %v", errs))
		return
	}

	var respErr *api.ResponseError
	if errors.As(err, &respErr) {
		output.PrintError(w, respErr.Error())
		if verbose {
			fmt.Fprintf(w, "  %s\n", respErr.Detail())
		}
		return
	}

	output.PrintError(w, err.Error())
}
