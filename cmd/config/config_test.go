package config

import (
	"bytes"
	"encoding/json"
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

func TestConfigShowEmptyBaseURL(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// 设置 environment 使 DefaultBaseURL 返回已知值
	internalconfig.SetEnvironment("development")

	// 保存一个 base_url 为空的配置
	cfg := &internalconfig.Config{
		BaseURL: "",
		APIKey:  "test-api-key-12345",
	}
	if err := internalconfig.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 通过 cobra 命令执行 config show，捕获输出
	cmd := NewCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"show"})

	// 重定向 stdout 以捕获 output.Print 的输出
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var captured bytes.Buffer
	captured.ReadFrom(r)

	// 验证输出中 base_url 为环境默认值，不是空字符串
	var result map[string]string
	if err := json.Unmarshal(captured.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v, output: %s", err, captured.String())
	}
	expected := internalconfig.DefaultBaseURL()
	if result["base_url"] != expected {
		t.Errorf("base_url = %q, want %q", result["base_url"], expected)
	}
}

func TestConfigInitSavesEmptyBaseURL(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// 模拟用户输入：只输入 api_key
	input := "test-api-key-12345\n"
	r, w, _ := os.Pipe()
	w.WriteString(input)
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// 重定向 stdout 避免输出到终端
	oldStdout := os.Stdout
	_, wOut, _ := os.Pipe()
	os.Stdout = wOut
	defer func() {
		wOut.Close()
		os.Stdout = oldStdout
	}()

	cmd := NewCommand()
	cmd.SetArgs([]string{"init"})
	cmd.Execute()

	wOut.Close()
	os.Stdout = oldStdout

	// 验证保存后的配置
	loaded, err := internalconfig.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.BaseURL != "" {
		t.Errorf("BaseURL = %q, want empty string", loaded.BaseURL)
	}
	if loaded.APIKey != "test-api-key-12345" {
		t.Errorf("APIKey = %q, want test-api-key-12345", loaded.APIKey)
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
