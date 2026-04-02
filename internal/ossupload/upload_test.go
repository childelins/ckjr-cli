package ossupload

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
