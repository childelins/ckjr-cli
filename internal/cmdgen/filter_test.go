package cmdgen

import (
	"reflect"
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
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
