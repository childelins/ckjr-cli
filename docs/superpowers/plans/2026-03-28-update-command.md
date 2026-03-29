# Update 命令实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 为 ckjr-cli 添加 `ckjr-cli update` 命令，自动检测 GitHub Release 最新版本并替换当前二进制。

**Architecture:** 新增 `internal/updater` 包封装核心逻辑（版本查询、比较、下载、替换），`cmd/update` 包提供 Cobra 命令。updater 的网络调用通过接口抽象以支持 httptest 测试。遵循项目现有的 `NewCommand()` + `SetVersion()` 模式。

**Tech Stack:** Go 1.24, Cobra, net/http (标准库), archive/tar, compress/gzip, archive/zip

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `internal/updater/updater.go` | 核心更新逻辑：版本比较、GitHub API 查询、产物匹配、下载、替换 |
| `internal/updater/updater_test.go` | updater 单元测试 |
| `cmd/update/update.go` | Cobra 命令定义 |
| `cmd/update/update_test.go` | 命令集成测试 |
| `cmd/root.go` | 注册 update 命令（修改） |

---

## Phase 1: 版本比较

### Task 1: 实现 CompareVersions

**Files:**
- Create: `internal/updater/updater.go`
- Create: `internal/updater/updater_test.go`

- [ ] **Step 1: 写失败测试**

`internal/updater/updater_test.go`:

```go
package updater

import "testing"

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
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /home/childelins/code/ckjr-cli && go test ./internal/updater/ -run TestCompareVersions -v
```

Expected: FAIL - `CompareVersions` 未定义

- [ ] **Step 3: 实现最小代码**

`internal/updater/updater.go`:

```go
package updater

import (
	"strconv"
	"strings"
)

// CompareVersions 比较两个 semver 版本。
// 返回 >0 表示 current 比 latest 新，<0 表示有更新可用，0 表示相同。
func CompareVersions(current, latest string) (int, error) {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	curParts := strings.Split(current, ".")
	latParts := strings.Split(latest, ".")

	maxLen := len(curParts)
	if len(latParts) > maxLen {
		maxLen = len(latParts)
	}

	for i := 0; i < maxLen; i++ {
		cur := partToInt(curParts, i)
		lat := partToInt(latParts, i)
		if cur != lat {
			return cur - lat, nil
		}
	}
	return 0, nil
}

func partToInt(parts []string, idx int) int {
	if idx >= len(parts) {
		return 0
	}
	n, _ := strconv.Atoi(parts[idx])
	return n
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /home/childelins/code/ckjr-cli && go test ./internal/updater/ -run TestCompareVersions -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/updater/updater.go internal/updater/updater_test.go
git commit -m "feat(update): add CompareVersions for semver comparison"
```

---

## Phase 2: GitHub API 版本查询 + 产物匹配

### Task 2: 实现 CheckLatestVersion 和 ParseAssetURL

**Files:**
- Modify: `internal/updater/updater.go`
- Modify: `internal/updater/updater_test.go`

- [ ] **Step 1: 写失败测试**

在 `internal/updater/updater_test.go` 追加：

```go
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestCheckLatestVersion(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       interface{}
		wantVer    string
		wantURL    string
		wantErr    bool
	}{
		{
			name: "正常响应",
			statusCode: http.StatusOK,
			body: map[string]interface{}{
				"tag_name": "v0.2.0",
				"assets": []interface{}{
					map[string]interface{}{
						"name":               "ckjr-cli_v0.2.0_linux_amd64.tar.gz",
						"browser_download_url": "https://example.com/ckjr-cli_v0.2.0_linux_amd64.tar.gz",
					},
					map[string]interface{}{
						"name":               "ckjr-cli_v0.2.0_darwin_arm64.tar.gz",
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
		name     string
		version  string
		goos     string
		goarch   string
		wantURL  string
		wantErr  bool
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
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /home/childelins/code/ckjr-cli && go test ./internal/updater/ -run "TestCheckLatestVersion|TestParseAssetURL" -v
```

Expected: FAIL - 函数未定义

- [ ] **Step 3: 实现最小代码**

在 `internal/updater/updater.go` 追加：

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// GitHubRelease 表示 GitHub Release API 响应
type GitHubRelease struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset 表示 Release 中的产物
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckLatestVersion 查询 GitHub Release 最新版本。
// apiURL 参数支持自定义 base URL（用于测试）。
func CheckLatestVersion(apiURL string) (version string, downloadURL string, err error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(apiURL)
	if err != nil {
		return "", "", fmt.Errorf("无法检查更新: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("检查更新失败: GitHub API 返回 %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("解析版本信息失败: %w", err)
	}

	url, err := ParseAssetURL(release.Assets, release.TagName, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", "", err
	}

	return release.TagName, url, nil
}

// ParseAssetURL 从 Release assets 中匹配当前平台的下载 URL
func ParseAssetURL(assets []Asset, version, goos, goarch string) (string, error) {
	ver := strings.TrimPrefix(version, "v")
	var suffix string
	if goos == "windows" {
		suffix = ".zip"
	} else {
		suffix = ".tar.gz"
	}
	pattern := fmt.Sprintf("ckjr-cli_%s_%s_%s%s", ver, goos, goarch, suffix)

	for _, a := range assets {
		if a.Name == pattern {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("未找到 %s/%s 平台的更新包", goos, goarch)
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /home/childelins/code/ckjr-cli && go test ./internal/updater/ -v
```

Expected: PASS（所有测试）

- [ ] **Step 5: 提交**

```bash
git add internal/updater/updater.go internal/updater/updater_test.go
git commit -m "feat(update): add CheckLatestVersion and ParseAssetURL"
```

---

## Phase 3: 下载与替换

### Task 3: 实现 DownloadAndReplace

**Files:**
- Modify: `internal/updater/updater.go`
- Modify: `internal/updater/updater_test.go`

- [ ] **Step 1: 写失败测试**

在 `internal/updater/updater_test.go` 追加：

```go
import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

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
		if runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}
		originalContent := []byte("old-binary")
		if err := os.WriteFile(binaryPath, originalContent, 0755); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}

		// 注意: DownloadAndReplace 需要知道当前二进制路径，通过参数传入
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
		binaryPath := filepath.Join(tmpDir, "ckjr-cli")
		originalContent := []byte("original-binary")
		if err := os.WriteFile(binaryPath, originalContent, 0755); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}

		// 让目标目录不可写以模拟替换失败
		readOnlyDir := filepath.Join(tmpDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0555); err != nil {
			t.Fatalf("创建只读目录失败: %v", err)
		}
		readOnlyPath := filepath.Join(readOnlyDir, "ckjr-cli")
		if err := os.WriteFile(readOnlyPath, originalContent, 0555); err != nil {
			t.Fatalf("写入测试文件失败: %v", err)
		}

		err := DownloadAndReplace(ts.URL+"/test-binary", readOnlyPath)
		if err == nil {
			t.Error("期望返回错误")
		}

		// 验证原始文件未被修改
		got, _ := os.ReadFile(readOnlyPath)
		if string(got) != string(originalContent) {
			t.Error("回滚失败: 原始文件内容已被修改")
		}
	})
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /home/childelins/code/ckjr-cli && go test ./internal/updater/ -run TestDownloadAndReplace -v
```

Expected: FAIL - `DownloadAndReplace` 未定义，`createTestTarGz` 未定义

- [ ] **Step 3: 实现最小代码**

在 `internal/updater/updater_test.go` 顶部追加测试辅助函数：

```go
import (
	"archive/tar"
	"compress/gzip"
	"bytes"
)

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
```

在 `internal/updater/updater.go` 追加：

```go
import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DownloadAndReplace 下载新版本并替换当前二进制文件。
// binaryPath 参数用于测试时传入自定义路径；生产环境传空字符串，自动获取当前可执行文件路径。
func DownloadAndReplace(downloadURL string, binaryPath string) error {
	if binaryPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("获取当前可执行文件路径失败: %w", err)
		}
		binaryPath, err = filepath.EvalSymlinks(exe)
		if err != nil {
			return fmt.Errorf("解析可执行文件路径失败: %w", err)
		}
	}

	// 下载到临时目录
	tmpDir, err := os.MkdirTemp("", "ckjr-cli-update-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archiveName := filepath.Base(downloadURL)
	archivePath := filepath.Join(tmpDir, archiveName)

	if err := downloadFile(downloadURL, archivePath); err != nil {
		return fmt.Errorf("下载更新失败: %w", err)
	}

	// 解压
	var newBinaryPath string
	if strings.HasSuffix(archiveName, ".tar.gz") {
		newBinaryPath, err = extractTarGz(archivePath, tmpDir)
	} else if strings.HasSuffix(archiveName, ".zip") {
		newBinaryPath, err = extractZip(archivePath, tmpDir)
	} else {
		return fmt.Errorf("不支持的归档格式: %s", archiveName)
	}
	if err != nil {
		return fmt.Errorf("解压更新包失败: %w", err)
	}

	// 替换二进制
	if err := replaceBinary(binaryPath, newBinaryPath); err != nil {
		return err
	}

	return nil
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func extractTarGz(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag == tar.TypeReg && !strings.HasSuffix(hdr.Name, "/") {
			// 只提取文件名包含 ckjr-cli 的文件（跳过 README 等）
			baseName := filepath.Base(hdr.Name)
			if strings.HasPrefix(baseName, "ckjr-cli") {
				outPath := filepath.Join(destDir, baseName)
				outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
				if err != nil {
					return "", err
				}
				if _, err := io.Copy(outFile, tr); err != nil {
					outFile.Close()
					return "", err
				}
				outFile.Close()
				return outPath, nil
			}
		}
	}
	return "", fmt.Errorf("归档中未找到 ckjr-cli 二进制")
}

func extractZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Mode().IsRegular() {
			baseName := filepath.Base(f.Name)
			if strings.HasPrefix(baseName, "ckjr-cli") {
				outPath := filepath.Join(destDir, baseName)
				outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, f.Mode())
				if err != nil {
					return "", err
				}
				rc, err := f.Open()
				if err != nil {
					outFile.Close()
					return "", err
				}
				if _, err := io.Copy(outFile, rc); err != nil {
					rc.Close()
					outFile.Close()
					return "", err
				}
				rc.Close()
				outFile.Close()
				return outPath, nil
			}
		}
	}
	return "", fmt.Errorf("归档中未找到 ckjr-cli 二进制")
}

func replaceBinary(currentPath, newBinaryPath string) error {
	// 备份当前二进制
	bakPath := currentPath + ".bak"
	if err := os.Rename(currentPath, bakPath); err != nil {
		return fmt.Errorf("备份当前二进制失败: %w", err)
	}

	// 复制新文件到原路径
	newContent, err := os.ReadFile(newBinaryPath)
	if err != nil {
		// 回滚
		os.Rename(bakPath, currentPath)
		return fmt.Errorf("读取新二进制失败: %w", err)
	}

	if err := os.WriteFile(currentPath, newContent, 0755); err != nil {
		// 回滚
		if renameErr := os.Rename(bakPath, currentPath); renameErr != nil {
			return fmt.Errorf("更新失败且回滚失败，请手动安装: %s.bak 备份已保留: %w", currentPath, renameErr)
		}
		return fmt.Errorf("替换二进制失败: %w，已回滚", err)
	}

	// 成功，删除备份
	os.Remove(bakPath)
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /home/childelins/code/ckjr-cli && go test ./internal/updater/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/updater/updater.go internal/updater/updater_test.go
git commit -m "feat(update): add DownloadAndReplace with tar.gz/zip extraction and rollback"
```

---

## Phase 4: Cobra 命令集成

### Task 4: 实现 cmd/update 命令

**Files:**
- Create: `cmd/update/update.go`
- Create: `cmd/update/update_test.go`

- [ ] **Step 1: 写失败测试**

`cmd/update/update_test.go`:

```go
package update

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
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
		cmd.SetArgs("--api-url", ts.URL)
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
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /home/childelins/code/ckjr-cli && go test ./cmd/update/ -v
```

Expected: FAIL - 包未定义

- [ ] **Step 3: 实现最小代码**

`cmd/update/update.go`:

```go
package update

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/updater"
)

var currentVersion = "dev"

// SetVersion 设置当前版本号，由 cmd/root.go 调用
func SetVersion(v string) {
	currentVersion = v
}

// apiURL 可在测试中覆盖
var defaultAPIURL = "https://api.github.com/repos/childelins/ckjr-cli/releases/latest"

// NewCommand 创建 update 命令
func NewCommand() *cobra.Command {
	var apiURL string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "检查并更新 ckjr-cli 到最新版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, apiURL)
		},
	}

	cmd.Flags().StringVar(&apiURL, "api-url", defaultAPIURL, "GitHub API URL（用于测试）")
	_ = cmd.Flags().MarkHidden("api-url")

	return cmd
}

func runUpdate(cmd *cobra.Command, apiURL string) error {
	if currentVersion == "dev" {
		return fmt.Errorf("当前为开发版本 (dev)，请使用 install.sh 安装正式版本")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "正在检查更新...")

	latestVersion, downloadURL, err := updater.CheckLatestVersion(apiURL)
	if err != nil {
		return err
	}

	cmp, err := updater.CompareVersions(currentVersion, latestVersion)
	if err != nil {
		return err
	}

	if cmp >= 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "已是最新版本 (%s)\n", currentVersion)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "发现新版本: %s → %s\n", currentVersion, latestVersion)
	fmt.Fprintln(cmd.OutOrStdout(), "正在下载更新...")

	if err := updater.DownloadAndReplace(downloadURL, ""); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "更新成功！%s → %s\n", currentVersion, latestVersion)
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /home/childelins/code/ckjr-cli && go test ./cmd/update/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/update/update.go cmd/update/update_test.go
git commit -m "feat(update): add ckjr-cli update command with version check and self-update"
```

---

## Phase 5: 注册命令

### Task 5: 在 root.go 中注册 update 命令

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: 修改 cmd/root.go**

在 import 块中追加：

```go
updatecmd "github.com/childelins/ckjr-cli/cmd/update"
```

在 `init()` 函数末尾追加：

```go
updatecmd.SetVersion(version)
rootCmd.AddCommand(updatecmd.NewCommand())
```

完整修改后的 `init()` 函数：

```go
func init() {
	rootCmd.PersistentFlags().Bool("pretty", false, "格式化 JSON 输出")
	rootCmd.PersistentFlags().Bool("verbose", false, "显示详细调试信息")
	cobra.OnInitialize(initLogging)

	rootCmd.AddCommand(configcmd.NewCommand())
	rootCmd.AddCommand(routecmd.NewCommand())
	updatecmd.SetVersion(version)
	rootCmd.AddCommand(updatecmd.NewCommand())
}
```

- [ ] **Step 2: 验证编译通过**

```bash
cd /home/childelins/code/ckjr-cli && go build ./cmd/ckjr-cli/
```

Expected: 编译成功，无错误

- [ ] **Step 3: 验证 update 子命令可见**

```bash
cd /home/childelins/code/ckjr-cli && ./ckjr-cli --help
```

Expected: 输出中包含 `update` 子命令

- [ ] **Step 4: 运行全量测试**

```bash
cd /home/childelins/code/ckjr-cli && go test ./...
```

Expected: 所有测试通过

- [ ] **Step 5: 提交**

```bash
git add cmd/root.go
git commit -m "feat(update): register update command in root"
```
