package cmdgen

import (
	"strings"

	"github.com/childelins/ckjr-cli/internal/router"
)

// getNestedValue 沿点号路径在 map 中取值，遇到数组自动穿透
// "list.data.courseId" -> 穿透 data 数组，对每个元素取 courseId
func getNestedValue(m map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = m
	for i, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		case []interface{}:
			remaining := strings.Join(parts[i:], ".")
			var results []interface{}
			for _, elem := range v {
				em, ok := elem.(map[string]interface{})
				if !ok {
					continue
				}
				if val, ok := getNestedValue(em, remaining); ok {
					results = append(results, val)
				}
			}
			if len(results) == 0 {
				return nil, false
			}
			return results, true
		default:
			return nil, false
		}
	}
	return current, true
}

// setNestedValue 沿点号路径在 map 中设值，自动创建中间 map
func setNestedValue(m map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := m
	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]]
		if !ok {
			next = make(map[string]interface{})
			current[parts[i]] = next
		}
		current = next.(map[string]interface{})
	}
	current[parts[len(parts)-1]] = value
}

// filterByFields 仅保留 fields 中列出的 key，支持点号路径和数组穿透
func filterByFields(m map[string]interface{}, fields []string) map[string]interface{} {
	filtered := make(map[string]interface{})
	for _, f := range fields {
		applyFieldPath(m, filtered, strings.Split(f, "."))
	}
	return filtered
}

// applyFieldPath 递归构建过滤后的 map，遇到数组自动穿透
func applyFieldPath(src, dst map[string]interface{}, parts []string) {
	if len(parts) == 0 {
		return
	}
	key := parts[0]
	val, ok := src[key]
	if !ok {
		return
	}
	if len(parts) == 1 {
		dst[key] = val
		return
	}
	remaining := parts[1:]
	switch v := val.(type) {
	case map[string]interface{}:
		sub, ok := dst[key].(map[string]interface{})
		if !ok {
			sub = make(map[string]interface{})
		}
		applyFieldPath(v, sub, remaining)
		if len(sub) > 0 {
			dst[key] = sub
		}
	case []interface{}:
		existingArr, _ := dst[key].([]interface{})
		if existingArr == nil {
			existingArr = make([]interface{}, len(v))
			dst[key] = existingArr
		}
		for i, elem := range v {
			em, ok := elem.(map[string]interface{})
			if !ok {
				continue
			}
			dm, ok := existingArr[i].(map[string]interface{})
			if !ok {
				dm = make(map[string]interface{})
				existingArr[i] = dm
			}
			applyFieldPath(em, dm, remaining)
		}
	}
}

// FilterResponse 根据 Route 的 response 配置过滤 result 的字段
// 返回过滤后的新 map，不修改原始 result
func FilterResponse(result interface{}, respFilter *router.ResponseFilter) interface{} {
	if respFilter == nil || len(respFilter.Fields) == 0 {
		return result
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		return result
	}
	return filterByFields(m, respFilter.FieldPaths())
}
