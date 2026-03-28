package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
		wantErr bool
	}{
		{"相等", "0.1.0", "0.1.0", 0, false},
		{"小版本更新", "0.1.0", "0.2.0", -1, false},
		{"大版本更新", "0.9.0", "1.0.0", -1, false},
		{"当前更新", "0.2.0", "0.1.0", 1, false},
		{"带 v 前缀", "v0.1.0", "v0.2.0", -1, false},
		{"混合前缀", "0.1.0", "v0.2.0", -1, false},
		{"不同段数相等", "0.1.0", "0.1", 0, false},
		{"不同段数不等", "0.1", "0.1.1", -1, false},
		{"空字符串", "", "0.1.0", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareVersions(tt.current, tt.latest)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersions(%q, %q) error = %v, wantErr %v", tt.current, tt.latest, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestCheckLatestVersion(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       interface{}
		wantVer    string
		wantErr    bool
	}{
		{
			name:       "正常响应",
			statusCode: http.StatusOK,
			body: map[string]interface{}{
				"tag_name": "v0.2.0",
				"assets": []interface{}{
					map[string]interface{}{
						"name":                 "ckjr-cli_v0.2.0_linux_amd64.tar.gz",
						"browser_download_url": "https://example.com/ckjr-cli_v0.2.0_linux_amd64.tar.gz",
					},
					map[string]interface{}{
						"name":                 "ckjr-cli_v0.2.0_darwin_arm64.tar.gz",
						"browser_download_url": "https://example.com/ckjr-cli_v0.2.0_darwin_arm64.tar.gz",
					},
				},
			},
			wantVer: "v0.2.0",
			wantErr: false,
		},
		{
			name:       "API 返回非 200",
			statusCode: http.StatusNotFound,
			body:       "not found",
			wantErr:    true,
		},
		{
			name:       "无效 JSON",
			statusCode: http.StatusOK,
			body:       "invalid json",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			switch b := tt.body.(type) {
			case string:
				bodyBytes = []byte(b)
			default:
				bodyBytes, _ = json.Marshal(b)
			}

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write(bodyBytes)
			}))
			defer ts.Close()

			gotVer, _, err := CheckLatestVersion(ts.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gotVer != tt.wantVer {
				t.Errorf("CheckLatestVersion() version = %v, want %v", gotVer, tt.wantVer)
			}
		})
	}
}

func TestParseAssetURL(t *testing.T) {
	assets := []Asset{
		{Name: "ckjr-cli_v0.2.0_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux_amd64.tar.gz"},
		{Name: "ckjr-cli_v0.2.0_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin_arm64.tar.gz"},
		{Name: "ckjr-cli_v0.2.0_windows_amd64.zip", BrowserDownloadURL: "https://example.com/windows_amd64.zip"},
	}

	tests := []struct {
		name    string
		version string
		goos    string
		goarch  string
		wantURL string
		wantErr bool
	}{
		{"linux amd64", "v0.2.0", "linux", "amd64", "https://example.com/linux_amd64.tar.gz", false},
		{"darwin arm64", "v0.2.0", "darwin", "arm64", "https://example.com/darwin_arm64.tar.gz", false},
		{"windows amd64", "v0.2.0", "windows", "amd64", "https://example.com/windows_amd64.zip", false},
		{"无匹配平台", "v0.2.0", "freebsd", "amd64", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAssetURL(assets, tt.version, tt.goos, tt.goarch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAssetURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantURL {
				t.Errorf("ParseAssetURL() = %v, want %v", got, tt.wantURL)
			}
		})
	}
}

func createTestTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{Name: name, Mode: 0755, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("写入 tar header 失败: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("写入 tar 内容失败: %v", err)
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func TestDownloadAndReplace(t *testing.T) {
	// 创建模拟的新二进制 HTTP 服务
	newBinaryContent := []byte("new-binary-content")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test-binary" {
			w.Write(newBinaryContent)
			return
		}
		// 模拟 tar.gz 产物下载
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(createTestTarGz(t, "ckjr-cli", newBinaryContent))
	}))
	defer ts.Close()

	t.Run("替换成功", func(t *testing.T) {
		// 创建模拟的当前二进制文件
		tmpDir := t.TempDir()
		binaryPath := filepath.Join(tmpDir, "ckjr-cli")

		originalContent := []byte("old-binary")
		if err := os.WriteFile(binaryPath, originalContent, 0755); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}

		err := DownloadAndReplace(ts.URL+"/test-binary", binaryPath)
		if err != nil {
			t.Fatalf("DownloadAndReplace() error = %v", err)
		}

		// 验证文件已被替换
		got, err := os.ReadFile(binaryPath)
		if err != nil {
			t.Fatalf("读取文件失败: %v", err)
		}
		if string(got) != string(newBinaryContent) {
			t.Errorf("文件内容 = %q, want %q", string(got), string(newBinaryContent))
		}

		// 验证权限
		info, _ := os.Stat(binaryPath)
		if info.Mode()&0111 == 0 {
			t.Error("文件应具有可执行权限")
		}

		// 验证 .bak 已清理
		if _, err := os.Stat(binaryPath + ".bak"); !os.IsNotExist(err) {
			t.Error(".bak 文件应已被删除")
		}
	})

	t.Run("替换失败时回滚", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalContent := []byte("original-binary")

		// 先在可写目录中创建文件和目录，然后设为只读
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
		readOnlyPath := filepath.Join(readOnlyDir, "ckjr-cli")
		if err := os.WriteFile(readOnlyPath, originalContent, 0755); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}
		// 将目录设为只读，使后续写文件失败
		if err := os.Chmod(readOnlyDir, 0555); err != nil {
			t.Fatalf("设置目录只读失败: %v", err)
		}

		err := DownloadAndReplace(ts.URL+"/test-binary", readOnlyPath)
		if err == nil {
			t.Error("期望返回错误")
		}

		// 恢复目录权限以便清理和读取
		os.Chmod(readOnlyDir, 0755)

		// 验证原始文件未被修改
		got, _ := os.ReadFile(readOnlyPath)
		if string(got) != string(originalContent) {
			t.Error("回滚失败: 原始文件内容已被修改")
		}
	})
}
