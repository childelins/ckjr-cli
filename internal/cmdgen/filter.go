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
