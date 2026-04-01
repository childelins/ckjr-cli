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

// deleteNestedPath 沿点号路径从 map 中删除，遇到数组自动穿透
func deleteNestedPath(m map[string]interface{}, path string) bool {
	parts := strings.Split(path, ".")
	return deleteNestedParts(m, parts)
}

func deleteNestedParts(m map[string]interface{}, parts []string) bool {
	if len(parts) == 1 {
		_, exists := m[parts[0]]
		if !exists {
			return false
		}
		delete(m, parts[0])
		return true
	}
	val, ok := m[parts[0]]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case map[string]interface{}:
		return deleteNestedParts(v, parts[1:])
	case []interface{}:
		deleted := false
		for _, elem := range v {
			if em, ok := elem.(map[string]interface{}); ok {
				if deleteNestedParts(em, parts[1:]) {
					deleted = true
				}
			}
		}
		return deleted
	default:
		return false
	}
}

// deepCopyValue 递归深拷贝 map/array/原始值
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return deepCopyMap(val)
	case []interface{}:
		cp := make([]interface{}, len(val))
		for i, elem := range val {
			cp[i] = deepCopyValue(elem)
		}
		return cp
	default:
		return v
	}
}

// deepCopyMap 递归深拷贝 map
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{}, len(m))
	for k, v := range m {
		cp[k] = deepCopyValue(v)
	}
	return cp
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

// filterByExclude 移除 exclude 中列出的 key，支持点号路径删除嵌套字段
// 返回深拷贝后的 map，不修改原始 m
func filterByExclude(m map[string]interface{}, exclude []string) map[string]interface{} {
	filtered := deepCopyMap(m)
	for _, e := range exclude {
		if strings.Contains(e, ".") {
			deleteNestedPath(filtered, e)
		} else {
			delete(filtered, e)
		}
	}
	return filtered
}

// FilterResponse 根据 Route 的 response 配置过滤 result 的字段
// 返回过滤后的新 map，不修改原始 result
func FilterResponse(result interface{}, respFilter *router.ResponseFilter) interface{} {
	if respFilter == nil {
		return result
	}

	m, ok := result.(map[string]interface{})
	if !ok {
		return result
	}

	if len(respFilter.Fields) > 0 {
		return filterByFields(m, respFilter.Fields)
	}

	if len(respFilter.Exclude) > 0 {
		return filterByExclude(m, respFilter.Exclude)
	}

	return result
}
