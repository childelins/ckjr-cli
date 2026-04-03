package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".ckjr", "config.json")
	if ConfigPath != expected {
		t.Errorf("ConfigPath = %s, want %s", ConfigPath, expected)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	// 保存原始 ConfigPath 并恢复
	originalPath := ConfigPath
	defer func() { ConfigPath = originalPath }()

	// 设置临时配置路径
	tmpDir := t.TempDir()
	ConfigPath = filepath.Join(tmpDir, ".ckjr", "config.json")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when config not found")
	}
}

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	ConfigPath = filepath.Join(tmpDir, ".ckjr", "config.json")

	cfg := &Config{
		BaseURL: "https://api.example.com",
		APIKey:  "test-key",
	}

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.BaseURL != cfg.BaseURL {
		t.Errorf("BaseURL = %s, want %s", loaded.BaseURL, cfg.BaseURL)
	}
	if loaded.APIKey != cfg.APIKey {
		t.Errorf("APIKey = %s, want %s", loaded.APIKey, cfg.APIKey)
	}
}

func TestMaskedAPIKey(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"", ""},
		{"abc", "***"},
		{"abcdef", "abc***"},
		{"abcdefghij", "abcde*****"},
	}

	for _, tt := range tests {
		cfg := &Config{APIKey: tt.key}
		if got := cfg.MaskedAPIKey(); got != tt.expected {
			t.Errorf("MaskedAPIKey(%s) = %s, want %s", tt.key, got, tt.expected)
		}
	}
}

func TestDefaultBaseURL_Development(t *testing.T) {
	SetEnvironment("development")
	got := DefaultBaseURL()
	want := "https://kpapi-cs.ckjr001.com/api"
	if got != want {
		t.Errorf("DefaultBaseURL() = %s, want %s", got, want)
	}
}

func TestDefaultBaseURL_Production(t *testing.T) {
	SetEnvironment("production")
	got := DefaultBaseURL()
	want := "http://kpapiop.ckjr001.com/api"
	if got != want {
		t.Errorf("DefaultBaseURL() = %s, want %s", got, want)
	}
}

func TestDefaultBaseURL_UnknownFallback(t *testing.T) {
	SetEnvironment("unknown")
	got := DefaultBaseURL()
	want := "https://kpapi-cs.ckjr001.com/api"
	if got != want {
		t.Errorf("DefaultBaseURL() = %s, want %s", got, want)
	}
}

func TestResolveBaseURL_WithBaseURL(t *testing.T) {
	cfg := &Config{BaseURL: "https://custom.example.com/api"}
	got := cfg.ResolveBaseURL()
	if got != cfg.BaseURL {
		t.Errorf("ResolveBaseURL() = %s, want %s", got, cfg.BaseURL)
	}
}

func TestResolveBaseURL_EmptyBaseURL(t *testing.T) {
	SetEnvironment("production")
	cfg := &Config{BaseURL: ""}
	got := cfg.ResolveBaseURL()
	want := "http://kpapiop.ckjr001.com/api"
	if got != want {
		t.Errorf("ResolveBaseURL() = %s, want %s", got, want)
	}
}
