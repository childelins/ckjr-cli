package cmdgen

import (
	"testing"

	"github.com/childelins/ckjr-cli/internal/router"
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

func intPtr(v int) *int          { return &v }
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
