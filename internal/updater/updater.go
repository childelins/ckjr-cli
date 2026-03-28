package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// CompareVersions 比较两个 semver 版本。
// 返回 >0 表示 current 比 latest 新，<0 表示有更新可用，0 表示相同。
func CompareVersions(current, latest string) (int, error) {
	if current == "" {
		return 0, nil
	}
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
	var suffix string
	if goos == "windows" {
		suffix = ".zip"
	} else {
		suffix = ".tar.gz"
	}
	pattern := fmt.Sprintf("ckjr-cli_%s_%s_%s%s", version, goos, goarch, suffix)

	for _, a := range assets {
		if a.Name == pattern {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("未找到 %s/%s 平台的更新包", goos, goarch)
}
