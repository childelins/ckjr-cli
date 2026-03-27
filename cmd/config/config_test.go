package config

import (
	"os"
	"path/filepath"
	"testing"

	internalconfig "github.com/childelins/ckjr-cli/internal/config"
)

func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	originalPath := internalconfig.ConfigPath
	internalconfig.ConfigPath = filepath.Join(tmpDir, ".ckjr", "config.json")
	cleanup := func() {
		internalconfig.ConfigPath = originalPath
	}
	return tmpDir, cleanup
}

func TestConfigShowNoConfig(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()
	_, err := internalconfig.Load()
	if err == nil {
		t.Error("Load() should return error when config not found")
	}
}

func TestConfigSetAndShow(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()
	cfg := &internalconfig.Config{
		BaseURL: "https://api.example.com",
		APIKey:  "test-api-key-12345",
	}
	if err := internalconfig.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := internalconfig.Load()
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
	cfg := &internalconfig.Config{}
	if err := internalconfig.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, _ := internalconfig.Load()
	loaded.BaseURL = "https://new-api.example.com"
	if err := internalconfig.Save(loaded); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	reloaded, err := internalconfig.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if reloaded.BaseURL != "https://new-api.example.com" {
		t.Errorf("BaseURL = %s, want https://new-api.example.com", reloaded.BaseURL)
	}
	reloaded.APIKey = "new-key-value"
	if err := internalconfig.Save(reloaded); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	final, err := internalconfig.Load()
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
	cfg := &internalconfig.Config{
		BaseURL: "https://api.example.com",
		APIKey:  "abcdefghij",
	}
	if err := internalconfig.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := internalconfig.Load()
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
	cfg := &internalconfig.Config{
		BaseURL: "https://api.example.com",
		APIKey:  "secret-key",
	}
	if err := internalconfig.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Stat(internalconfig.ConfigPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}
