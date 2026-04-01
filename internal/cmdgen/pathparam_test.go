package cmdgen

import (
	"strings"
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

// --- PathParamError ---

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

// --- ReplacePath ---

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
	if _, ok := input["courseId"]; ok {
		t.Error("courseId should be removed from input")
	}
	if _, ok := input["name"]; !ok {
		t.Error("name should remain in input")
	}
}

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

func TestReplacePath_UndeclaredPlaceholder(t *testing.T) {
	input := map[string]interface{}{"courseId": float64(1)}
	tmpl := map[string]router.Field{
		"courseId": {Type: "int"},
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

func TestReplacePath_LargeFloat64NotScientific(t *testing.T) {
	// JSON 反序列化后大整数变成 float64，%v 会输出科学计数法
	input := map[string]interface{}{"courseId": float64(15427611)}
	tmpl := map[string]router.Field{
		"courseId": {Type: "path", Required: true},
	}
	path, err := ReplacePath("/admin/courses/{courseId}", input, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if path != "/admin/courses/15427611" {
		t.Errorf("got %q, want /admin/courses/15427611", path)
	}
}

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
