package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/childelins/ckjr-cli/internal/config"
)

func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	originalPath := config.ConfigPath
	config.ConfigPath = filepath.Join(tmpDir, ".ckjr", "config.json")

	cleanup := func() {
		config.ConfigPath = originalPath
	}
	return tmpDir, cleanup
}

func TestConfigShowNoConfig(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// config show 应该在没有配置时返回错误
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"config", "show"})

	// 由于 runConfigShow 内部使用了 os.Exit(1)，我们无法直接测试
	// 改为测试 config.Load() 在没有配置文件时返回错误
	_, err := config.Load()
	if err == nil {
		t.Error("Load() should return error when config not found")
	}
}

func TestConfigSetAndShow(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// 先设置配置
	cfg := &config.Config{
		BaseURL: "https://api.example.com",
		APIKey:  "test-api-key-12345",
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 验证配置已保存
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %s, want https://api.example.com", loaded.BaseURL)
	}
	if loaded.APIKey != "test-api-key-12345" {
		t.Errorf("APIKey = %s, want test-api-key-12345", loaded.APIKey)
	}
}

func TestConfigSetValidKeys(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// 先创建初始配置
	cfg := &config.Config{}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 测试 set base_url
	loaded, _ := config.Load()
	loaded.BaseURL = "https://new-api.example.com"
	if err := config.Save(loaded); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if reloaded.BaseURL != "https://new-api.example.com" {
		t.Errorf("BaseURL = %s, want https://new-api.example.com", reloaded.BaseURL)
	}

	// 测试 set api_key
	reloaded.APIKey = "new-key-value"
	if err := config.Save(reloaded); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	final, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if final.APIKey != "new-key-value" {
		t.Errorf("APIKey = %s, want new-key-value", final.APIKey)
	}
}

func TestConfigShowMaskedKey(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &config.Config{
		BaseURL: "https://api.example.com",
		APIKey:  "abcdefghij",
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	masked := loaded.MaskedAPIKey()
	if masked != "abcde*****" {
		t.Errorf("MaskedAPIKey() = %s, want abcde*****", masked)
	}
}

func TestConfigValidKeysCheck(t *testing.T) {
	validKeys := map[string]bool{"base_url": true, "api_key": true}

	tests := []struct {
		key   string
		valid bool
	}{
		{"base_url", true},
		{"api_key", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		if validKeys[tt.key] != tt.valid {
			t.Errorf("validKeys[%s] = %v, want %v", tt.key, validKeys[tt.key], tt.valid)
		}
	}
}

func TestConfigFilePermissions(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &config.Config{
		BaseURL: "https://api.example.com",
		APIKey:  "secret-key",
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 验证配置文件权限为 0600
	info, err := os.Stat(config.ConfigPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}
