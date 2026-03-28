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
