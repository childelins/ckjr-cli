package updater

import (
	"strconv"
	"strings"
)

// CompareVersions 比较两个 semver 版本。
// 返回 >0 表示 current 比 latest 新，<0 表示有更新可用，0 表示相同。
func CompareVersions(current, latest string) (int, error) {
	if current == "" {
		return 0, nil
	}
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	curParts := strings.Split(current, ".")
	latParts := strings.Split(latest, ".")

	maxLen := len(curParts)
	if len(latParts) > maxLen {
		maxLen = len(latParts)
	}

	for i := 0; i < maxLen; i++ {
		cur := partToInt(curParts, i)
		lat := partToInt(latParts, i)
		if cur != lat {
			return cur - lat, nil
		}
	}
	return 0, nil
}

func partToInt(parts []string, idx int) int {
	if idx >= len(parts) {
		return 0
	}
	n, _ := strconv.Atoi(parts[idx])
	return n
}
