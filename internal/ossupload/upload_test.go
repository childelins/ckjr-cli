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
