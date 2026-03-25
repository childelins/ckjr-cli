package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

// Do 执行 API 请求
func (c *Client) Do(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var apiResp Response
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 处理错误状态
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return &ValidationError{
			Message: apiResp.Message,
			Errors:  apiResp.Errors,
		}
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API 错误 (%d): %s", resp.StatusCode, apiResp.Message)
	}

	// 解析 data 到 result
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
