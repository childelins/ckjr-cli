package curlparse

import (
	"testing"
)

func TestParse_PostWithBody(t *testing.T) {
	curl := `curl 'https://kpapi-cs.ckjr001.com/api/admin/aiCreationCenter/modifyApp' \
  -H 'content-type: application/json' \
  --data-raw '{"name":"test","aikbId":3550,"desc":"描述"}'`

	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Method != "POST" {
		t.Errorf("Method = %q, want POST", result.Method)
	}
	if result.Path != "/admin/aiCreationCenter/modifyApp" {
		t.Errorf("Path = %q, want /admin/aiCreationCenter/modifyApp", result.Path)
	}
	if len(result.Fields) != 3 {
		t.Fatalf("Fields count = %d, want 3", len(result.Fields))
	}
	if f, ok := result.Fields["aikbId"]; !ok || f.Type != "int" {
		t.Errorf("aikbId field: got %+v", result.Fields["aikbId"])
	}
	if f, ok := result.Fields["name"]; !ok || f.Type != "string" {
		t.Errorf("name field: got %+v", result.Fields["name"])
	}
}

func TestParse_GetRequest(t *testing.T) {
	curl := `curl 'https://example.com/api/users?page=1&name=test'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.Method != "GET" {
		t.Errorf("Method = %q, want GET", result.Method)
	}
	if result.Path != "/users" {
		t.Errorf("Path = %q, want /users", result.Path)
	}
	if len(result.Fields) != 2 {
		t.Fatalf("Fields count = %d, want 2", len(result.Fields))
	}
	if f, ok := result.Fields["page"]; !ok || f.Type != "int" || f.Example != 1 {
		t.Errorf("page field: got %+v", result.Fields["page"])
	}
	if f, ok := result.Fields["name"]; !ok || f.Type != "string" || f.Example != "test" {
		t.Errorf("name field: got %+v", result.Fields["name"])
	}
}

func TestParse_GetQueryTypes(t *testing.T) {
	curl := `curl 'https://example.com/api/items?count=42&flag=true&tag=hello'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tests := map[string]struct {
		typ     string
		example interface{}
	}{
		"count": {"int", 42},
		"flag":  {"bool", true},
		"tag":   {"string", "hello"},
	}
	for name, want := range tests {
		f, ok := result.Fields[name]
		if !ok {
			t.Errorf("field %q not found", name)
			continue
		}
		if f.Type != want.typ {
			t.Errorf("%s.Type = %q, want %q", name, f.Type, want.typ)
		}
		if f.Example != want.example {
			t.Errorf("%s.Example = %v, want %v", name, f.Example, want.example)
		}
	}
}

func TestParse_ExplicitMethod(t *testing.T) {
	curl := `curl -X PUT 'https://example.com/api/users' --data-raw '{"name":"test"}'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result.Method != "PUT" {
		t.Errorf("Method = %q, want PUT", result.Method)
	}
}

func TestParse_NestedBody(t *testing.T) {
	curl := `curl 'https://example.com/api' --data-raw '{"name":"test","items":[1,2],"config":{"key":"val"},"count":5}'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	// 嵌套字段（items, config）应被跳过，只保留顶层简单类型
	if len(result.Fields) != 2 {
		t.Errorf("Fields count = %d, want 2 (name + count)", len(result.Fields))
	}
	if _, ok := result.Fields["items"]; ok {
		t.Error("items (array) should be skipped")
	}
	if _, ok := result.Fields["config"]; ok {
		t.Error("config (object) should be skipped")
	}
}

func TestParse_TypeInference(t *testing.T) {
	curl := `curl 'https://example.com/api' --data-raw '{"str":"hello","num":42,"flag":true,"empty":null}'`
	result, err := Parse(curl)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tests := map[string]string{
		"str":   "string",
		"num":   "int",
		"flag":  "bool",
		"empty": "string",
	}
	for name, wantType := range tests {
		f, ok := result.Fields[name]
		if !ok {
			t.Errorf("field %q not found", name)
			continue
		}
		if f.Type != wantType {
			t.Errorf("%s.Type = %q, want %q", name, f.Type, wantType)
		}
	}
}

func TestParse_Invalid(t *testing.T) {
	tests := []struct {
		name string
		curl string
	}{
		{"empty", ""},
		{"not curl", "wget https://example.com"},
		{"bad json", "curl 'https://example.com' --data-raw 'not json'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.curl)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}
