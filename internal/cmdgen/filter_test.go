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
