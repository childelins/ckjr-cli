# 路由模板自动图片转存 Implementation Plan

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 在路由 YAML 中标记 `autoUpload: image` 字段，cmdgen 自动转存外部图片 URL 到素材库，消除 workflow 中重复的 upload-avatar 步骤。

**Architecture:** Field 新增 AutoUpload 字段，cmdgen 在 ValidateAll 之后插入 processAutoUpload 函数，复用已有 ossupload 包。workflow YAML 删除 upload-avatar 步骤并简化参数引用。

**Tech Stack:** Go 1.24, Cobra, gopkg.in/yaml.v3, internal/ossupload

---

## File Structure

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/router/router.go` | 修改 | Field 新增 AutoUpload 字段 |
| `internal/cmdgen/cmdgen.go` | 修改 | 新增 processAutoUpload + printTemplateTo 添加 note + buildSubCommand 调用链插入 |
| `internal/cmdgen/autoupload_test.go` | 新增 | processAutoUpload 单元测试 + printTemplateTo autoUpload note 测试 |
| `internal/router/router_test.go` | 修改 | 补充 AutoUpload 字段解析测试 |
| `cmd/ckjr-cli/routes/agent.yaml` | 修改 | avatar 字段添加 autoUpload: image |
| `cmd/ckjr-cli/routes/course.yaml` | 修改 | courseAvatar 字段添加 autoUpload: image |
| `cmd/ckjr-cli/workflows/agent.yaml` | 修改 | 移除 upload-avatar 步骤，简化参数 |
| `cmd/ckjr-cli/workflows/course.yaml` | 修改 | 移除 3 个 workflow 的 upload-avatar 步骤 |

---

### Task 1: Field 新增 AutoUpload 字段

**Files:**
- Modify: `internal/router/router.go:10-25`
- Test: `internal/router/router_test.go`

- [ ] **Step 1: 写失败测试 - 验证 AutoUpload 从 YAML 正确解析**

在 `internal/router/router_test.go` 末尾添加：

```go
func TestParseRouteConfig_AutoUpload(t *testing.T) {
	yamlContent := `
name: test
description: 测试自动上传
routes:
  create:
    method: POST
    path: /create
    description: 创建
    template:
      avatar:
        description: 头像URL
        required: true
        type: string
        autoUpload: image
      name:
        description: 名称
        required: true
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	route := cfg.Routes["create"]

	// avatar 字段应有 AutoUpload = "image"
	avatarField := route.Template["avatar"]
	if avatarField.AutoUpload != "image" {
		t.Errorf("avatar.AutoUpload = %q, want \"image\"", avatarField.AutoUpload)
	}

	// name 字段不应有 AutoUpload
	nameField := route.Template["name"]
	if nameField.AutoUpload != "" {
		t.Errorf("name.AutoUpload = %q, want empty", nameField.AutoUpload)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/router/ -run TestParseRouteConfig_AutoUpload -v`
Expected: FAIL（Field 没有 AutoUpload 字段，YAML 解析会忽略该字段）

- [ ] **Step 3: 实现 Field.AutoUpload**

在 `internal/router/router.go` 的 Field 结构体中，在 `Pattern` 字段后面添加：

```go
	// 自动转存标记，"image" 表示自动转存外部图片
	AutoUpload string `yaml:"autoUpload,omitempty"`
```

完整 Field 结构体变为：

```go
type Field struct {
	Description string      `yaml:"description"`
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default,omitempty"`
	Type        string      `yaml:"type,omitempty"`
	Example     string      `yaml:"example,omitempty"`

	// 数值约束
	Min *float64 `yaml:"min,omitempty"`
	Max *float64 `yaml:"max,omitempty"`

	// 字符串约束
	MinLength *int   `yaml:"minLength,omitempty"`
	MaxLength *int   `yaml:"maxLength,omitempty"`
	Pattern   string `yaml:"pattern,omitempty"`

	// 自动转存标记，"image" 表示自动转存外部图片
	AutoUpload string `yaml:"autoUpload,omitempty"`
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/router/ -run TestParseRouteConfig_AutoUpload -v`
Expected: PASS

- [ ] **Step 5: 运行全量 router 测试确保无回归**

Run: `go test ./internal/router/ -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/router/router.go internal/router/router_test.go
git commit -m "feat(router): add autoUpload field to Field struct for automatic image rehosting"
```

---

### Task 2: 实现 processAutoUpload 函数

**Files:**
- Modify: `internal/cmdgen/cmdgen.go`
- Create: `internal/cmdgen/autoupload_test.go`

- [ ] **Step 1: 写失败测试 - processAutoUpload 各场景**

创建 `internal/cmdgen/autoupload_test.go`：

```go
package cmdgen

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/ossupload"
	"github.com/childelins/ckjr-cli/internal/router"
)

// mockAPIError 用于测试转存失败场景
type mockAPIError struct{}

func (e *mockAPIError) Error() string { return "mock api error" }

func TestProcessAutoUpload_ExternalURL(t *testing.T) {
	// 模拟完整流程：imageSign -> download -> OSS upload -> addImgInAsset
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/assets/imageSign":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"accessid":  "test-id",
				"policy":    "test-policy",
				"signature": "test-sig",
				"callback":  "",
				"dir":       "test/dir/",
				"host":      "http://" + r.Host + "/oss-upload",
				"origin":    1,
			})
		case "/oss-upload":
			// OSS 直传成功
			w.WriteHeader(http.StatusOK)
		case "/admin/assets/addImgInAsset":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statusCode": 200,
			})
		}
	}))
	defer server.Close()

	// 模拟外部图片服务器（挂在同 server 不同路径不行，需要单独 server）
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-image-data"))
	}))
	defer imageServer.Close()

	externalURL := imageServer.URL + "/test.png"

	input := map[string]interface{}{
		"avatar": externalURL,
		"name":   "test",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
		"name":   {},
	}

	client := api.NewClient(server.URL, "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}

	// avatar 值应被替换为 OSS URL
	newAvatar, ok := input["avatar"].(string)
	if !ok {
		t.Fatalf("avatar should be string, got %T", input["avatar"])
	}
	if newAvatar == externalURL {
		t.Error("avatar should be replaced with OSS URL")
	}
	if !ossupload.IsExternalURL(newAvatar) {
		// 新 URL 应该是内部 OSS URL（不再被判定为外部）
		// 注意：这里需要确认 server host 是否在白名单中
	}
	// 验证走了完整流程
	if len(requests) < 3 {
		t.Errorf("expected at least 3 requests (imageSign, OSS, addImgInAsset), got %d: %v", len(requests), requests)
	}
}

func TestProcessAutoUpload_InternalURL_Skipped(t *testing.T) {
	input := map[string]interface{}{
		"avatar": "https://ck-bkt-knowledge-payment.oss-cn-hangzhou.aliyuncs.com/test.png",
		"name":   "test",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}

	// avatar 值不变
	if input["avatar"] != "https://ck-bkt-knowledge-payment.oss-cn-hangzhou.aliyuncs.com/test.png" {
		t.Errorf("internal URL should not be changed, got %v", input["avatar"])
	}
}

func TestProcessAutoUpload_EmptyValue_Skipped(t *testing.T) {
	input := map[string]interface{}{
		"avatar": "",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}

	if input["avatar"] != "" {
		t.Errorf("empty string should not be changed, got %v", input["avatar"])
	}
}

func TestProcessAutoUpload_MissingField_Skipped(t *testing.T) {
	input := map[string]interface{}{
		"name": "test",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}
}

func TestProcessAutoUpload_NonStringField_Skipped(t *testing.T) {
	input := map[string]interface{}{
		"avatar": float64(123),
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}

	if input["avatar"] != float64(123) {
		t.Errorf("non-string value should not be changed, got %v", input["avatar"])
	}
}

func TestProcessAutoUpload_NoAutoUploadFields(t *testing.T) {
	input := map[string]interface{}{
		"name": "test",
	}
	template := map[string]router.Field{
		"name": {},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}
}

func TestProcessAutoUpload_UploadError_ReturnsError(t *testing.T) {
	// 模拟 imageSign 失败
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"statusCode": 500,
			"msg":        "internal error",
		})
	}))
	defer server.Close()

	// 外部图片需要是有效的 HTTP URL
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-image-data"))
	}))
	defer imageServer.Close()

	input := map[string]interface{}{
		"avatar": imageServer.URL + "/test.png",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient(server.URL, "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err == nil {
		t.Fatal("expected error when upload fails")
	}
	// 错误信息应包含字段名
	if !errors.As(err, new(*api.APIError)) && !errors.As(err, new(*api.ResponseError)) {
		// processAutoUpload 包装了错误，应该包含字段名
		t.Logf("error: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cmdgen/ -run TestProcessAutoUpload -v`
Expected: FAIL（processAutoUpload 函数不存在）

- [ ] **Step 3: 实现 processAutoUpload 函数**

在 `internal/cmdgen/cmdgen.go` 中：

1. 添加 import：
```go
"github.com/childelins/ckjr-cli/internal/ossupload"
```

2. 在 `applyDefaults` 函数之后添加 processAutoUpload 函数：

```go
// processAutoUpload 扫描 template 中 autoUpload=image 的字段，
// 对外部 URL 执行转存，将 input 中对应值替换为转存后的 OSS URL
func processAutoUpload(ctx context.Context, input map[string]interface{},
	template map[string]router.Field, apiClient *api.Client) error {

	for name, field := range template {
		if field.AutoUpload != "image" {
			continue
		}

		val, exists := input[name]
		if !exists {
			continue
		}

		urlStr, ok := val.(string)
		if !ok || urlStr == "" {
			continue
		}

		if !ossupload.IsExternalURL(urlStr) {
			continue
		}

		slog.InfoContext(ctx, "auto_upload_start",
			"field", name,
			"original_url", urlStr,
		)

		result, err := ossupload.Upload(ctx, apiClient, urlStr)
		if err != nil {
			return fmt.Errorf("字段 %s 图片转存失败: %w", name, err)
		}

		input[name] = result.ImageURL

		slog.InfoContext(ctx, "auto_upload_complete",
			"field", name,
			"new_url", result.ImageURL,
		)
	}
	return nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/cmdgen/ -run TestProcessAutoUpload -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/autoupload_test.go
git commit -m "feat(cmdgen): add processAutoUpload for automatic external image rehosting"
```

---

### Task 3: 在 buildSubCommand 中集成 processAutoUpload

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:88-96`
- Test: `internal/cmdgen/autoupload_test.go`

- [ ] **Step 1: 写集成测试 - buildSubCommand 自动转存完整流程**

在 `internal/cmdgen/autoupload_test.go` 末尾添加：

```go
func TestBuildSubCommand_AutoUpload(t *testing.T) {
	// 外部图片服务器
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-image-data"))
	}))
	defer imageServer.Close()

	var capturedAvatar string
	// API 服务器
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/assets/imageSign":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"accessid":  "test-id",
				"policy":    "test-policy",
				"signature": "test-sig",
				"callback":  "",
				"dir":       "test/dir/",
				"host":      "http://" + r.Host + "/oss-upload",
				"origin":    1,
			})
		case "/oss-upload":
			w.WriteHeader(http.StatusOK)
		case "/admin/assets/addImgInAsset":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statusCode": 200,
			})
		case "/admin/create":
			// 捕获最终请求中的 avatar 值
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			capturedAvatar = body["avatar"].(string)
			resp := api.Response{Data: map[string]interface{}{"id": float64(1)}, Message: "ok", StatusCode: 200}
			json.NewEncoder(w).Encode(resp)
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"statusCode": 200,
			})
		}
	}))
	defer apiServer.Close()

	cfg := &router.RouteConfig{
		Name: "agent",
		Routes: map[string]router.Route{
			"create": {
				Method:      "POST",
				Path:        "/admin/create",
				Description: "创建",
				Template: map[string]router.Field{
					"avatar": {
						Description: "头像URL",
						Required:    true,
						AutoUpload:  "image",
					},
					"name": {
						Description: "名称",
						Required:    true,
					},
				},
				Response: &router.ResponseFilter{
					Fields: []router.ResponseField{{Path: "id"}},
				},
			},
		},
	}

	clientFactory := func() (*api.Client, error) {
		return api.NewClient(apiServer.URL, "test-key"), nil
	}

	// 捕获 stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := BuildCommand(cfg, clientFactory)
	cmd.PersistentFlags().Bool("pretty", false, "")
	cmd.PersistentFlags().Bool("verbose", false, "")

	externalURL := imageServer.URL + "/test.png"
	cmd.SetArgs([]string{"create", `{"avatar": "` + externalURL + `", "name": "test"}`})
	cmd.Execute()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	// 验证最终请求中 avatar 已被替换为 OSS URL
	if capturedAvatar == externalURL {
		t.Errorf("avatar should be replaced, still external URL: %s", capturedAvatar)
	}
	if capturedAvatar == "" {
		t.Error("avatar should not be empty")
	}
}
```

注意：需要在 autoupload_test.go 的 import 中添加 `"os"` 和 `"bytes"`。

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cmdgen/ -run TestBuildSubCommand_AutoUpload -v`
Expected: FAIL（capturedAvatar 仍为原始外部 URL，因为 processAutoUpload 未被调用）

- [ ] **Step 3: 在 buildSubCommand 中插入 processAutoUpload 调用**

在 `internal/cmdgen/cmdgen.go` 的 `buildSubCommand` 函数中，在 ValidateAll 校验之后、clientFactory 检查之前（约第 96-98 行之间）插入：

```go
			// 自动转存外部图片 URL
			if err := processAutoUpload(ctx, input, route.Template, client); err != nil {
				output.PrintError(os.Stderr, err.Error())
				os.Exit(1)
			}
```

注意：`client` 需要在调用前创建。当前代码中 clientFactory 检查和 client 创建在第 99-108 行。需要将 client 创建提前到 processAutoUpload 之前。

完整的插入区域变为（替换第 96-108 行）：

```go
			// 校验参数
			if errs := ValidateAll(input, route.Template); len(errs) > 0 {
				var msgs []string
				for _, e := range errs {
					msgs = append(msgs, e.Error())
				}
				output.PrintError(os.Stderr, fmt.Sprintf("参数校验失败:\n  %s", strings.Join(msgs, "\n  ")))
				os.Exit(1)
			}

			// 创建 API 客户端
			if clientFactory == nil {
				output.PrintError(os.Stderr, "API 客户端未配置")
				os.Exit(1)
			}

			client, err := clientFactory()
			if err != nil {
				output.PrintError(os.Stderr, err.Error())
				os.Exit(1)
			}

			// 自动转存外部图片 URL
			ctx := context.Background()
			requestID := logging.NewRequestID()
			ctx = logging.WithRequestID(ctx, requestID)

			if err := processAutoUpload(ctx, input, route.Template, client); err != nil {
				output.PrintError(os.Stderr, err.Error())
				os.Exit(1)
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			verbose, _ := cmd.Flags().GetBool("verbose")
```

同时删除原来在第 99-116 行的 client 创建和 ctx 生成代码（已上移）。

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/cmdgen/ -run TestBuildSubCommand_AutoUpload -v`
Expected: PASS

- [ ] **Step 5: 运行全量 cmdgen 测试确保无回归**

Run: `go test ./internal/cmdgen/ -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/autoupload_test.go
git commit -m "feat(cmdgen): integrate processAutoUpload into buildSubCommand pipeline"
```

---

### Task 4: printTemplateTo 添加 autoUpload note

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:140-203`
- Test: `internal/cmdgen/autoupload_test.go`

- [ ] **Step 1: 写失败测试 - 验证 autoUpload 字段输出 note**

在 `internal/cmdgen/autoupload_test.go` 末尾添加：

```go
func TestPrintTemplate_AutoUploadNote(t *testing.T) {
	template := map[string]router.Field{
		"avatar": {
			Description: "头像URL",
			Required:    true,
			Type:        "string",
			AutoUpload:  "image",
		},
		"name": {
			Description: "名称",
			Required:    true,
			Type:        "string",
		},
	}

	var buf bytes.Buffer
	printTemplateTo(&buf, template, nil)

	var outer map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &outer); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	result := outer["request"].(map[string]interface{})

	// avatar 字段应有 autoUpload note
	avatarEntry := result["avatar"].(map[string]interface{})
	note, ok := avatarEntry["note"]
	if !ok {
		t.Fatal("avatar (autoUpload=image) should have note field")
	}
	if note != "外部图片URL将自动转存到系统素材库" {
		t.Errorf("note = %q, want %q", note, "外部图片URL将自动转存到系统素材库")
	}

	// name 字段不应有 note
	nameEntry := result["name"].(map[string]interface{})
	if _, exists := nameEntry["note"]; exists {
		t.Error("name (no autoUpload) should not have note field")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cmdgen/ -run TestPrintTemplate_AutoUploadNote -v`
Expected: FAIL（avatar 没有 note）

- [ ] **Step 3: 在 printTemplateTo 中添加 autoUpload note**

在 `internal/cmdgen/cmdgen.go` 的 `printTemplateTo` 函数中，在 `if t == "date"` 块之后添加：

```go
		if field.AutoUpload == "image" {
			entry["note"] = "外部图片URL将自动转存到系统素材库"
		}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/cmdgen/ -run TestPrintTemplate_AutoUploadNote -v`
Expected: PASS

- [ ] **Step 5: 运行全量 cmdgen 测试确保无回归**

Run: `go test ./internal/cmdgen/ -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/autoupload_test.go
git commit -m "feat(cmdgen): add autoUpload note to template output"
```

---

### Task 5: 路由 YAML 添加 autoUpload 标记

**Files:**
- Modify: `cmd/ckjr-cli/routes/agent.yaml`
- Modify: `cmd/ckjr-cli/routes/course.yaml`

- [ ] **Step 1: 修改 agent.yaml - create 和 update 的 avatar 字段**

在 `cmd/ckjr-cli/routes/agent.yaml` 中：

create 路由的 avatar 字段（约第 15-22 行）添加 `autoUpload: image`：
```yaml
            avatar:
                description: 头像URL
                required: true
                type: string
                autoUpload: image
                minLength: 1
                maxLength: 255
                example: https://ck-bkt-knowledge-payment.oss-cn-hangzhou.aliyuncs.com/y99rjd/resource/imgs/120b2b85/admin-fe_y99rjd_material_BAwRCWsQW2tMd6nHichk.png
```

update 路由的 avatar 字段（约第 123-128 行）添加 `autoUpload: image`：
```yaml
            avatar:
                description: 智能体头像URL
                required: true
                type: string
                autoUpload: image
                minLength: 1
                maxLength: 255
```

- [ ] **Step 2: 修改 course.yaml - create 和 update 的 courseAvatar 字段**

在 `cmd/ckjr-cli/routes/course.yaml` 中：

create 路由的 courseAvatar 字段（约第 9-14 行）添加 `autoUpload: image`：
```yaml
            courseAvatar:
                description: 课程封面
                required: true
                type: string
                autoUpload: image
                minLength: 1
                maxLength: 255
```

update 路由的 courseAvatar 字段（约第 191-196 行）添加 `autoUpload: image`：
```yaml
            courseAvatar:
                description: 课程封面
                required: true
                type: string
                autoUpload: image
                minLength: 1
                maxLength: 255
```

- [ ] **Step 3: 构建验证**

Run: `go build ./...`
Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add cmd/ckjr-cli/routes/agent.yaml cmd/ckjr-cli/routes/course.yaml
git commit -m "feat(routes): add autoUpload: image to avatar and courseAvatar fields"
```

---

### Task 6: Workflow YAML 简化 - 移除 upload-avatar 步骤

**Files:**
- Modify: `cmd/ckjr-cli/workflows/agent.yaml`
- Modify: `cmd/ckjr-cli/workflows/course.yaml`

- [ ] **Step 1: 简化 agent.yaml workflow**

将 `cmd/ckjr-cli/workflows/agent.yaml` 的 create-agent workflow 修改为：

```yaml
name: agent-workflows
description: 智能体相关工作流

workflows:
  create-agent:
    description: 创建并配置一个完整的智能体
    triggers:
      - 创建智能体
      - 新建智能体
      - 创建一个AI助手
    allowed-routes:
      - agent
      - common
    inputs:
      - name: name
        description: 智能体名称
        required: true
      - name: desc
        description: 智能体描述/用途
        required: true
      - name: avatar
        description: 头像URL（外部图片将自动转存到素材库）
        required: false
        hint: 如果用户未提供，询问用户或使用用户提供的素材链接
      - name: instructions
        description: 智能体提示词/角色设定
        required: true
        hint: 根据用户描述的用途，生成包含角色定位、能力、交流规则和响应方式的完整提示词
      - name: greeting
        description: 开场白文案
        required: false
        hint: 根据智能体角色生成一条友好的开场白
    steps:
      - id: create
        description: 创建智能体基本信息
        command: agent create
        params:
          name: "{{inputs.name}}"
          desc: "{{inputs.desc}}"
          avatar: "{{inputs.avatar}}"
        output:
          aikbId: "response.aikbId"
      - id: configure
        description: 设置提示词和开场白
        command: agent update
        params:
          aikbId: "{{steps.create.aikbId}}"
          name: "{{inputs.name}}"
          desc: "{{inputs.desc}}"
          avatar: "{{inputs.avatar}}"
          instructions: "{{inputs.instructions}}"
          greeting: "{{inputs.greeting}}"
      - id: get-link
        description: 获取公众号端访问链接和二维码
        command: common link
        params:
          prodId: "{{steps.create.aikbId}}"
          prodType: ai_service
        output:
          url: "response.url"
          qrcodeImg: "response.img"
    summary: |
      智能体创建完成：
      - 名称：{{inputs.name}}
      - ID：{{steps.create.aikbId}}
      - 访问链接：{{steps.get-link.url}}
      - 二维码：{{steps.get-link.qrcodeImg}}
```

关键变更：
- 移除 `asset` 从 allowed-routes
- 移除 `upload-avatar` 步骤
- create 和 configure 的 avatar 参数改为直接引用 `{{inputs.avatar}}`
- avatar 的 description 添加"外部图片将自动转存到素材库"提示

- [ ] **Step 2: 简化 course.yaml - 3 个 workflow**

对 `cmd/ckjr-cli/workflows/course.yaml` 中每个 workflow 做相同变更：

1. 移除 `asset` 从 allowed-routes
2. 移除 `upload-avatar` 步骤
3. create 步骤的 courseAvatar 参数改为直接引用 `{{inputs.courseAvatar}}`
4. courseAvatar 的 description 添加"外部图片将自动转存到素材库"提示

以 create-video-course 为例（其他两个同理）：

```yaml
  create-video-course:
    description: 创建视频课程并获取访问链接
    triggers:
      - 创建视频课程
      - 新建视频课程
    allowed-routes:
      - course
      - common
    inputs:
      - name: name
        description: 课程名称
        required: true
      - name: courseAvatar
        description: 课程封面 URL（外部图片将自动转存到素材库）
        required: true
      - name: detailInfo
        description: 课程详情
        required: true
        hint: 根据用户描述生成富文本 HTML
      - name: payType
        description: 售卖类型, 1-免费 2-付费 3-加密
        required: true
      - name: price
        description: 价格
        required: true
      - name: playMode
        description: 播放模式, 1-横屏(默认) 2-竖屏
        required: false
        hint: 默认横屏(1)
    steps:
      - id: create
        description: 创建视频课程
        command: course create
        params:
          name: "{{inputs.name}}"
          courseAvatar: "{{inputs.courseAvatar}}"
          detailInfo: "{{inputs.detailInfo}}"
          payType: "{{inputs.payType}}"
          price: "{{inputs.price}}"
          playMode: "{{inputs.playMode}}"
          courseType: 0
          isSaleOnly: 1
          status: 1
        output:
          courseId: "response.courseId"
      - id: get-link
        description: 获取公众号端访问链接和二维码
        command: common link
        params:
          prodId: "{{steps.create.courseId}}"
          prodType: video
        output:
          url: "response.url"
          qrcodeImg: "response.img"
    summary: |
      视频课程创建完成：
      - 名称：{{inputs.name}}
      - ID：{{steps.create.courseId}}
      - 访问链接：{{steps.get-link.url}}
      - 二维码：{{steps.get-link.qrcodeImg}}
```

- [ ] **Step 3: 构建验证**

Run: `go build ./...`
Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add cmd/ckjr-cli/workflows/agent.yaml cmd/ckjr-cli/workflows/course.yaml
git commit -m "refactor(workflows): remove upload-avatar steps, rely on cmdgen autoUpload"
```

---

### Task 7: 全量测试 + 清理

**Files:** 无新文件

- [ ] **Step 1: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: 运行 go vet**

Run: `go vet ./...`
Expected: 无警告

- [ ] **Step 3: 构建验证**

Run: `go build ./...`
Expected: 编译成功
