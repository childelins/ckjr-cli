# 路由路径参数替换 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 在 YAML 路由的 template 中通过 `type: path` 声明路径参数，请求时自动替换 path 中的 `{xxx}` 占位符。

**Architecture:** 新增 `internal/cmdgen/pathparam.go` 处理路径参数提取与替换逻辑，修改 `validate.go` 跳过 `type: path` 字段的校验，在 `cmdgen.go` 的 `buildSubCommand` 中于 `ValidateAll` 之前调用 `ReplacePath`。

**Tech Stack:** Go, regexp, net/url, cobra

**Source Spec:** `docs/superpowers/specs/2026-03-31-path-params-design.md`

---

## File Structure

| 操作 | 文件 | 职责 |
|------|------|------|
| 新建 | `internal/cmdgen/pathparam.go` | 路径参数提取、替换、错误类型 |
| 新建 | `internal/cmdgen/pathparam_test.go` | pathparam 单元测试 |
| 修改 | `internal/cmdgen/validate.go:46-61,37-44,104-165` | 跳过 type: path 字段 |
| 修改 | `internal/cmdgen/validate_test.go` | 补充 path 类型跳过测试 |
| 修改 | `internal/cmdgen/cmdgen.go:79-115` | 集成 ReplacePath |
| 修改 | `internal/cmdgen/cmdgen_test.go` | 补充集成测试 |
| 修改 | `cmd/ckjr-cli/routes/course.yaml:83-132` | update 路由添加 courseId 字段 |

不修改: `internal/router/router.go`（Field.Type 已是 string）、`internal/api/client.go`

---

## Phase 1: IsPathParam + extractPlaceholders

### Task 1: IsPathParam 函数

**Files:**
- Create: `internal/cmdgen/pathparam.go`
- Create: `internal/cmdgen/pathparam_test.go`

- [ ] **Step 1: 写测试**

```go
// pathparam_test.go
package cmdgen

import (
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
)

func TestIsPathParam_True(t *testing.T) {
	field := router.Field{Type: "path"}
	if !IsPathParam(field) {
		t.Error("expected true for type=path")
	}
}

func TestIsPathParam_False(t *testing.T) {
	for _, typ := range []string{"string", "int", "float", "", "bool"} {
		if IsPathParam(router.Field{Type: typ}) {
			t.Errorf("expected false for type=%s", typ)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestIsPathParam -v`
Expected: FAIL（函数未定义）

- [ ] **Step 3: 实现 IsPathParam**

```go
// pathparam.go
package cmdgen

import "github.com/childelins/ckjr-cli/internal/router"

// IsPathParam 判断字段是否为路径参数
func IsPathParam(field router.Field) bool {
	return field.Type == "path"
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestIsPathParam -v`
Expected: PASS

### Task 2: extractPlaceholders 函数

**Files:**
- Modify: `internal/cmdgen/pathparam.go`
- Modify: `internal/cmdgen/pathparam_test.go`

- [ ] **Step 1: 写测试**

```go
// 追加到 pathparam_test.go
func TestExtractPlaceholders_None(t *testing.T) {
	result := extractPlaceholders("/admin/courses")
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestExtractPlaceholders_Single(t *testing.T) {
	result := extractPlaceholders("/admin/courses/{courseId}")
	if len(result) != 1 || result[0] != "courseId" {
		t.Errorf("expected [courseId], got %v", result)
	}
}

func TestExtractPlaceholders_Multiple(t *testing.T) {
	result := extractPlaceholders("/courses/{courseId}/chapters/{chapterId}")
	if len(result) != 2 || result[0] != "courseId" || result[1] != "chapterId" {
		t.Errorf("expected [courseId chapterId], got %v", result)
	}
}

func TestExtractPlaceholders_Duplicate(t *testing.T) {
	result := extractPlaceholders("/a/{id}/b/{id}")
	if len(result) != 1 || result[0] != "id" {
		t.Errorf("expected [id], got %v", result)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestExtractPlaceholders -v`
Expected: FAIL

- [ ] **Step 3: 实现 extractPlaceholders**

```go
// 追加到 pathparam.go
import "regexp"

var pathParamRe = regexp.MustCompile(`\{(\w+)\}`)

// extractPlaceholders 从 path 中提取 {xxx} 占位符名，去重保序
func extractPlaceholders(path string) []string {
	matches := pathParamRe.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var params []string
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			params = append(params, name)
		}
	}
	return params
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestExtractPlaceholders -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/pathparam.go internal/cmdgen/pathparam_test.go
git commit -m "feat(cmdgen): add IsPathParam and extractPlaceholders"
```

---

## Phase 2: PathParamError + ReplacePath

### Task 3: PathParamError 错误类型

**Files:**
- Modify: `internal/cmdgen/pathparam.go`
- Modify: `internal/cmdgen/pathparam_test.go`

- [ ] **Step 1: 写测试**

```go
func TestPathParamError_Missing(t *testing.T) {
	e := &PathParamError{Missing: []string{"courseId"}}
	expected := "缺少路径参数: courseId"
	if e.Error() != expected {
		t.Errorf("got %q, want %q", e.Error(), expected)
	}
}

func TestPathParamError_Undeclared(t *testing.T) {
	e := &PathParamError{Undeclared: []string{"courseId"}}
	expected := "路径占位符 {courseId} 未在 template 中声明为 type: path"
	if e.Error() != expected {
		t.Errorf("got %q, want %q", e.Error(), expected)
	}
}

func TestPathParamError_Both(t *testing.T) {
	e := &PathParamError{Undeclared: []string{"a"}, Missing: []string{"b"}}
	msg := e.Error()
	if !strings.Contains(msg, "路径占位符") || !strings.Contains(msg, "缺少路径参数") {
		t.Errorf("unexpected message: %s", msg)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestPathParamError -v`
Expected: FAIL

- [ ] **Step 3: 实现 PathParamError**

```go
// 追加到 pathparam.go
import (
	"fmt"
	"strings"
)

// PathParamError 路径参数错误
type PathParamError struct {
	Missing    []string
	Undeclared []string
}

func (e *PathParamError) Error() string {
	var parts []string
	if len(e.Undeclared) > 0 {
		parts = append(parts, fmt.Sprintf(
			"路径占位符 {%s} 未在 template 中声明为 type: path",
			strings.Join(e.Undeclared, "}, {")))
	}
	if len(e.Missing) > 0 {
		parts = append(parts, fmt.Sprintf(
			"缺少路径参数: %s", strings.Join(e.Missing, ", ")))
	}
	return strings.Join(parts, "; ")
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestPathParamError -v`
Expected: PASS

### Task 4: ReplacePath 函数

**Files:**
- Modify: `internal/cmdgen/pathparam.go`
- Modify: `internal/cmdgen/pathparam_test.go`

- [ ] **Step 1: 写测试 -- 无占位符**

```go
func TestReplacePath_NoPlaceholders(t *testing.T) {
	input := map[string]interface{}{"name": "test"}
	tmpl := map[string]router.Field{"name": {Type: "string"}}
	path, err := ReplacePath("/admin/courses", input, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/admin/courses" {
		t.Errorf("got %q", path)
	}
	if _, ok := input["name"]; !ok {
		t.Error("input should not be modified when no placeholders")
	}
}
```

- [ ] **Step 2: 写测试 -- 单参数替换**

```go
func TestReplacePath_SingleParam(t *testing.T) {
	input := map[string]interface{}{"courseId": float64(123), "name": "test"}
	tmpl := map[string]router.Field{
		"courseId": {Type: "path", Required: true},
		"name":     {Type: "string"},
	}
	path, err := ReplacePath("/admin/courses/{courseId}", input, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/admin/courses/123" {
		t.Errorf("got %q", path)
	}
	// courseId 应从 input 中移除
	if _, ok := input["courseId"]; ok {
		t.Error("courseId should be removed from input")
	}
	// name 应保留
	if _, ok := input["name"]; !ok {
		t.Error("name should remain in input")
	}
}
```

- [ ] **Step 3: 写测试 -- 多参数替换**

```go
func TestReplacePath_MultipleParams(t *testing.T) {
	input := map[string]interface{}{
		"courseId":  float64(1),
		"chapterId": float64(2),
		"title":    "hello",
	}
	tmpl := map[string]router.Field{
		"courseId":  {Type: "path", Required: true},
		"chapterId": {Type: "path", Required: true},
		"title":    {Type: "string"},
	}
	path, err := ReplacePath("/courses/{courseId}/chapters/{chapterId}", input, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/courses/1/chapters/2" {
		t.Errorf("got %q", path)
	}
	if _, ok := input["courseId"]; ok {
		t.Error("courseId should be removed")
	}
	if _, ok := input["chapterId"]; ok {
		t.Error("chapterId should be removed")
	}
}
```

- [ ] **Step 4: 写测试 -- 缺失参数**

```go
func TestReplacePath_MissingParam(t *testing.T) {
	input := map[string]interface{}{"name": "test"}
	tmpl := map[string]router.Field{
		"courseId": {Type: "path", Required: true},
	}
	_, err := ReplacePath("/admin/courses/{courseId}", input, tmpl)
	if err == nil {
		t.Fatal("expected error")
	}
	pErr, ok := err.(*PathParamError)
	if !ok {
		t.Fatalf("expected *PathParamError, got %T", err)
	}
	if len(pErr.Missing) != 1 || pErr.Missing[0] != "courseId" {
		t.Errorf("unexpected Missing: %v", pErr.Missing)
	}
}
```

- [ ] **Step 5: 写测试 -- 占位符未声明**

```go
func TestReplacePath_UndeclaredPlaceholder(t *testing.T) {
	input := map[string]interface{}{"courseId": float64(1)}
	tmpl := map[string]router.Field{
		"courseId": {Type: "int"}, // type 不是 path
	}
	_, err := ReplacePath("/admin/courses/{courseId}", input, tmpl)
	if err == nil {
		t.Fatal("expected error")
	}
	pErr, ok := err.(*PathParamError)
	if !ok {
		t.Fatalf("expected *PathParamError, got %T", err)
	}
	if len(pErr.Undeclared) != 1 || pErr.Undeclared[0] != "courseId" {
		t.Errorf("unexpected Undeclared: %v", pErr.Undeclared)
	}
}
```

- [ ] **Step 6: 写测试 -- nil 值视为缺失**

```go
func TestReplacePath_NilValue(t *testing.T) {
	input := map[string]interface{}{"courseId": nil}
	tmpl := map[string]router.Field{
		"courseId": {Type: "path", Required: true},
	}
	_, err := ReplacePath("/admin/courses/{courseId}", input, tmpl)
	if err == nil {
		t.Fatal("expected error for nil value")
	}
	pErr := err.(*PathParamError)
	if len(pErr.Missing) != 1 {
		t.Errorf("expected 1 missing, got %v", pErr.Missing)
	}
}
```

- [ ] **Step 7: 写测试 -- URL 编码特殊字符**

```go
func TestReplacePath_SpecialChars(t *testing.T) {
	input := map[string]interface{}{"name": "hello world/test"}
	tmpl := map[string]router.Field{
		"name": {Type: "path", Required: true},
	}
	path, err := ReplacePath("/items/{name}", input, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/items/hello%20world%2Ftest" {
		t.Errorf("got %q", path)
	}
}
```

- [ ] **Step 8: 运行所有 ReplacePath 测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestReplacePath -v`
Expected: FAIL

- [ ] **Step 9: 实现 ReplacePath**

```go
// 追加到 pathparam.go
import "net/url"

// ReplacePath 将 path 中 {xxx} 替换为 input 中的值
// 仅替换 template 中 type: path 的字段
// 替换后从 input 中移除路径参数
func ReplacePath(path string, input map[string]interface{}, template map[string]router.Field) (string, error) {
	placeholders := extractPlaceholders(path)
	if len(placeholders) == 0 {
		return path, nil
	}

	// 校验：占位符必须在 template 中声明为 type: path
	var undeclared []string
	for _, name := range placeholders {
		field, exists := template[name]
		if !exists || !IsPathParam(field) {
			undeclared = append(undeclared, name)
		}
	}
	if len(undeclared) > 0 {
		return "", &PathParamError{Undeclared: undeclared}
	}

	// 校验：所有路径参数在 input 中存在且非 nil
	var missing []string
	for _, name := range placeholders {
		val, exists := input[name]
		if !exists || val == nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return "", &PathParamError{Missing: missing}
	}

	// 执行替换
	result := pathParamRe.ReplaceAllStringFunc(path, func(match string) string {
		name := match[1 : len(match)-1]
		val := input[name]
		return url.PathEscape(fmt.Sprintf("%v", val))
	})

	// 从 input 中移除路径参数
	for _, name := range placeholders {
		delete(input, name)
	}

	return result, nil
}
```

- [ ] **Step 10: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run "TestReplacePath|TestPathParamError|TestIsPathParam|TestExtractPlaceholders" -v`
Expected: ALL PASS

- [ ] **Step 11: 提交**

```bash
git add internal/cmdgen/pathparam.go internal/cmdgen/pathparam_test.go
git commit -m "feat(cmdgen): add ReplacePath with type:path support"
```

---

## Phase 3: validate.go 修改

### Task 5: validateTypes 跳过 type: path

**Files:**
- Modify: `internal/cmdgen/validate.go:46-61`
- Modify: `internal/cmdgen/validate_test.go`

- [ ] **Step 1: 写测试**

```go
// 追加到 validate_test.go
func TestValidateTypes_SkipsPathParam(t *testing.T) {
	input := map[string]interface{}{
		"courseId": float64(123),
	}
	template := map[string]router.Field{
		"courseId": {Type: "path", Required: true},
	}
	errs := validateTypes(input, template)
	if len(errs) != 0 {
		t.Errorf("expected no errors for type=path, got %v", errs)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestValidateTypes_SkipsPathParam -v`
Expected: FAIL（当前 validateType 的 default 分支会报 "未知类型 path"）

- [ ] **Step 3: 修改 validateTypes**

在 `validate.go` 的 `validateTypes` 函数中，跳过条件从 `field.Type == ""` 改为 `field.Type == "" || IsPathParam(field)`：

```go
// validate.go:validateTypes
if field.Type == "" || IsPathParam(field) {
    continue
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestValidateTypes -v`
Expected: PASS

### Task 6: validateRequiredErrors 跳过 type: path

**Files:**
- Modify: `internal/cmdgen/validate.go:37-44`
- Modify: `internal/cmdgen/validate_test.go`

- [ ] **Step 1: 写测试**

```go
func TestValidateRequiredErrors_SkipsPathParam(t *testing.T) {
	// courseId 已被 ReplacePath 从 input 中移除的场景
	input := map[string]interface{}{"name": "test"}
	template := map[string]router.Field{
		"courseId": {Type: "path", Required: true},
		"name":     {Type: "string", Required: true},
	}
	errs := validateRequiredErrors(input, template)
	// courseId 虽然 required 且不在 input 中，但 type=path 应被跳过
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestValidateRequiredErrors_SkipsPathParam -v`
Expected: FAIL（当前会报 courseId 为必填字段）

- [ ] **Step 3: 修改 validateRequiredErrors**

在 `validate.go` 的 `validateRequiredErrors` 中调用 `validateRequired` 前，需要过滤掉 path 类型字段。最简单的做法是直接在 `validateRequiredErrors` 中内联并跳过：

在函数中为 path 类型字段添加 continue：

```go
func validateRequiredErrors(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	var errs []*FieldValidationError
	for name, field := range template {
		if IsPathParam(field) {
			continue
		}
		if field.Required {
			if _, exists := input[name]; !exists {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: "为必填字段",
				})
			}
		}
	}
	return errs
}
```

注意：这替换了原来调用 `validateRequired()`（cmdgen.go:194）的间接方式。`validateRequired()` 函数如无其他调用者可保留不动。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run "TestValidateRequired" -v`
Expected: PASS

### Task 7: validateConstraints 显式跳过 type: path

**Files:**
- Modify: `internal/cmdgen/validate.go:104-165`
- Modify: `internal/cmdgen/validate_test.go`

- [ ] **Step 1: 写测试**

```go
func TestValidateConstraints_SkipsPathParam(t *testing.T) {
	input := map[string]interface{}{
		"courseId": float64(123),
	}
	template := map[string]router.Field{
		"courseId": {Type: "path", Required: true, Min: floatPtr(1)},
	}
	errs := validateConstraints(input, template)
	if len(errs) != 0 {
		t.Errorf("expected no errors for path param, got %v", errs)
	}
}
```

- [ ] **Step 2: 运行测试确认通过或失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestValidateConstraints_SkipsPathParam -v`

注意：由于 validateConstraints 的 switch 不命中 "path"，此测试可能已通过。如已通过则添加显式跳过以提高可读性。

- [ ] **Step 3: 修改 validateConstraints（可选，提高可读性）**

在 `validateConstraints` 循环开头添加跳过：

```go
for name, field := range template {
    if IsPathParam(field) {
        continue
    }
    // ... 原有逻辑
}
```

- [ ] **Step 4: 运行全部校验测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run "TestValidate" -v`
Expected: ALL PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/validate.go internal/cmdgen/validate_test.go
git commit -m "feat(cmdgen): skip type:path fields in validation"
```

---

## Phase 4: cmdgen.go 集成

### Task 8: buildSubCommand 集成 ReplacePath

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:79-115`
- Modify: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 写集成测试 -- 路径参数正常替换**

```go
// 追加到 cmdgen_test.go
func TestBuildSubCommand_PathParam(t *testing.T) {
	var capturedPath string
	var capturedBody map[string]interface{}

	route := router.Route{
		Method: "PUT",
		Path:   "/admin/courses/{courseId}",
		Template: map[string]router.Field{
			"courseId": {Type: "path", Required: true},
			"name":    {Type: "string", Required: true},
		},
	}

	factory := func() (*api.Client, error) {
		return api.NewTestClient(func(method, path string, body map[string]interface{}) (interface{}, error) {
			capturedPath = path
			capturedBody = body
			return map[string]interface{}{"ok": true}, nil
		}), nil
	}

	cmd := buildSubCommand("course", "update", route, factory)
	cmd.SetArgs([]string{`{"courseId": 123, "name": "Go入门"}`})
	// 注意：实际测试方式取决于现有 cmdgen_test.go 的测试模式
	// 如果使用 cmd.Execute()，需要捕获 os.Exit
	// 参考现有测试文件的方式
}
```

注意：此步骤的测试代码需要根据现有 `cmdgen_test.go` 的测试模式（如 mock client 的创建方式）进行适配。执行时先阅读现有测试确定模式。

- [ ] **Step 2: 修改 buildSubCommand**

在 `cmdgen.go` 的 `buildSubCommand` 函数中，`applyDefaults` 之后、`ValidateAll` 之前插入：

```go
// 第 79 行之后插入
applyDefaults(input, route.Template)

// 路径参数替换
resolvedPath, err := ReplacePath(route.Path, input, route.Template)
if err != nil {
    output.PrintError(os.Stderr, err.Error())
    os.Exit(1)
}

if errs := ValidateAll(input, route.Template); len(errs) > 0 {
```

并将第 112 行 `route.Path` 改为 `resolvedPath`：

```go
if err := client.DoCtx(ctx, route.Method, resolvedPath, input, &result); err != nil {
```

- [ ] **Step 3: 运行全部测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -v`
Expected: ALL PASS（包括所有原有测试不受影响）

- [ ] **Step 4: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/cmdgen_test.go
git commit -m "feat(cmdgen): integrate ReplacePath in buildSubCommand"
```

---

## Phase 5: YAML 更新

### Task 9: 更新 course.yaml

**Files:**
- Modify: `cmd/ckjr-cli/routes/course.yaml:83-132`

- [ ] **Step 1: 在 update 路由 template 中添加 courseId 字段**

在 `update` 路由的 `template:` 下，`courseType:` 之前添加：

```yaml
        courseId:
            description: 课程ID
            required: true
            type: path
```

- [ ] **Step 2: 运行全部测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./... 2>&1 | tail -20`
Expected: ALL PASS

- [ ] **Step 3: 提交**

```bash
git add cmd/ckjr-cli/routes/course.yaml
git commit -m "feat(course): add courseId path param to update route"
```

---

## 验证清单

- [ ] `go test ./internal/cmdgen/ -v` 全部通过
- [ ] `go test ./... ` 全部通过
- [ ] `go vet ./...` 无报错
- [ ] course.yaml update 路由的 `{courseId}` 在 template 中有 `type: path` 声明
