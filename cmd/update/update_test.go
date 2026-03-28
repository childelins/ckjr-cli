package update

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateCommand(t *testing.T) {
	t.Run("dev 版本报错", func(t *testing.T) {
		SetVersion("dev")
		cmd := NewCommand()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		if err == nil {
			t.Error("期望返回错误")
		}
		if !bytes.Contains(buf.Bytes(), []byte("开发版本")) {
			t.Errorf("输出应包含 '开发版本'，实际: %s", buf.String())
		}
	})

	t.Run("已是最新版本", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"tag_name": "v0.1.0",
				"assets": []interface{}{
					map[string]interface{}{
						"name":                 "ckjr-cli_v0.1.0_linux_amd64.tar.gz",
						"browser_download_url": "http://example.com/ckjr-cli_v0.1.0_linux_amd64.tar.gz",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer ts.Close()

		SetVersion("v0.1.0")
		cmd := NewCommand()
		cmd.SetArgs([]string{"--api-url", ts.URL})
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		if err != nil {
			t.Errorf("不期望错误: %v", err)
		}
		if !bytes.Contains(buf.Bytes(), []byte("已是最新版本")) {
			t.Errorf("输出应包含 '已是最新版本'，实际: %s", buf.String())
		}
	})
}

func TestNewCommand(t *testing.T) {
	cmd := NewCommand()
	if cmd.Use != "update" {
		t.Errorf("Use = %q, want 'update'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short 不应为空")
	}
}
