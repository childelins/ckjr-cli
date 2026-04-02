package ossupload

import (
	"bytes"
	"context"
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
