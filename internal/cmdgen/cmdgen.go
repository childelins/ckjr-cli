package cmdgen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
				printTemplate(route.Template, route.Response)
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

			// 路径参数替换
			resolvedPath, err := ReplacePath(route.Path, input, route.Template)
			if err != nil {
				output.PrintError(os.Stderr, err.Error())
				os.Exit(1)
			}

			// 校验参数
			if errs := ValidateAll(input, route.Template); len(errs) > 0 {
				var msgs []string
				for _, e := range errs {
					msgs = append(msgs, e.Error())
				}
				output.PrintError(os.Stderr, fmt.Sprintf("参数校验失败:\n  %s", strings.Join(msgs, "\n  ")))
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
			if err := client.DoCtx(ctx, route.Method, resolvedPath, input, &result); err != nil {
				handleAPIError(err, verbose)
				os.Exit(1)
			}

			// 响应字段过滤
			result = FilterResponse(result, route.Response)

			output.Print(os.Stdout, result, pretty)
		},
	}

	cmd.Flags().BoolVar(&showTemplate, "template", false, "显示参数模板")

	return cmd
}

func printTemplate(template map[string]router.Field, response *router.ResponseFilter) {
	printTemplateTo(os.Stdout, template, response)
}

func printTemplateTo(w io.Writer, template map[string]router.Field, response *router.ResponseFilter) {
	result := make(map[string]interface{})

	// request 部分
	request := make(map[string]interface{})
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
		if t == "path" {
			entry["note"] = "路径参数，必须包含在 JSON 中，将自动替换 URL 中的占位符"
		}
		if t == "date" {
			entry["note"] = "日期格式: YYYY-MM-DD HH:MM:SS"
		}
		if field.Example != "" {
			entry["example"] = field.Example
		}

		// 约束信息
		constraints := map[string]interface{}{}
		if field.Min != nil {
			constraints["min"] = *field.Min
		}
		if field.Max != nil {
			constraints["max"] = *field.Max
		}
		if field.MinLength != nil {
			constraints["minLength"] = *field.MinLength
		}
		if field.MaxLength != nil {
			constraints["maxLength"] = *field.MaxLength
		}
		if field.Pattern != "" {
			constraints["pattern"] = field.Pattern
		}
		if len(constraints) > 0 {
			entry["constraints"] = constraints
		}

		request[name] = entry
	}
	result["request"] = request

	// response 部分（仅在配置了 fields 时输出）
	if response != nil && len(response.Fields) > 0 {
		respFields := make(map[string]interface{})
		for _, f := range response.Fields {
			respFields[f.Path] = f.Description
		}
		result["response"] = respFields
	}

	output.Print(w, result, true)
}

func applyDefaults(input map[string]interface{}, template map[string]router.Field) {
	for name, field := range template {
		if _, exists := input[name]; !exists && field.Default != nil {
			input[name] = normalizeDefault(field.Default)
		}
	}
}

// normalizeDefault 将 YAML 解析的整数默认值转为 float64，与 JSON 反序列化保持一致
func normalizeDefault(val interface{}) interface{} {
	switch v := val.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return val
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
	// 1. Unauthorized -- 构造类似服务端格式的 JSON
	if api.IsUnauthorized(err) {
		resp := map[string]interface{}{
			"msg":        "api_key 已过期，请重新登录获取",
			"statusCode": 401,
		}
		output.Print(w, resp, false)
		return
	}

	// 2. ValidationError -- 透传服务端原始结构
	if api.IsValidationError(err) {
		errs := api.GetValidationErrors(err)
		msg := api.GetValidationMessage(err)
		resp := map[string]interface{}{
			"msg":        msg,
			"statusCode": 422,
			"errors":     errs,
		}
		output.Print(w, resp, false)
		return
	}

	// 3. APIError -- 透传服务端原始结构
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		resp := map[string]interface{}{
			"msg":        apiErr.Message,
			"statusCode": apiErr.ServerCode,
		}
		if apiErr.Errors != nil {
			resp["errors"] = apiErr.Errors
		}
		output.Print(w, resp, false)
		return
	}

	// 4. ResponseError (非 JSON 响应) -- 构造结构化输出
	var respErr *api.ResponseError
	if errors.As(err, &respErr) {
		detail := map[string]interface{}{
			"msg":          respErr.Error(),
			"statusCode":   respErr.StatusCode,
			"content_type": respErr.ContentType,
		}
		if verbose {
			detail["body"] = respErr.Body
		}
		output.Print(w, detail, false)
		return
	}

	// 5. 客户端侧错误（网络、序列化等）-- 保持简单格式
	output.PrintError(w, err.Error())
}
