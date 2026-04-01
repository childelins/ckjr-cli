package cmdgen

import "github.com/childelins/ckjr-cli/internal/router"

// filterByFields 仅保留 fields 中列出的顶层 key
func filterByFields(m map[string]interface{}, fields []string) map[string]interface{} {
	allowed := make(map[string]bool, len(fields))
	for _, f := range fields {
		allowed[f] = true
	}

	filtered := make(map[string]interface{})
	for k, v := range m {
		if allowed[k] {
			filtered[k] = v
		}
	}
	return filtered
}

// filterByExclude 移除 exclude 中列出的顶层 key
func filterByExclude(m map[string]interface{}, exclude []string) map[string]interface{} {
	excluded := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		excluded[e] = true
	}

	filtered := make(map[string]interface{}, len(m))
	for k, v := range m {
		if !excluded[k] {
			filtered[k] = v
		}
	}
	return filtered
}

// FilterResponse 根据 Route 的 response 配置过滤 result 的顶层字段
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
