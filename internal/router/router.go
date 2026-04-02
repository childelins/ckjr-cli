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

	// 数值约束
	Min *float64 `yaml:"min,omitempty"`
	Max *float64 `yaml:"max,omitempty"`

	// 字符串约束
	MinLength *int   `yaml:"minLength,omitempty"`
	MaxLength *int   `yaml:"maxLength,omitempty"`
	Pattern   string `yaml:"pattern,omitempty"`

	// 自动转存标记，"image" 表示自动转存外部图片
	AutoUpload string `yaml:"autoUpload,omitempty"`
}

// ResponseField 定义响应字段（路径 + 可选描述）
type ResponseField struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description,omitempty"`
}

// ResponseFilter 定义响应字段过滤规则
type ResponseFilter struct {
	Fields []ResponseField
}

// FieldPaths 返回所有字段的路径列表（供过滤逻辑使用）
func (rf *ResponseFilter) FieldPaths() []string {
	paths := make([]string, len(rf.Fields))
	for i, f := range rf.Fields {
		paths[i] = f.Path
	}
	return paths
}

// UnmarshalYAML 自定义解析，支持纯字符串和 path+description 对象两种格式
func (rf *ResponseFilter) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return fmt.Errorf("response should be a list, got %v", value.Kind)
	}
	for _, node := range value.Content {
		switch node.Kind {
		case yaml.ScalarNode:
			rf.Fields = append(rf.Fields, ResponseField{Path: node.Value})
		case yaml.MappingNode:
			var field ResponseField
			if err := node.Decode(&field); err != nil {
				return err
			}
			rf.Fields = append(rf.Fields, field)
		}
	}
	return nil
}

// Route 定义单个路由
type Route struct {
	Method      string           `yaml:"method"`
	Path        string           `yaml:"path"`
	Description string           `yaml:"description"`
	Template    map[string]Field `yaml:"template"`
	Response    *ResponseFilter  `yaml:"response,omitempty"`
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
