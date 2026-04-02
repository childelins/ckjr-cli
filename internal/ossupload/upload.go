package ossupload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const maxImageSize = 10 * 1024 * 1024 // 10MB

// ImageSignResponse imageSign 接口响应
type ImageSignResponse struct {
	Key                 string `json:"key"`
	Policy              string `json:"policy"`
	OSSAccessKeyID      string `json:"OSSAccessKeyId"`
	Signature           string `json:"signature"`
	Callback            string `json:"callback"`
	SuccessActionStatus string `json:"success_action_status"`
	Origin              string `json:"origin"`
	Host                string `json:"host"`
}

// AssetImage 素材库图片信息
type AssetImage struct {
	ImageURL string  `json:"imageUrl"`
	Name     string  `json:"name"`
	Suffix   string  `json:"suffix"`
	Size     float64 `json:"size"`
	Width    string  `json:"width"`
	Height   string  `json:"height"`
}

// IsExternalURL 检查 URL 是否为外部图片（非系统 OSS 域名）
func IsExternalURL(imageURL string) bool {
	if imageURL == "" {
		return false
	}
	u, err := url.Parse(imageURL)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, "aliyuncs.com") {
		return false
	}
	if strings.HasSuffix(host, "ckjr001.com") {
		return false
	}
	return true
}

// downloadImage 下载外部图片，返回字节流和 Content-Type
func downloadImage(ctx context.Context, imageURL string) ([]byte, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("创建下载请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "ckjr-cli/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("下载外部图片失败: %s: %w", imageURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("下载外部图片失败: %s: HTTP %d", imageURL, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !isImageContentType(contentType) {
		return nil, "", fmt.Errorf("不支持的内容类型: %s，仅支持图片文件", contentType)
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取图片数据失败: %w", err)
	}

	if len(imgBytes) > maxImageSize {
		return nil, "", fmt.Errorf("图片过大: %d 字节，最大支持 %d 字节", len(imgBytes), maxImageSize)
	}

	return imgBytes, contentType, nil
}

// isImageContentType 检查 Content-Type 是否为图片类型
func isImageContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.HasPrefix(ct, "image/")
}

// parseFileName 从 URL 和 Content-Type 中提取文件名和扩展名
func parseFileName(imageURL, contentType string) (base, suffix string) {
	u, err := url.Parse(imageURL)
	if err != nil {
		return "image", extFromContentType(contentType)
	}

	fileName := path.Base(u.Path)
	ext := path.Ext(fileName)
	base = strings.TrimSuffix(fileName, ext)

	if ext == "" || !isKnownImageExt(ext) {
		return base, extFromContentType(contentType)
	}

	return base, ext
}

// isKnownImageExt 检查是否为已知图片扩展名
func isKnownImageExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg":
		return true
	}
	return false
}

// extFromContentType 从 Content-Type 推断扩展名
func extFromContentType(ct string) string {
	exts, _ := mime.ExtensionsByType(ct)
	if len(exts) > 0 {
		// mime.ExtensionsByType 对 image/jpeg 返回 [.jpe .jpeg .jpg]，优先选常见扩展名
		for _, e := range exts {
			if e == ".jpg" {
				return ".jpg"
			}
		}
		for _, e := range exts {
			if e == ".jpeg" || e == ".png" || e == ".gif" || e == ".webp" {
				return e
			}
		}
		return exts[0]
	}
	return ".png"
}

// uploadToOSS 直传图片到阿里云 OSS（multipart/form-data）
func uploadToOSS(ctx context.Context, signResp *ImageSignResponse, imageBytes []byte, fileName, suffix string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 按顺序写入签名字段
	fields := []struct{ key, value string }{
		{"key", signResp.Key},
		{"policy", signResp.Policy},
		{"OSSAccessKeyId", signResp.OSSAccessKeyID},
		{"success_action_status", signResp.SuccessActionStatus},
		{"callback", signResp.Callback},
		{"signature", signResp.Signature},
		{"origin", signResp.Origin},
		{"name", fileName},
		{"x:realname", fileName},
	}
	for _, f := range fields {
		if err := writer.WriteField(f.key, f.value); err != nil {
			return fmt.Errorf("写入 OSS 表单字段 %s 失败: %w", f.key, err)
		}
	}

	part, err := writer.CreateFormFile("file", fileName+suffix)
	if err != nil {
		return fmt.Errorf("创建 OSS 文件表单字段失败: %w", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		return fmt.Errorf("写入 OSS 文件数据失败: %w", err)
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", signResp.Host, body)
	if err != nil {
		return fmt.Errorf("创建 OSS 上传请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	slog.InfoContext(ctx, "oss_upload_request",
		"url", signResp.Host,
		"key", signResp.Key,
		"size_bytes", len(imageBytes),
	)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("OSS 上传请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OSS 上传失败: HTTP %d", resp.StatusCode)
	}

	slog.InfoContext(ctx, "oss_upload_response", "status", resp.StatusCode)
	return nil
}
