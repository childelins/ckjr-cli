package curlparse

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
)

// Result 保存 curl 解析结果
type Result struct {
	Method string           // HTTP method
	Path   string           // URL path
	Fields map[string]Field // 从 JSON body 提取的顶层字段
}

// Field 解析出的字段信息
type Field struct {
	Type    string      // 推断类型: string/int/bool
	Example interface{} // 原始值
}

// Parse 解析 curl 命令字符串
func Parse(curl string) (*Result, error) {
	// 预处理：去除续行符，合并为单行
	curl = strings.ReplaceAll(curl, "\\\n", " ")
	curl = strings.ReplaceAll(curl, "\\\r\n", " ")

	tokens := tokenize(curl)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("空的 curl 命令")
	}

	// 第一个 token 应该是 "curl"
	if tokens[0] != "curl" {
		return nil, fmt.Errorf("不是有效的 curl 命令")
	}

	result := &Result{Fields: make(map[string]Field)}
	var rawURL, dataRaw, method string

	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case tok == "-X" || tok == "--request":
			if i+1 < len(tokens) {
				method = strings.ToUpper(tokens[i+1])
				i++
			}
		case tok == "--data-raw" || tok == "-d" || tok == "--data":
			if i+1 < len(tokens) {
				dataRaw = tokens[i+1]
				i++
			}
		case tok == "-H" || tok == "--header":
			i++ // 跳过 header 值
		case !strings.HasPrefix(tok, "-") && rawURL == "":
			rawURL = tok
		}
	}

	if rawURL == "" {
		return nil, fmt.Errorf("未找到 URL")
	}

	// 解析 URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("URL 解析失败: %w", err)
	}
	result.Path = strings.TrimPrefix(u.Path, "/api")

	// 解析 query parameters
	for key, vals := range u.Query() {
		if len(vals) > 0 {
			result.Fields[key] = inferQueryParam(vals[0])
		}
	}

	// 确定 method
	if method != "" {
		result.Method = method
	} else if dataRaw != "" {
		result.Method = "POST"
	} else {
		result.Method = "GET"
	}

	// 解析 JSON body
	if dataRaw != "" {
		var body map[string]interface{}
		if err := json.Unmarshal([]byte(dataRaw), &body); err != nil {
			return nil, fmt.Errorf("JSON body 解析失败: %w", err)
		}
		for key, val := range body {
			f, ok := inferField(val)
			if ok {
				result.Fields[key] = f
			}
		}
	}

	return result, nil
}

// inferQueryParam 从 query string 值推断字段类型
func inferQueryParam(val string) Field {
	if i, err := strconv.Atoi(val); err == nil {
		return Field{Type: "int", Example: i}
	}
	if val == "true" || val == "false" {
		return Field{Type: "bool", Example: val == "true"}
	}
	return Field{Type: "string", Example: val}
}

// inferField 从 JSON 值推断字段类型，跳过数组和对象
func inferField(val interface{}) (Field, bool) {
	switch v := val.(type) {
	case float64:
		if v == math.Trunc(v) {
			return Field{Type: "int", Example: int(v)}, true
		}
		return Field{Type: "float", Example: v}, true
	case bool:
		return Field{Type: "bool", Example: v}, true
	case string:
		return Field{Type: "string", Example: v}, true
	case nil:
		return Field{Type: "string", Example: nil}, true
	default:
		// 数组、对象跳过
		return Field{}, false
	}
}

// tokenize 将 curl 命令分词，处理单引号和双引号
func tokenize(input string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') && !inSingle && !inDouble:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}
