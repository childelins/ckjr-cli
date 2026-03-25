package router

import (
	"testing"
)

func TestParseRouteConfig(t *testing.T) {
	yamlContent := `
resource: agent
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

	if cfg.Resource != "agent" {
		t.Errorf("Resource = %s, want agent", cfg.Resource)
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

func TestGetTemplate(t *testing.T) {
	cfg := &RouteConfig{
		Resource: "agent",
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
