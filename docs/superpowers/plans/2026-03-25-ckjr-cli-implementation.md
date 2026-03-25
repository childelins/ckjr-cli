# ckjr-cli 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建基于 Go 的 CLI 工具，作为 Claude Code Skills 与公司 SaaS 平台之间的桥梁。

**Architecture:**
- CLI 框架使用 cobra，命令通过 YAML 路由配置自动生成
- 配置存储在 `~/.ckjr/config.json`，API 请求统一使用 POST + JSON body
- 输出默认 JSON，支持 `--pretty` 格式化

**Tech Stack:** Go 1.21+, cobra, viper, go-yaml, embed

---

## 文件结构

```
ckjr-cli/
├── cmd/
│   ├── root.go              # 根命令，加载配置，注册模块
│   └── config.go            # ckjr config 子命令（手动实现）
├── routes/
│   └── agent.yaml           # 智能体路由映射（embed 打包）
├── internal/
│   ├── api/
│   │   └── client.go        # HTTP 客户端，统一认证和请求
│   ├── config/
│   │   └── config.go        # 配置加载（~/.ckjr/config.json）
│   ├── output/
│   │   └── output.go        # JSON 输出格式化
│   ├── router/
│   │   └── router.go        # 读取 YAML 路由，提供路径查询
│   └── cmdgen/
│       └── cmdgen.go        # 根据 YAML 自动生成 cobra 子命令
├── go.mod
├── go.sum
└── main.go
```

---

### Task 1: 项目初始化

**Files:**
- Create: `go.mod`
- Create: `main.go`（骨架）

- [ ] **Step 1: 初始化 Go 模块**

```bash
cd /home/childelins/code/ckjr-cli
go mod init github.com/childelins/ckjr-cli
```

- [ ] **Step 2: 创建 main.go 骨架**

```go
package main

import "fmt"

func main() {
	fmt.Println("ckjr-cli")
}
```

- [ ] **Step 3: 验证编译**

```bash
go build -o ckjr .
./ckjr
# Expected: ckjr-cli
```

- [ ] **Step 4: 提交**

```bash
git add go.mod main.go
git commit -m "chore: initialize Go module"
```

---

### Task 2: 配置模块 (internal/config)

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: 编写配置加载测试**

```go
// internal/config/config_test.go
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
	// 设置临时 HOME 目录
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when config not found")
	}
}

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

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
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/config/... -v
# Expected: FAIL (config.go not exists)
```

- [ ] **Step 3: 实现配置模块**

```go
// internal/config/config.go
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	if len(c.APIKey) <= 3 {
		return "***"
	}
	if len(c.APIKey) <= 5 {
		return c.APIKey[:3] + "***"
	}
	return c.APIKey[:5] + "*****"
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/config/... -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/config/
git commit -m "feat(config): add config load/save with masking"
```

---

### Task 3: 输出模块 (internal/output)

**Files:**
- Create: `internal/output/output.go`
- Create: `internal/output/output_test.go`

- [ ] **Step 1: 编写输出测试**

```go
// internal/output/output_test.go
package output

import (
	"bytes"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		pretty   bool
		expected string
	}{
		{
			name:     "compact object",
			data:     map[string]string{"key": "value"},
			pretty:   false,
			expected: `{"key":"value"}`,
		},
		{
			name:     "pretty object",
			data:     map[string]string{"key": "value"},
			pretty:   true,
			expected: "{\n  \"key\": \"value\"\n}",
		},
		{
			name:     "nil data",
			data:     nil,
			pretty:   false,
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Print(&buf, tt.data, tt.pretty)
			if buf.String() != tt.expected+"\n" {
				t.Errorf("Print() = %q, want %q", buf.String(), tt.expected+"\n")
			}
		})
	}
}

func TestPrintError(t *testing.T) {
	var buf bytes.Buffer
	PrintError(&buf, "test error")
	expected := `{"error":"test error"}` + "\n"
	if buf.String() != expected {
		t.Errorf("PrintError() = %q, want %q", buf.String(), expected)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/output/... -v
# Expected: FAIL
```

- [ ] **Step 3: 实现输出模块**

```go
// internal/output/output.go
package output

import (
	"encoding/json"
	"io"
)

// Printer 控制输出格式
type Printer struct {
	Pretty bool
	Writer io.Writer
}

// Print 输出 JSON 数据
func Print(w io.Writer, data interface{}, pretty bool) {
	var bytes []byte
	var err error
	if pretty {
		bytes, err = json.MarshalIndent(data, "", "  ")
	} else {
		bytes, err = json.Marshal(data)
	}
	if err != nil {
		PrintError(w, err.Error())
		return
	}
	w.Write(bytes)
	w.Write([]byte("\n"))
}

// PrintError 输出错误信息
func PrintError(w io.Writer, msg string) {
	data, _ := json.Marshal(map[string]string{"error": msg})
	w.Write(data)
	w.Write([]byte("\n"))
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/output/... -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/output/
git commit -m "feat(output): add JSON output with pretty option"
```

---

### Task 4: 路由模块 (internal/router)

**Files:**
- Create: `internal/router/router.go`
- Create: `internal/router/router_test.go`

- [ ] **Step 1: 编写路由测试**

```go
// internal/router/router_test.go
package router

import (
	"testing"
)

func TestParseRouteConfig(t *testing.T) {
	yamlContent := `
resource: agent
description: AI智能体的增删改查
routes:
  list:
    method: POST
    path: /admin/aiCreationCenter/listApp
    description: 获取智能体列表
    template:
      page:
        description: 页码
        required: false
        default: 1
  get:
    method: POST
    path: /admin/aiCreationCenter/getAppInfo
    description: 获取智能体详情
    template:
      aikbId:
        description: 智能体ID
        required: true
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Resource != "agent" {
		t.Errorf("Resource = %s, want agent", cfg.Resource)
	}

	if len(cfg.Routes) != 2 {
		t.Fatalf("Routes count = %d, want 2", len(cfg.Routes))
	}

	listRoute, ok := cfg.Routes["list"]
	if !ok {
		t.Fatal("list route not found")
	}
	if listRoute.Method != "POST" {
		t.Errorf("list.Method = %s, want POST", listRoute.Method)
	}
	if listRoute.Path != "/admin/aiCreationCenter/listApp" {
		t.Errorf("list.Path = %s", listRoute.Path)
	}
}

func TestGetTemplate(t *testing.T) {
	cfg := &RouteConfig{
		Resource: "agent",
		Routes: map[string]Route{
			"create": {
				Method:      "POST",
				Path:        "/create",
				Description: "创建",
				Template: map[string]Field{
					"name": {
						Description: "名称",
						Required:    true,
					},
					"page": {
						Description: "页码",
						Required:    false,
						Default:     1,
					},
				},
			},
		},
	}

	tmpl := cfg.GetTemplate("create")
	if len(tmpl) != 2 {
		t.Fatalf("GetTemplate() count = %d, want 2", len(tmpl))
	}

	if tmpl["name"].Default != nil {
		t.Error("name should not have default")
	}
	if tmpl["page"].Default == nil {
		t.Error("page should have default")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/router/... -v
# Expected: FAIL
```

- [ ] **Step 3: 实现路由模块**

```go
// internal/router/router.go
package router

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Field 定义模板字段
type Field struct {
	Description string      `yaml:"description"`
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default"`
}

// Route 定义单个路由
type Route struct {
	Method      string           `yaml:"method"`
	Path        string           `yaml:"path"`
	Description string           `yaml:"description"`
	Template    map[string]Field `yaml:"template"`
}

// RouteConfig 路由配置
type RouteConfig struct {
	Resource    string           `yaml:"resource"`
	Description string           `yaml:"description"`
	Routes      map[string]Route `yaml:"routes"`
}

// Parse 解析 YAML 路由配置
func Parse(data []byte) (*RouteConfig, error) {
	var cfg RouteConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析路由配置失败: %w", err)
	}
	return &cfg, nil
}

// GetRoute 获取指定路由
func (c *RouteConfig) GetRoute(name string) (Route, bool) {
	route, ok := c.Routes[name]
	return route, ok
}

// GetTemplate 获取路由模板
func (c *RouteConfig) GetTemplate(routeName string) map[string]Field {
	route, ok := c.GetRoute(routeName)
	if !ok {
		return nil
	}
	return route.Template
}

// RequiredFields 获取必填字段列表
func (c *RouteConfig) RequiredFields(routeName string) []string {
	tmpl := c.GetTemplate(routeName)
	if tmpl == nil {
		return nil
	}

	var required []string
	for name, field := range tmpl {
		if field.Required {
			required = append(required, name)
		}
	}
	return required
}
```

- [ ] **Step 4: 安装依赖并运行测试**

```bash
go get gopkg.in/yaml.v3
go test ./internal/router/... -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/router/
git commit -m "feat(router): add YAML route config parser"
```

---

### Task 5: API 客户端模块 (internal/api)

**Files:**
- Create: `internal/api/client.go`
- Create: `internal/api/client_test.go`

- [ ] **Step 1: 编写 API 客户端测试**

```go
// internal/api/client_test.go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("Missing Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Missing Content-Type header")
		}

		// 返回模拟响应
		resp := Response{
			Data:       map[string]string{"id": "123"},
			Message:    "success",
			StatusCode: 200,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")

	var result map[string]string
	err := client.Do("POST", "/test", nil, &result)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("result = %v, want id=123", result)
	}
}

func TestClientUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := Response{
			Message:    "Unauthorized",
			StatusCode: 401,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "invalid-key")

	var result interface{}
	err := client.Do("POST", "/test", nil, &result)
	if err == nil {
		t.Fatal("Do() should return error for 401")
	}

	if !IsUnauthorized(err) {
		t.Errorf("error should be ErrUnauthorized, got %v", err)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/api/... -v
# Expected: FAIL
```

- [ ] **Step 3: 实现 API 客户端**

```go
// internal/api/client.go
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrUnauthorized API Key 无效或过期
var ErrUnauthorized = errors.New("api_key 已过期，请重新登录获取")

// ErrValidation 参数校验失败
var ErrValidation = errors.New("参数校验失败")

// Response Dingo API 响应格式
type Response struct {
	Data       interface{}            `json:"data"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"status_code"`
	Errors     map[string]interface{} `json:"errors,omitempty"`
}

// ValidationError 验证错误详情
type ValidationError struct {
	Message string
	Errors  map[string]interface{}
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Client API 客户端
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient 创建新的 API 客户端
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{},
	}
}

// Do 执行 API 请求
func (c *Client) Do(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var apiResp Response
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 处理错误状态
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return &ValidationError{
			Message: apiResp.Message,
			Errors:  apiResp.Errors,
		}
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiResp.Message)
	}

	// 解析 data 到 result
	if result != nil && apiResp.Data != nil {
		data, err := json.Marshal(apiResp.Data)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, result); err != nil {
			return err
		}
	}

	return nil
}

// IsUnauthorized 检查是否是认证错误
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsValidationError 检查是否是验证错误
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// GetValidationErrors 获取验证错误详情
func GetValidationErrors(err error) map[string]interface{} {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve.Errors
	}
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/api/... -v
# Expected: PASS
```

- [ ] **Step 5: 提交**

```bash
git add internal/api/
git commit -m "feat(api): add HTTP client with auth and error handling"
```

---

### Task 6: 命令生成模块 (internal/cmdgen)

**Files:**
- Create: `internal/cmdgen/cmdgen.go`
- Create: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 编写命令生成测试**

```go
// internal/cmdgen/cmdgen_test.go
package cmdgen

import (
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
)

func TestBuildCommand(t *testing.T) {
	cfg := &router.RouteConfig{
		Resource:    "agent",
		Description: "AI智能体管理",
		Routes: map[string]router.Route{
			"list": {
				Method:      "POST",
				Path:        "/admin/list",
				Description: "获取列表",
				Template: map[string]router.Field{
					"page": {
						Description: "页码",
						Required:    false,
						Default:     1,
					},
				},
			},
			"get": {
				Method:      "POST",
				Path:        "/admin/get",
				Description: "获取详情",
				Template: map[string]router.Field{
					"id": {
						Description: "ID",
						Required:    true,
					},
				},
			},
		},
	}

	cmd := BuildCommand(cfg, nil)
	if cmd.Use != "agent" {
		t.Errorf("Use = %s, want agent", cmd.Use)
	}

	if cmd.Short != "AI智能体管理" {
		t.Errorf("Short = %s", cmd.Short)
	}

	// 验证子命令
	subCmds := cmd.Commands()
	if len(subCmds) != 2 {
		t.Fatalf("子命令数量 = %d, want 2", len(subCmds))
	}

	// 验证 list 子命令
	listCmd, _, _ := cmd.Find([]string{"list"})
	if listCmd == nil {
		t.Error("list 子命令未找到")
	}
}

func TestTemplateFlag(t *testing.T) {
	cfg := &router.RouteConfig{
		Resource: "agent",
		Routes: map[string]router.Route{
			"create": {
				Method: "POST",
				Path:   "/create",
				Template: map[string]router.Field{
					"name": {
						Description: "名称",
						Required:    true,
					},
				},
			},
		},
	}

	cmd := BuildCommand(cfg, nil)
	createCmd, _, _ := cmd.Find([]string{"create"})
	if createCmd == nil {
		t.Fatal("create 子命令未找到")
	}

	// 验证 --template flag 存在
	templateFlag := createCmd.Flags().Lookup("template")
	if templateFlag == nil {
		t.Error("--template flag 未找到")
	}
}
```

- [ ] **Step 2: 安装 cobra 依赖**

```bash
go get github.com/spf13/cobra
```

- [ ] **Step 3: 运行测试确认失败**

```bash
go test ./internal/cmdgen/... -v
# Expected: FAIL
```

- [ ] **Step 4: 实现命令生成模块**

```go
// internal/cmdgen/cmdgen.go
package cmdgen

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/output"
	"github.com/childelins/ckjr-cli/internal/router"
)

// APIClientFactory 创建 API 客户端的工厂函数
type APIClientFactory func() (*api.Client, error)

// BuildCommand 从路由配置构建 cobra 命令
func BuildCommand(cfg *router.RouteConfig, clientFactory APIClientFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   cfg.Resource,
		Short: cfg.Description,
	}

	for name, route := range cfg.Routes {
		subCmd := buildSubCommand(cfg.Resource, name, route, clientFactory)
		cmd.AddCommand(subCmd)
	}

	return cmd
}

func buildSubCommand(resource, name string, route router.Route, clientFactory APIClientFactory) *cobra.Command {
	var showTemplate bool
	var inputJSON string

	cmd := &cobra.Command{
		Use:   name + " [json]",
		Short: route.Description,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// --template 模式：输出模板并退出
			if showTemplate {
				printTemplate(route.Template)
				return
			}

			// 获取输入 JSON
			var input map[string]interface{}
			if len(args) > 0 {
				if args[0] == "-" {
					// 从 stdin 读取
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						output.PrintError(os.Stderr, "读取 stdin 失败: "+err.Error())
						os.Exit(1)
					}
					inputJSON = string(data)
				} else {
					inputJSON = args[0]
				}
			}

			if inputJSON != "" {
				if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
					output.PrintError(os.Stderr, "JSON 解析失败: "+err.Error())
					os.Exit(1)
				}
			} else {
				input = make(map[string]interface{})
			}

			// 应用默认值
			applyDefaults(input, route.Template)

			// 校验必填字段
			if missing := validateRequired(input, route.Template); len(missing) > 0 {
				output.PrintError(os.Stderr, fmt.Sprintf("缺少必填字段: %v", missing))
				os.Exit(1)
			}

			// 执行 API 请求
			if clientFactory == nil {
				output.PrintError(os.Stderr, "API 客户端未配置")
				os.Exit(1)
			}

			client, err := clientFactory()
			if err != nil {
				output.PrintError(os.Stderr, err.Error())
				os.Exit(1)
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			var result interface{}
			if err := client.Do(route.Method, route.Path, input, &result); err != nil {
				handleAPIError(err)
				os.Exit(1)
			}

			output.Print(os.Stdout, result, pretty)
		},
	}

	cmd.Flags().BoolVar(&showTemplate, "template", false, "显示参数模板")

	return cmd
}

func printTemplate(template map[string]router.Field) {
	tmpl := make(map[string]interface{})
	for name, field := range template {
		entry := map[string]interface{}{
			"description": field.Description,
			"required":    field.Required,
		}
		if field.Default != nil {
			entry["default"] = field.Default
		}
		tmpl[name] = entry
	}
	output.Print(os.Stdout, tmpl, true)
}

func applyDefaults(input map[string]interface{}, template map[string]router.Field) {
	for name, field := range template {
		if _, exists := input[name]; !exists && field.Default != nil {
			input[name] = field.Default
		}
	}
}

func validateRequired(input map[string]interface{}, template map[string]router.Field) []string {
	var missing []string
	for name, field := range template {
		if field.Required {
			if _, exists := input[name]; !exists {
				missing = append(missing, name)
			}
		}
	}
	return missing
}

func handleAPIError(err error) {
	if api.IsUnauthorized(err) {
		output.PrintError(os.Stderr, "api_key 已过期，请重新登录获取")
		return
	}

	if api.IsValidationError(err) {
		errors := api.GetValidationErrors(err)
		output.PrintError(os.Stderr, fmt.Sprintf("参数校验失败: %v", errors))
		return
	}

	output.PrintError(os.Stderr, err.Error())
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go test ./internal/cmdgen/... -v
# Expected: PASS
```

- [ ] **Step 6: 提交**

```bash
git add internal/cmdgen/
git commit -m "feat(cmdgen): add cobra command generator from route config"
```

---

### Task 7: 路由 YAML 文件

**Files:**
- Create: `routes/agent.yaml`

- [ ] **Step 1: 创建智能体路由配置**

```yaml
# routes/agent.yaml
resource: agent
description: AI智能体的增删改查
routes:
  list:
    method: POST
    path: /admin/aiCreationCenter/listApp
    description: 获取智能体列表
    template:
      page:
        description: 页码
        required: false
        default: 1
      limit:
        description: 每页数量
        required: false
        default: 10
      name:
        description: 按名称搜索
        required: false
  get:
    method: POST
    path: /admin/aiCreationCenter/getAppInfo
    description: 获取智能体详情
    template:
      aikbId:
        description: 智能体ID
        required: true
  create:
    method: POST
    path: /admin/aiCreationCenter/createApp
    description: 创建智能体
    template:
      name:
        description: 智能体名称
        required: true
      avatar:
        description: 头像URL
        required: true
      desc:
        description: 描述
        required: true
      modelId:
        description: 模型ID
        required: false
      botType:
        description: 类型
        required: false
      isSaleOnly:
        description: 1-交付型 0-工具型
        required: false
        default: 1
  update:
    method: POST
    path: /admin/aiCreationCenter/modifyApp
    description: 更新智能体
    template:
      aikbId:
        description: 智能体ID
        required: true
      name:
        description: 智能体名称
        required: true
      avatar:
        description: 头像URL
        required: true
      desc:
        description: 描述
        required: true
  delete:
    method: POST
    path: /admin/aiCreationCenter/deleteApp
    description: 删除智能体
    template:
      aikbId:
        description: 智能体ID
        required: true
```

- [ ] **Step 2: 提交**

```bash
git add routes/
git commit -m "feat(routes): add agent route configuration"
```

---

### Task 8: Config 命令 (cmd/config.go)

**Files:**
- Create: `cmd/config.go`

- [ ] **Step 1: 实现 config 命令**

```go
// cmd/config.go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/config"
	"github.com/childelins/ckjr-cli/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理 CLI 配置",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "交互式初始化配置",
	Run:   runConfigInit,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置配置项",
	Args:  cobra.ExactArgs(2),
	Run:   runConfigSet,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "查看当前配置",
	Run:   runConfigShow,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	// 输入 base_url
	fmt.Print("请输入 API 地址 (base_url): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)

	// 引导获取 api_key
	fmt.Println("\n请按以下步骤获取 API Key:")
	fmt.Println("1. 访问公司 SaaS 平台并登录")
	fmt.Println("2. 进入个人设置 -> API 密钥")
	fmt.Println("3. 复制 API Key")
	fmt.Print("\n请粘贴 API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	cfg := &config.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n配置已保存到:", config.ConfigPath)
}

func runConfigSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]

	// 验证 key
	validKeys := map[string]bool{"base_url": true, "api_key": true}
	if !validKeys[key] {
		fmt.Fprintf(os.Stderr, "无效的配置项: %s\n合法值: base_url, api_key\n", key)
		os.Exit(1)
	}

	// 加载现有配置或创建新配置
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	switch key {
	case "base_url":
		cfg.BaseURL = value
	case "api_key":
		cfg.APIKey = value
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "保存配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已设置 %s\n", key)
}

func runConfigShow(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n请先执行 ckjr config init\n", err)
		os.Exit(1)
	}

	pretty, _ := cmd.Flags().GetBool("pretty")
	result := map[string]string{
		"base_url": cfg.BaseURL,
		"api_key":  cfg.MaskedAPIKey(),
	}
	output.Print(os.Stdout, result, pretty)
}
```

- [ ] **Step 2: 提交**

```bash
git add cmd/config.go
git commit -m "feat(cmd): add config init/set/show commands"
```

---

### Task 9: 根命令 (cmd/root.go)

**Files:**
- Create: `cmd/root.go`

- [ ] **Step 1: 实现根命令**

```go
// cmd/root.go
package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/cmdgen"
	"github.com/childelins/ckjr-cli/internal/config"
	"github.com/childelins/ckjr-cli/internal/router"
)

//go:embed routes
var routesFS embed.FS

var (
	// 版本信息，通过 ldflags 注入
	Version = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "ckjr",
	Short:   "Claude Code 与公司 SaaS 平台的桥梁",
	Version: Version,
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// 添加 --pretty 全局 flag
	rootCmd.PersistentFlags().Bool("pretty", false, "格式化 JSON 输出")

	// 注册 config 命令
	rootCmd.AddCommand(configCmd)

	// 注册动态生成的命令
	registerRouteCommands()
}

func registerRouteCommands() {
	// 读取 embed 的路由文件
	entries, err := routesFS.ReadDir("routes")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取路由目录失败: %v\n", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只处理 .yaml 文件
		name := entry.Name()
		if len(name) < 5 || name[len(name)-5:] != ".yaml" {
			continue
		}

		// 读取并解析路由配置
		data, err := routesFS.ReadFile("routes/" + name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取路由文件 %s 失败: %v\n", name, err)
			continue
		}

		cfg, err := router.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "解析路由文件 %s 失败: %v\n", name, err)
			continue
		}

		// 生成命令并注册
		cmd := cmdgen.BuildCommand(cfg, createClient)
		rootCmd.AddCommand(cmd)
	}
}

// createClient 创建 API 客户端
func createClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("未找到配置文件，请先执行 ckjr config init")
	}

	return api.NewClient(cfg.BaseURL, cfg.APIKey), nil
}
```

- [ ] **Step 2: 提交**

```bash
git add cmd/root.go
git commit -m "feat(cmd): add root command with route auto-registration"
```

---

### Task 10: 主入口更新

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 更新 main.go**

```go
// main.go
package main

import "github.com/childelins/ckjr-cli/cmd"

func main() {
	cmd.Execute()
}
```

- [ ] **Step 2: 验证编译和基本功能**

```bash
go build -o ckjr .
./ckjr --help
# Expected: 显示帮助信息

./ckjr config --help
# Expected: 显示 config 子命令帮助

./ckjr agent --help
# Expected: 显示 agent 子命令帮助
```

- [ ] **Step 3: 提交**

```bash
git add main.go
git commit -m "feat: wire up main entry point"
```

---

### Task 11: 集成测试与修复

**Files:**
- Modify: 各模块根据测试结果修复

- [ ] **Step 1: 运行所有测试**

```bash
go test ./... -v
```

- [ ] **Step 2: 修复发现的问题**

根据测试输出修复任何问题。

- [ ] **Step 3: 提交修复**

```bash
git add -A
git commit -m "fix: resolve test failures"
```

---

### Task 12: 最终验证

- [ ] **Step 1: 完整构建**

```bash
go build -o ckjr .
```

- [ ] **Step 2: 验证所有命令**

```bash
# 帮助
./ckjr --help
./ckjr config --help
./ckjr agent --help

# 模板
./ckjr agent list --template
./ckjr agent create --template
./ckjr agent update --template

# 配置（会提示未配置）
./ckjr config show
```

- [ ] **Step 3: 最终提交**

```bash
git add -A
git commit -m "feat: complete ckjr-cli MVP implementation"
```

---

## 验收标准

1. `ckjr config init` 能交互式创建配置
2. `ckjr config show` 能显示配置（api_key 脱敏）
3. `ckjr agent list --template` 能显示参数模板
4. `ckjr agent list` 能调用 API 并返回结果
5. `--pretty` 能格式化 JSON 输出
6. 所有测试通过
