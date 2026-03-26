package yamlgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/childelins/ckjr-cli/internal/curlparse"
	"github.com/childelins/ckjr-cli/internal/router"
)

func TestGenerateRoute(t *testing.T) {
	result := &curlparse.Result{
		Method: "POST",
		Path:   "/admin/aiCreationCenter/modifyApp",
		Fields: map[string]curlparse.Field{
			"name":   {Type: "string", Example: "test"},
			"aikbId": {Type: "int", Example: 3550},
		},
	}

	route := GenerateRoute(result)

	if route.Method != "POST" {
		t.Errorf("Method = %q, want POST", route.Method)
	}
	if route.Path != "/admin/aiCreationCenter/modifyApp" {
		t.Errorf("Path = %q, want /admin/aiCreationCenter/modifyApp", route.Path)
	}
	if route.Description != "" {
		t.Errorf("Description = %q, want empty", route.Description)
	}
	if len(route.Template) != 2 {
		t.Fatalf("Template count = %d, want 2", len(route.Template))
	}
	f := route.Template["aikbId"]
	if f.Type != "int" {
		t.Errorf("aikbId.Type = %q, want int", f.Type)
	}
	if f.Example != "" {
		t.Errorf("aikbId.Example = %q, want empty", f.Example)
	}
	if f.Description != "" {
		t.Errorf("aikbId.Description = %q, want empty", f.Description)
	}
}

func TestAppendToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	// 写入初始文件
	initial := `resource: agent
description: AI智能体管理
routes:
    list:
        method: POST
        path: /admin/list
        description: 获取列表
        template:
            page:
                description: 页码
                required: false
`
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	route := router.Route{
		Method:   "POST",
		Path:     "/admin/create",
		Template: map[string]router.Field{
			"name": {Required: false},
		},
	}

	if err := AppendToFile(path, "create", route); err != nil {
		t.Fatalf("AppendToFile() error = %v", err)
	}

	// 验证结果
	data, _ := os.ReadFile(path)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(cfg.Routes) != 2 {
		t.Errorf("Routes count = %d, want 2", len(cfg.Routes))
	}
	if _, ok := cfg.Routes["create"]; !ok {
		t.Error("create route not found")
	}
}

func TestAppendToFile_Conflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	initial := `resource: agent
description: test
routes:
    list:
        method: POST
        path: /list
        description: 列表
`
	os.WriteFile(path, []byte(initial), 0644)

	route := router.Route{Method: "POST", Path: "/list2"}
	err := AppendToFile(path, "list", route)
	if err == nil {
		t.Error("expected conflict error")
	}
}

func TestCreateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.yaml")

	route := router.Route{
		Method: "POST",
		Path:   "/admin/order/list",
		Template: map[string]router.Field{
			"page": {Required: false, Type: "int"},
		},
	}

	if err := CreateFile(path, "order", "订单管理", "list", route); err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Resource != "order" {
		t.Errorf("Resource = %q, want order", cfg.Resource)
	}
	if cfg.Description != "订单管理" {
		t.Errorf("Description = %q", cfg.Description)
	}
	if _, ok := cfg.Routes["list"]; !ok {
		t.Error("list route not found")
	}
}

func TestCreateFile_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.yaml")
	os.WriteFile(path, []byte("test"), 0644)

	route := router.Route{Method: "POST", Path: "/test"}
	err := CreateFile(path, "test", "", "get", route)
	if err == nil {
		t.Error("expected file exists error")
	}
}
