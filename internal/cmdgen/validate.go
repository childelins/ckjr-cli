package cmdgen

import (
	"fmt"
	"math"
	"regexp"
	"time"

	"github.com/childelins/ckjr-cli/internal/router"
)

// FieldValidationError 单个字段的校验错误
type FieldValidationError struct {
	Field   string
	Message string
}

func (e *FieldValidationError) Error() string {
	return fmt.Sprintf("字段 %q %s", e.Field, e.Message)
}

// ValidateAll 校验输入数据，返回所有校验错误
func ValidateAll(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	var errs []*FieldValidationError

	// 1. required 校验
	errs = append(errs, validateRequiredErrors(input, template)...)

	// 2. 类型校验
	errs = append(errs, validateTypes(input, template)...)

	// 3. 约束校验
	errs = append(errs, validateConstraints(input, template)...)

	return errs
}

func validateRequiredErrors(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	var errs []*FieldValidationError
	for name, field := range template {
		if IsPathParam(field) {
			continue
		}
		if field.Required {
			if _, exists := input[name]; !exists {
				errs = append(errs, &FieldValidationError{Field: name, Message: "为必填字段"})
			}
		}
	}
	return errs
}

func validateTypes(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	var errs []*FieldValidationError
	for name, field := range template {
		if field.Type == "" || IsPathParam(field) {
			continue
		}
		val, exists := input[name]
		if !exists {
			continue
		}
		if err := validateType(name, val, field.Type); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func validateType(fieldName string, value interface{}, expectedType string) *FieldValidationError {
	if expectedType == "" {
		return nil
	}

	if value == nil {
		return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 %s，实际为 null", expectedType)}
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 string，实际为 %T", value)}
		}
	case "int":
		v, ok := value.(float64)
		if !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 int，实际为 %T", value)}
		}
		if v != math.Trunc(v) {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 int，实际为浮点数 %v", v)}
		}
	case "float":
		if _, ok := value.(float64); !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 float，实际为 %T", value)}
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 bool，实际为 %T", value)}
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("类型应为 array，实际为 %T", value)}
		}
	case "date":
		str, ok := value.(string)
		if !ok {
			return &FieldValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("类型应为 date（字符串格式），实际为 %T", value),
			}
		}
		const dateLayout = "2006-01-02 15:04:05"
		if _, err := time.Parse(dateLayout, str); err != nil {
			return &FieldValidationError{
				Field:   fieldName,
				Message: fmt.Sprintf("日期格式应为 YYYY-MM-DD HH:MM:SS，实际值 %q 无效: %s", str, err.Error()),
			}
		}
	default:
		return &FieldValidationError{Field: fieldName, Message: fmt.Sprintf("未知类型 %q", expectedType)}
	}

	return nil
}

func validateConstraints(input map[string]interface{}, template map[string]router.Field) []*FieldValidationError {
	patterns, patternErr := compilePatterns(template)
	if patternErr != nil {
		return []*FieldValidationError{patternErr}
	}

	var errs []*FieldValidationError
	for name, field := range template {
		if IsPathParam(field) {
			continue
		}
		val, exists := input[name]
		if !exists || val == nil {
			continue
		}

		switch field.Type {
		case "int", "float":
			v, ok := val.(float64)
			if !ok {
				continue
			}
			if field.Min != nil && v < *field.Min {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: fmt.Sprintf("值 %v 小于最小值 %v", v, *field.Min),
				})
			}
			if field.Max != nil && v > *field.Max {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: fmt.Sprintf("值 %v 大于最大值 %v", v, *field.Max),
				})
			}

		case "string":
			str, ok := val.(string)
			if !ok {
				continue
			}
			if field.MinLength != nil && len(str) < *field.MinLength {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: fmt.Sprintf("长度 %d 小于最小长度 %d", len(str), *field.MinLength),
				})
			}
			if field.MaxLength != nil && len(str) > *field.MaxLength {
				errs = append(errs, &FieldValidationError{
					Field:   name,
					Message: fmt.Sprintf("长度 %d 大于最大长度 %d", len(str), *field.MaxLength),
				})
			}
			if field.Pattern != "" {
				re := patterns[name]
				if !re.MatchString(str) {
					errs = append(errs, &FieldValidationError{
						Field:   name,
						Message: fmt.Sprintf("值 %q 不匹配正则 %q", str, field.Pattern),
					})
				}
			}
		}
	}
	return errs
}

// compilePatterns 预编译 template 中的正则表达式
func compilePatterns(template map[string]router.Field) (map[string]*regexp.Regexp, *FieldValidationError) {
	patterns := make(map[string]*regexp.Regexp)
	for name, field := range template {
		if field.Pattern == "" {
			continue
		}
		re, err := regexp.Compile(field.Pattern)
		if err != nil {
			return nil, &FieldValidationError{
				Field:   name,
				Message: fmt.Sprintf("正则表达式编译失败: %s", field.Pattern),
			}
		}
		patterns[name] = re
	}
	return patterns, nil
}
