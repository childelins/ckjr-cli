package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
		wantErr bool
	}{
		{"相等", "0.1.0", "0.1.0", 0, false},
		{"小版本更新", "0.1.0", "0.2.0", -1, false},
		{"大版本更新", "0.9.0", "1.0.0", -1, false},
		{"当前更新", "0.2.0", "0.1.0", 1, false},
		{"带 v 前缀", "v0.1.0", "v0.2.0", -1, false},
		{"混合前缀", "0.1.0", "v0.2.0", -1, false},
		{"不同段数相等", "0.1.0", "0.1", 0, false},
		{"不同段数不等", "0.1", "0.1.1", -1, false},
		{"空字符串", "", "0.1.0", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareVersions(tt.current, tt.latest)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersions(%q, %q) error = %v, wantErr %v", tt.current, tt.latest, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
