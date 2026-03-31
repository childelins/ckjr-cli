package cmdgen

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/childelins/ckjr-cli/internal/router"
)

var pathParamRe = regexp.MustCompile(`\{(\w+)\}`)

// IsPathParam 判断字段是否为路径参数
func IsPathParam(field router.Field) bool {
	return field.Type == "path"
}

// extractPlaceholders 从 path 中提取 {xxx} 占位符名，去重保序
func extractPlaceholders(path string) []string {
	matches := pathParamRe.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var params []string
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			params = append(params, name)
		}
	}
	return params
}

// PathParamError 路径参数错误
type PathParamError struct {
	Missing    []string
	Undeclared []string
}

func (e *PathParamError) Error() string {
	var parts []string
	if len(e.Undeclared) > 0 {
		parts = append(parts, fmt.Sprintf(
			"路径占位符 {%s} 未在 template 中声明为 type: path",
			strings.Join(e.Undeclared, "}, {")))
	}
	if len(e.Missing) > 0 {
		parts = append(parts, fmt.Sprintf(
			"缺少路径参数: %s", strings.Join(e.Missing, ", ")))
	}
	return strings.Join(parts, "; ")
}

// ReplacePath 将 path 中 {xxx} 替换为 input 中的值
// 仅替换 template 中 type: path 的字段
// 替换后从 input 中移除路径参数
func ReplacePath(path string, input map[string]interface{}, template map[string]router.Field) (string, error) {
	placeholders := extractPlaceholders(path)
	if len(placeholders) == 0 {
		return path, nil
	}

	// 校验：占位符必须在 template 中声明为 type: path
	var undeclared []string
	for _, name := range placeholders {
		field, exists := template[name]
		if !exists || !IsPathParam(field) {
			undeclared = append(undeclared, name)
		}
	}
	if len(undeclared) > 0 {
		return "", &PathParamError{Undeclared: undeclared}
	}

	// 校验：所有路径参数在 input 中存在且非 nil
	var missing []string
	for _, name := range placeholders {
		val, exists := input[name]
		if !exists || val == nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return "", &PathParamError{Missing: missing}
	}

	// 执行替换
	result := pathParamRe.ReplaceAllStringFunc(path, func(match string) string {
		name := match[1 : len(match)-1]
		val := input[name]
		return url.PathEscape(fmt.Sprintf("%v", val))
	})

	// 从 input 中移除路径参数
	for _, name := range placeholders {
		delete(input, name)
	}

	return result, nil
}
