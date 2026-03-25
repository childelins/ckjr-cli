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

// ErrUnauthorized API Key 无效或过期
var ErrUnauthorized = errors.New("api_key 已过期，请重新登录获取")

// ErrValidation 参数校验失败
var ErrValidation = errors.New("参数校验失败")

// Response Dingo API 响应格式
type Response struct {
	Data       interface{}            `json:"data"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"status_code"`
	Errors     map[string]interface{} `json:"errors,omitempty"`
}

// ValidationError 验证错误详情
type ValidationError struct {
	Message string
	Errors  map[string]interface{}
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

	slog.InfoContext(ctx, "api_request",
		"request_id", requestID,
		"method", method,
		"url", url,
	)

	start := time.Now()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

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
			slog.ErrorContext(ctx, "api_response",
				"request_id", requestID,
				"method", method,
				"url", url,
				"status", resp.StatusCode,
				"duration_ms", duration.Milliseconds(),
				"error", respErr.Message,
			)
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
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", respErr.Message,
		)
		return respErr
	}

	// 3. JSON 解码
	var apiResp Response
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 4. 业务错误处理
	if resp.StatusCode == http.StatusUnauthorized {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", "unauthorized",
		)
		return ErrUnauthorized
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		)
		return &ValidationError{
			Message: apiResp.Message,
			Errors:  apiResp.Errors,
		}
	}

	if resp.StatusCode >= 400 {
		slog.ErrorContext(ctx, "api_response",
			"request_id", requestID,
			"method", method,
			"url", url,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"error", apiResp.Message,
		)
		return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiResp.Message)
	}

	// 5. 成功日志
	slog.InfoContext(ctx, "api_response",
		"request_id", requestID,
		"method", method,
		"url", url,
		"status", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	)

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
		return ve.Errors
	}
	return nil
}

// IsResponseError 检查是否是非预期响应错误
func IsResponseError(err error) bool {
	var re *ResponseError
	return errors.As(err, &re)
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
