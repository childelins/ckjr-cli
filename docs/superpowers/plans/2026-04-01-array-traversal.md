# Response Filter 自动数组穿透实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 增强 response filter 的 dot notation 路径解析，遇到 `[]interface{}` 自动穿透数组，对每个元素应用剩余路径，支持 `list.data.courseId` 等分页列表场景。

**Architecture:** 改造 `filter.go` 中 5 个核心函数：`deepCopyMap` 增加数组深拷贝；`getNestedValue`/`deleteNestedPath` 增加 `[]interface{}` type switch 分支；`filterByFields` 从 get-then-set 重构为 `applyFieldPath` 递归构建模式；`filterByExclude` 无需改动（依赖底层函数增强自动获得数组穿透能力）。

**Tech Stack:** Go, `strings` 标准库

**Spec:** `docs/superpowers/specs/2026-04-01-array-traversal-design.md`

---

## Phase 1: deepCopyMap 数组支持

### Task 1: deepCopyMap 深拷贝数组内 map

**Files:**
- Modify: `internal/cmdgen/filter_test.go`
- Modify: `internal/cmdgen/filter.go:62-72`

- [ ] **Step 1: 写失败测试 — 数组内 map 独立性**

```go
func TestDeepCopyMap_ArrayWithMaps(t *testing.T) {
	original := map[string]interface{}{
		"list": []interface{}{
			map[string]interface{}{"id": float64(1), "name": "Go"},
			map[string]interface{}{"id": float64(2), "name": "Rust"},
		},
	}
	cp := deepCopyMap(original)

	// 修改拷贝中数组内的 map，不应影响原始数据
	cpList := cp["list"].([]interface{})
	cpList[0].(map[string]interface{})["name"] = "Python"

	origList := original["list"].([]interface{})
	if origList[0].(map[string]interface{})["name"] != "Go" {
		t.Error("modifying copy's array element should not affect original")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestDeepCopyMap_ArrayWithMaps -v`
Expected: FAIL — 当前 deepCopyMap 对数组只做浅拷贝

- [ ] **Step 3: 实现 deepCopyValue + 改造 deepCopyMap**

将 `filter.go:62-72` 的 `deepCopyMap` 改为：

```go
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return deepCopyMap(val)
	case []interface{}:
		cp := make([]interface{}, len(val))
		for i, elem := range val {
			cp[i] = deepCopyValue(elem)
		}
		return cp
	default:
		return v
	}
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{}, len(m))
	for k, v := range m {
		cp[k] = deepCopyValue(v)
	}
	return cp
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestDeepCopyMap -v`
Expected: ALL PASS（包括已有的 TestDeepCopyMap 和新增的 TestDeepCopyMap_ArrayWithMaps）

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go
git commit -m "feat(filter): deep copy arrays with nested maps in deepCopyMap"
```

## Phase 2: getNestedValue 数组穿透

### Task 2: getNestedValue 遇到数组自动穿透

**Files:**
- Modify: `internal/cmdgen/filter_test.go`
- Modify: `internal/cmdgen/filter.go:11-25`

- [ ] **Step 1: 写失败测试 — 数组穿透取值**

```go
func TestGetNestedValue_ArrayTraversal(t *testing.T) {
	m := map[string]interface{}{
		"list": map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{"courseId": float64(1), "name": "Go"},
				map[string]interface{}{"courseId": float64(2), "name": "Rust"},
			},
			"total": float64(2),
		},
	}

	t.Run("traverses array elements", func(t *testing.T) {
		val, ok := getNestedValue(m, "list.data.courseId")
		if !ok {
			t.Fatal("expected ok=true")
		}
		want := []interface{}{float64(1), float64(2)}
		if !reflect.DeepEqual(val, want) {
			t.Errorf("got %v, want %v", val, want)
		}
	})

	t.Run("non-array path still works", func(t *testing.T) {
		val, ok := getNestedValue(m, "list.total")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if val != float64(2) {
			t.Errorf("got %v, want 2", val)
		}
	})

	t.Run("empty array returns false", func(t *testing.T) {
		m := map[string]interface{}{
			"list": map[string]interface{}{
				"data": []interface{}{},
			},
		}
		_, ok := getNestedValue(m, "list.data.courseId")
		if ok {
			t.Fatal("expected ok=false for empty array")
		}
	})

	t.Run("array with non-map elements skips them", func(t *testing.T) {
		m := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"id": float64(1)},
				"not a map",
				map[string]interface{}{"id": float64(3)},
			},
		}
		val, ok := getNestedValue(m, "items.id")
		if !ok {
			t.Fatal("expected ok=true")
		}
		want := []interface{}{float64(1), float64(3)}
		if !reflect.DeepEqual(val, want) {
			t.Errorf("got %v, want %v", val, want)
		}
	})
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestGetNestedValue_ArrayTraversal -v`
Expected: FAIL — 当前遇到 `[]interface{}` 返回 false

- [ ] **Step 3: 改造 getNestedValue 添加数组分支**

将 `filter.go:11-25` 替换为：

```go
func getNestedValue(m map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = m
	for i, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		case []interface{}:
			remaining := strings.Join(parts[i:], ".")
			var results []interface{}
			for _, elem := range v {
				em, ok := elem.(map[string]interface{})
				if !ok {
					continue
				}
				if val, ok := getNestedValue(em, remaining); ok {
					results = append(results, val)
				}
			}
			if len(results) == 0 {
				return nil, false
			}
			return results, true
		default:
			return nil, false
		}
	}
	return current, true
}
```

- [ ] **Step 4: 运行全部 getNestedValue 测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestGetNestedValue -v`
Expected: ALL PASS（向后兼容 + 新增数组穿透）

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go
git commit -m "feat(filter): auto array traversal in getNestedValue"
```

## Phase 3: deleteNestedPath 数组穿透

### Task 3: deleteNestedPath 穿透数组删除

**Files:**
- Modify: `internal/cmdgen/filter_test.go`
- Modify: `internal/cmdgen/filter.go:43-59`

- [ ] **Step 1: 写失败测试**

```go
func TestDeleteNestedPath_ArrayTraversal(t *testing.T) {
	t.Run("deletes field in each array element", func(t *testing.T) {
		m := map[string]interface{}{
			"list": map[string]interface{}{
				"data": []interface{}{
					map[string]interface{}{"id": float64(1), "secret": "x"},
					map[string]interface{}{"id": float64(2), "secret": "y"},
				},
			},
		}
		deleted := deleteNestedPath(m, "list.data.secret")
		if !deleted {
			t.Fatal("expected deleted=true")
		}
		data := m["list"].(map[string]interface{})["data"].([]interface{})
		for i, elem := range data {
			em := elem.(map[string]interface{})
			if _, exists := em["secret"]; exists {
				t.Errorf("element %d: secret should be deleted", i)
			}
			if _, exists := em["id"]; !exists {
				t.Errorf("element %d: id should be preserved", i)
			}
		}
	})

	t.Run("empty array returns false", func(t *testing.T) {
		m := map[string]interface{}{
			"list": map[string]interface{}{
				"data": []interface{}{},
			},
		}
		deleted := deleteNestedPath(m, "list.data.secret")
		if deleted {
			t.Fatal("expected deleted=false for empty array")
		}
	})

	t.Run("skips non-map elements in array", func(t *testing.T) {
		m := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"id": float64(1), "secret": "x"},
				"not a map",
				map[string]interface{}{"id": float64(2), "secret": "y"},
			},
		}
		deleted := deleteNestedPath(m, "items.secret")
		if !deleted {
			t.Fatal("expected deleted=true")
		}
	})
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestDeleteNestedPath_ArrayTraversal -v`
Expected: FAIL

- [ ] **Step 3: 改造 deleteNestedPath**

将 `filter.go:43-59` 替换为：

```go
func deleteNestedPath(m map[string]interface{}, path string) bool {
	parts := strings.Split(path, ".")
	return deleteNestedParts(m, parts)
}

func deleteNestedParts(m map[string]interface{}, parts []string) bool {
	if len(parts) == 1 {
		_, exists := m[parts[0]]
		if !exists {
			return false
		}
		delete(m, parts[0])
		return true
	}
	val, ok := m[parts[0]]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case map[string]interface{}:
		return deleteNestedParts(v, parts[1:])
	case []interface{}:
		deleted := false
		for _, elem := range v {
			if em, ok := elem.(map[string]interface{}); ok {
				if deleteNestedParts(em, parts[1:]) {
					deleted = true
				}
			}
		}
		return deleted
	default:
		return false
	}
}
```

- [ ] **Step 4: 运行全部 deleteNestedPath 测试**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestDeleteNestedPath -v`
Expected: ALL PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go
git commit -m "feat(filter): auto array traversal in deleteNestedPath"
```

## Phase 4: filterByExclude 数组穿透验证

### Task 4: filterByExclude 利用底层增强自动获得穿透能力

**Files:**
- Modify: `internal/cmdgen/filter_test.go`

- [ ] **Step 1: 写测试验证 exclude 穿透数组**

```go
func TestFilterByExclude_ArrayTraversal(t *testing.T) {
	m := map[string]interface{}{
		"list": map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{"id": float64(1), "name": "Go", "secret": "x"},
				map[string]interface{}{"id": float64(2), "name": "Rust", "secret": "y"},
			},
			"total": float64(2),
		},
	}
	result := filterByExclude(m, []string{"list.data.secret"})

	want := map[string]interface{}{
		"list": map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{"id": float64(1), "name": "Go"},
				map[string]interface{}{"id": float64(2), "name": "Rust"},
			},
			"total": float64(2),
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}

	// 原始数据不应被修改
	origData := m["list"].(map[string]interface{})["data"].([]interface{})
	if _, exists := origData[0].(map[string]interface{})["secret"]; !exists {
		t.Error("original data should not be modified")
	}
}
```

- [ ] **Step 2: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestFilterByExclude_ArrayTraversal -v`
Expected: PASS（deepCopyMap + deleteNestedPath 已增强，filterByExclude 自动获得穿透能力）

- [ ] **Step 3: 提交**

```bash
git add internal/cmdgen/filter_test.go
git commit -m "test(filter): verify filterByExclude array traversal"
```

## Phase 5: filterByFields 重构为 applyFieldPath

### Task 5: 重构 filterByFields 支持数组穿透

**Files:**
- Modify: `internal/cmdgen/filter_test.go`
- Modify: `internal/cmdgen/filter.go:74-87`

- [ ] **Step 1: 写失败测试 — 分页列表 fields 过滤**

```go
func TestFilterByFields_ArrayTraversal(t *testing.T) {
	m := map[string]interface{}{
		"list": map[string]interface{}{
			"current_page": float64(1),
			"data": []interface{}{
				map[string]interface{}{"courseId": float64(1), "name": "Go", "secret": "x"},
				map[string]interface{}{"courseId": float64(2), "name": "Rust", "secret": "y"},
			},
			"total": float64(2),
		},
	}
	fields := []string{"list.data.courseId", "list.data.name", "list.total"}

	result := filterByFields(m, fields)

	want := map[string]interface{}{
		"list": map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{"courseId": float64(1), "name": "Go"},
				map[string]interface{}{"courseId": float64(2), "name": "Rust"},
			},
			"total": float64(2),
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterByFields_ArrayTraversal_EmptyArray(t *testing.T) {
	m := map[string]interface{}{
		"list": map[string]interface{}{
			"data":  []interface{}{},
			"total": float64(0),
		},
	}
	fields := []string{"list.data.courseId", "list.total"}

	result := filterByFields(m, fields)

	want := map[string]interface{}{
		"list": map[string]interface{}{
			"data":  []interface{}{},
			"total": float64(0),
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterByFields_ArrayTraversal_NonMapElements(t *testing.T) {
	m := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": float64(1)},
			"not a map",
			map[string]interface{}{"id": float64(3)},
		},
	}
	fields := []string{"items.id"}

	result := filterByFields(m, fields)

	want := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": float64(1)},
			nil,
			map[string]interface{}{"id": float64(3)},
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run "TestFilterByFields_ArrayTraversal" -v`
Expected: FAIL

- [ ] **Step 3: 重构 filterByFields + 新增 applyFieldPath**

将 `filter.go:74-87` 替换为：

```go
func filterByFields(m map[string]interface{}, fields []string) map[string]interface{} {
	filtered := make(map[string]interface{})
	for _, f := range fields {
		applyFieldPath(m, filtered, strings.Split(f, "."))
	}
	return filtered
}

func applyFieldPath(src, dst map[string]interface{}, parts []string) {
	if len(parts) == 0 {
		return
	}
	key := parts[0]
	val, ok := src[key]
	if !ok {
		return
	}
	if len(parts) == 1 {
		dst[key] = val
		return
	}
	remaining := parts[1:]
	switch v := val.(type) {
	case map[string]interface{}:
		sub, ok := dst[key].(map[string]interface{})
		if !ok {
			sub = make(map[string]interface{})
			dst[key] = sub
		}
		applyFieldPath(v, sub, remaining)
	case []interface{}:
		existingArr, _ := dst[key].([]interface{})
		if existingArr == nil {
			existingArr = make([]interface{}, len(v))
			dst[key] = existingArr
		}
		for i, elem := range v {
			em, ok := elem.(map[string]interface{})
			if !ok {
				continue
			}
			dm, ok := existingArr[i].(map[string]interface{})
			if !ok {
				dm = make(map[string]interface{})
				existingArr[i] = dm
			}
			applyFieldPath(em, dm, remaining)
		}
	}
}
```

- [ ] **Step 4: 运行全部 filterByFields 测试（含向后兼容）**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run TestFilterByFields -v`
Expected: ALL PASS（新增数组穿透 + 全部已有测试通过）

- [ ] **Step 5: 提交**

```bash
git add internal/cmdgen/filter.go internal/cmdgen/filter_test.go
git commit -m "feat(filter): refactor filterByFields to applyFieldPath with array traversal"
```

## Phase 6: FilterResponse 集成测试 + YAML 配置

### Task 6: 端到端集成测试

**Files:**
- Modify: `internal/cmdgen/filter_test.go`

- [ ] **Step 1: 写集成测试**

```go
func TestFilterResponse_ListWithFields(t *testing.T) {
	m := map[string]interface{}{
		"list": map[string]interface{}{
			"current_page": float64(1),
			"data": []interface{}{
				map[string]interface{}{
					"courseId":     float64(15427611),
					"name":        "AI 红利期",
					"courseAvatar": "https://example.com/avatar.png",
					"price":       "0.00",
					"permission":  float64(19),
					"secret":      "hidden",
				},
			},
			"total":    float64(1),
			"per_page": "10",
		},
	}
	filter := &router.ResponseFilter{
		Fields: []string{
			"list.data.courseId",
			"list.data.name",
			"list.data.courseAvatar",
			"list.data.price",
			"list.total",
			"list.current_page",
			"list.per_page",
		},
	}

	result := FilterResponse(m, filter)

	rm := result.(map[string]interface{})
	list := rm["list"].(map[string]interface{})
	if list["total"] != float64(1) {
		t.Errorf("total: got %v, want 1", list["total"])
	}
	if list["current_page"] != float64(1) {
		t.Errorf("current_page: got %v, want 1", list["current_page"])
	}
	data := list["data"].([]interface{})
	item := data[0].(map[string]interface{})
	if item["courseId"] != float64(15427611) {
		t.Errorf("courseId: got %v, want 15427611", item["courseId"])
	}
	if _, exists := item["secret"]; exists {
		t.Error("secret should be filtered out")
	}
	if _, exists := item["permission"]; exists {
		t.Error("permission should be filtered out")
	}
}

func TestFilterResponse_ListWithExclude(t *testing.T) {
	m := map[string]interface{}{
		"list": map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{"id": float64(1), "name": "Go", "secret": "x"},
				map[string]interface{}{"id": float64(2), "name": "Rust", "secret": "y"},
			},
			"total": float64(2),
		},
	}
	filter := &router.ResponseFilter{
		Exclude: []string{"list.data.secret"},
	}

	result := FilterResponse(m, filter)

	rm := result.(map[string]interface{})
	data := rm["list"].(map[string]interface{})["data"].([]interface{})
	for i, elem := range data {
		em := elem.(map[string]interface{})
		if _, exists := em["secret"]; exists {
			t.Errorf("element %d: secret should be excluded", i)
		}
		if _, exists := em["name"]; !exists {
			t.Errorf("element %d: name should be preserved", i)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认通过**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -run "TestFilterResponse_List" -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add internal/cmdgen/filter_test.go
git commit -m "test(filter): add FilterResponse integration tests for list array traversal"
```

### Task 7: 更新 course.yaml list 路由配置

**Files:**
- Modify: `cmd/ckjr-cli/routes/course.yaml:83-111`

- [ ] **Step 1: 添加 list 路由的 response 配置**

在 course.yaml 的 `list` 路由末尾（`status` 字段配置之后）添加：

```yaml
        response:
            fields:
                - list.data.courseId
                - list.data.name
                - list.data.courseType
                - list.data.status
                - list.data.isSaleOnly
                - list.data.price
                - list.data.payType
                - list.data.courseAvatar
                - list.total
                - list.current_page
                - list.per_page
```

- [ ] **Step 2: 运行全量测试确认向后兼容**

Run: `cd /home/childelins/code/ckjr-cli && go test ./internal/cmdgen/ -v`
Expected: ALL PASS

- [ ] **Step 3: 提交**

```bash
git add cmd/ckjr-cli/routes/course.yaml
git commit -m "feat(course): add response fields filter to list route"
```
