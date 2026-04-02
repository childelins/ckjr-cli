package ossupload

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
