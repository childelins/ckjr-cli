package cmdgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/router"
)

func TestBuildCommand(t *testing.T) {
	cfg := &router.RouteConfig{
		Name:        "agent",
		Description: "AI智能体管理",
		Routes: map[string]router.Route{
			"list": {
				Method:      "POST",
				Path:        "/admin/list",
				Description: "获取列表",
				Template: map[string]router.Field{
					"page": {
						Description: "页码",
						Required:    false,
						Default:     1,
					},
				},
			},
			"get": {
				Method:      "POST",
				Path:        "/admin/get",
				Description: "获取详情",
				Template: map[string]router.Field{
					"id": {
						Description: "ID",
						Required:    true,
					},
				},
			},
		},
	}

	cmd := BuildCommand(cfg, nil)
	if cmd.Use != "agent" {
		t.Errorf("Use = %s, want agent", cmd.Use)
	}

	if cmd.Short != "AI智能体管理" {
		t.Errorf("Short = %s", cmd.Short)
	}

	// 验证子命令
	subCmds := cmd.Commands()
	if len(subCmds) != 2 {
		t.Fatalf("子命令数量 = %d, want 2", len(subCmds))
	}

	// 验证 list 子命令
	listCmd, _, _ := cmd.Find([]string{"list"})
	if listCmd == nil {
		t.Error("list 子命令未找到")
	}
}

func TestTemplateFlag(t *testing.T) {
	cfg := &router.RouteConfig{
		Name:     "agent",
		Routes: map[string]router.Route{
			"create": {
				Method: "POST",
				Path:   "/create",
				Template: map[string]router.Field{
					"name": {
						Description: "名称",
						Required:    true,
					},
				},
			},
		},
	}

	cmd := BuildCommand(cfg, nil)
	createCmd, _, _ := cmd.Find([]string{"create"})
	if createCmd == nil {
		t.Fatal("create 子命令未找到")
	}

	// 验证 --template flag 存在
	templateFlag := createCmd.Flags().Lookup("template")
	if templateFlag == nil {
		t.Error("--template flag 未找到")
	}
}

func TestHandleAPIError_ResponseError(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试",
	}

	handleAPIErrorTo(&buf, respErr, false)

	got := buf.String()
	if !strings.Contains(got, "服务端返回异常") {
		t.Errorf("output should contain friendly message, got: %s", got)
	}
	// 非verbose不应包含body字段
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q", got)
	}
	if _, exists := result["body"]; exists {
		t.Error("non-verbose should not contain body field")
	}
}

func TestBuildSubCommand_GeneratesRequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := api.Response{Data: map[string]string{"id": "1"}, Message: "ok", StatusCode: 200}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// 捕获日志输出
	var logBuf bytes.Buffer
	handler := slog.NewJSONHandler(&logBuf, nil)
	old := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(old)

	cfg := &router.RouteConfig{
		Name: "test",
		Routes: map[string]router.Route{
			"list": {
				Method:      "POST",
				Path:        "/test/list",
				Description: "test list",
			},
		},
	}

	clientFactory := func() (*api.Client, error) {
		return api.NewClient(server.URL, "test-key"), nil
	}

	cmd := BuildCommand(cfg, clientFactory)
	cmd.PersistentFlags().Bool("pretty", false, "")
	cmd.PersistentFlags().Bool("verbose", false, "")

	listCmd, _, _ := cmd.Find([]string{"list"})
	if listCmd == nil {
		t.Fatal("list subcommand not found")
	}

	// 执行命令
	cmd.SetArgs([]string{"list", "{}"})
	cmd.Execute()

	// 检查日志中包含 UUID v4 格式的 request_id
	logOutput := logBuf.String()
	uuidPattern := `[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`
	if !regexp.MustCompile(uuidPattern).MatchString(logOutput) {
		t.Errorf("log should contain UUID v4 request_id, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "api_request") {
		t.Errorf("log should contain api_request, got: %s", logOutput)
	}
}

func TestPrintTemplate_TypeAndExample(t *testing.T) {
	template := map[string]router.Field{
		"count": {
			Description: "数量",
			Required:    false,
			Default:     10,
			Type:        "int",
			Example:     "10",
		},
		"name": {
			Description: "名称",
			Required:    true,
		},
	}

	var buf bytes.Buffer
	printTemplateTo(&buf, template)
	var result map[string]map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	// count: 有 type=int 和 example
	countEntry := result["count"]
	if countEntry["type"] != "int" {
		t.Errorf("count.type = %v, want \"int\"", countEntry["type"])
	}
	if countEntry["example"] != "10" {
		t.Errorf("count.example = %v, want \"10\"", countEntry["example"])
	}

	// name: 无 type 应默认 string，无 example 应不存在
	nameEntry := result["name"]
	if nameEntry["type"] != "string" {
		t.Errorf("name.type = %v, want \"string\"", nameEntry["type"])
	}
	if _, exists := nameEntry["example"]; exists {
		t.Error("name should not have example field")
	}
}

func TestHandleAPIError_ResponseError_Verbose(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)，请检查 base_url 配置或稍后重试",
	}

	handleAPIErrorTo(&buf, respErr, true)

	got := buf.String()
	if !strings.Contains(got, "服务端返回异常") {
		t.Errorf("output should contain friendly message, got: %s", got)
	}
	if !strings.Contains(got, "502") || !strings.Contains(got, "text/html") {
		t.Errorf("verbose should contain debug info, got: %s", got)
	}
}

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

func TestHandleAPIErrorTo_Unauthorized_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	handleAPIErrorTo(&buf, api.ErrUnauthorized, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["msg"] != "api_key 已过期，请重新登录获取" {
		t.Errorf("msg = %v, want api_key 已过期", result["msg"])
	}
	if result["statusCode"] != float64(401) {
		t.Errorf("statusCode = %v, want 401", result["statusCode"])
	}
}

func TestHandleAPIErrorTo_ValidationError_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	err := &api.ValidationError{
		Message: "参数校验失败",
		Errors:  map[string]interface{}{"name": []interface{}{"required"}},
	}
	handleAPIErrorTo(&buf, err, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["msg"] != "参数校验失败" {
		t.Errorf("msg = %v, want 参数校验失败", result["msg"])
	}
	if result["statusCode"] != float64(422) {
		t.Errorf("statusCode = %v, want 422", result["statusCode"])
	}
	// errors 字段应该是 map 而非字符串
	errorsMap, ok := result["errors"].(map[string]interface{})
	if !ok {
		t.Fatalf("errors should be a map, got %T: %v", result["errors"], result["errors"])
	}
	nameErrors, ok := errorsMap["name"].([]interface{})
	if !ok || len(nameErrors) == 0 || nameErrors[0] != "required" {
		t.Errorf("errors.name = %v, want [required]", errorsMap["name"])
	}
}

func TestHandleAPIErrorTo_APIError_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	err := &api.APIError{
		StatusCode: 403,
		Message:    "无权访问",
		ServerCode: 403,
	}
	handleAPIErrorTo(&buf, err, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["msg"] != "无权访问" {
		t.Errorf("msg = %v, want 无权访问", result["msg"])
	}
	if result["statusCode"] != float64(403) {
		t.Errorf("statusCode = %v, want 403", result["statusCode"])
	}
}

func TestHandleAPIErrorTo_APIError_WithErrors(t *testing.T) {
	var buf bytes.Buffer
	err := &api.APIError{
		StatusCode: 402,
		Message:    "余额不足",
		ServerCode: 402,
		Errors:     map[string]interface{}{"detail": "账户余额为0"},
	}
	handleAPIErrorTo(&buf, err, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["statusCode"] != float64(402) {
		t.Errorf("statusCode = %v, want 402", result["statusCode"])
	}
	errorsMap, ok := result["errors"].(map[string]interface{})
	if !ok {
		t.Fatalf("errors should be a map, got %T", result["errors"])
	}
	if errorsMap["detail"] != "账户余额为0" {
		t.Errorf("errors.detail = %v", errorsMap["detail"])
	}
}

func TestHandleAPIErrorTo_ResponseError_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)",
	}
	handleAPIErrorTo(&buf, respErr, false)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["statusCode"] != float64(502) {
		t.Errorf("statusCode = %v, want 502", result["statusCode"])
	}
	if result["content_type"] != "text/html" {
		t.Errorf("content_type = %v, want text/html", result["content_type"])
	}
	// 非verbose不应包含body
	if _, exists := result["body"]; exists {
		t.Error("non-verbose should not contain body field")
	}
}

func TestHandleAPIErrorTo_ResponseError_Verbose_StructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	respErr := &api.ResponseError{
		StatusCode:  502,
		ContentType: "text/html",
		Body:        "<html>Bad Gateway</html>",
		Message:     "服务端返回异常 (HTTP 502)",
	}
	handleAPIErrorTo(&buf, respErr, true)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["body"] != "<html>Bad Gateway</html>" {
		t.Errorf("verbose should contain body, got: %v", result["body"])
	}
}

func TestHandleAPIErrorTo_GenericError_FlatJSON(t *testing.T) {
	var buf bytes.Buffer
	err := fmt.Errorf("网络连接超时")
	handleAPIErrorTo(&buf, err, false)

	// 非 API 错误保持 {"error":"msg"} 格式
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output should be valid JSON, got: %q, error: %v", buf.String(), err)
	}
	if result["error"] != "网络连接超时" {
		t.Errorf("error = %v, want 网络连接超时", result["error"])
	}
}

func TestHandleAPIErrorTo_Integration_ServerResponsePassthrough(t *testing.T) {
	tests := []struct {
		name       string
		setupError error
		verbose    bool
		wantFields map[string]interface{}
	}{
		{
			name:       "403_forbidden",
			setupError: &api.APIError{StatusCode: 403, Message: "无权访问", ServerCode: 403},
			wantFields: map[string]interface{}{
				"msg":        "无权访问",
				"statusCode": float64(403),
			},
		},
		{
			name:       "500_server_error",
			setupError: &api.APIError{StatusCode: 500, Message: "内部错误", ServerCode: 500},
			wantFields: map[string]interface{}{
				"msg":        "内部错误",
				"statusCode": float64(500),
			},
		},
		{
			name: "422_validation_with_field_errors",
			setupError: &api.ValidationError{
				Message: "参数校验失败",
				Errors:  map[string]interface{}{"name": "required"},
			},
			wantFields: map[string]interface{}{
				"msg":        "参数校验失败",
				"statusCode": float64(422),
			},
		},
		{
			name: "502_gateway_non_json_verbose",
			setupError: &api.ResponseError{
				StatusCode:  502,
				ContentType: "text/html",
				Body:        "Bad Gateway",
				Message:     "服务端返回异常 (HTTP 502)",
			},
			verbose: true,
			wantFields: map[string]interface{}{
				"msg":          "服务端返回异常 (HTTP 502)",
				"statusCode":   float64(502),
				"content_type": "text/html",
				"body":         "Bad Gateway",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handleAPIErrorTo(&buf, tt.setupError, tt.verbose)

			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("output should be valid JSON, got: %q", buf.String())
			}
			for key, wantVal := range tt.wantFields {
				gotVal, ok := result[key]
				if !ok {
					t.Errorf("missing field %q in output: %v", key, result)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("field %q = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}
