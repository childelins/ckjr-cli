package router

import (
	"testing"

	"gopkg.in/yaml.v3"
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

func TestParseRouteConfig_Constraints(t *testing.T) {
	yamlContent := `
name: test
description: 测试约束
routes:
  create:
    method: POST
    path: /create
    template:
      page:
        description: 页码
        required: false
        default: 1
        type: int
        min: 1
        max: 1000
      keyword:
        description: 关键词
        required: false
        type: string
        minLength: 1
        maxLength: 100
      email:
        description: 邮箱
        required: true
        type: string
        pattern: "^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$"
      score:
        description: 评分
        required: false
        type: float
        min: 0.0
        max: 10.0
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	route := cfg.Routes["create"]

	// page: min/max
	page := route.Template["page"]
	if page.Min == nil || *page.Min != 1.0 {
		t.Errorf("page.Min = %v, want 1.0", page.Min)
	}
	if page.Max == nil || *page.Max != 1000.0 {
		t.Errorf("page.Max = %v, want 1000.0", page.Max)
	}

	// keyword: minLength/maxLength
	keyword := route.Template["keyword"]
	if keyword.MinLength == nil || *keyword.MinLength != 1 {
		t.Errorf("keyword.MinLength = %v, want 1", keyword.MinLength)
	}
	if keyword.MaxLength == nil || *keyword.MaxLength != 100 {
		t.Errorf("keyword.MaxLength = %v, want 100", keyword.MaxLength)
	}

	// email: pattern
	email := route.Template["email"]
	if email.Pattern != `^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$` {
		t.Errorf("email.Pattern = %q", email.Pattern)
	}

	// score: float type with min/max
	score := route.Template["score"]
	if score.Type != "float" {
		t.Errorf("score.Type = %q, want \"float\"", score.Type)
	}
	if score.Min == nil || *score.Min != 0.0 {
		t.Errorf("score.Min = %v, want 0.0", score.Min)
	}
	if score.Max == nil || *score.Max != 10.0 {
		t.Errorf("score.Max = %v, want 10.0", score.Max)
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

func TestRoute_ResponseFilter_Unmarshal(t *testing.T) {
	yamlData := `
method: GET
path: /admin/courses/{courseId}/edit
description: 获取课程详情
response:
    - courseId
    - name
    - status
`
	var route Route
	if err := yaml.Unmarshal([]byte(yamlData), &route); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if route.Response == nil {
		t.Fatal("Response should not be nil")
	}
	if len(route.Response.Fields) != 3 {
		t.Errorf("Fields count = %d, want 3", len(route.Response.Fields))
	}
}

func TestResponseFilter_MixedFieldFormats(t *testing.T) {
	yamlData := `
- data.courseId
- path: data.courseType
  description: "课程类型, 0-视频 1-音频 2-图文"
- path: data.status
  description: "上架状态, 1-已上架 2-已下架"
- data.name
`
	var rf ResponseFilter
	if err := yaml.Unmarshal([]byte(yamlData), &rf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rf.Fields) != 4 {
		t.Fatalf("Fields count = %d, want 4", len(rf.Fields))
	}

	// 纯字符串格式
	if rf.Fields[0].Path != "data.courseId" {
		t.Errorf("Fields[0].Path = %q, want data.courseId", rf.Fields[0].Path)
	}
	if rf.Fields[0].Description != "" {
		t.Errorf("Fields[0].Description = %q, want empty", rf.Fields[0].Description)
	}

	// 对象格式
	if rf.Fields[1].Path != "data.courseType" {
		t.Errorf("Fields[1].Path = %q, want data.courseType", rf.Fields[1].Path)
	}
	if rf.Fields[1].Description != "课程类型, 0-视频 1-音频 2-图文" {
		t.Errorf("Fields[1].Description = %q", rf.Fields[1].Description)
	}

	// FieldPaths 返回纯路径列表
	paths := rf.FieldPaths()
	want := []string{"data.courseId", "data.courseType", "data.status", "data.name"}
	for i, p := range paths {
		if p != want[i] {
			t.Errorf("FieldPaths()[%d] = %q, want %q", i, p, want[i])
		}
	}
}

func TestResponseFilter_BackwardCompat_PureStrings(t *testing.T) {
	yamlData := `
- courseId
- name
- status
`
	var rf ResponseFilter
	if err := yaml.Unmarshal([]byte(yamlData), &rf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rf.Fields) != 3 {
		t.Fatalf("Fields count = %d, want 3", len(rf.Fields))
	}
	paths := rf.FieldPaths()
	if paths[0] != "courseId" || paths[1] != "name" || paths[2] != "status" {
		t.Errorf("FieldPaths() = %v", paths)
	}
}

func TestResponseFilter_InvalidFormat(t *testing.T) {
	yamlData := `
key: value
nested: true
`
	var rf ResponseFilter
	err := yaml.Unmarshal([]byte(yamlData), &rf)
	if err == nil {
		t.Error("expected error for non-sequence YAML")
	}
}

func TestParseRouteConfig_AutoUpload(t *testing.T) {
	yamlContent := `
name: test
description: 测试自动上传
routes:
  create:
    method: POST
    path: /create
    description: 创建
    template:
      avatar:
        description: 头像URL
        required: true
        type: string
        autoUpload: image
      name:
        description: 名称
        required: true
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	route := cfg.Routes["create"]

	// avatar 字段应有 AutoUpload = "image"
	avatarField := route.Template["avatar"]
	if avatarField.AutoUpload != "image" {
		t.Errorf("avatar.AutoUpload = %q, want \"image\"", avatarField.AutoUpload)
	}

	// name 字段不应有 AutoUpload
	nameField := route.Template["name"]
	if nameField.AutoUpload != "" {
		t.Errorf("name.AutoUpload = %q, want empty", nameField.AutoUpload)
	}
}

func TestRoute_ResponseFilter_Nil(t *testing.T) {
	yamlData := `
method: GET
path: /admin/courses
`
	var route Route
	if err := yaml.Unmarshal([]byte(yamlData), &route); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if route.Response != nil {
		t.Error("Response should be nil when not configured")
	}
}
