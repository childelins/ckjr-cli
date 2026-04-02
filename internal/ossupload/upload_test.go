package ossupload

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/childelins/ckjr-cli/internal/api"
)

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

func TestDownloadImage_Success(t *testing.T) {
	pngImage := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngImage)
	}))
	defer server.Close()

	imgBytes, contentType, err := downloadImage(context.Background(), server.URL+"/test/avatar.png")
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

func TestParseFileName(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		contentType string
		wantBase    string
		wantSuffix  string
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
