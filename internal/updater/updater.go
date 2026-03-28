package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
		// 非 tar.gz/.zip 后缀，直接当作裸二进制下载
		newBinaryPath = archivePath
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
