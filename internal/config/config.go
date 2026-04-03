package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Config 存储 CLI 配置
type Config struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

// ConfigPath 返回配置文件路径
var ConfigPath string

func init() {
	home, _ := os.UserHomeDir()
	ConfigPath = filepath.Join(home, ".ckjr", "config.json")
}

// envBaseURLs 各环境对应的默认 base_url
var envBaseURLs = map[string]string{
	"development": "https://kpapi-cs.ckjr001.com/api",
	"production":  "http://kpapiop.ckjr001.com/api",
}

// environment 由 main 包通过 SetEnvironment 注入
var environment string

// SetEnvironment 注入编译时的 Environment 值
func SetEnvironment(env string) {
	environment = env
}

// DefaultBaseURL 根据当前 environment 返回对应的默认 base_url
func DefaultBaseURL() string {
	if u, ok := envBaseURLs[environment]; ok {
		return u
	}
	return envBaseURLs["development"]
}

// ResolveBaseURL 返回最终使用的 base_url
// 优先级: config.json 中的 base_url > DefaultBaseURL()
func (c *Config) ResolveBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL()
}

// ErrConfigNotFound 配置文件不存在
var ErrConfigNotFound = errors.New("配置文件不存在")

// Load 从文件加载配置
func Load() (*Config, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save 保存配置到文件
func Save(cfg *Config) error {
	// 确保目录存在
	dir := filepath.Dir(ConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath, data, 0600)
}

// MaskedAPIKey 返回脱敏的 API Key
func (c *Config) MaskedAPIKey() string {
	if c.APIKey == "" {
		return ""
	}
	n := len(c.APIKey)
	if n <= 3 {
		return "***"
	}
	// 显示前 n/2 位（向下取整到3或5），其余用*替代
	visible := n / 2
	if visible < 3 {
		visible = 3
	} else if visible > 5 {
		visible = 5
	}
	masked := n - visible
	if masked < 3 {
		masked = 3
	} else if masked > 5 {
		masked = 5
	}
	return c.APIKey[:visible] + strings.Repeat("*", masked)
}
