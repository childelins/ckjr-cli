# Response Filter 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 route YAML 中支持 `response` 字段定义，通过 fields(白名单) 或 exclude(黑名单) 过滤 API 响应的顶层字段输出。

**Architecture:** Route 结构新增 `Response *ResponseFilter`，新增 `filter.go` 包含纯函数过滤逻辑，在 `cmdgen.buildSubCommand` 的 `output.Print` 之前插入一行调用。

**Tech Stack:** Go, gopkg.in/yaml.v3

---

### Task 1: Route 结构扩展 — 新增 ResponseFilter

**Files:**
- Modify: `internal/router/router.go:28-33`
- Test: `internal/router/router_test.go`

- [ ] **Step 1: 写失败的测试 — 验证 ResponseFilter 反序列化**

在 `internal/router/router_test.go` 中新增测试（如不存在则创建）：

```go
func TestRoute_ResponseFilter_Unmarshal(t *testing.T) {
	yamlData := `
method: GET
path: /admin/courses/{courseId}/edit
description: 获取课程详情
response:
  fields:
    - courseId
    - name
    - status
`
	var route Route
	if err := yaml.Unmarshal([]byte(yamlData), &route); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if route.Response == nil {
		t.Fatal("Response should not be nil")
	}
	if len(route.Response.Fields) != 3 {
		t.Errorf("Fields count = %d, want 3", len(route.Response.Fields))
	}
	if len(route.Response.Exclude) != 0 {
		t.Errorf("Exclude count = %d, want 0", len(route.Response.Exclude))
	}
}

func TestRoute_ResponseFilter_Exclude(t *testing.T) {
	yamlData := `
method: GET
path: /admin/courses
response:
  exclude:
    - detailInfo
    - internalFlag
`
	var route Route
	if err := yaml.Unmarshal([]byte(yamlData), &route); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(route.Response.Exclude) != 2 {
		t.Errorf("Exclude count = %d, want 2", len(route.Response.Exclude))
	}
}

func TestRoute_ResponseFilter_Nil(t *testing.T) {
	yamlData := `
method: GET
path: /admin/courses
`
	var route Route
	if err := yaml.Unmarshal([]byte(yamlData), &route); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if route.Response != nil {
		t.Error("Response should be nil when not configured")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/router/ -run TestRoute_ResponseFilter -v`
Expected: FAIL — `Route` 没有 `Response` 字段

- [ ] **Step 3: 实现 ResponseFilter 结构体**

在 `internal/router/router.go` 的 `Route` 结构之前新增：

```go
// ResponseFilter 定义响应字段过滤规则
type ResponseFilter struct {
	Fields  []string `yaml:"fields,omitempty"`
	Exclude []string `yaml:"exclude,omitempty"`
}
```

在 `Route` 结构中新增字段：

```go
Response    *ResponseFilter    `yaml:"response,omitempty"`
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/router/ -run TestRoute_ResponseFilter -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/router/router.go internal/router/router_test.go
git commit -m "feat(router): add ResponseFilter struct for response field filtering"
```

---

### Task 2: filterByFields — 白名单过滤函数

**Files:**
- Create: `internal/cmdgen/filter_test.go`
- Create: `internal/cmdgen/filter.go`

- [ ] **Step 1: 写失败的测试 — filterByFields**

在 `internal/cmdgen/filter_test.go` 中：

```go
package cmdgen

import (
	"reflect"
	"testing"
)

func TestFilterByFields_AllMatch(t *testing.T) {
	m := map[string]interface{}{
		"courseId": float64(1),
		"name":     "Go",
		"status":   float64(1),
	}
	fields := []string{"courseId", "name"}

	result := filterByFields(m, fields)

	want := map[string]interface{}{
		"courseId": float64(1),
		"name":     "Go",
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterByFields_PartialMatch(t *testing.T) {
	m := map[string]interface{}{
		"courseId": float64(1),
		"name":     "Go",
	}
	fields := []string{"courseId", "createdAt"}

	result := filterByFields(m, fields)

	want := map[string]interface{}{
		"courseId": float64(1),
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterByFields_NoneMatch(t *testing.T) {
	m := map[string]interface{}{"a": float64(1)}
	fields := []string{"x", "y"}

	result := filterByFields(m, fields)

	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestFilterByFields_PreservesNested(t *testing.T) {
	m := map[string]interface{}{
		"courseId": float64(1),
		"detailInfo": []interface{}{
			map[string]interface{}{"type": float64(1), "content": "<p>hello</p>"},
		},
	}
	fields := []string{"detailInfo"}

	result := filterByFields(m, fields)

	if len(result) != 1 {
		t.Fatalf("expected 1 key, got %d", len(result))
	}
	detail, ok := result["detailInfo"].([]interface{})
	if !ok || len(detail) != 1 {
		t.Fatalf("detailInfo should be preserved as-is, got %v", result["detailInfo"])
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cmdgen/ -run TestFilterByFields -v`
Expected: FAIL — `filterByFields` 未定义

- [ ] **Step 3: 实现 filterByFields**

在 `internal/cmdgen/filter.go` 中：

```go
package cmdgen

// filterByFields 仅保留 fields 中列出的顶层 key
func filterByFields(m map[string]interface{}, fields []string) map[string]interface{} {
	allowed := make(map[string]bool, len(fields))
	for _, f := range fields {
		allowed[f] = true
	}

	filtered := make(map[string]interface{})
	for k, v := range m {
		if allowed[k] {
			filtered[k] = v
		}
	}
	return filtered
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/cmdgen/ -run TestFilterByFields -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go
git commit -m "feat(cmdgen): add filterByFields whitelist function"
```

---

### Task 3: filterByExclude — 黑名单过滤函数

**Files:**
- Modify: `internal/cmdgen/filter.go`
- Modify: `internal/cmdgen/filter_test.go`

- [ ] **Step 1: 写失败的测试 — filterByExclude**

在 `internal/cmdgen/filter_test.go` 中追加：

```go
func TestFilterByExclude_AllMatch(t *testing.T) {
	m := map[string]interface{}{
		"courseId":     float64(1),
		"name":         "Go",
		"detailInfo":   "big data",
		"internalFlag": true,
	}
	exclude := []string{"detailInfo", "internalFlag"}

	result := filterByExclude(m, exclude)

	want := map[string]interface{}{
		"courseId": float64(1),
		"name":     "Go",
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterByExclude_PartialMatch(t *testing.T) {
	m := map[string]interface{}{"a": float64(1), "b": float64(2)}
	exclude := []string{"a", "nonexistent"}

	result := filterByExclude(m, exclude)

	want := map[string]interface{}{"b": float64(2)}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterByExclude_NoneMatch(t *testing.T) {
	m := map[string]interface{}{"a": float64(1), "b": float64(2)}
	exclude := []string{"x", "y"}

	result := filterByExclude(m, exclude)

	if !reflect.DeepEqual(result, m) {
		t.Errorf("should return original when nothing to exclude, got %v", result)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cmdgen/ -run TestFilterByExclude -v`
Expected: FAIL — `filterByExclude` 未定义

- [ ] **Step 3: 实现 filterByExclude**

在 `internal/cmdgen/filter.go` 中追加：

```go
// filterByExclude 移除 exclude 中列出的顶层 key
func filterByExclude(m map[string]interface{}, exclude []string) map[string]interface{} {
	excluded := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		excluded[e] = true
	}

	filtered := make(map[string]interface{}, len(m))
	for k, v := range m {
		if !excluded[k] {
			filtered[k] = v
		}
	}
	return filtered
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/cmdgen/ -run TestFilterByExclude -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go
git commit -m "feat(cmdgen): add filterByExclude blacklist function"
```

---

### Task 4: FilterResponse — 顶层过滤入口函数

**Files:**
- Modify: `internal/cmdgen/filter.go`
- Modify: `internal/cmdgen/filter_test.go`

- [ ] **Step 1: 写失败的测试 — FilterResponse 边界情况**

在 `internal/cmdgen/filter_test.go` 中追加：

```go
import "github.com/childelins/ckjr-cli/internal/router"

func TestFilterResponse_NilFilter(t *testing.T) {
	m := map[string]interface{}{"a": float64(1)}
	result := FilterResponse(m, nil)
	if !reflect.DeepEqual(result, m) {
		t.Errorf("nil filter should return original, got %v", result)
	}
}

func TestFilterResponse_NonMapResult(t *testing.T) {
	tests := []struct {
		name   string
		result interface{}
	}{
		{"nil", nil},
		{"string", "hello"},
		{"slice", []interface{}{float64(1), float64(2)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &router.ResponseFilter{Fields: []string{"a"}}
			got := FilterResponse(tt.result, filter)
			if got != tt.result {
				t.Errorf("non-map should pass through, got %v", got)
			}
		})
	}
}

func TestFilterResponse_FieldsOnly(t *testing.T) {
	m := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(3)}
	filter := &router.ResponseFilter{Fields: []string{"a", "c"}}

	result := FilterResponse(m, filter)

	want := map[string]interface{}{"a": float64(1), "c": float64(3)}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterResponse_ExcludeOnly(t *testing.T) {
	m := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(3)}
	filter := &router.ResponseFilter{Exclude: []string{"b"}}

	result := FilterResponse(m, filter)

	want := map[string]interface{}{"a": float64(1), "c": float64(3)}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterResponse_FieldsAndExclude(t *testing.T) {
	// 同时配置时 fields 优先，exclude 被忽略
	m := map[string]interface{}{"a": float64(1), "b": float64(2), "c": float64(3)}
	filter := &router.ResponseFilter{
		Fields:  []string{"a"},
		Exclude: []string{"a", "b"},
	}

	result := FilterResponse(m, filter)

	want := map[string]interface{}{"a": float64(1)}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("fields should take priority, got %v, want %v", result, want)
	}
}

func TestFilterResponse_EmptyFields(t *testing.T) {
	m := map[string]interface{}{"a": float64(1)}
	filter := &router.ResponseFilter{Fields: []string{}}

	result := FilterResponse(m, filter)

	// 空 fields 等同于未配置，全量返回
	if !reflect.DeepEqual(result, m) {
		t.Errorf("empty fields should return original, got %v", result)
	}
}

func TestFilterResponse_EmptyExclude(t *testing.T) {
	m := map[string]interface{}{"a": float64(1)}
	filter := &router.ResponseFilter{Exclude: []string{}}

	result := FilterResponse(m, filter)

	if !reflect.DeepEqual(result, m) {
		t.Errorf("empty exclude should return original, got %v", result)
	}
}

func TestFilterResponse_FieldNotFound(t *testing.T) {
	m := map[string]interface{}{"a": float64(1)}
	filter := &router.ResponseFilter{Fields: []string{"a", "nonexistent"}}

	result := FilterResponse(m, filter)

	// 不存在的字段静默跳过
	want := map[string]interface{}{"a": float64(1)}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("missing fields should be silently skipped, got %v", result)
	}
}

func TestFilterResponse_EmptyResult(t *testing.T) {
	m := map[string]interface{}{}
	filter := &router.ResponseFilter{Fields: []string{"a"}}

	result := FilterResponse(m, filter)

	if len(result.(map[string]interface{})) != 0 {
		t.Errorf("empty result should remain empty, got %v", result)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cmdgen/ -run TestFilterResponse -v`
Expected: FAIL — `FilterResponse` 未定义

- [ ] **Step 3: 实现 FilterResponse**

在 `internal/cmdgen/filter.go` 头部添加 import，并新增函数：

```go
import "github.com/childelins/ckjr-cli/internal/router"

// FilterResponse 根据 Route 的 response 配置过滤 result 的顶层字段
// 返回过滤后的新 map，不修改原始 result
func FilterResponse(result interface{}, respFilter *router.ResponseFilter) interface{} {
	if respFilter == nil {
		return result
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		return result
	}

	if len(respFilter.Fields) > 0 {
		return filterByFields(m, respFilter.Fields)
	}

	if len(respFilter.Exclude) > 0 {
		return filterByExclude(m, respFilter.Exclude)
	}

	return result
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/cmdgen/ -run TestFilterResponse -v`
Expected: PASS

- [ ] **Step 5: 全量测试确认无回归**

Run: `go test ./internal/cmdgen/ -v`
Expected: ALL PASS

- [ ] **Step 6: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go
git commit -m "feat(cmdgen): add FilterResponse with fields/exclude support"
```

---

### Task 5: 集成到 cmdgen — 在输出前调用 FilterResponse

**Files:**
- Modify: `internal/cmdgen/cmdgen.go:122-124`
- Modify: `internal/cmdgen/cmdgen_test.go`

- [ ] **Step 1: 写失败的集成测试**

在 `internal/cmdgen/cmdgen_test.go` 中追加：

```go
func TestBuildSubCommand_ResponseFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := api.Response{
			Data: map[string]interface{}{
				"courseId":     float64(1),
				"name":         "Go",
				"status":       float64(1),
				"detailInfo":   "sensitive data",
				"internalFlag": true,
			},
			Message:    "ok",
			StatusCode: 200,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &router.RouteConfig{
		Name: "course",
		Routes: map[string]router.Route{
			"get": {
				Method:      "GET",
				Path:        "/admin/courses/{courseId}/edit",
				Description: "获取课程详情",
				Template: map[string]router.Field{
					"courseId": {Type: "path", Required: true},
				},
				Response: &router.ResponseFilter{
					Fields: []string{"courseId", "name", "status"},
				},
			},
		},
	}

	clientFactory := func() (*api.Client, error) {
		return api.NewClient(server.URL, "test-key"), nil
	}

	var buf bytes.Buffer
	cmd := BuildCommand(cfg, clientFactory)
	cmd.SetOut(&buf)
	cmd.PersistentFlags().Bool("pretty", false, "")
	cmd.PersistentFlags().Bool("verbose", false, "")

	cmd.SetArgs([]string{"get", `{"courseId": 1}`})
	cmd.Execute()

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	// 验证仅输出 fields 中指定的字段
	if _, ok := result["detailInfo"]; ok {
		t.Error("detailInfo should be filtered out")
	}
	if _, ok := result["internalFlag"]; ok {
		t.Error("internalFlag should be filtered out")
	}
	if _, ok := result["courseId"]; !ok {
		t.Error("courseId should be present")
	}
	if _, ok := result["name"]; !ok {
		t.Error("name should be present")
	}
	if _, ok := result["status"]; !ok {
		t.Error("status should be present")
	}
}

func TestBuildSubCommand_ResponseFilter_Exclude(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := api.Response{
			Data: map[string]interface{}{
				"courseId":     float64(1),
				"name":         "Go",
				"detailInfo":   "sensitive",
				"internalFlag": true,
			},
			Message:    "ok",
			StatusCode: 200,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &router.RouteConfig{
		Name: "course",
		Routes: map[string]router.Route{
			"list": {
				Method:      "GET",
				Path:        "/admin/courses",
				Description: "课程列表",
				Response: &router.ResponseFilter{
					Exclude: []string{"detailInfo", "internalFlag"},
				},
			},
		},
	}

	clientFactory := func() (*api.Client, error) {
		return api.NewClient(server.URL, "test-key"), nil
	}

	var buf bytes.Buffer
	cmd := BuildCommand(cfg, clientFactory)
	cmd.SetOut(&buf)
	cmd.PersistentFlags().Bool("pretty", false, "")
	cmd.PersistentFlags().Bool("verbose", false, "")

	cmd.SetArgs([]string{"list", "{}"})
	cmd.Execute()

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if _, ok := result["detailInfo"]; ok {
		t.Error("detailInfo should be excluded")
	}
	if _, ok := result["internalFlag"]; ok {
		t.Error("internalFlag should be excluded")
	}
	if _, ok := result["courseId"]; !ok {
		t.Error("courseId should be present")
	}
}

func TestBuildSubCommand_NoResponseFilter(t *testing.T) {
	// 无 response 配置时行为不变
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := api.Response{
			Data: map[string]interface{}{
				"a": float64(1),
				"b": float64(2),
			},
			Message:    "ok",
			StatusCode: 200,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &router.RouteConfig{
		Name: "test",
		Routes: map[string]router.Route{
			"get": {
				Method:      "GET",
				Path:        "/test",
				Description: "test",
			},
		},
	}

	clientFactory := func() (*api.Client, error) {
		return api.NewClient(server.URL, "test-key"), nil
	}

	var buf bytes.Buffer
	cmd := BuildCommand(cfg, clientFactory)
	cmd.SetOut(&buf)
	cmd.PersistentFlags().Bool("pretty", false, "")
	cmd.PersistentFlags().Bool("verbose", false, "")

	cmd.SetArgs([]string{"get", "{}"})
	cmd.Execute()

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if _, ok := result["a"]; !ok {
		t.Error("a should be present (no filter)")
	}
	if _, ok := result["b"]; !ok {
		t.Error("b should be present (no filter)")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/cmdgen/ -run TestBuildSubCommand_ResponseFilter -v`
Run: `go test ./internal/cmdgen/ -run TestBuildSubCommand_NoResponseFilter -v`
Expected: FAIL — `detailInfo` 和 `internalFlag` 仍然出现在输出中

- [ ] **Step 3: 在 cmdgen.go 中集成 FilterResponse**

在 `internal/cmdgen/cmdgen.go` 第 122 行（`output.Print` 之前）插入：

```go
			// 响应字段过滤
			result = FilterResponse(result, route.Response)

			output.Print(os.Stdout, result, pretty)
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/cmdgen/ -run TestBuildSubCommand_ResponseFilter -v`
Run: `go test ./internal/cmdgen/ -run TestBuildSubCommand_NoResponseFilter -v`
Expected: ALL PASS

- [ ] **Step 5: 全量测试确认无回归**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 6: 提交**

```bash
git add internal/cmdgen/cmdgen.go internal/cmdgen/cmdgen_test.go
git commit -m "feat(cmdgen): integrate FilterResponse into buildSubCommand output"
```

---

### Task 6: 更新 course.yaml — 为 get 路由添加 response 过滤

**Files:**
- Modify: `cmd/ckjr-cli/routes/course.yaml:63-71`

- [ ] **Step 1: 在 course get 路由添加 response fields**

在 `cmd/ckjr-cli/routes/course.yaml` 的 `get` 路由中，`template` 后面添加 `response` 配置。字段列表需根据实际 API 返回确定，以下为示例：

```yaml
    get:
        method: GET
        path: /admin/courses/{courseId}/edit
        description: 获取课程详情
        template:
            courseId:
                description: 课程ID
                required: true
                type: path
        response:
            fields:
                - courseId
                - name
                - courseType
                - status
                - price
                - payType
                - courseAvatar
```

注意：具体 fields 列表需要根据 `ckjr-cli course get '{"courseId":15427611}'` 的实际输出来确定。上述列表为常见字段，实际实现时应先运行命令查看完整返回后确定。

- [ ] **Step 2: 验证 YAML 解析无报错**

Run: `go build ./cmd/ckjr-cli/`
Expected: 编译成功

- [ ] **Step 3: 提交**

```bash
git add cmd/ckjr-cli/routes/course.yaml
git commit -m "feat(course): add response fields filter to get route"
```
