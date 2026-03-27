# YAML 配置文件迁移到 config/ 实施计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 将 cmd/routes/ 和 cmd/workflows/ 下的 YAML 文件迁移到根目录 config/routes/ 和 config/workflows/，并通过 internal/config/yaml 包集中管理 embed 和加载逻辑。

**Architecture:** 由于 Go `//go:embed` 只能引用当前 Go 文件所在目录及子目录，embed 声明放在根目录 main 包的 embed.go 中。加载逻辑集中在 internal/config/yaml 包，cmd 包通过注入的 yaml.FS 加载配置。

**Tech Stack:** Go embed, testing/fstest

---

### Task 1: 创建 internal/config/yaml 包（TDD）

**Files:**
- Create: `internal/config/yaml/yaml_test.go`
- Create: `internal/config/yaml/yaml.go`

- [ ] **Step 1: 创建目录并编写失败测试**

```bash
mkdir -p internal/config/yaml
```

```go
// internal/config/yaml/yaml_test.go
package yaml

import (
	"testing"
	"testing/fstest"
)

func TestLoadRoutes(t *testing.T) {
	memFS := fstest.MapFS{
		"config/routes/agent.yaml":  {Data: []byte("name: agent\ndescription: test\nroutes: {}")},
		"config/routes/common.yaml": {Data: []byte("name: common\ndescription: common\nroutes: {}")},
		"config/routes/readme.txt":  {Data: []byte("ignored")},
		"config/routes/sub/.keep":   {Data: []byte("")},
	}
	loader := New(memFS)
	files, err := loader.LoadRoutes()
	if err != nil {
		t.Fatalf("LoadRoutes() error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("LoadRoutes() got %d files, want 2", len(files))
	}
	if _, ok := files["agent.yaml"]; !ok {
		t.Error("LoadRoutes() missing agent.yaml")
	}
	if _, ok := files["common.yaml"]; !ok {
		t.Error("LoadRoutes() missing common.yaml")
	}
	if _, ok := files["readme.txt"]; ok {
		t.Error("LoadRoutes() should skip .txt files")
	}
}

func TestLoadRoutes_EmptyDir(t *testing.T) {
	memFS := fstest.MapFS{
		"config/routes/readme.txt": {Data: []byte("ignored")},
	}
	loader := New(memFS)
	files, err := loader.LoadRoutes()
	if err != nil {
		t.Fatalf("LoadRoutes() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("LoadRoutes() got %d files, want 0", len(files))
	}
}

func TestLoadRoutes_NonexistentDir(t *testing.T) {
	memFS := fstest.MapFS{}
	loader := New(memFS)
	_, err := loader.LoadRoutes()
	if err == nil {
		t.Fatal("LoadRoutes() expected error for nonexistent dir")
	}
}

func TestLoadWorkflows(t *testing.T) {
	memFS := fstest.MapFS{
		"config/workflows/agent.yaml": {Data: []byte("name: agent\nworkflows: {}")},
		"config/workflows/note.txt":   {Data: []byte("ignored")},
	}
	loader := New(memFS)
	files, err := loader.LoadWorkflows()
	if err != nil {
		t.Fatalf("LoadWorkflows() error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("LoadWorkflows() got %d files, want 1", len(files))
	}
	if _, ok := files["agent.yaml"]; !ok {
		t.Error("LoadWorkflows() missing agent.yaml")
	}
}

func TestLoadWorkflows_NonexistentDir(t *testing.T) {
	memFS := fstest.MapFS{}
	loader := New(memFS)
	_, err := loader.LoadWorkflows()
	if err == nil {
		t.Fatal("LoadWorkflows() expected error for nonexistent dir")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/config/yaml/ -v`
Expected: FAIL (包不存在)

- [ ] **Step 3: 实现 yaml.go**

```go
// internal/config/yaml/yaml.go
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
		data, err := f.fs.ReadFile(dir + "/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("读取文件 %s 失败: %w", entry.Name(), err)
		}
		result[entry.Name()] = data
	}
	return result, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/config/yaml/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/yaml/yaml.go internal/config/yaml/yaml_test.go
git commit -m "feat: add internal/config/yaml package for centralized YAML loading"
```

---

### Task 2: 迁移 YAML 文件 + 创建 embed.go

**Files:**
- Create: `config/routes/agent.yaml`（从 cmd/routes/ 复制）
- Create: `config/routes/common.yaml`（从 cmd/routes/ 复制）
- Create: `config/workflows/agent.yaml`（从 cmd/workflows/ 复制）
- Create: `embed.go`（根目录）
- Delete: `cmd/routes/agent.yaml`
- Delete: `cmd/routes/common.yaml`
- Delete: `cmd/workflows/agent.yaml`

- [ ] **Step 1: 创建 config 目录并复制 YAML 文件**

```bash
mkdir -p config/routes config/workflows
cp cmd/routes/agent.yaml config/routes/agent.yaml
cp cmd/routes/common.yaml config/routes/common.yaml
cp cmd/workflows/agent.yaml config/workflows/agent.yaml
```

- [ ] **Step 2: 创建根目录 embed.go**

```go
// embed.go
package main

import "embed"

//go:embed all:config
var configFS embed.FS
```

- [ ] **Step 3: 编译验证 embed 正常**

Run: `go build ./...`
Expected: 编译成功（此时 cmd/ 包仍用旧的 embed，不影响编译）

- [ ] **Step 4: Commit**

```bash
git add config/ embed.go
git commit -m "feat: add config/ directory with YAML files and root embed.go"
```

---

### Task 3: 修改 cmd/ 包使用 yaml.FS

**Files:**
- Modify: `cmd/root.go` — 移除 embed，使用 yaml.FS
- Modify: `cmd/workflow.go` — 移除 embed，使用 yaml.FS
- Modify: `main.go` — 注入 configFS

- [ ] **Step 1: 修改 cmd/root.go**

移除 `//go:embed routes`、`routesFS` 变量、`embed` 导入。新增 `yamlFS` 变量和 `SetYAMLFS` 函数。修改 `registerRouteCommands()` 使用 `yamlFS.LoadRoutes()`。

修改后 `cmd/root.go` 关键部分：

```go
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/api"
	"github.com/childelins/ckjr-cli/internal/cmdgen"
	"github.com/childelins/ckjr-cli/internal/config"
	"github.com/childelins/ckjr-cli/internal/logging"
	"github.com/childelins/ckjr-cli/internal/router"
	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
)

var yamlFS *configyaml.FS

// SetYAMLFS 设置 YAML 配置加载器，由 main 包调用
func SetYAMLFS(fs *configyaml.FS) {
	yamlFS = fs
}
```

`registerRouteCommands()` 修改为：

```go
func registerRouteCommands() {
	if yamlFS == nil {
		fmt.Fprintf(os.Stderr, "YAML 文件系统未初始化\n")
		return
	}

	files, err := yamlFS.LoadRoutes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取路由目录失败: %v\n", err)
		return
	}

	for name, data := range files {
		cfg, err := router.Parse(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "解析路由文件 %s 失败: %v\n", name, err)
			continue
		}

		cmd := cmdgen.BuildCommand(cfg, createClient)
		rootCmd.AddCommand(cmd)
	}
}
```

- [ ] **Step 2: 修改 cmd/workflow.go**

移除 `//go:embed workflows`、`workflowsFS` 变量、`embed` 和 `io/fs` 导入。修改 `loadAllWorkflows()` 使用 `yamlFS.LoadWorkflows()`。

`loadAllWorkflows()` 修改为：

```go
func loadAllWorkflows() ([]*workflow.Config, error) {
	if yamlFS == nil {
		return nil, fmt.Errorf("YAML 文件系统未初始化")
	}

	files, err := yamlFS.LoadWorkflows()
	if err != nil {
		return nil, err
	}

	var configs []*workflow.Config
	for name, data := range files {
		cfg, err := workflow.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %w", name, err)
		}
		configs = append(configs, cfg)
	}
	return configs, nil
}
```

- [ ] **Step 3: 修改 main.go**

```go
package main

import (
	"github.com/childelins/ckjr-cli/cmd"
	configyaml "github.com/childelins/ckjr-cli/internal/config/yaml"
)

func main() {
	cmd.SetYAMLFS(configyaml.New(configFS))
	cmd.Execute()
}
```

- [ ] **Step 4: 编译验证**

Run: `go build ./...`
Expected: PASS

- [ ] **Step 5: 运行全部测试**

Run: `go test ./... -v`
Expected: PASS（除了 workflow_test.go 路径问题，在 Task 4 修复）

- [ ] **Step 6: Commit**

```bash
git add cmd/root.go cmd/workflow.go main.go
git commit -m "refactor: cmd package uses yaml.FS instead of direct embed"
```

---

### Task 4: 更新测试和文档路径

**Files:**
- Modify: `internal/workflow/workflow_test.go:158` — 路径 `../../cmd/workflows/agent.yaml` → `../../config/workflows/agent.yaml`
- Modify: `wiki/core-concepts.md` — `cmd/routes/` → `config/routes/`，`cmd/workflows/` → `config/workflows/`
- Modify: `wiki/extending.md` — `cmd/routes/` → `config/routes/`
- Modify: `wiki/project-structure.md` — 更新目录结构和数据流描述

- [ ] **Step 1: 更新 workflow_test.go 路径**

在 `internal/workflow/workflow_test.go` 第 158 行：
```
../../cmd/workflows/agent.yaml → ../../config/workflows/agent.yaml
```

- [ ] **Step 2: 运行测试确认通过**

Run: `go test ./internal/workflow/ -v`
Expected: PASS

- [ ] **Step 3: 更新 wiki/core-concepts.md**

替换所有 `cmd/routes/` 为 `config/routes/`，`cmd/workflows/` 为 `config/workflows/`（共 3 处）。

- [ ] **Step 4: 更新 wiki/extending.md**

替换所有 `cmd/routes/` 为 `config/routes/`（共 4 处）。

- [ ] **Step 5: 更新 wiki/project-structure.md**

替换数据流中的路径描述（1 处），目录结构中的路径描述。

- [ ] **Step 6: 运行全部测试确认通过**

Run: `go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 7: Commit**

```bash
git add internal/workflow/workflow_test.go wiki/core-concepts.md wiki/extending.md wiki/project-structure.md
git commit -m "docs: update file path references from cmd/ to config/"
```

---

### Task 5: 清理旧文件

**Files:**
- Delete: `cmd/routes/agent.yaml`
- Delete: `cmd/routes/common.yaml`
- Delete: `cmd/workflows/agent.yaml`
- Delete: `cmd/routes/`（空目录）
- Delete: `cmd/workflows/`（空目录）

- [ ] **Step 1: 删除旧的 YAML 文件和目录**

```bash
rm cmd/routes/agent.yaml cmd/routes/common.yaml cmd/workflows/agent.yaml
rmdir cmd/routes cmd/workflows
```

- [ ] **Step 2: 编译和测试**

Run: `go build ./... && go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 3: Commit**

```bash
git add -u cmd/routes/ cmd/workflows/
git commit -m "chore: remove old cmd/routes and cmd/workflows YAML files"
```
