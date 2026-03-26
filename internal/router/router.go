package router

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Field 定义模板字段
type Field struct {
	Description string      `yaml:"description"`
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default,omitempty"`
	Type        string      `yaml:"type,omitempty"`
	Example     string      `yaml:"example,omitempty"`
}

// Route 定义单个路由
type Route struct {
	Method      string           `yaml:"method"`
	Path        string           `yaml:"path"`
	Description string           `yaml:"description,omitempty"`
	Template    map[string]Field `yaml:"template"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Routes      map[string]Route `yaml:"routes"`
}

// Parse 解析 YAML 路由配置
func Parse(data []byte) (*RouteConfig, error) {
	var cfg RouteConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析路由配置失败: %w", err)
	}
	return &cfg, nil
}

// GetRoute 获取指定路由
func (c *RouteConfig) GetRoute(name string) (Route, bool) {
	route, ok := c.Routes[name]
	return route, ok
}

// GetTemplate 获取路由模板
func (c *RouteConfig) GetTemplate(routeName string) map[string]Field {
	route, ok := c.GetRoute(routeName)
	if !ok {
		return nil
	}
	return route.Template
}

// RequiredFields 获取必填字段列表
func (c *RouteConfig) RequiredFields(routeName string) []string {
	tmpl := c.GetTemplate(routeName)
	if tmpl == nil {
		return nil
	}

	var required []string
	for name, field := range tmpl {
		if field.Required {
			required = append(required, name)
		}
	}
	return required
}
