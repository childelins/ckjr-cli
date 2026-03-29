# Field 类型与约束校验实施计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 为 route YAML 参数模板增加类型校验和约束校验（min/max/minLength/maxLength/pattern）

**Architecture:** 扩展 router.Field 结构增加约束字段，新建 cmdgen/validate.go 实现校验逻辑，修改 cmdgen 调用方统一通过 ValidateAll 入口校验，修改 curlparse 的 float64 推断逻辑。

**Tech Stack:** Go 1.24, gopkg.in/yaml.v3, regexp 标准库

---

## Phase 1: Field 结构扩展

**Files:**
- Modify: `internal/router/router.go:10-16`
- Test: `internal/router/router_test.go`

- [ ] **Step 1: 写失败测试 - 解析含约束字段的 YAML**

在 `internal/router/router_test.go` 末尾添加：

```go
func TestParseRouteConfig_Constraints(t *testing.T) {
	yamlContent := `
name: test
description: 测试约束
routes:
  create:
    method: POST
    path: /create
    template:
      page:
        description: 页码
        required: false
        default: 1
        type: int
        min: 1
        max: 1000
      keyword:
        description: 关键词
        required: false
        type: string
        minLength: 1
        maxLength: 100
      email:
        description: 邮箱
        required: true
        type: string
        pattern: "^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$"
      score:
        description: 评分
        required: false
        type: float
        min: 0.0
        max: 10.0
`
	cfg, err := Parse([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	route := cfg.Routes["create"]

	// page: min/max
	page := route.Template["page"]
	if page.Min == nil || *page.Min != 1.0 {
		t.Errorf("page.Min = %v, want 1.0", page.Min)
	}
	if page.Max == nil || *page.Max != 1000.0 {
		t.Errorf("page.Max = %v, want 1000.0", page.Max)
	}

	// keyword: minLength/maxLength
	keyword := route.Template["keyword"]
	if keyword.MinLength == nil || *keyword.MinLength != 1 {
		t.Errorf("keyword.MinLength = %v, want 1", keyword.MinLength)
	}
	if keyword.MaxLength == nil || *keyword.MaxLength != 100 {
		t.Errorf("keyword.MaxLength = %v, want 100", keyword.MaxLength)
	}

	// email: pattern
	email := route.Template["email"]
	if email.Pattern != `^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$` {
		t.Errorf("email.Pattern = %q", email.Pattern)
	}

	// score: float type with min/max
	score := route.Template["score"]
	if score.Type != "float" {
		t.Errorf("score.Type = %q, want \"float\"", score.Type)
	}
	if score.Min == nil || *score.Min != 0.0 {
		t.Errorf("score.Min = %v, want 0.0", score.Min)
	}
	if score.Max == nil || *score.Max != 10.0 {
		t.Errorf("score.Max = %v, want 10.0", score.Max)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/router/ -run TestParseRouteConfig_Constraints -v`
Expected: 编译失败（Field 结构缺少 Min/Max/MinLength/MaxLength/Pattern 字段）

- [ ] **Step 3: 修改 Field 结构**

在 `internal/router/router.go` 的 Field 结构中追加约束字段：

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
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/router/ -v`
Expected: 所有测试 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/router/router.go internal/router/router_test.go
git commit -m "feat(validation): extend Field struct with constraint fields (min/max/minLength/maxLength/pattern)"
```

---

## Phase 2: 类型校验

**Files:**
- Create: `internal/cmdgen/validate.go`
- Create: `internal/cmdgen/validate_test.go`

- [ ] **Step 1: 写失败测试 - validateType**

创建 `internal/cmdgen/validate_test.go`：

```go
package cmdgen

import (
	"testing"
)

func TestValidateType_String(t *testing.T) {
	// string 值通过
	if err := validateType("name", "hello", "string"); err != nil {
		t.Errorf("string value should pass: %v", err)
	}
	// 非 string 失败
	if err := validateType("name", float64(1), "string"); err == nil {
		t.Error("float64 value should fail for string type")
	}
}

func TestValidateType_Int(t *testing.T) {
	// 整数 float64 通过
	if err := validateType("count", float64(10), "int"); err != nil {
		t.Errorf("integer float64 should pass: %v", err)
	}
	// 浮点 float64 失败
	if err := validateType("count", float64(10.5), "int"); err == nil {
		t.Error("10.5 should fail for int type")
	}
	// string 失败
	if err := validateType("count", "10", "int"); err == nil {
		t.Error("string should fail for int type")
	}
}

func TestValidateType_Float(t *testing.T) {
	// float64 通过
	if err := validateType("score", float64(3.14), "float"); err != nil {
		t.Errorf("float64 should pass: %v", err)
	}
	// 整数值也通过
	if err := validateType("score", float64(10), "float"); err != nil {
		t.Errorf("integer float64 should pass for float type: %v", err)
	}
}

func TestValidateType_Bool(t *testing.T) {
	if err := validateType("flag", true, "bool"); err != nil {
		t.Errorf("bool should pass: %v", err)
	}
	if err := validateType("flag", "true", "bool"); err == nil {
		t.Error("string should fail for bool type")
	}
}

func TestValidateType_Array(t *testing.T) {
	if err := validateType("tags", []interface{}{"a", "b"}, "array"); err != nil {
		t.Errorf("array should pass: %v", err)
	}
	// map 不是 array
	if err := validateType("tags", map[string]interface{}{"key": "val"}, "array"); err == nil {
		t.Error("map should fail for array type")
	}
}

func TestValidateType_Empty(t *testing.T) {
	// type 为空不校验
	if err := validateType("field", "anything", ""); err != nil {
		t.Errorf("empty type should not validate: %v", err)
	}
}

func TestValidateType_Unknown(t *testing.T) {
	if err := validateType("field", "val", "unknown"); err == nil {
		t.Error("unknown type should return error")
	}
}

func TestValidateType_Nil(t *testing.T) {
	// nil 值应返回类型不匹配
	if err := validateType("field", nil, "string"); err == nil {
		t.Error("nil should fail for string type")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cmdgen/ -run TestValidateType -v`
Expected: 编译失败（validateType 未定义）

- [ ] **Step 3: 实现 validateType 和 ValidationError**

创建 `internal/cmdgen/validate.go`：

```go
package cmdgen

import (
	"fmt"
	"math"
	"regexp"

	"github.com/childelins/ckjr-cli/internal/router"
)

// FieldValidationError 单个字段的校验错误
type FieldValidationError struct {
	Field   string
	Message string
}

func (e *FieldValidationError) Error() string {
	return fmt.Sprintf("字段 %q %s", e.Field, e.Message)
}

// ValidateAll 校验输入数据，返回所有校验错误
func ValidateAll(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	var errs []*FieldValidationError

	// 1. required 校验
	errs = append(errs, validateRequiredErrors(input, template)...)

	// 2. 类型校验
	errs = append(errs, validateTypes(input, template)...)

	// 3. 约束校验
	errs = append(errs, validateConstraints(input, template)...)

	return errs
}

func validateRequiredErrors(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	missing := validateRequired(input, template)
	var errs []*FieldValidationError
	for _, name := range missing {
		errs = append(errs, &FieldValidationError{Field: name, Message: "为必填字段"})
	}
	return errs
}

func validateTypes(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	var errs []*FieldValidationError
	for name, field := range template {
		if field.Type == "" {
			continue
		}
		val, exists := input[name]
		if !exists {
			continue
		}
		if err := validateType(name, val, field.Type); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func validateType(fieldName string, value interface{}, expectedType string) *FieldValidationError {
	if expectedType == "" {
		return nil
	}

	if value == nil {
		return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 %s，实际为 null", expectedType)}
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 string，实际为 %T", value)}
		}
	case "int":
		v, ok := value.(float64)
		if !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 int，实际为 %T", value)}
		}
		if v != math.Trunc(v) {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 int，实际为浮点数 %v", v)}
		}
	case "float":
		if _, ok := value.(float64); !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 float，实际为 %T", value)}
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 bool，实际为 %T", value)}
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 array，实际为 %T", value)}
		}
	default:
		return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("未知类型 %q", expectedType)}
	}

	return nil
}

func validateConstraints(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	// Phase 3 实现
	return nil
}

// compilePatterns 预编译 template 中的正则表达式
func compilePatterns(template map[string]router.Field) (map[string]*regexp.Regexp, *FieldValidationError) {
	patterns := make(map[string]*regexp.Regexp)
	for name, field := range template {
		if field.Pattern == "" {
			continue
		}
		re, err := regexp.Compile(field.Pattern)
		if err != nil {
			return nil, &FieldValidationError{
				Field:   name,
				Message: fmt.Sprintf("正则表达式编译失败: %s", field.Pattern),
			}
		}
		patterns[name] = re
	}
	return patterns, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/cmdgen/ -run TestValidateType -v`
Expected: 所有测试 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/validate.go internal/cmdgen/validate_test.go
git commit -m "feat(validation): add type validation (string/int/float/bool/array)"
```

---

## Phase 3: 约束校验

**Files:**
- Modify: `internal/cmdgen/validate.go`
- Modify: `internal/cmdgen/validate_test.go`

- [ ] **Step 1: 写失败测试 - 约束校验**

在 `internal/cmdgen/validate_test.go` 末尾追加：

```go
func intPtr(v int) *int       { return &v }
func floatPtr(v float64) *float64 { return &v }

func TestValidateConstraints_MinMax(t *testing.T) {
	template := map[string]router.Field{
		"page": {
			Type: "int",
			Min:  floatPtr(1),
			Max:  floatPtr(100),
		},
	}

	tests := []struct {
		name  string
		input map[string]interface{}
		errs  int
	}{
		{"within range", map[string]interface{}{"page": float64(50)}, 0},
		{"at min", map[string]interface{}{"page": float64(1)}, 0},
		{"at max", map[string]interface{}{"page": float64(100)}, 0},
		{"below min", map[string]interface{}{"page": float64(0)}, 1},
		{"above max", map[string]interface{}{"page": float64(101)}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateConstraints(tt.input, template)
			if len(errs) != tt.errs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.errs, errs)
			}
		})
	}
}

func TestValidateConstraints_MinLengthMaxLength(t *testing.T) {
	template := map[string]router.Field{
		"name": {
			Type:      "string",
			MinLength: intPtr(2),
			MaxLength: intPtr(10),
		},
	}

	tests := []struct {
		name  string
		input map[string]interface{}
		errs  int
	}{
		{"valid length", map[string]interface{}{"name": "hello"}, 0},
		{"at min", map[string]interface{}{"name": "ab"}, 0},
		{"at max", map[string]interface{}{"name": "0123456789"}, 0},
		{"too short", map[string]interface{}{"name": "a"}, 1},
		{"too long", map[string]interface{}{"name": "01234567890"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateConstraints(tt.input, template)
			if len(errs) != tt.errs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.errs, errs)
			}
		})
	}
}

func TestValidateConstraints_Pattern(t *testing.T) {
	template := map[string]router.Field{
		"email": {
			Type:    "string",
			Pattern: `^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$`,
		},
	}

	tests := []struct {
		name  string
		input map[string]interface{}
		errs  int
	}{
		{"valid email", map[string]interface{}{"email": "test@example.com"}, 0},
		{"invalid email", map[string]interface{}{"email": "not-email"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateConstraints(tt.input, template)
			if len(errs) != tt.errs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.errs, errs)
			}
		})
	}
}

func TestValidateConstraints_Irrelevant(t *testing.T) {
	// 约束与 type 不匹配时不报错
	template := map[string]router.Field{
		"name": {
			Type: "string",
			Min:  floatPtr(1),
			Max:  floatPtr(10),
		},
	}
	input := map[string]interface{}{"name": "hello"}
	errs := validateConstraints(input, template)
	if len(errs) != 0 {
		t.Errorf("min/max on string should be ignored, got: %v", errs)
	}
}

func TestValidateConstraints_FloatMinMax(t *testing.T) {
	template := map[string]router.Field{
		"score": {
			Type: "float",
			Min:  floatPtr(0.0),
			Max:  floatPtr(10.0),
		},
	}

	tests := []struct {
		name  string
		input map[string]interface{}
		errs  int
	}{
		{"within range", map[string]interface{}{"score": float64(5.5)}, 0},
		{"below min", map[string]interface{}{"score": float64(-0.1)}, 1},
		{"above max", map[string]interface{}{"score": float64(10.1)}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateConstraints(tt.input, template)
			if len(errs) != tt.errs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.errs, errs)
			}
		})
	}
}
```

注意：测试文件顶部需要增加 import `"github.com/childelins/ckjr-cli/internal/router"`

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cmdgen/ -run TestValidateConstraints -v`
Expected: 约束校验返回 nil（validateConstraints 是空实现）

- [ ] **Step 3: 实现 validateConstraints**

替换 `internal/cmdgen/validate.go` 中 `validateConstraints` 的空实现：

```go
func validateConstraints(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	patterns, patternErr := compilePatterns(template)
	if patternErr != nil {
		return []*FieldValidationError{patternErr}
	}

	var errs []*FieldValidationError
	for name, field := range template {
		val, exists := input[name]
		if !exists || val == nil {
			continue
		}

		switch field.Type {
		case "int", "float":
			v, ok := val.(float64)
			if !ok {
				continue
			}
			if field.Min != nil && v < *field.Min {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: fmt.Sprintf("值 %v 小于最小值 %v", v, *field.Min),
				})
			}
			if field.Max != nil && v > *field.Max {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: fmt.Sprintf("值 %v 大于最大值 %v", v, *field.Max),
				})
			}

		case "string":
			str, ok := val.(string)
			if !ok {
				continue
			}
			if field.MinLength != nil && len(str) < *field.MinLength {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: fmt.Sprintf("长度 %d 小于最小长度 %d", len(str), *field.MinLength),
				})
			}
			if field.MaxLength != nil && len(str) > *field.MaxLength {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: fmt.Sprintf("长度 %d 大于最大长度 %d", len(str), *field.MaxLength),
				})
			}
			if field.Pattern != "" {
				re := patterns[name]
				if !re.MatchString(str) {
					errs = append(errs, &FieldValidationError{
						Field:   name,
						Message: fmt.Sprintf("值 %q 不匹配正则 %q", str, field.Pattern),
					})
				}
			}
		}
	}
	return errs
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/cmdgen/ -run TestValidateConstraints -v`
Expected: 所有测试 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/validate.go internal/cmdgen/validate_test.go
git commit -m "feat(validation): add constraint validation (min/max/minLength/maxLength/pattern)"
```

---

## Phase 4: ValidateAll 集成 + cmdgen 调用方修改

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:80-84`
- Modify: `internal/cmdgen/validate_test.go`

- [ ] **Step 1: 写集成测试 - ValidateAll**

在 `internal/cmdgen/validate_test.go` 末尾追加：

```go
func TestValidateAll_MultipleErrors(t *testing.T) {
	template := map[string]router.Field{
		"name": {
			Description: "名称",
			Required:    true,
			Type:        "string",
			MinLength:   intPtr(2),
		},
		"age": {
			Description: "年龄",
			Required:    true,
			Type:        "int",
			Min:         floatPtr(0),
			Max:         floatPtr(150),
		},
		"email": {
			Description: "邮箱",
			Required:    false,
			Type:        "string",
			Pattern:     `^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$`,
		},
	}

	// 缺少 name 和 age（required），email 格式错误
	input := map[string]interface{}{
		"email": "bad-email",
	}

	errs := ValidateAll(input, template)
	if len(errs) < 3 {
		t.Errorf("expected at least 3 errors (2 missing + 1 pattern), got %d: %v", len(errs), errs)
	}
}

func TestValidateAll_Pass(t *testing.T) {
	template := map[string]router.Field{
		"name": {
			Description: "名称",
			Required:    true,
			Type:        "string",
			MinLength:   intPtr(1),
		},
	}
	input := map[string]interface{}{
		"name": "hello",
	}

	errs := ValidateAll(input, template)
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}
```

- [ ] **Step 2: 运行测试确认通过**

Run: `go test ./internal/cmdgen/ -run TestValidateAll -v`
Expected: PASS（ValidateAll 已在 Phase 2 实现）

- [ ] **Step 3: 修改 cmdgen.go 调用方**

将 `internal/cmdgen/cmdgen.go` 中的校验调用（约 80-84 行）：

```go
// 校验必填字段
if missing := validateRequired(input, route.Template); len(missing) > 0 {
    output.PrintError(os.Stderr, fmt.Sprintf("缺少必填字段: %v", missing))
    os.Exit(1)
}
```

替换为：

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
```

同时在 import 中添加 `"strings"`。

- [ ] **Step 4: 运行全部测试**

Run: `go test ./internal/cmdgen/ -v`
Expected: 所有测试 PASS（包括已有的 TestBuildSubCommand_GeneratesRequestID）

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/validate_test.go
git commit -m "feat(validation): integrate ValidateAll into cmdgen, replace validateRequired call"
```

---

## Phase 5: printTemplate 展示约束信息

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:125-146`
- Modify: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 写失败测试 - printTemplate 展示约束**

在 `internal/cmdgen/cmdgen_test.go` 末尾追加：

```go
func TestPrintTemplate_Constraints(t *testing.T) {
	minVal := 1.0
	maxVal := 100.0
	minLen := 2
	maxLen := 50

	template := map[string]router.Field{
		"page": {
			Description: "页码",
			Required:    false,
			Default:     1,
			Type:        "int",
			Min:         &minVal,
			Max:         &maxVal,
		},
		"keyword": {
			Description: "关键词",
			Required:    true,
			Type:        "string",
			MinLength:   &minLen,
			MaxLength:   &maxLen,
			Pattern:     `^\w+$`,
		},
	}

	var buf bytes.Buffer
	printTemplateTo(&buf, template)

	var result map[string]map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	// page: 有 constraints
	pageEntry := result["page"]
	constraints, ok := pageEntry["constraints"]
	if !ok {
		t.Fatal("page should have constraints")
	}
	cm := constraints.(map[string]interface{})
	if cm["min"] != 1.0 {
		t.Errorf("constraints.min = %v, want 1.0", cm["min"])
	}
	if cm["max"] != 100.0 {
		t.Errorf("constraints.max = %v, want 100.0", cm["max"])
	}

	// keyword: 有 constraints
	keywordEntry := result["keyword"]
	kc := keywordEntry["constraints"].(map[string]interface{})
	if kc["minLength"] != 2.0 { // JSON 数字解析为 float64
		t.Errorf("constraints.minLength = %v, want 2", kc["minLength"])
	}
	if kc["maxLength"] != 50.0 {
		t.Errorf("constraints.maxLength = %v, want 50", kc["maxLength"])
	}
	if kc["pattern"] != `^\w+$` {
		t.Errorf("constraints.pattern = %v", kc["pattern"])
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cmdgen/ -run TestPrintTemplate_Constraints -v`
Expected: FAIL（constraints 字段不存在）

- [ ] **Step 3: 修改 printTemplateTo 追加约束信息**

在 `internal/cmdgen/cmdgen.go` 的 `printTemplateTo` 函数中，在 `tmpl[name] = entry` 之前追加约束输出：

```go
func printTemplateTo(w io.Writer, template map[string]router.Field) {
	tmpl := make(map[string]interface{})
	for name, field := range template {
		entry := map[string]interface{}{
			"description": field.Description,
			"required":    field.Required,
		}
		if field.Default != nil {
			entry["default"] = field.Default
		}
		t := field.Type
		if t == "" {
			t = "string"
		}
		entry["type"] = t
		if field.Example != "" {
			entry["example"] = field.Example
		}

		// 约束信息
		constraints := map[string]interface{}{}
		if field.Min != nil {
			constraints["min"] = *field.Min
		}
		if field.Max != nil {
			constraints["max"] = *field.Max
		}
		if field.MinLength != nil {
			constraints["minLength"] = *field.MinLength
		}
		if field.MaxLength != nil {
			constraints["maxLength"] = *field.MaxLength
		}
		if field.Pattern != "" {
			constraints["pattern"] = field.Pattern
		}
		if len(constraints) > 0 {
			entry["constraints"] = constraints
		}

		tmpl[name] = entry
	}
	output.Print(w, tmpl, true)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/cmdgen/ -run TestPrintTemplate -v`
Expected: 所有 printTemplate 测试 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/cmdgen_test.go
git commit -m "feat(validation): display constraint info in --template output"
```

---

## Phase 6: curlparse float 类型推断

**Files:**
- Modify: `internal/curlparse/parse.go:126`
- Modify: `internal/curlparse/parse_test.go`

- [ ] **Step 1: 写失败测试 - float 推断**

在 `internal/curlparse/parse_test.go` 末尾追加：

```go
func TestParse_FloatType(t *testing.T) {
	curl := `curl 'https://example.com/api' --data-raw '{"score":3.14,"count":5}'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	score, ok := result.Fields["score"]
	if !ok {
		t.Fatal("score field not found")
	}
	if score.Type != "float" {
		t.Errorf("score.Type = %q, want \"float\"", score.Type)
	}
	if score.Example != 3.14 {
		t.Errorf("score.Example = %v, want 3.14", score.Example)
	}

	count, ok := result.Fields["count"]
	if !ok {
		t.Fatal("count field not found")
	}
	if count.Type != "int" {
		t.Errorf("count.Type = %q, want \"int\"", count.Type)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/curlparse/ -run TestParse_FloatType -v`
Expected: FAIL（score.Type 为 "string" 而非 "float"）

- [ ] **Step 3: 修改 inferField**

在 `internal/curlparse/parse.go` 的 `inferField` 函数中，修改 float64 分支：

```go
case float64:
    if v == math.Trunc(v) {
        return Field{Type: "int", Example: int(v)}, true
    }
    return Field{Type: "float", Example: v}, true
```

将原来的 `Field{Type: "string", Example: v}` 改为 `Field{Type: "float", Example: v}`。

- [ ] **Step 4: 运行全部 curlparse 测试**

Run: `go test ./internal/curlparse/ -v`
Expected: 所有测试 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/curlparse/parse.go internal/curlparse/parse_test.go
git commit -m "feat(curlparse): infer float type for non-integer numbers"
```

---

## Phase 7: 全量测试

**Files:**
- 无新增

- [ ] **Step 1: 运行全量测试**

Run: `go test ./... -v`
Expected: 所有包测试 PASS

- [ ] **Step 2: 验证已有 YAML 无需修改**

Run: `go test ./internal/router/ -v`
Expected: PASS（新字段全部 omitempty，已有解析不受影响）

- [ ] **Step 3: 最终提交（如有遗漏修复）**

```bash
git add -A
git commit -m "chore(validation): fix any remaining test issues"
```
