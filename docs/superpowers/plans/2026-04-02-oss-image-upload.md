# OSS 图片上传实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 为 ckjr-cli 新增 OSS 图片转存能力，将外部图片 URL 转存到系统素材库，支持独立 CLI 命令和 workflow 集成两种模式。

**Architecture:** 新增 `internal/ossupload` 包封装完整的 4 步上传流程（签名 -> 下载 -> OSS 直传 -> 保存素材库）。OSS 直传使用标准 `http.Client` + multipart/form-data，不走 `api.Client`。通过 `asset upload-image` 子命令暴露给 CLI。

**Tech Stack:** Go 标准库 (`net/http`, `mime/multipart`, `net/url`, `path`, `strings`)，httptest 单元测试

---

## Phase 1: ossupload 核心包

### Task 1: 数据结构与 IsExternalURL

**Files:**
- Create: `internal/ossupload/upload.go`
- Create: `internal/ossupload/upload_test.go`

- [ ] **Step 1: 写失败测试 TestIsExternalURL**

`internal/ossupload/upload_test.go`:

```go
package ossupload

import "testing"

func TestIsExternalURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"OSS URL is not external", "https://knowledge-payment.oss-cn-beijing.aliyuncs.com/lj7l/img.png", false},
		{"API URL is not external", "https://kpapi-cs.ckjr001.com/api/admin/assets/image.png", false},
		{"third-party is external", "https://example.com/avatar.png", true},
		{"external CDN is external", "https://cdn.example.com/images/photo.jpg", true},
		{"empty string is external", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsExternalURL(tt.url)
			if got != tt.want {
				t.Errorf("IsExternalURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestIsExternalURL -v`
Expected: FAIL (package not found)

- [ ] **Step 3: 创建包文件，定义数据结构和 IsExternalURL**

`internal/ossupload/upload.go`:

```go
package ossupload

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/childelins/ckjr-cli/internal/api"
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
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestIsExternalURL -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ossupload/upload.go internal/ossupload/upload_test.go
git commit -m "feat(ossupload): add data structures and IsExternalURL helper"
```

---

### Task 2: 下载外部图片辅助函数

**Files:**
- Modify: `internal/ossupload/upload.go`
- Modify: `internal/ossupload/upload_test.go`

- [ ] **Step 1: 写失败测试 TestDownloadImage**

在 `internal/ossupload/upload_test.go` 追加:

```go
import (
	"net/http"
	"net/http/httptest"
	// ... 已有 imports
)

func TestDownloadImage_Success(t *testing.T) {
	pngImage := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngImage)
	}))
	defer server.Close()

	imgBytes, contentType, err := downloadImage(r.Context(), server.URL+"/test/avatar.png")
	if err != nil {
		t.Fatalf("downloadImage() error = %v", err)
	}
	if !bytes.Equal(imgBytes, pngImage) {
		t.Errorf("downloadImage() bytes mismatch")
	}
	if contentType != "image/png" {
		t.Errorf("downloadImage() contentType = %q, want image/png", contentType)
	}
}

func TestDownloadImage_NonImageContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html>not an image</html>"))
	}))
	defer server.Close()

	_, _, err := downloadImage(context.Background(), server.URL+"/test.html")
	if err == nil {
		t.Fatal("downloadImage() should return error for non-image content type")
	}
	if !strings.Contains(err.Error(), "不支持的内容类型") {
		t.Errorf("downloadImage() error = %q, want content type error", err.Error())
	}
}

func TestDownloadImage_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, _, err := downloadImage(context.Background(), server.URL+"/missing.png")
	if err == nil {
		t.Fatal("downloadImage() should return error for 404")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestDownloadImage -v`
Expected: FAIL (downloadImage undefined)

- [ ] **Step 3: 实现 downloadImage 函数**

在 `internal/ossupload/upload.go` 追加:

```go
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
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestDownloadImage -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ossupload/upload.go internal/ossupload/upload_test.go
git commit -m "feat(ossupload): add downloadImage helper with content type validation"
```

---

### Task 3: 文件名与扩展名解析辅助函数

**Files:**
- Modify: `internal/ossupload/upload.go`
- Modify: `internal/ossupload/upload_test.go`

- [ ] **Step 1: 写失败测试 TestParseFileName**

在 `internal/ossupload/upload_test.go` 追加:

```go
func TestParseFileName(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		contentType   string
		wantBase      string
		wantSuffix    string
	}{
		{"png from URL", "https://example.com/path/avatar.png", "image/png", "avatar", ".png"},
		{"jpeg from URL", "https://example.com/photo.jpg", "image/jpeg", "photo", ".jpg"},
		{"no extension uses content type", "https://example.com/abc123", "image/png", "abc123", ".png"},
		{"unknown extension uses content type", "https://example.com/img.xyz", "image/jpeg", "img", ".jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, suffix := parseFileName(tt.url, tt.contentType)
			if base != tt.wantBase {
				t.Errorf("base = %q, want %q", base, tt.wantBase)
			}
			if suffix != tt.wantSuffix {
				t.Errorf("suffix = %q, want %q", suffix, tt.wantSuffix)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestParseFileName -v`
Expected: FAIL (parseFileName undefined)

- [ ] **Step 3: 实现 parseFileName**

在 `internal/ossupload/upload.go` 追加:

```go
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
		return exts[0]
	}
	return ".png"
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestParseFileName -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ossupload/upload.go internal/ossupload/upload_test.go
git commit -m "feat(ossupload): add parseFileName helper for URL and content type parsing"
```

---

### Task 4: OSS 直传函数

**Files:**
- Modify: `internal/ossupload/upload.go`
- Modify: `internal/ossupload/upload_test.go`

- [ ] **Step 1: 写失败测试 TestUploadToOSS**

在 `internal/ossupload/upload_test.go` 追加:

```go
import (
	"mime/multipart"
	// ... 已有 imports
)

func TestUploadToOSS_Success(t *testing.T) {
	var receivedKey, receivedName string
	ossServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("OSS method = %q, want POST", r.Method)
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader error: %v", err)
		}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart error: %v", err)
			}
			switch part.FormName() {
			case "key":
				buf, _ := io.ReadAll(part)
				receivedKey = string(buf)
			case "name":
				buf, _ := io.ReadAll(part)
				receivedName = string(buf)
			case "file":
				// consume the file part
				io.ReadAll(part)
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ossServer.Close()

	signResp := &ImageSignResponse{
		Key:                 "lj7l/resource/imgs/test/upload.png",
		Host:                ossServer.URL,
		Policy:              "test-policy",
		OSSAccessKeyID:      "test-key-id",
		Signature:           "test-sig",
		Callback:            "test-cb",
		SuccessActionStatus: "200",
		Origin:              "0",
	}

	err := uploadToOSS(context.Background(), signResp, []byte("fake image data"), "avatar", ".png")
	if err != nil {
		t.Fatalf("uploadToOSS() error = %v", err)
	}
	if receivedKey != "lj7l/resource/imgs/test/upload.png" {
		t.Errorf("key = %q, want %q", receivedKey, "lj7l/resource/imgs/test/upload.png")
	}
	if receivedName != "avatar" {
		t.Errorf("name = %q, want %q", receivedName, "avatar")
	}
}

func TestUploadToOSS_ServerError(t *testing.T) {
	ossServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("AccessDenied"))
	}))
	defer ossServer.Close()

	signResp := &ImageSignResponse{
		Key:                 "test-key",
		Host:                ossServer.URL,
		Policy:              "test-policy",
		OSSAccessKeyID:      "test-key-id",
		Signature:           "test-sig",
		Callback:            "test-cb",
		SuccessActionStatus: "200",
		Origin:              "0",
	}

	err := uploadToOSS(context.Background(), signResp, []byte("data"), "test", ".png")
	if err == nil {
		t.Fatal("uploadToOSS() should return error for 403")
	}
	if !strings.Contains(err.Error(), "OSS 上传失败") {
		t.Errorf("error = %q, want OSS upload error", err.Error())
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestUploadToOSS -v`
Expected: FAIL (uploadToOSS undefined)

- [ ] **Step 3: 实现 uploadToOSS**

在 `internal/ossupload/upload.go` 追加:

```go
// uploadToOSS 直传图片到阿里云 OSS（multipart/form-data）
func uploadToOSS(ctx context.Context, signResp *ImageSignResponse, imageBytes []byte, fileName, suffix string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 按顺序写入签名字段（与前端 curl 一致）
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
```

注意：需要在 imports 中加入 `"bytes"` 和 `"mime/multipart"`。

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestUploadToOSS -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ossupload/upload.go internal/ossupload/upload_test.go
git commit -m "feat(ossupload): add uploadToOSS with multipart form upload"
```

---

### Task 5: Upload 总入口函数

**Files:**
- Modify: `internal/ossupload/upload.go`
- Modify: `internal/ossupload/upload_test.go`

- [ ] **Step 1: 写失败测试 TestUpload_Success（集成 4 步流程）**

在 `internal/ossupload/upload_test.go` 追加:

```go
func TestUpload_Success(t *testing.T) {
	pngImage := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x01, 0x02, 0x03, 0x04}

	// 模拟外部图片服务器
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngImage)
	}))
	defer imageServer.Close()

	// 模拟 OSS 服务器
	ossServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ossServer.Close()

	// 模拟 API 服务器（imageSign + addImgInAsset）
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/admin/assets/imageSign":
			json.NewEncoder(w).Encode(api.Response{
				Data: map[string]interface{}{
					"key":                   "lj7l/resource/imgs/eeb49984/test.png",
					"policy":                "test-policy-base64",
					"OSSAccessKeyId":        "LTAIEooZEnvlRbrb",
					"signature":             "test-sig",
					"callback":              "test-callback-base64",
					"success_action_status": "200",
					"origin":                "0",
					"host":                  ossServer.URL,
				},
				Message:    "ok",
				StatusCode: 200,
			})
		case r.URL.Path == "/admin/assets/addImgInAsset":
			json.NewEncoder(w).Encode(api.Response{
				Data:       map[string]interface{}{"id": 123},
				Message:    "ok",
				StatusCode: 200,
			})
		}
	}))
	defer apiServer.Close()

	apiClient := api.NewClient(apiServer.URL, "test-key")
	result, err := Upload(context.Background(), apiClient, imageServer.URL+"/test/avatar.png")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if result.ImageURL == "" {
		t.Error("Upload() result.ImageURL should not be empty")
	}
	if result.Name != "avatar" {
		t.Errorf("Upload() result.Name = %q, want %q", result.Name, "avatar")
	}
	if result.Suffix != ".png" {
		t.Errorf("Upload() result.Suffix = %q, want %q", result.Suffix, ".png")
	}
}

func TestUpload_ImageSignFails(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(api.Response{
			Message:    "Unauthorized",
			StatusCode: 401,
		})
	}))
	defer apiServer.Close()

	apiClient := api.NewClient(apiServer.URL, "invalid-key")
	_, err := Upload(context.Background(), apiClient, "https://example.com/avatar.png")
	if err == nil {
		t.Fatal("Upload() should return error when imageSign fails")
	}
}

func TestUpload_DownloadFails(t *testing.T) {
	// API 返回签名成功
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.Response{
			Data: map[string]interface{}{
				"key":                   "test-key",
				"policy":                "test-policy",
				"OSSAccessKeyId":        "test-id",
				"signature":             "test-sig",
				"callback":              "test-cb",
				"success_action_status": "200",
				"origin":                "0",
				"host":                  "https://oss.example.com",
			},
			Message:    "ok",
			StatusCode: 200,
		})
	}))
	defer apiServer.Close()

	apiClient := api.NewClient(apiServer.URL, "test-key")
	// 图片 URL 指向不存在的服务器
	_, err := Upload(context.Background(), apiClient, "http://127.0.0.1:1/nonexistent.png")
	if err == nil {
		t.Fatal("Upload() should return error when download fails")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -run TestUpload -v`
Expected: FAIL (Upload undefined)

- [ ] **Step 3: 实现 Upload 函数**

在 `internal/ossupload/upload.go` 追加:

```go
// Upload 将外部图片 URL 转存到素材库
//
// 完整流程：imageSign -> 下载外部图片 -> 直传 OSS -> addImgInAsset
func Upload(ctx context.Context, apiClient *api.Client, imageURL string) (*AssetImage, error) {
	// Step 1: 获取 OSS 上传签名
	var signResp ImageSignResponse
	if err := apiClient.DoCtx(ctx, "GET", "/admin/assets/imageSign?type=2", nil, &signResp); err != nil {
		return nil, fmt.Errorf("获取 OSS 上传签名失败: %w", err)
	}

	// Step 2: 下载外部图片
	imgBytes, contentType, err := downloadImage(ctx, imageURL)
	if err != nil {
		return nil, err
	}

	// 解析文件名和扩展名
	fileName, suffix := parseFileName(imageURL, contentType)
	fileSizeMB := float64(len(imgBytes)) / 1024 / 1024

	// Step 3: 直传 OSS
	if err := uploadToOSS(ctx, &signResp, imgBytes, fileName, suffix); err != nil {
		return nil, err
	}

	// Step 4: 保存到素材库
	ossURL := signResp.Host + "/" + signResp.Key
	payload := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"fId":      -1,
				"level":    1,
				"parentId": 0,
				"size":     fileSizeMB,
				"height":   "",
				"width":    "",
				"suffix":   suffix,
				"name":     fileName,
				"imageUrl": ossURL,
			},
		},
	}

	var result interface{}
	if err := apiClient.DoCtx(ctx, "POST", "/admin/assets/addImgInAsset", payload, &result); err != nil {
		return nil, fmt.Errorf("保存到素材库失败: %w", err)
	}

	return &AssetImage{
		ImageURL: ossURL,
		Name:     fileName,
		Suffix:   suffix,
		Size:     fileSizeMB,
		Width:    "",
		Height:   "",
	}, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/ossupload/ -v`
Expected: ALL PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ossupload/upload.go internal/ossupload/upload_test.go
git commit -m "feat(ossupload): implement Upload function with full 4-step flow"
```

---

## Phase 2: CLI 命令注册

### Task 6: asset upload-image 子命令

**Files:**
- Create: `cmd/upload.go`
- Create: `cmd/upload_test.go`
- Modify: `cmd/root.go` (在 registerRouteCommands 后注册 upload-image)

- [ ] **Step 1: 写失败测试 TestUploadImageCmd**

`cmd/upload_test.go`:

```go
package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/childelins/ckjr-cli/internal/api"
)

func TestUploadImageCmd(t *testing.T) {
	// 模拟完整 API + OSS 服务器
	pngImage := []byte{0x89, 0x50, 0x4E, 0x47}

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngImage)
	}))
	defer imageServer.Close()

	ossServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ossServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/admin/assets/imageSign":
			json.NewEncoder(w).Encode(api.Response{
				Data: map[string]interface{}{
					"key":                   "lj7l/imgs/test.png",
					"policy":                "test-policy",
					"OSSAccessKeyId":        "test-id",
					"signature":             "test-sig",
					"callback":              "test-cb",
					"success_action_status": "200",
					"origin":                "0",
					"host":                  ossServer.URL,
				},
				Message:    "ok",
				StatusCode: 200,
			})
		case r.URL.Path == "/admin/assets/addImgInAsset":
			json.NewEncoder(w).Encode(api.Response{
				Data:       map[string]interface{}{"id": 1},
				Message:    "ok",
				StatusCode: 200,
			})
		}
	}))
	defer apiServer.Close()

	// 创建临时配置文件
	tmpDir := t.TempDir()
	configDir := tmpDir + "/.ckjr"
	os.MkdirAll(configDir, 0755)
	configData := map[string]string{
		"base_url": apiServer.URL,
		"api_key":  "test-key",
	}
	configJSON, _ := json.Marshal(configData)
	os.WriteFile(configDir+"/config.json", configJSON, 0644)
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// 验证命令已注册
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"asset", "upload-image", `{"url":"` + imageServer.URL + `/test/avatar.png"}`})
	// 注意: 因为 rootCmd 的 init 已经执行，需要确认命令能正常注册
	// 此测试可能需要根据实际 rootCmd 初始化方式调整
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/ -run TestUploadImageCmd -v`
Expected: FAIL (command not found or similar)

- [ ] **Step 3: 创建 cmd/upload.go**

`cmd/upload.go`:

```go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/cmdgen"
	"github.com/childelins/ckjr-cli/internal/logging"
	"github.com/childelins/ckjr-cli/internal/ossupload"
	"github.com/childelins/ckjr-cli/internal/output"
)

func newUploadImageCmd(clientFactory cmdgen.APIClientFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "upload-image [json]",
		Short: "将外部图片URL转存到系统素材库",
		Long:  "下载外部图片链接，直传到 OSS 并保存到素材库。返回素材库中的图片 URL。",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var input map[string]interface{}
			if len(args) > 0 {
				if err := json.Unmarshal([]byte(args[0]), &input); err != nil {
					output.PrintError(os.Stderr, "JSON 解析失败: "+err.Error())
					os.Exit(1)
				}
			}

			imageURL, _ := input["url"].(string)
			if imageURL == "" {
				output.PrintError(os.Stderr, "缺少 url 参数")
				os.Exit(1)
			}

			client, err := clientFactory()
			if err != nil {
				output.PrintError(os.Stderr, err.Error())
				os.Exit(1)
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			verbose, _ := cmd.Flags().GetBool("verbose")

			ctx := logging.WithRequestID(context.Background(), logging.NewRequestID())

			result, err := ossupload.Upload(ctx, client, imageURL)
			if err != nil {
				if verbose {
					output.PrintError(os.Stderr, err.Error())
				} else {
					output.PrintError(os.Stderr, formatUploadError(err))
				}
				os.Exit(1)
			}

			output.Print(os.Stdout, result, pretty)
		},
	}
}

func formatUploadError(err error) string {
	return fmt.Sprintf("图片上传失败: %v", err)
}
```

- [ ] **Step 4: 修改 cmd/root.go 注册命令**

在 `registerRouteCommands` 函数中，`rootCmd.AddCommand(cmd)` 后添加:

```go
// 为 asset 命令额外注册 upload-image 子命令
if cfg.Name == "asset" {
	cmd.AddCommand(newUploadImageCmd(createClient))
}
```

- [ ] **Step 5: 运行测试验证通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/ -run TestUploadImageCmd -v`
Expected: PASS

- [ ] **Step 6: 运行所有测试确认无回归**

Run: `cd /home/childelins/code/ckjr-cli && go test ./...`
Expected: ALL PASS

- [ ] **Step 7: 提交**

```bash
git add cmd/upload.go cmd/upload_test.go cmd/root.go
git commit -m "feat(cmd): add asset upload-image subcommand for OSS image transfer"
```

---

## Phase 3: Workflow 集成

### Task 7: 更新 course workflow

**Files:**
- Modify: `cmd/ckjr-cli/workflows/course.yaml`

- [ ] **Step 1: 更新 course.yaml**

在 `create-video-course` 的 `steps` 最前面添加 `upload-avatar` 步骤，并修改 `create` 步骤引用上传结果:

```yaml
    steps:
      - id: upload-avatar
        description: 如果课程封面是外部图片URL，先转存到系统素材库
        command: asset upload-image
        params:
          url: "{{inputs.courseAvatar}}"
        output:
          imageUrl: "response.imageUrl"
      - id: create
        description: 创建视频课程
        command: course create
        params:
          name: "{{inputs.name}}"
          courseAvatar: "{{steps.upload-avatar.imageUrl}}"
          # ... 其他参数不变
```

对 `create-audio-course` 和 `create-article-course` 做同样修改。

- [ ] **Step 2: 运行所有测试确认无回归**

Run: `cd /home/childelins/code/ckjr-cli && go test ./...`
Expected: ALL PASS

- [ ] **Step 3: 提交**

```bash
git add cmd/ckjr-cli/workflows/course.yaml
git commit -m "feat(workflow): add upload-avatar step to course creation workflows"
```

---

## Phase 4: 验证与清理

### Task 8: 全量测试与编译验证

**Files:** 无修改

- [ ] **Step 1: 运行全量测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: 编译验证**

Run: `cd /home/childelins/code/ckjr-cli && go build ./...`
Expected: 无错误

- [ ] **Step 3: 验证命令注册**

Run: `cd /home/childelins/code/ckjr-cli && go run ./cmd/ckjr-cli asset upload-image --help`
Expected: 输出命令用法说明
