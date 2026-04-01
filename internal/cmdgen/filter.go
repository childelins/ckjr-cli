package cmdgen

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
