package yaml

import (
	"fmt"
	"io/fs"
	"strings"
)

// FS 持有嵌入的文件系统，提供 YAML 配置加载功能
type FS struct {
	fs fs.FS
}

// New 创建一个新的 YAML 配置加载器
func New(embedFS fs.FS) *FS {
	return &FS{fs: embedFS}
}

// LoadRoutes 读取 config/routes/ 下所有 .yaml 文件，返回文件名到内容的映射
func (f *FS) LoadRoutes() (map[string][]byte, error) {
	return f.loadDir("config/routes")
}

// LoadWorkflows 读取 config/workflows/ 下所有 .yaml 文件，返回文件名到内容的映射
func (f *FS) LoadWorkflows() (map[string][]byte, error) {
	return f.loadDir("config/workflows")
}

func (f *FS) loadDir(dir string) (map[string][]byte, error) {
	entries, err := fs.ReadDir(f.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录 %s 失败: %w", dir, err)
	}
	result := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := fs.ReadFile(f.fs, dir+"/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("读取文件 %s 失败: %w", entry.Name(), err)
		}
		result[entry.Name()] = data
	}
	return result, nil
}
