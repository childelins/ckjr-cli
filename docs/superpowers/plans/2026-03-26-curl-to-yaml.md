# curl-to-yaml 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 实现 `ckjr-cli route import` 命令，从 curl 命令自动生成 YAML 路由配置

**Architecture:** 新增 `internal/curlparse` 解析 curl 命令提取 method/path/fields，新增 `internal/yamlgen` 生成符合现有 agent.yaml 格式的 YAML 配置，新增 `cmd/route.go` 注册 route import 子命令。不修改现有模块。

**Tech Stack:** Go 1.24.3, gopkg.in/yaml.v3, cobra

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `internal/curlparse/parse.go` | curl 命令解析：提取 method、path、JSON body fields |
| `internal/curlparse/parse_test.go` | curlparse 单元测试 |
| `internal/yamlgen/generate.go` | YAML 路由生成：GenerateRoute + AppendToFile + CreateFile |
| `internal/yamlgen/generate_test.go` | yamlgen 单元测试 |
| `cmd/route.go` | route 命令组 + import 子命令 |
| `cmd/route_test.go` | route import 集成测试 |

---

### Task 1: curlparse - curl 命令解析器

**Files:**
- Create: `internal/curlparse/parse.go`
- Create: `internal/curlparse/parse_test.go`

- [ ] **Step 1: 写失败测试 - 基本 POST 解析**

```go
// internal/curlparse/parse_test.go
package curlparse

import (
	"testing"
)

func TestParse_PostWithBody(t *testing.T) {
	curl := `curl 'https://kpapi-cs.ckjr001.com/api/admin/aiCreationCenter/modifyApp' \
  -H 'content-type: application/json' \
  --data-raw '{"name":"test","aikbId":3550,"desc":"描述"}'`

	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Method != "POST" {
		t.Errorf("Method = %q, want POST", result.Method)
	}
	if result.Path != "/api/admin/aiCreationCenter/modifyApp" {
		t.Errorf("Path = %q", result.Path)
	}
	if len(result.Fields) != 3 {
		t.Fatalf("Fields count = %d, want 3", len(result.Fields))
	}
	if f, ok := result.Fields["aikbId"]; !ok || f.Type != "int" {
		t.Errorf("aikbId field: got %+v", result.Fields["aikbId"])
	}
	if f, ok := result.Fields["name"]; !ok || f.Type != "string" {
		t.Errorf("name field: got %+v", result.Fields["name"])
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/curlparse/...`
Expected: FAIL (package not found)

- [ ] **Step 3: 实现 curlparse.Parse**

```go
// internal/curlparse/parse.go
package curlparse

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
)

// Result 保存 curl 解析结果
type Result struct {
	Method string           // HTTP method
	Path   string           // URL path
	Fields map[string]Field // 从 JSON body 提取的顶层字段
}

// Field 解析出的字段信息
type Field struct {
	Type    string      // 推断类型: string/int/bool
	Example interface{} // 原始值
}

// Parse 解析 curl 命令字符串
func Parse(curl string) (*Result, error) {
	// 预处理：去除续行符，合并为单行
	curl = strings.ReplaceAll(curl, "\\\n", " ")
	curl = strings.ReplaceAll(curl, "\\\r\n", " ")

	tokens := tokenize(curl)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("空的 curl 命令")
	}

	// 第一个 token 应该是 "curl"
	if tokens[0] != "curl" {
		return nil, fmt.Errorf("不是有效的 curl 命令")
	}

	result := &Result{Fields: make(map[string]Field)}
	var rawURL, dataRaw, method string

	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case tok == "-X" || tok == "--request":
			if i+1 < len(tokens) {
				method = strings.ToUpper(tokens[i+1])
				i++
			}
		case tok == "--data-raw" || tok == "-d" || tok == "--data":
			if i+1 < len(tokens) {
				dataRaw = tokens[i+1]
				i++
			}
		case tok == "-H" || tok == "--header":
			i++ // 跳过 header 值
		case !strings.HasPrefix(tok, "-") && rawURL == "":
			rawURL = tok
		}
	}

	if rawURL == "" {
		return nil, fmt.Errorf("未找到 URL")
	}

	// 解析 URL path
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("URL 解析失败: %w", err)
	}
	result.Path = u.Path

	// 确定 method
	if method != "" {
		result.Method = method
	} else if dataRaw != "" {
		result.Method = "POST"
	} else {
		result.Method = "GET"
	}

	// 解析 JSON body
	if dataRaw != "" {
		var body map[string]interface{}
		if err := json.Unmarshal([]byte(dataRaw), &body); err != nil {
			return nil, fmt.Errorf("JSON body 解析失败: %w", err)
		}
		for key, val := range body {
			f, ok := inferField(val)
			if ok {
				result.Fields[key] = f
			}
		}
	}

	return result, nil
}

// inferField 从 JSON 值推断字段类型，跳过数组和对象
func inferField(val interface{}) (Field, bool) {
	switch v := val.(type) {
	case float64:
		if v == math.Trunc(v) {
			return Field{Type: "int", Example: int(v)}, true
		}
		return Field{Type: "string", Example: v}, true
	case bool:
		return Field{Type: "bool", Example: v}, true
	case string:
		return Field{Type: "string", Example: v}, true
	case nil:
		return Field{Type: "string", Example: nil}, true
	default:
		// 数组、对象跳过
		return Field{}, false
	}
}

// tokenize 将 curl 命令分词，处理单引号和双引号
func tokenize(input string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') && !inSingle && !inDouble:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/curlparse/... -v`
Expected: PASS

- [ ] **Step 5: 补充更多测试用例**

```go
// 追加到 internal/curlparse/parse_test.go

func TestParse_GetRequest(t *testing.T) {
	curl := `curl 'https://example.com/api/users?page=1'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.Method != "GET" {
		t.Errorf("Method = %q, want GET", result.Method)
	}
	if result.Path != "/api/users" {
		t.Errorf("Path = %q", result.Path)
	}
	if len(result.Fields) != 0 {
		t.Errorf("Fields count = %d, want 0", len(result.Fields))
	}
}

func TestParse_ExplicitMethod(t *testing.T) {
	curl := `curl -X PUT 'https://example.com/api/users' --data-raw '{"name":"test"}'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.Method != "PUT" {
		t.Errorf("Method = %q, want PUT", result.Method)
	}
}

func TestParse_NestedBody(t *testing.T) {
	curl := `curl 'https://example.com/api' --data-raw '{"name":"test","items":[1,2],"config":{"key":"val"},"count":5}'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	// 嵌套字段（items, config）应被跳过，只保留顶层简单类型
	if len(result.Fields) != 2 {
		t.Errorf("Fields count = %d, want 2 (name + count)", len(result.Fields))
	}
	if _, ok := result.Fields["items"]; ok {
		t.Error("items (array) should be skipped")
	}
	if _, ok := result.Fields["config"]; ok {
		t.Error("config (object) should be skipped")
	}
}

func TestParse_TypeInference(t *testing.T) {
	curl := `curl 'https://example.com/api' --data-raw '{"str":"hello","num":42,"flag":true,"empty":null}'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tests := map[string]string{
		"str":   "string",
		"num":   "int",
		"flag":  "bool",
		"empty": "string",
	}
	for name, wantType := range tests {
		f, ok := result.Fields[name]
		if !ok {
			t.Errorf("field %q not found", name)
			continue
		}
		if f.Type != wantType {
			t.Errorf("%s.Type = %q, want %q", name, f.Type, wantType)
		}
	}
}

func TestParse_Invalid(t *testing.T) {
	tests := []struct {
		name string
		curl string
	}{
		{"empty", ""},
		{"not curl", "wget https://example.com"},
		{"bad json", "curl 'https://example.com' --data-raw 'not json'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.curl)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}
```

- [ ] **Step 6: 运行全部 curlparse 测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/curlparse/... -v`
Expected: ALL PASS

- [ ] **Step 7: 提交**

```bash
git add internal/curlparse/
git commit -m "feat(curlparse): add curl command parser"
```

---

### Task 2: yamlgen - YAML 路由生成器

**Files:**
- Create: `internal/yamlgen/generate.go`
- Create: `internal/yamlgen/generate_test.go`

- [ ] **Step 1: 写失败测试 - GenerateRoute**

```go
// internal/yamlgen/generate_test.go
package yamlgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/childelins/ckjr-cli/internal/curlparse"
	"github.com/childelins/ckjr-cli/internal/router"
)

func TestGenerateRoute(t *testing.T) {
	result := &curlparse.Result{
		Method: "POST",
		Path:   "/api/admin/aiCreationCenter/modifyApp",
		Fields: map[string]curlparse.Field{
			"name":   {Type: "string", Example: "test"},
			"aikbId": {Type: "int", Example: 3550},
		},
	}

	route := GenerateRoute(result)

	if route.Method != "POST" {
		t.Errorf("Method = %q, want POST", route.Method)
	}
	if route.Path != "/api/admin/aiCreationCenter/modifyApp" {
		t.Errorf("Path = %q", route.Path)
	}
	if route.Description != "TODO: 补充描述" {
		t.Errorf("Description = %q", route.Description)
	}
	if len(route.Template) != 2 {
		t.Fatalf("Template count = %d, want 2", len(route.Template))
	}
	f := route.Template["aikbId"]
	if f.Type != "int" {
		t.Errorf("aikbId.Type = %q, want int", f.Type)
	}
	if f.Example != "3550" {
		t.Errorf("aikbId.Example = %q, want 3550", f.Example)
	}
	if f.Description != "TODO" {
		t.Errorf("aikbId.Description = %q, want TODO", f.Description)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/yamlgen/...`
Expected: FAIL

- [ ] **Step 3: 实现 GenerateRoute**

```go
// internal/yamlgen/generate.go
package yamlgen

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/childelins/ckjr-cli/internal/curlparse"
	"github.com/childelins/ckjr-cli/internal/router"
)

// GenerateRoute 从 curlparse.Result 生成 Route
func GenerateRoute(result *curlparse.Result) router.Route {
	tmpl := make(map[string]router.Field, len(result.Fields))
	for name, f := range result.Fields {
		field := router.Field{
			Description: "TODO",
			Required:    false,
		}
		if f.Type != "" && f.Type != "string" {
			field.Type = f.Type
		}
		if f.Example != nil {
			field.Example = fmt.Sprintf("%v", f.Example)
		}
		tmpl[name] = field
	}
	return router.Route{
		Method:      result.Method,
		Path:        result.Path,
		Description: "TODO: 补充描述",
		Template:    tmpl,
	}
}

// AppendToFile 追加路由到已有 YAML 文件
func AppendToFile(path string, name string, route router.Route) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	cfg, err := router.Parse(data)
	if err != nil {
		return fmt.Errorf("解析 YAML 失败: %w", err)
	}

	if _, exists := cfg.Routes[name]; exists {
		return fmt.Errorf("路由 %q 已存在", name)
	}

	cfg.Routes[name] = route
	return writeConfig(path, cfg)
}

// CreateFile 创建新的 YAML 路由文件
func CreateFile(path string, resource string, resourceDesc string, name string, route router.Route) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("文件已存在: %s", path)
	}

	if resourceDesc == "" {
		resourceDesc = "TODO: 补充描述"
	}

	cfg := &router.RouteConfig{
		Resource:    resource,
		Description: resourceDesc,
		Routes:      map[string]router.Route{name: route},
	}
	return writeConfig(path, cfg)
}

func writeConfig(path string, cfg *router.RouteConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("YAML 序列化失败: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 4: 运行测试确认 GenerateRoute 通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/yamlgen/... -run TestGenerateRoute -v`
Expected: PASS

- [ ] **Step 5: 写文件操作测试**

```go
// 追加到 internal/yamlgen/generate_test.go

func TestAppendToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	// 写入初始文件
	initial := `resource: agent
description: AI智能体管理
routes:
    list:
        method: POST
        path: /admin/list
        description: 获取列表
        template:
            page:
                description: 页码
                required: false
`
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	route := router.Route{
		Method:      "POST",
		Path:        "/admin/create",
		Description: "TODO: 补充描述",
		Template: map[string]router.Field{
			"name": {Description: "TODO", Required: false},
		},
	}

	if err := AppendToFile(path, "create", route); err != nil {
		t.Fatalf("AppendToFile() error = %v", err)
	}

	// 验证结果
	data, _ := os.ReadFile(path)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(cfg.Routes) != 2 {
		t.Errorf("Routes count = %d, want 2", len(cfg.Routes))
	}
	if _, ok := cfg.Routes["create"]; !ok {
		t.Error("create route not found")
	}
}

func TestAppendToFile_Conflict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	initial := `resource: agent
description: test
routes:
    list:
        method: POST
        path: /list
        description: 列表
`
	os.WriteFile(path, []byte(initial), 0644)

	route := router.Route{Method: "POST", Path: "/list2"}
	err := AppendToFile(path, "list", route)
	if err == nil {
		t.Error("expected conflict error")
	}
}

func TestCreateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.yaml")

	route := router.Route{
		Method:      "POST",
		Path:        "/admin/order/list",
		Description: "TODO: 补充描述",
		Template: map[string]router.Field{
			"page": {Description: "TODO", Required: false, Type: "int"},
		},
	}

	if err := CreateFile(path, "order", "订单管理", "list", route); err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Resource != "order" {
		t.Errorf("Resource = %q, want order", cfg.Resource)
	}
	if cfg.Description != "订单管理" {
		t.Errorf("Description = %q", cfg.Description)
	}
	if _, ok := cfg.Routes["list"]; !ok {
		t.Error("list route not found")
	}
}

func TestCreateFile_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.yaml")
	os.WriteFile(path, []byte("test"), 0644)

	route := router.Route{Method: "POST", Path: "/test"}
	err := CreateFile(path, "test", "", "get", route)
	if err == nil {
		t.Error("expected file exists error")
	}
}
```

- [ ] **Step 6: 运行全部 yamlgen 测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/yamlgen/... -v`
Expected: ALL PASS

- [ ] **Step 7: 提交**

```bash
git add internal/yamlgen/
git commit -m "feat(yamlgen): add YAML route generator with file append/create"
```

---

### Task 3: route import CLI 命令

**Files:**
- Create: `cmd/route.go`
- Modify: `cmd/root.go:49` (在 `rootCmd.AddCommand(configCmd)` 后添加 `rootCmd.AddCommand(routeCmd)`)
- Create: `cmd/route_test.go`

- [ ] **Step 1: 写失败测试**

```go
// cmd/route_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
)

func TestRouteImport_Stdin_AppendToExisting(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "agent.yaml")

	initial := `resource: agent
description: AI智能体管理
routes:
    list:
        method: POST
        path: /admin/list
        description: 获取列表
`
	os.WriteFile(yamlPath, []byte(initial), 0644)

	curl := `curl 'https://kpapi-cs.ckjr001.com/api/admin/aiCreationCenter/modifyApp' -H 'content-type: application/json' --data-raw '{"name":"test","aikbId":3550}'`

	err := runImport(curl, yamlPath, "update", "", "")
	if err != nil {
		t.Fatalf("runImport() error = %v", err)
	}

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

	err := runImport(curl, yamlPath, "list", "order", "订单管理")
	if err != nil {
		t.Fatalf("runImport() error = %v", err)
	}

	data, _ := os.ReadFile(yamlPath)
	cfg, err := router.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Resource != "order" {
		t.Errorf("Resource = %q", cfg.Resource)
	}
	if _, ok := cfg.Routes["list"]; !ok {
		t.Error("list route not found")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/... -run TestRouteImport -v`
Expected: FAIL

- [ ] **Step 3: 实现 cmd/route.go**

```go
// cmd/route.go
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/curlparse"
	"github.com/childelins/ckjr-cli/internal/yamlgen"
)

var routeCmd = &cobra.Command{
	Use:   "route",
	Short: "路由配置管理",
}

var routeImportCmd = &cobra.Command{
	Use:   "import",
	Short: "从 curl 命令导入路由配置",
	Long:  "解析 curl 命令并生成 YAML 路由配置。支持 stdin 管道输入或 --curl 参数。",
	RunE: func(cmd *cobra.Command, args []string) error {
		curlStr, _ := cmd.Flags().GetString("curl")
		file, _ := cmd.Flags().GetString("file")
		name, _ := cmd.Flags().GetString("name")
		resource, _ := cmd.Flags().GetString("resource")
		resourceDesc, _ := cmd.Flags().GetString("resource-desc")

		// 从 stdin 读取
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

		if err := runImport(curlStr, file, name, resource, resourceDesc); err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "已添加路由 %s 到 %s\n", name, file)
		return nil
	},
}

// runImport 核心逻辑，方便测试
func runImport(curlStr, file, name, resource, resourceDesc string) error {
	result, err := curlparse.Parse(curlStr)
	if err != nil {
		return fmt.Errorf("curl 解析失败: %w", err)
	}

	// 自动推导 route name
	if name == "" {
		name = inferRouteName(result.Path)
	}

	route := yamlgen.GenerateRoute(result)

	// 判断追加还是新建
	if _, err := os.Stat(file); err == nil {
		return yamlgen.AppendToFile(file, name, route)
	}

	if resource == "" {
		return fmt.Errorf("新建文件需要通过 --resource 指定 resource 名称")
	}
	return yamlgen.CreateFile(file, resource, resourceDesc, name, route)
}

// inferRouteName 从 URL path 末段推导 route name
func inferRouteName(path string) string {
	// 取最后一个路径段
	parts := splitPath(path)
	if len(parts) == 0 {
		return "unknown"
	}
	last := parts[len(parts)-1]

	// 常见后缀映射
	prefixes := map[string]string{
		"modify": "update",
		"edit":   "update",
		"remove": "delete",
		"add":    "create",
		"query":  "list",
	}
	lower := toLower(last)
	for prefix, mapped := range prefixes {
		if len(lower) >= len(prefix) && lower[:len(prefix)] == prefix {
			return mapped
		}
	}

	// describe*/get*Info -> get
	if len(lower) >= 8 && lower[:8] == "describe" {
		return "get"
	}

	return last
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

func init() {
	routeImportCmd.Flags().String("curl", "", "curl 命令字符串")
	routeImportCmd.Flags().StringP("file", "f", "", "目标 YAML 文件路径")
	routeImportCmd.Flags().StringP("name", "n", "", "路由名称（默认从 URL 推导）")
	routeImportCmd.Flags().String("resource", "", "resource 名称（新建文件时必填）")
	routeImportCmd.Flags().String("resource-desc", "", "resource 描述")

	routeCmd.AddCommand(routeImportCmd)
}
```

- [ ] **Step 4: 在 cmd/root.go 注册 routeCmd**

在 `cmd/root.go` 的 `init()` 函数中，`rootCmd.AddCommand(configCmd)` 之后添加：

```go
	rootCmd.AddCommand(routeCmd)
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/... -run TestRouteImport -v`
Expected: ALL PASS

- [ ] **Step 6: 写 inferRouteName 测试**

```go
// 追加到 cmd/route_test.go

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
			got := inferRouteName(tt.path)
			if got != tt.want {
				t.Errorf("inferRouteName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 7: 运行全部 cmd 测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./cmd/... -v`
Expected: ALL PASS

- [ ] **Step 8: 提交**

```bash
git add cmd/route.go cmd/route_test.go cmd/root.go
git commit -m "feat(cmd): add route import command for curl-to-yaml conversion"
```

---

### Task 4: 全量测试验证

- [ ] **Step 1: 运行全部测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: 手动验证（用需求文档中的 curl 示例）**

Run: `cd /home/childelins/code/ckjr-cli && go build -o /tmp/ckjr-cli ./cmd/ckjr-cli/ && echo "curl 'https://kpapi-cs.ckjr001.com/api/admin/aiCreationCenter/modifyApp' --data-raw '{\"name\":\"test\",\"aikbId\":3550,\"desc\":\"描述\"}'" | /tmp/ckjr-cli route import -f /tmp/test-agent.yaml -n update --resource agent --resource-desc "AI智能体管理" && cat /tmp/test-agent.yaml`
Expected: 生成的 YAML 包含 resource: agent、routes.update 条目

- [ ] **Step 3: 提交（如有修复）**

```bash
git add -A
git commit -m "fix: address issues found in integration testing"
```
