package updater

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
