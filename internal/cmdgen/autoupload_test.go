package cmdgen

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/router"
)

func TestProcessAutoUpload_ExternalURL(t *testing.T) {
	// 模拟外部图片服务器
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-image-data"))
	}))
	defer imageServer.Close()

	// 模拟 OSS 服务器
	ossServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ossServer.Close()

	var requests []string
	// 模拟 API 服务器
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/assets/imageSign":
			json.NewEncoder(w).Encode(api.Response{
				Data: map[string]interface{}{
					"accessid":  "test-id",
					"policy":    "test-policy",
					"signature": "test-sig",
					"callback":  "",
					"dir":       "test/dir/",
					"host":      ossServer.URL,
					"origin":    1,
				},
				Message:    "ok",
				StatusCode: 200,
			})
		case "/admin/assets/addImgInAsset":
			json.NewEncoder(w).Encode(api.Response{
				Data:       map[string]interface{}{"id": 123},
				Message:    "ok",
				StatusCode: 200,
			})
		}
	}))
	defer apiServer.Close()

	externalURL := imageServer.URL + "/test.png"

	input := map[string]interface{}{
		"avatar": externalURL,
		"name":   "test",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
		"name":   {},
	}

	client := api.NewClient(apiServer.URL, "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}

	// avatar 值应被替换为 OSS URL
	newAvatar, ok := input["avatar"].(string)
	if !ok {
		t.Fatalf("avatar should be string, got %T", input["avatar"])
	}
	if newAvatar == externalURL {
		t.Error("avatar should be replaced with OSS URL")
	}
	if newAvatar == "" {
		t.Error("avatar should not be empty after upload")
	}

	// 验证走了完整流程（imageSign + addImgInAsset）
	foundImageSign := false
	foundAddImg := false
	for _, path := range requests {
		if path == "/admin/assets/imageSign" {
			foundImageSign = true
		}
		if path == "/admin/assets/addImgInAsset" {
			foundAddImg = true
		}
	}
	if !foundImageSign {
		t.Error("expected imageSign request")
	}
	if !foundAddImg {
		t.Error("expected addImgInAsset request")
	}

	// name 值不变
	if input["name"] != "test" {
		t.Errorf("name should not be changed, got %v", input["name"])
	}
}

func TestProcessAutoUpload_InternalURL_Skipped(t *testing.T) {
	input := map[string]interface{}{
		"avatar": "https://ck-bkt-knowledge-payment.oss-cn-hangzhou.aliyuncs.com/test.png",
		"name":   "test",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}

	// avatar 值不变
	if input["avatar"] != "https://ck-bkt-knowledge-payment.oss-cn-hangzhou.aliyuncs.com/test.png" {
		t.Errorf("internal URL should not be changed, got %v", input["avatar"])
	}
}

func TestProcessAutoUpload_EmptyValue_Skipped(t *testing.T) {
	input := map[string]interface{}{
		"avatar": "",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}

	if input["avatar"] != "" {
		t.Errorf("empty string should not be changed, got %v", input["avatar"])
	}
}

func TestProcessAutoUpload_MissingField_Skipped(t *testing.T) {
	input := map[string]interface{}{
		"name": "test",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}
}

func TestProcessAutoUpload_NonStringField_Skipped(t *testing.T) {
	input := map[string]interface{}{
		"avatar": float64(123),
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}

	if input["avatar"] != float64(123) {
		t.Errorf("non-string value should not be changed, got %v", input["avatar"])
	}
}

func TestProcessAutoUpload_NoAutoUploadFields(t *testing.T) {
	input := map[string]interface{}{
		"name": "test",
	}
	template := map[string]router.Field{
		"name": {},
	}

	client := api.NewClient("http://localhost", "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err != nil {
		t.Fatalf("processAutoUpload() error = %v", err)
	}
}

func TestProcessAutoUpload_UploadError_ReturnsError(t *testing.T) {
	// 模拟 imageSign 失败（返回 500）
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Response{
			Message:    "internal error",
			StatusCode: 500,
		})
	}))
	defer apiServer.Close()

	// 外部图片需要是有效的 HTTP URL
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-image-data"))
	}))
	defer imageServer.Close()

	input := map[string]interface{}{
		"avatar": imageServer.URL + "/test.png",
	}
	template := map[string]router.Field{
		"avatar": {AutoUpload: "image"},
	}

	client := api.NewClient(apiServer.URL, "test-key")
	err := processAutoUpload(context.Background(), input, template, client)
	if err == nil {
		t.Fatal("expected error when upload fails")
	}
	// 错误信息应包含字段名
	if !containsSubstring(err.Error(), "avatar") {
		t.Errorf("error should contain field name 'avatar', got: %v", err)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestBuildSubCommand_AutoUpload(t *testing.T) {
	// 外部图片服务器
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-image-data"))
	}))
	defer imageServer.Close()

	// 模拟 OSS 服务器
	ossServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ossServer.Close()

	var capturedAvatar string
	// API 服务器
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/assets/imageSign":
			json.NewEncoder(w).Encode(api.Response{
				Data: map[string]interface{}{
					"accessid":  "test-id",
					"policy":    "test-policy",
					"signature": "test-sig",
					"callback":  "",
					"dir":       "test/dir/",
					"host":      ossServer.URL,
					"origin":    1,
				},
				Message:    "ok",
				StatusCode: 200,
			})
		case "/admin/assets/addImgInAsset":
			json.NewEncoder(w).Encode(api.Response{
				Data:       map[string]interface{}{"id": 123},
				Message:    "ok",
				StatusCode: 200,
			})
		case "/admin/create":
			// 捕获最终请求中的 avatar 值
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			capturedAvatar = body["avatar"].(string)
			resp := api.Response{Data: map[string]interface{}{"id": float64(1)}, Message: "ok", StatusCode: 200}
			json.NewEncoder(w).Encode(resp)
		default:
			json.NewEncoder(w).Encode(api.Response{
				Data:       map[string]interface{}{"statusCode": 200},
				Message:    "ok",
				StatusCode: 200,
			})
		}
	}))
	defer apiServer.Close()

	cfg := &router.RouteConfig{
		Name: "agent",
		Routes: map[string]router.Route{
			"create": {
				Method:      "POST",
				Path:        "/admin/create",
				Description: "创建",
				Template: map[string]router.Field{
					"avatar": {
						Description: "头像URL",
						Required:    true,
						AutoUpload:  "image",
					},
					"name": {
						Description: "名称",
						Required:    true,
					},
				},
				Response: &router.ResponseFilter{
					Fields: []router.ResponseField{{Path: "id"}},
				},
			},
		},
	}

	clientFactory := func() (*api.Client, error) {
		return api.NewClient(apiServer.URL, "test-key"), nil
	}

	// 捕获 stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := BuildCommand(cfg, clientFactory)
	cmd.PersistentFlags().Bool("pretty", false, "")
	cmd.PersistentFlags().Bool("verbose", false, "")

	externalURL := imageServer.URL + "/test.png"
	cmd.SetArgs([]string{"create", `{"avatar": "` + externalURL + `", "name": "test"}`})
	cmd.Execute()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	// 验证最终请求中 avatar 已被替换为 OSS URL
	if capturedAvatar == externalURL {
		t.Errorf("avatar should be replaced, still external URL: %s", capturedAvatar)
	}
	if capturedAvatar == "" {
		t.Error("avatar should not be empty")
	}
}
