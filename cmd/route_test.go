package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
)

func TestRouteCmd_IsHidden(t *testing.T) {
	if !routeCmd.Hidden {
		t.Error("routeCmd should be hidden")
	}
}

func TestRouteImport_Stdin_AppendToExisting(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "agent.yaml")

	initial := `resource: agent
description: AI智能体管理
routes:
    list:
        method: POST
        path: /admin/list
        description: 获取列表
`
	os.WriteFile(yamlPath, []byte(initial), 0644)

	curl := `curl 'https://kpapi-cs.ckjr001.com/api/admin/aiCreationCenter/modifyApp' -H 'content-type: application/json' --data-raw '{"name":"test","aikbId":3550}'`

	err := runImport(curl, yamlPath, "update", "", "")
	if err != nil {
		t.Fatalf("runImport() error = %v", err)
	}

	data, _ := os.ReadFile(yamlPath)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(cfg.Routes) != 2 {
		t.Errorf("Routes count = %d, want 2", len(cfg.Routes))
	}
	route, ok := cfg.Routes["update"]
	if !ok {
		t.Fatal("update route not found")
	}
	if route.Method != "POST" {
		t.Errorf("Method = %q", route.Method)
	}
}

func TestRouteImport_CreateNewFile(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "order.yaml")

	curl := `curl 'https://example.com/api/admin/order/list' --data-raw '{"page":1,"limit":10}'`

	err := runImport(curl, yamlPath, "list", "order", "订单管理")
	if err != nil {
		t.Fatalf("runImport() error = %v", err)
	}

	data, _ := os.ReadFile(yamlPath)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Resource != "order" {
		t.Errorf("Resource = %q", cfg.Resource)
	}
	if _, ok := cfg.Routes["list"]; !ok {
		t.Error("list route not found")
	}
}

func TestInferRouteName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/admin/aiCreationCenter/modifyApp", "update"},
		{"/admin/aiCreationCenter/listApp", "listApp"},
		{"/admin/aiCreationCenter/createApp", "create"},
		{"/admin/aiCreationCenter/deleteApp", "deleteApp"},
		{"/admin/aiCreationCenter/describeApp", "get"},
		{"/admin/order/addOrder", "create"},
		{"/admin/order/removeOrder", "delete"},
		{"/admin/order/editOrder", "update"},
		{"/admin/order/queryList", "list"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := inferRouteName(tt.path)
			if got != tt.want {
				t.Errorf("inferRouteName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
