package router

import (
	"testing"
)

func TestParseRouteConfig(t *testing.T) {
	yamlContent := `
name: agent
description: AI智能体的增删改查
routes:
  list:
    method: POST
    path: /admin/aiCreationCenter/listApp
    description: 获取智能体列表
    template:
      page:
        description: 页码
        required: false
        default: 1
  get:
    method: POST
    path: /admin/aiCreationCenter/getAppInfo
    description: 获取智能体详情
    template:
      aikbId:
        description: 智能体ID
        required: true
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Name != "agent" {
		t.Errorf("Name = %s, want agent", cfg.Name)
	}

	if len(cfg.Routes) != 2 {
		t.Fatalf("Routes count = %d, want 2", len(cfg.Routes))
	}

	listRoute, ok := cfg.Routes["list"]
	if !ok {
		t.Fatal("list route not found")
	}
	if listRoute.Method != "POST" {
		t.Errorf("list.Method = %s, want POST", listRoute.Method)
	}
	if listRoute.Path != "/admin/aiCreationCenter/listApp" {
		t.Errorf("list.Path = %s", listRoute.Path)
	}
}

func TestParseRouteConfig_TypeAndExample(t *testing.T) {
	yamlContent := `
name: test
description: 测试模块
routes:
  create:
    method: POST
    path: /create
    description: 创建
    template:
      count:
        description: 数量
        required: false
        default: 10
        type: int
        example: "10"
      name:
        description: 名称
        required: true
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	route, ok := cfg.Routes["create"]
	if !ok {
		t.Fatal("create route not found")
	}

	// 有 type/example 的字段
	countField := route.Template["count"]
	if countField.Type != "int" {
		t.Errorf("count.Type = %q, want \"int\"", countField.Type)
	}
	if countField.Example != "10" {
		t.Errorf("count.Example = %q, want \"10\"", countField.Example)
	}

	// 未设置 type/example 的字段，应为零值
	nameField := route.Template["name"]
	if nameField.Type != "" {
		t.Errorf("name.Type = %q, want \"\"", nameField.Type)
	}
	if nameField.Example != "" {
		t.Errorf("name.Example = %q, want \"\"", nameField.Example)
	}
}

func TestGetTemplate(t *testing.T) {
	cfg := &RouteConfig{
		Name: "agent",
		Routes: map[string]Route{
			"create": {
				Method:      "POST",
				Path:        "/create",
				Description: "创建",
				Template: map[string]Field{
					"name": {
						Description: "名称",
						Required:    true,
					},
					"page": {
						Description: "页码",
						Required:    false,
						Default:     1,
					},
				},
			},
		},
	}

	tmpl := cfg.GetTemplate("create")
	if len(tmpl) != 2 {
		t.Fatalf("GetTemplate() count = %d, want 2", len(tmpl))
	}

	if tmpl["name"].Default != nil {
		t.Error("name should not have default")
	}
	if tmpl["page"].Default == nil {
		t.Error("page should have default")
	}
}
