package yamlgen

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/childelins/ckjr-cli/internal/curlparse"
	"github.com/childelins/ckjr-cli/internal/router"
)

// GenerateRoute 从 curlparse.Result 生成 Route
func GenerateRoute(result *curlparse.Result) router.Route {
	tmpl := make(map[string]router.Field, len(result.Fields))
	for name, f := range result.Fields {
		field := router.Field{}
		if f.Type != "" && f.Type != "string" {
			field.Type = f.Type
		}
		tmpl[name] = field
	}
	return router.Route{
		Method:   result.Method,
		Path:     result.Path,
		Template: tmpl,
	}
}

// AppendToFile 追加路由到已有 YAML 文件
func AppendToFile(path string, name string, route router.Route) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	cfg, err := router.Parse(data)
	if err != nil {
		return fmt.Errorf("解析 YAML 失败: %w", err)
	}

	if _, exists := cfg.Routes[name]; exists {
		return fmt.Errorf("路由 %q 已存在", name)
	}

	cfg.Routes[name] = route
	return writeConfig(path, cfg)
}

// CreateFile 创建新的 YAML 路由文件
func CreateFile(path string, resource string, resourceDesc string, name string, route router.Route) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("文件已存在: %s", path)
	}

	cfg := &router.RouteConfig{
		Resource:    resource,
		Description: resourceDesc,
		Routes:      map[string]router.Route{name: route},
	}
	return writeConfig(path, cfg)
}

func writeConfig(path string, cfg *router.RouteConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("YAML 序列化失败: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
