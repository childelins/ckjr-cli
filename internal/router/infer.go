package router

// InferRouteName 从 URL path 末段推导 route name
func InferRouteName(path string) string {
	parts := splitPath(path)
	if len(parts) == 0 {
		return "unknown"
	}
	last := parts[len(parts)-1]

	prefixes := map[string]string{
		"modify": "update",
		"edit":   "update",
		"remove": "delete",
		"add":    "create",
		"create": "create",
		"query":  "list",
	}
	lower := toLower(last)
	for prefix, mapped := range prefixes {
		if len(lower) >= len(prefix) && lower[:len(prefix)] == prefix {
			return mapped
		}
	}

	if len(lower) >= 8 && lower[:8] == "describe" {
		return "get"
	}

	return last
}

// InferNameFromPath 从文件路径推导 name（resource 名称）
func InferNameFromPath(path string) string {
	parts := splitPath(path)
	if len(parts) == 0 {
		return "unknown"
	}
	filename := parts[len(parts)-1]
	for i := range filename {
		if i > 0 && filename[i-1] == '.' {
			return filename[:i-1]
		}
	}
	return filename
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range split(path, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func split(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
