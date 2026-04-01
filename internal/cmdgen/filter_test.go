package cmdgen

import (
	"reflect"
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
)

func TestGetNestedValue(t *testing.T) {
	m := map[string]interface{}{
		"data": map[string]interface{}{
			"courseId": float64(15427611),
			"name":     "Go 入门",
		},
	}

	t.Run("existing path", func(t *testing.T) {
		val, ok := getNestedValue(m, "data.courseId")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if val != float64(15427611) {
			t.Errorf("got %v, want 15427611", val)
		}
	})

	t.Run("nonexistent leaf", func(t *testing.T) {
		_, ok := getNestedValue(m, "data.nonexistent")
		if ok {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("nonexistent root", func(t *testing.T) {
		_, ok := getNestedValue(m, "missing.key")
		if ok {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("intermediate not map", func(t *testing.T) {
		m := map[string]interface{}{"a": float64(1)}
		_, ok := getNestedValue(m, "a.b")
		if ok {
			t.Fatal("expected ok=false for non-map intermediate")
		}
	})

	t.Run("single segment", func(t *testing.T) {
		val, ok := getNestedValue(m, "data")
		if !ok {
			t.Fatal("expected ok=true")
		}
		if _, isMap := val.(map[string]interface{}); !isMap {
			t.Errorf("expected map, got %T", val)
		}
	})
}

func TestSetNestedValue(t *testing.T) {
	t.Run("single segment", func(t *testing.T) {
		m := make(map[string]interface{})
		setNestedValue(m, "a", float64(1))
		if m["a"] != float64(1) {
			t.Errorf("got %v, want 1", m["a"])
		}
	})

	t.Run("nested path creates intermediate maps", func(t *testing.T) {
		m := make(map[string]interface{})
		setNestedValue(m, "data.courseId", float64(42))
		data, ok := m["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected data to be map, got %T", m["data"])
		}
		if data["courseId"] != float64(42) {
			t.Errorf("got %v, want 42", data["courseId"])
		}
	})

	t.Run("overwrite existing intermediate map", func(t *testing.T) {
		m := map[string]interface{}{
			"data": map[string]interface{}{"old": "x"},
		}
		setNestedValue(m, "data.new", "y")
		data := m["data"].(map[string]interface{})
		if data["old"] != "x" {
			t.Error("should preserve existing keys in intermediate map")
		}
		if data["new"] != "y" {
			t.Error("should set new key")
		}
	})
}

func TestDeleteNestedPath(t *testing.T) {
	t.Run("delete nested key", func(t *testing.T) {
		m := map[string]interface{}{
			"data": map[string]interface{}{
				"courseId": float64(1),
				"name":     "Go",
			},
		}
		deleted := deleteNestedPath(m, "data.courseId")
		if !deleted {
			t.Fatal("expected deleted=true")
		}
		data := m["data"].(map[string]interface{})
		if _, exists := data["courseId"]; exists {
			t.Error("courseId should be deleted")
		}
		if data["name"] != "Go" {
			t.Error("name should be preserved")
		}
	})

	t.Run("nonexistent path", func(t *testing.T) {
		m := map[string]interface{}{"a": float64(1)}
		deleted := deleteNestedPath(m, "a.b")
		if deleted {
			t.Fatal("expected deleted=false for nonexistent path")
		}
	})

	t.Run("nonexistent root", func(t *testing.T) {
		m := map[string]interface{}{"a": float64(1)}
		deleted := deleteNestedPath(m, "missing.key")
		if deleted {
			t.Fatal("expected deleted=false")
		}
	})
}

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

func TestDeepCopyMap(t *testing.T) {
	original := map[string]interface{}{
		"a": float64(1),
		"nested": map[string]interface{}{
			"b": float64(2),
		},
	}
	copy := deepCopyMap(original)

	// Equal content
	if !reflect.DeepEqual(copy, original) {
		t.Errorf("copy should equal original, got %v", copy)
	}

	// Independent mutation
	copy["a"] = float64(99)
	if original["a"] != float64(1) {
		t.Error("modifying copy should not affect original")
	}
	copy["nested"].(map[string]interface{})["b"] = float64(99)
	if original["nested"].(map[string]interface{})["b"] != float64(2) {
		t.Error("nested mutation should not affect original")
	}
}

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

func TestFilterByFields_NestedPath(t *testing.T) {
	m := map[string]interface{}{
		"data": map[string]interface{}{
			"courseId": float64(15427611),
			"name":     "Go 入门",
			"secret":   "hidden",
		},
	}
	fields := []string{"data.courseId", "data.name"}

	result := filterByFields(m, fields)

	want := map[string]interface{}{
		"data": map[string]interface{}{
			"courseId": float64(15427611),
			"name":     "Go 入门",
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterByFields_MixedPaths(t *testing.T) {
	m := map[string]interface{}{
		"code": float64(0),
		"data": map[string]interface{}{
			"courseId": float64(15427611),
			"name":     "Go 入门",
		},
	}
	fields := []string{"code", "data.courseId"}

	result := filterByFields(m, fields)

	want := map[string]interface{}{
		"code": float64(0),
		"data": map[string]interface{}{
			"courseId": float64(15427611),
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

func TestFilterByFields_NestedPathNotFound(t *testing.T) {
	m := map[string]interface{}{
		"data": map[string]interface{}{
			"courseId": float64(1),
		},
	}
	fields := []string{"data.nonexistent"}

	result := filterByFields(m, fields)

	// 不存在的嵌套路径静默跳过
	want := map[string]interface{}{}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want empty map", result)
	}
}

func TestFilterByFields_NestedPathPreservesStructure(t *testing.T) {
	m := map[string]interface{}{
		"data": map[string]interface{}{
			"courseId": float64(1),
			"name":     "Go",
		},
	}
	fields := []string{"data.courseId", "data.name"}

	result := filterByFields(m, fields)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be map, got %T", result["data"])
	}
	if len(data) != 2 {
		t.Errorf("expected 2 keys in data, got %d", len(data))
	}
}

func TestFilterByExclude_NestedPath(t *testing.T) {
	m := map[string]interface{}{
		"data": map[string]interface{}{
			"courseId": float64(1),
			"name":     "Go",
			"secret":   "hidden",
		},
	}
	exclude := []string{"data.secret"}

	result := filterByExclude(m, exclude)

	want := map[string]interface{}{
		"data": map[string]interface{}{
			"courseId": float64(1),
			"name":     "Go",
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
	// 原始不应被修改
	origData := m["data"].(map[string]interface{})
	if _, exists := origData["secret"]; !exists {
		t.Error("original map should not be modified")
	}
}

func TestFilterByExclude_NestedPathPreservesOriginal(t *testing.T) {
	m := map[string]interface{}{
		"data": map[string]interface{}{
			"a": float64(1),
		},
	}
	filterByExclude(m, []string{"data.a"})

	// 原始 map 不应被修改
	if _, exists := m["data"].(map[string]interface{})["a"]; !exists {
		t.Error("exclude should deep copy before deleting")
	}
}

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

// TestFilterByFields_NestedDotKey tests that plain keys with no dot work as before
func TestFilterByFields_BackwardCompatNoDot(t *testing.T) {
	m := map[string]interface{}{
		"a": float64(1),
		"b": float64(2),
	}
	fields := []string{"a"}

	result := filterByFields(m, fields)

	want := map[string]interface{}{"a": float64(1)}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

// TestFilterByExclude_BackwardCompatNoDot tests that plain exclude keys work as before
func TestFilterByExclude_BackwardCompatNoDot(t *testing.T) {
	m := map[string]interface{}{"a": float64(1), "b": float64(2)}
	exclude := []string{"b"}

	result := filterByExclude(m, exclude)

	want := map[string]interface{}{"a": float64(1)}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

// TestFilterResponse_NestedFields tests FilterResponse with nested dot paths
func TestFilterResponse_NestedFields(t *testing.T) {
	m := map[string]interface{}{
		"code": float64(0),
		"data": map[string]interface{}{
			"courseId": float64(15427611),
			"name":     "Go 入门",
			"ext":      map[string]interface{}{"foo": "bar"},
		},
	}
	filter := &router.ResponseFilter{Fields: []string{"code", "data.courseId", "data.name"}}

	result := FilterResponse(m, filter)

	want := map[string]interface{}{
		"code": float64(0),
		"data": map[string]interface{}{
			"courseId": float64(15427611),
			"name":     "Go 入门",
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

// TestFilterResponse_NestedExclude tests FilterResponse with nested dot exclude paths
func TestFilterResponse_NestedExclude(t *testing.T) {
	m := map[string]interface{}{
		"code": float64(0),
		"data": map[string]interface{}{
			"courseId": float64(1),
			"ext":      map[string]interface{}{"foo": "bar"},
		},
	}
	filter := &router.ResponseFilter{Exclude: []string{"data.ext"}}

	result := FilterResponse(m, filter)

	want := map[string]interface{}{
		"code": float64(0),
		"data": map[string]interface{}{
			"courseId": float64(1),
		},
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("got %v, want %v", result, want)
	}
}

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

func TestFilterResponse_SliceResult(t *testing.T) {
	slice := []interface{}{float64(1), float64(2)}
	filter := &router.ResponseFilter{Fields: []string{"a"}}
	got := FilterResponse(slice, filter)
	gotSlice, ok := got.([]interface{})
	if !ok {
		t.Fatalf("expected slice, got %T", got)
	}
	if len(gotSlice) != 2 {
		t.Errorf("slice should pass through unchanged, got %v", gotSlice)
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
