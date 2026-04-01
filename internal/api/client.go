package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/childelins/ckjr-cli/internal/logging"
)

// readableJSON 将 JSON 中的 Unicode 转义序列解码为 UTF-8 字符
func readableJSON(raw []byte) string {
	var v interface{}
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if enc.Encode(v) != nil {
		return string(raw)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// ErrUnauthorized API Key 无效或过期
var ErrUnauthorized = errors.New("api_key 已过期，请重新登录获取")

// ErrValidation 参数校验失败
var ErrValidation = errors.New("参数校验失败")

// Response API 响应格式
type Response struct {
	Data       interface{} `json:"data"`
	Message    string      `json:"msg"`
	StatusCode int         `json:"statusCode"`
	Errors     interface{} `json:"errors,omitempty"`
}

// ValidationError 验证错误详情
type ValidationError struct {
	Message string
	Errors  interface{}
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ResponseError 非预期响应错误（非 JSON、非 2xx 等）
type ResponseError struct {
	StatusCode  int
	ContentType string
	Body        string // 响应体前 512 字符
	Message     string // 用户友好的错误信息
}

func (e *ResponseError) Error() string {
	return e.Message
}

// Detail 返回包含调试信息的详细错误描述
func (e *ResponseError) Detail() string {
	return fmt.Sprintf("HTTP %d | Content-Type: %s\n响应体: %s", e.StatusCode, e.ContentType, e.Body)
}

// APIError 服务端返回的业务错误（JSON 格式，如 402/403/500 等）
type APIError struct {
	StatusCode int         // HTTP 状态码
	Message    string      // 服务端 message 字段
	ServerCode int         // 服务端 status_code 字段
	Errors     interface{} // 服务端 errors 字段（string 或 map[string]interface{}）
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API 错误 (%d): %s", e.StatusCode, e.Message)
}

// Client API 客户端
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient 创建新的 API 客户端
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{},
	}
}

// Do 执行 API 请求（向后兼容）
func (c *Client) Do(method, path string, body interface{}, result interface{}) error {
	return c.DoCtx(context.Background(), method, path, body, result)
}

// DoCtx 执行 API 请求（带 context，支持日志追踪）
func (c *Client) DoCtx(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	requestID := logging.RequestIDFrom(ctx)
	url := c.baseURL + path

	// body 序列化提前，使 data 在日志时可用
	var data []byte
	var reqBody io.Reader
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	attrs := []interface{}{
		"request_id", requestID,
		"method", method,
		"url", url,
	}
	if logging.IsDev() {
		attrs = append(attrs, "request_body", string(data))
	}
	slog.InfoContext(ctx, "api_request", attrs...)

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		duration := time.Since(start)
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")

	// 1. 非 2xx 状态码 + 非 JSON -> ResponseError
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if !isJSONContentType(contentType) {
			respErr := &ResponseError{
				StatusCode:  resp.StatusCode,
				ContentType: contentType,
				Body:        truncate(string(bodyBytes), 512),
				Message:     fmt.Sprintf("服务端返回异常 (HTTP %d)，请检查 base_url 配置或稍后重试", resp.StatusCode),
			}
			errAttrs := []interface{}{
				"request_id", requestID,
				"method", method,
				"url", url,
				"status", resp.StatusCode,
				"duration_ms", duration.Milliseconds(),
				"error", respErr.Message,
			}
			if logging.IsDev() {
				errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
			}
			slog.ErrorContext(ctx, "api_response", errAttrs...)
			return respErr
		}
	}

	// 2. 2xx 但 Content-Type 非 JSON 且非空 -> ResponseError
	if contentType != "" && !isJSONContentType(contentType) {
		respErr := &ResponseError{
			StatusCode:  resp.StatusCode,
			ContentType: contentType,
			Body:        truncate(string(bodyBytes), 512),
			Message:     "服务端返回非 JSON 响应，可能是 base_url 配置错误或需要重新认证",
		}
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", respErr.Message,
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
		return respErr
	}

	// 3. JSON 解码
	var apiResp Response
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 4. 业务错误处理
	if resp.StatusCode == http.StatusUnauthorized {
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", "unauthorized",
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
		return ErrUnauthorized
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
		return &ValidationError{
			Message: apiResp.Message,
			Errors:  apiResp.Errors,
		}
	}

	if resp.StatusCode >= 400 {
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    apiResp.Message,
			ServerCode: apiResp.StatusCode,
			Errors:     apiResp.Errors,
		}
	}

	// 4.5 HTTP 200 但 body 中业务码为错误（部分 API 用 HTTP 200 + body statusCode 报错）
	if apiResp.StatusCode >= 400 {
		errAttrs := []interface{}{
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		}
		if logging.IsDev() {
			errAttrs = append(errAttrs, "response_body", readableJSON(bodyBytes))
		}
		slog.ErrorContext(ctx, "api_response", errAttrs...)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    apiResp.Message,
			ServerCode: apiResp.StatusCode,
			Errors:     apiResp.Errors,
		}
	}

	// 5. 成功日志
	respAttrs := []interface{}{
		"request_id", requestID,
		"method", method,
		"url", url,
		"status", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	}
	if logging.IsDev() {
		respAttrs = append(respAttrs, "response_body", readableJSON(bodyBytes))
	}
	slog.InfoContext(ctx, "api_response", respAttrs...)

	// 6. 解析 data 到 result
	if result != nil && apiResp.Data != nil {
		data, err := json.Marshal(apiResp.Data)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, result); err != nil {
			return err
		}
	}

	return nil
}

// IsUnauthorized 检查是否是认证错误
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsValidationError 检查是否是验证错误
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// GetValidationErrors 获取验证错误详情
func GetValidationErrors(err error) map[string]interface{} {
	var ve *ValidationError
	if errors.As(err, &ve) {
		if m, ok := ve.Errors.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// GetValidationMessage 获取验证错误的消息
func GetValidationMessage(err error) string {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve.Message
	}
	return ""
}

// IsResponseError 检查是否是非预期响应错误
func IsResponseError(err error) bool {
	var re *ResponseError
	return errors.As(err, &re)
}

// IsAPIError 检查是否是 API 业务错误
func IsAPIError(err error) bool {
	var ae *APIError
	return errors.As(err, &ae)
}

// isJSONContentType 检查 Content-Type 是否包含 application/json
func isJSONContentType(ct string) bool {
	return strings.Contains(ct, "application/json")
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
