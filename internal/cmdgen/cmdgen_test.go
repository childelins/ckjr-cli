package cmdgen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/router"
)

func TestBuildCommand(t *testing.T) {
	cfg := &router.RouteConfig{
		Resource:    "agent",
		Description: "AI智能体管理",
		Routes: map[string]router.Route{
			"list": {
				Method:      "POST",
				Path:        "/admin/list",
				Description: "获取列表",
				Template: map[string]router.Field{
					"page": {
						Description: "页码",
						Required:    false,
						Default:     1,
					},
				},
			},
			"get": {
				Method:      "POST",
				Path:        "/admin/get",
				Description: "获取详情",
				Template: map[string]router.Field{
					"id": {
						Description: "ID",
						Required:    true,
					},
				},
			},
		},
	}

	cmd := BuildCommand(cfg, nil)
	if cmd.Use != "agent" {
		t.Errorf("Use = %s, want agent", cmd.Use)
	}

	if cmd.Short != "AI智能体管理" {
		t.Errorf("Short = %s", cmd.Short)
	}

	// 验证子命令
	subCmds := cmd.Commands()
	if len(subCmds) != 2 {
		t.Fatalf("子命令数量 = %d, want 2", len(subCmds))
	}

	// 验证 list 子命令
	listCmd, _, _ := cmd.Find([]string{"list"})
	if listCmd == nil {
		t.Error("list 子命令未找到")
	}
}

func TestTemplateFlag(t *testing.T) {
	cfg := &router.RouteConfig{
		Resource: "agent",
		Routes: map[string]router.Route{
			"create": {
				Method: "POST",
				Path:   "/create",
				Template: map[string]router.Field{
					"name": {
						Description: "名称",
						Required:    true,
					},
				},
			},
		},
	}

	cmd := BuildCommand(cfg, nil)
	createCmd, _, _ := cmd.Find([]string{"create"})
	if createCmd == nil {
		t.Fatal("create 子命令未找到")
	}

	// 验证 --template flag 存在
	templateFlag := createCmd.Flags().Lookup("template")
	if templateFlag == nil {
		t.Error("--template flag 未找到")
	}
}

func TestHandleAPIError_ResponseError(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试",
	}

	handleAPIErrorTo(&buf, respErr, false)

	got := buf.String()
	if !strings.Contains(got, "服务端返回异常") {
		t.Errorf("output should contain friendly message, got: %s", got)
	}
	if strings.Contains(got, "text/html") {
		t.Error("non-verbose should not contain Content-Type")
	}
}

func TestHandleAPIError_ResponseError_Verbose(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试",
	}

	handleAPIErrorTo(&buf, respErr, true)

	got := buf.String()
	if !strings.Contains(got, "服务端返回异常") {
		t.Errorf("output should contain friendly message, got: %s", got)
	}
	if !strings.Contains(got, "502") || !strings.Contains(got, "text/html") {
		t.Errorf("verbose should contain debug info, got: %s", got)
	}
}
