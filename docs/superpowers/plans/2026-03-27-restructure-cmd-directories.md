# cmd 目录结构重组实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 cmd/ 平铺文件拆分为子包 + 精简 YAML 嵌入路径

**Architecture:** 两阶段变更：(1) YAML 路径精简：embed 指令和加载路径从 `config/routes` 改为 `routes`；(2) cmd 子包拆分：config.go/route.go/workflow.go 各自独立为子包，暴露 NewCommand() 工厂函数

**Tech Stack:** Go, cobra, go:embed

---

## Part A: YAML 路径精简

### Task 1: 更新 internal/config/yaml 加载路径

**Files:**
- Modify: `internal/config/yaml/yaml.go:19-26`
- Modify: `internal/config/yaml/yaml_test.go:10,36,59`

- [ ] **Step 1: 修改 yaml_test.go MapFS key（让测试先失败）**

将所有 `"config/routes/..."` 改为 `"routes/..."`，`"config/workflows/..."` 改为 `"workflows/..."`：

```go
// internal/config/yaml/yaml_test.go
func TestLoadRoutes(t *testing.T) {
	memFS := fstest.MapFS{
		"routes/agent.yaml":          {Data: []byte("name: agent\ndescription: test\nroutes: {}")},
		"routes/common.yaml":         {Data: []byte("name: common\ndescription: common\nroutes: {}")},
		"routes/readme.txt":          {Data: []byte("ignored")},
		"routes/sub/.keep":           {Data: []byte("")},
	}
	// ... 其余不变
}

func TestLoadRoutes_EmptyDir(t *testing.T) {
	memFS := fstest.MapFS{
		"routes/readme.txt": {Data: []byte("ignored")},
	}
	// ... 其余不变
}

func TestLoadWorkflows(t *testing.T) {
	memFS := fstest.MapFS{
		"workflows/agent.yaml": {Data: []byte("name: agent\nworkflows: {}")},
		"workflows/note.txt":   {Data: []byte("ignored")},
	}
	// ... 其余不变
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/config/yaml/ -v`
Expected: FAIL（loadDir 找不到 "routes" 目录）

- [ ] **Step 3: 修改 yaml.go loadDir 路径**

```go
// internal/config/yaml/yaml.go
// LoadRoutes 读取 routes/ 下所有 .yaml 文件
func (f *FS) LoadRoutes() (map[string][]byte, error) {
	return f.loadDir("routes")
}

// LoadWorkflows 读取 workflows/ 下所有 .yaml 文件
func (f *FS) LoadWorkflows() (map[string][]byte, error) {
	return f.loadDir("workflows")
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/config/yaml/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/config/yaml/yaml.go internal/config/yaml/yaml_test.go
git commit -m "refactor: simplify YAML load paths from config/routes to routes"
```

### Task 2: 迁移 YAML 物理文件 + 更新 embed 指令

**Files:**
- Move: `cmd/ckjr-cli/config/routes/*.yaml` -> `cmd/ckjr-cli/routes/`
- Move: `cmd/ckjr-cli/config/workflows/*.yaml` -> `cmd/ckjr-cli/workflows/`
- Modify: `cmd/ckjr-cli/embed.go:5`
- Modify: `cmd/embed_test.go:11`
- Delete: `cmd/ckjr-cli/config/` (空目录)

- [ ] **Step 1: 创建新目录并迁移 YAML 文件**

```bash
mkdir -p cmd/ckjr-cli/routes cmd/ckjr-cli/workflows
mv cmd/ckjr-cli/config/routes/*.yaml cmd/ckjr-cli/routes/
mv cmd/ckjr-cli/config/workflows/*.yaml cmd/ckjr-cli/workflows/
```

- [ ] **Step 2: 更新 cmd/ckjr-cli/embed.go**

```go
package main

import "embed"

//go:embed all:routes all:workflows
var configFS embed.FS
```

- [ ] **Step 3: 更新 cmd/embed_test.go embed 指令**

```go
package cmd

import (
	"embed"
	"io/fs"
	"testing"

	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
)

//go:embed all:ckjr-cli/routes all:ckjr-cli/workflows
var testEmbedFS embed.FS

func TestMain(m *testing.M) {
	subFS, err := fs.Sub(testEmbedFS, "ckjr-cli")
	if err != nil {
		panic(err)
	}
	yamlFS = configyaml.New(subFS)
	registerRouteCommands()
	m.Run()
}
```

- [ ] **Step 4: 删除空 config 目录**

```bash
rm -rf cmd/ckjr-cli/config/
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./cmd/... -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add cmd/ckjr-cli/embed.go cmd/embed_test.go cmd/ckjr-cli/routes/ cmd/ckjr-cli/workflows/
git rm -r cmd/ckjr-cli/config/
git commit -m "refactor: move YAML configs from config/ to routes/ and workflows/"
```

### Task 3: 更新 workflow_test.go 路径 + wiki 文档

**Files:**
- Modify: `internal/workflow/workflow_test.go:158`
- Modify: `wiki/core-concepts.md`
- Modify: `wiki/extending.md`
- Modify: `wiki/project-structure.md`

- [ ] **Step 1: 更新 workflow_test.go 中的 os.ReadFile 路径**

```go
// internal/workflow/workflow_test.go:158
data, err := os.ReadFile("../../cmd/ckjr-cli/workflows/agent.yaml")
```

- [ ] **Step 2: 运行测试确认通过**

Run: `go test ./internal/workflow/ -v`
Expected: PASS

- [ ] **Step 3: 更新 wiki 文档中的路径引用**

在 `wiki/core-concepts.md`、`wiki/extending.md`、`wiki/project-structure.md` 中将所有 `cmd/ckjr-cli/config/routes/` 替换为 `cmd/ckjr-cli/routes/`，`cmd/ckjr-cli/config/workflows/` 替换为 `cmd/ckjr-cli/workflows/`。

- [ ] **Step 4: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 5: 提交**

```bash
git add internal/workflow/workflow_test.go wiki/
git commit -m "docs: update file path references after YAML directory restructure"
```

---

## Part B: cmd 子包拆分

### Task 4: 提取辅助函数到 internal/router

**Files:**
- Create: `internal/router/infer_test.go`
- Create: `internal/router/infer.go`

- [ ] **Step 1: 创建 infer_test.go（从 cmd/route_test.go 迁移 TestInferRouteName）**

```go
// internal/router/infer_test.go
package router

import "testing"

func TestInferRouteName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/admin/aiCreationCenter/modifyApp", "update"},
		{"/admin/aiCreationCenter/listApp", "listApp"},
		{"/admin/aiCreationCenter/createApp", "create"},
		{"/admin/aiCreationCenter/deleteApp", "deleteApp"},
		{"/admin/aiCreationCenter/describeApp", "get"},
		{"/admin/order/addOrder", "create"},
		{"/admin/order/removeOrder", "delete"},
		{"/admin/order/editOrder", "update"},
		{"/admin/order/queryList", "list"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := InferRouteName(tt.path)
			if got != tt.want {
				t.Errorf("InferRouteName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestInferNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"agent.yaml", "agent"},
		{"order.yaml", "order"},
		{"sub/dir/test.yaml", "test"},
		{"noext", "noext"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := InferNameFromPath(tt.path)
			if got != tt.want {
				t.Errorf("InferNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/router/ -run TestInfer -v`
Expected: FAIL（InferRouteName 未定义）

- [ ] **Step 3: 创建 infer.go**

从 `cmd/route.go:85-170` 提取辅助函数，改为导出名：

```go
// internal/router/infer.go
package router

// InferRouteName 从 URL path 末段推导 route name
func InferRouteName(path string) string {
	parts := splitPath(path)
	if len(parts) == 0 {
		return "unknown"
	}
	last := parts[len(parts)-1]

	prefixes := map[string]string{
		"modify": "update",
		"edit":   "update",
		"remove": "delete",
		"add":    "create",
		"create": "create",
		"query":  "list",
	}
	lower := toLower(last)
	for prefix, mapped := range prefixes {
		if len(lower) >= len(prefix) && lower[:len(prefix)] == prefix {
			return mapped
		}
	}

	if len(lower) >= 8 && lower[:8] == "describe" {
		return "get"
	}

	return last
}

// InferNameFromPath 从文件路径推导 name（resource 名称）
func InferNameFromPath(path string) string {
	parts := splitPath(path)
	if len(parts) == 0 {
		return "unknown"
	}
	filename := parts[len(parts)-1]
	for i := range filename {
		if i > 0 && filename[i-1] == '.' {
			return filename[:i-1]
		}
	}
	return filename
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range split(path, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func split(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/router/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/router/infer.go internal/router/infer_test.go
git commit -m "refactor: extract route inference helpers to internal/router"
```

### Task 5: 创建 cmd/config/ 子包

**Files:**
- Create: `cmd/config/config_test.go`
- Create: `cmd/config/config.go`

- [ ] **Step 1: 创建 cmd/config/config_test.go**

从 `cmd/config_test.go` 迁移，移除对 `rootCmd` 的引用（line 29 的 `cmd := rootCmd` 是死代码，不需要迁移）：

```go
package config

import (
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
	_, err := config.Load()
	if err == nil {
		t.Error("Load() should return error when config not found")
	}
}

func TestConfigSetAndShow(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()
	cfg := &config.Config{
		BaseURL: "https://api.example.com",
		APIKey:  "test-api-key-12345",
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
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
	cfg := &config.Config{}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
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
	info, err := os.Stat(config.ConfigPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./cmd/config/ -v`
Expected: FAIL（package 不存在）

- [ ] **Step 3: 创建 cmd/config/config.go**

```go
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/config"
	"github.com/childelins/ckjr-cli/internal/output"
)

func NewCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "管理 CLI 配置",
	}

	configInitCmd := &cobra.Command{
		Use:   "init",
		Short: "交互式初始化配置",
		Run:   runConfigInit,
	}

	configSetCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "设置配置项",
		Args:  cobra.ExactArgs(2),
		Run:   runConfigSet,
	}

	configShowCmd := &cobra.Command{
		Use:   "show",
		Short: "查看当前配置",
		Run:   runConfigShow,
	}

	configCmd.AddCommand(configInitCmd, configSetCmd, configShowCmd)
	return configCmd
}

func runConfigInit(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入 API 地址 (base_url): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
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
	validKeys := map[string]bool{"base_url": true, "api_key": true}
	if !validKeys[key] {
		fmt.Fprintf(os.Stderr, "无效的配置项: %s\n合法值: base_url, api_key\n", key)
		os.Exit(1)
	}
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
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n请先执行 ckjr-cli config init\n", err)
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

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./cmd/config/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/config/
git commit -m "refactor: extract config command to cmd/config subpackage"
```

### Task 6: 创建 cmd/route/ 子包

**Files:**
- Create: `cmd/route/route_test.go`
- Create: `cmd/route/route.go`

- [ ] **Step 1: 创建 cmd/route/route_test.go**

从 `cmd/route_test.go` 迁移命令测试，辅助函数测试已在 Task 4 处理。使用 `router.InferRouteName` 替换原来的 `inferRouteName`：

```go
package route

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
)

func TestRouteCmd_IsHidden(t *testing.T) {
	cmd := NewCommand()
	if !cmd.Hidden {
		t.Error("routeCmd should be hidden")
	}
}

func TestRouteImport_Stdin_AppendToExisting(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "agent.yaml")

	initial := `name: agent
description: AI智能体管理
routes:
    list:
        method: POST
        path: /admin/list
        description: 获取列表
`
	os.WriteFile(yamlPath, []byte(initial), 0644)

	curl := `curl 'https://kpapi-cs.ckjr001.com/api/admin/aiCreationCenter/modifyApp' -H 'content-type: application/json' --data-raw '{"name":"test","aikbId":3550}'`

	cmd := NewCommand()
	importCmd, _, _ := cmd.Find([]string{"import"})
	err := runImport(curl, yamlPath, "update", "")
	if err != nil {
		t.Fatalf("runImport() error = %v", err)
	}
	_ = importCmd // 仅验证 importCmd 存在

	data, _ := os.ReadFile(yamlPath)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(cfg.Routes) != 2 {
		t.Errorf("Routes count = %d, want 2", len(cfg.Routes))
	}
	route, ok := cfg.Routes["update"]
	if !ok {
		t.Fatal("update route not found")
	}
	if route.Method != "POST" {
		t.Errorf("Method = %q", route.Method)
	}
}

func TestRouteImport_CreateNewFile(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "order.yaml")

	curl := `curl 'https://example.com/api/admin/order/list' --data-raw '{"page":1,"limit":10}'`

	err := runImport(curl, yamlPath, "list", "订单管理")
	if err != nil {
		t.Fatalf("runImport() error = %v", err)
	}

	data, _ := os.ReadFile(yamlPath)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Name != "order" {
		t.Errorf("Name = %q", cfg.Name)
	}
	if _, ok := cfg.Routes["list"]; !ok {
		t.Error("list route not found")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./cmd/route/ -v`
Expected: FAIL（package 不存在）

- [ ] **Step 3: 创建 cmd/route/route.go**

```go
package route

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/curlparse"
	"github.com/childelins/ckjr-cli/internal/router"
	"github.com/childelins/ckjr-cli/internal/yamlgen"
)

func NewCommand() *cobra.Command {
	routeCmd := &cobra.Command{
		Use:    "route",
		Short:  "路由配置管理",
		Hidden: true,
	}

	routeImportCmd := &cobra.Command{
		Use:   "import",
		Short: "从 curl 命令导入路由配置",
		Long:  "解析 curl 命令并生成 YAML 路由配置。支持 stdin 管道输入或 --curl 参数。",
		RunE: func(cmd *cobra.Command, args []string) error {
			curlStr, _ := cmd.Flags().GetString("curl")
			file, _ := cmd.Flags().GetString("file")
			routeName, _ := cmd.Flags().GetString("name")
			nameDesc, _ := cmd.Flags().GetString("name-desc")

			if curlStr == "" {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("读取 stdin 失败: %w", err)
					}
					curlStr = string(data)
				}
			}

			if curlStr == "" {
				return fmt.Errorf("请通过 --curl 参数或 stdin 管道提供 curl 命令")
			}
			if file == "" {
				return fmt.Errorf("请通过 --file 参数指定目标 YAML 文件路径")
			}

			if err := runImport(curlStr, file, routeName, nameDesc); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "已添加路由 %s 到 %s\n", routeName, file)
			return nil
		},
	}

	routeImportCmd.Flags().String("curl", "", "curl 命令字符串")
	routeImportCmd.Flags().StringP("file", "f", "", "目标 YAML 文件路径")
	routeImportCmd.Flags().StringP("name", "n", "", "路由名称（默认从 URL 推导）")
	routeImportCmd.Flags().String("name-desc", "", "资源名称描述")

	routeCmd.AddCommand(routeImportCmd)
	return routeCmd
}

func runImport(curlStr, file, routeName, nameDesc string) error {
	result, err := curlparse.Parse(curlStr)
	if err != nil {
		return fmt.Errorf("curl 解析失败: %w", err)
	}

	if routeName == "" {
		routeName = router.InferRouteName(result.Path)
	}

	r := yamlgen.GenerateRoute(result)

	if _, err := os.Stat(file); err == nil {
		return yamlgen.AppendToFile(file, routeName, r)
	}

	name := router.InferNameFromPath(file)
	if nameDesc == "" {
		return fmt.Errorf("新建文件需要通过 --name-desc 指定资源描述")
	}
	return yamlgen.CreateFile(file, name, nameDesc, routeName, r)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./cmd/route/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/route/
git commit -m "refactor: extract route command to cmd/route subpackage"
```

### Task 7: 创建 cmd/workflow/ 子包

**Files:**
- Create: `cmd/workflow/workflow_test.go`
- Create: `cmd/workflow/workflow.go`

- [ ] **Step 1: 创建 cmd/workflow/workflow_test.go**

使用 MapFS mock yamlFS，不再依赖 rootCmd：

```go
package workflow

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
)

func setupTestYAMLFS(t *testing.T) *configyaml.FS {
	t.Helper()
	memFS := fstest.MapFS{
		"workflows/agent.yaml": {Data: []byte(`
name: agent
workflows:
  create-agent:
    description: 创建并配置一个完整的智能体
    triggers:
      - 用户请求创建智能体
    info:
      - name: name (必填)
      - name: instructions (必填)
    steps:
      - agent create
      - agent update
      - common getLink
      - qrcodeImg
    summary:
      - 智能体配置完成
`)},
	}
	return configyaml.New(memFS)
}

func TestWorkflowList(t *testing.T) {
	yamlFS := setupTestYAMLFS(t)
	cmd := NewCommand(yamlFS)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "create-agent") {
		t.Errorf("输出缺少 create-agent, got: %s", output)
	}
	if !strings.Contains(output, "创建并配置一个完整的智能体") {
		t.Errorf("输出缺少 workflow 描述, got: %s", output)
	}
}

func TestWorkflowDescribe(t *testing.T) {
	yamlFS := setupTestYAMLFS(t)
	cmd := NewCommand(yamlFS)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"describe", "create-agent"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	output := buf.String()
	checks := []string{
		"Workflow: create-agent",
		"== 需要收集的信息 ==",
		"name (必填)",
		"instructions (必填)",
		"== 执行步骤 ==",
		"agent create",
		"agent update",
		"common getLink",
		"qrcodeImg",
		"== 完成摘要 ==",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("输出缺少 %q", check)
		}
	}
}

func TestWorkflowDescribe_NotFound(t *testing.T) {
	yamlFS := setupTestYAMLFS(t)
	cmd := NewCommand(yamlFS)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"describe", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("期望返回错误")
	}
	if !strings.Contains(err.Error(), "未找到工作流") {
		t.Errorf("错误信息 = %q, 期望包含 '未找到工作流'", err.Error())
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./cmd/workflow/ -v`
Expected: FAIL（package 不存在）

- [ ] **Step 3: 创建 cmd/workflow/workflow.go**

```go
package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/workflow"
	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
)

func NewCommand(yamlFS *configyaml.FS) *cobra.Command {
	workflowCmd := &cobra.Command{
		Use:   "workflow",
		Short: "工作流管理",
	}

	workflowListCmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有可用的工作流",
		RunE: func(cmd *cobra.Command, args []string) error {
			configs, err := loadAllWorkflows(yamlFS)
			if err != nil {
				return err
			}

			type item struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Triggers    []string `json:"triggers"`
			}

			var items []item
			for _, cfg := range configs {
				for name, wf := range cfg.Workflows {
					items = append(items, item{
						Name:        name,
						Description: wf.Description,
						Triggers:    wf.Triggers,
					})
				}
			}

			data, err := json.Marshal(items)
			if err != nil {
				return err
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			if pretty {
				var indented bytes.Buffer
				json.Indent(&indented, data, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), indented.String())
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
			}
			return nil
		},
	}

	workflowDescribeCmd := &cobra.Command{
		Use:   "describe <name>",
		Short: "输出工作流的完整描述",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			configs, err := loadAllWorkflows(yamlFS)
			if err != nil {
				return err
			}

			for _, cfg := range configs {
				if wf, ok := cfg.Workflows[name]; ok {
					fmt.Fprint(cmd.OutOrStdout(), workflow.Describe(&wf, name))
					return nil
				}
			}

			return fmt.Errorf("未找到工作流: %s", name)
		},
	}

	workflowCmd.AddCommand(workflowListCmd, workflowDescribeCmd)
	return workflowCmd
}

func loadAllWorkflows(yamlFS *configyaml.FS) ([]*workflow.Config, error) {
	if yamlFS == nil {
		return nil, fmt.Errorf("YAML 文件系统未初始化")
	}

	files, err := yamlFS.LoadWorkflows()
	if err != nil {
		return nil, err
	}

	var configs []*workflow.Config
	for name, data := range files {
		cfg, err := workflow.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %w", name, err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./cmd/workflow/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/workflow/
git commit -m "refactor: extract workflow command to cmd/workflow subpackage"
```

### Task 8: 重构 cmd/root.go，集成子包

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: 重写 cmd/root.go**

移除子命令定义（configCmd/routeCmd/workflowCmd），改为 import 子包注册。config 和 route 在 init() 注册，workflow 在 Execute() 注册（因为需要 yamlFS，而 yamlFS 由 main 包在 init 中设置，Go init 顺序决定 cmd init 先于 main init 执行）：

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/cmdgen"
	"github.com/childelins/ckjr-cli/internal/config"
	"github.com/childelins/ckjr-cli/internal/logging"
	"github.com/childelins/ckjr-cli/internal/router"
	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"

	"github.com/childelins/ckjr-cli/cmd/config"
	"github.com/childelins/ckjr-cli/cmd/route"
	"github.com/childelins/ckjr-cli/cmd/workflow"
)

var yamlFS *configyaml.FS

// SetYAMLFS 设置 YAML 配置加载器，由 main 包调用
func SetYAMLFS(fs *configyaml.FS) {
	yamlFS = fs
}

var (
	version     = "dev"
	environment = "production"
)

// SetVersion 由 main 包调用，通过 ldflags 注入版本号
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// SetEnvironment 由 main 包调用，通过 ldflags 注入环境模式
func SetEnvironment(e string) {
	environment = e
}

var rootCmd = &cobra.Command{
	Use:               "ckjr-cli",
	Short:             "创客匠人 CLI - 知识付费 SaaS 系统的命令行工具",
	Version:           version,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

// Execute 执行根命令
func Execute() {
	registerRouteCommands()
	rootCmd.AddCommand(workflow.NewCommand(yamlFS))
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("pretty", false, "格式化 JSON 输出")
	rootCmd.PersistentFlags().Bool("verbose", false, "显示详细调试信息")
	cobra.OnInitialize(initLogging)

	rootCmd.AddCommand(config.NewCommand())
	rootCmd.AddCommand(route.NewCommand())
}

func initLogging() {
	verbose, _ := rootCmd.Flags().GetBool("verbose")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取用户目录失败: %v\n", err)
		return
	}
	baseDir := filepath.Join(homeDir, ".ckjr")
	env := logging.ParseEnvironment(environment)
	if err := logging.Init(verbose, baseDir, env); err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
	}
}

func registerRouteCommands() {
	if yamlFS == nil {
		fmt.Fprintf(os.Stderr, "YAML 文件系统未初始化\n")
		return
	}

	files, err := yamlFS.LoadRoutes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取路由目录失败: %v\n", err)
		return
	}

	for name, data := range files {
		cfg, err := router.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "解析路由文件 %s 失败: %v\n", name, err)
			continue
		}

		cmd := cmdgen.BuildCommand(cfg, createClient)
		rootCmd.AddCommand(cmd)
	}
}

// createClient 创建 API 客户端
func createClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("未找到配置文件，请先执行 ckjr-cli config init")
	}
	return api.NewClient(cfg.BaseURL, cfg.APIKey), nil
}
```

- [ ] **Step 2: 运行 cmd 包测试确认通过**

Run: `go test ./cmd/ -v`
Expected: PASS（root_test.go 和 embed_test.go 应全部通过）

- [ ] **Step 3: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/root.go
git commit -m "refactor: wire cmd subpackages into root.go"
```

### Task 9: 删除旧文件

**Files:**
- Delete: `cmd/config.go`
- Delete: `cmd/config_test.go`
- Delete: `cmd/route.go`
- Delete: `cmd/route_test.go`
- Delete: `cmd/workflow.go`
- Delete: `cmd/workflow_test.go`

- [ ] **Step 1: 删除已迁移的旧文件**

```bash
git rm cmd/config.go cmd/config_test.go cmd/route.go cmd/route_test.go cmd/workflow.go cmd/workflow_test.go
```

- [ ] **Step 2: 运行全量测试确认无遗漏**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 3: 提交**

```bash
git commit -m "chore: remove old flat cmd files after subpackage migration"
```

### Task 10: 最终验证

- [ ] **Step 1: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: 运行 go vet**

Run: `go vet ./...`
Expected: 无警告

- [ ] **Step 3: 确认编译通过**

Run: `go build ./cmd/ckjr-cli/`
Expected: 编译成功

- [ ] **Step 4: 确认最终目录结构**

```
cmd/
  ckjr-cli/
    main.go
    embed.go
    routes/*.yaml
    workflows/*.yaml
  root.go
  root_test.go
  embed_test.go
  config/
    config.go
    config_test.go
  route/
    route.go
    route_test.go
  workflow/
    workflow.go
    workflow_test.go
internal/router/
  router.go
  router_test.go
  infer.go
  infer_test.go
```
