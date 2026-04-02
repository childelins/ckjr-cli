package ossupload

import (
	"net/url"
	"strings"
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
