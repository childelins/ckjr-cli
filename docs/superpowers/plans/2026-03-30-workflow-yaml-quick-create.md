# Workflow YAML 快速创建实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 提供 `InitWorkflowFile(path, moduleName)` 函数，快速生成 workflow YAML 骨架文件

**Architecture:** 扩展 `internal/yamlgen` 包新增 `workflow.go`，与现有 `generate.go` 的 `CreateFile` 模式对称。函数接受 path 参数确保可测试性。

**Tech Stack:** Go, gopkg.in/yaml.v3

---

## 文件结构

- Create: `internal/yamlgen/workflow.go` — InitWorkflowFile 函数
- Create: `internal/yamlgen/workflow_test.go` — 测试

## 函数签名

与 route 的 `CreateFile(path, name, nameDesc, routeName, route)` 对称：

```go
// InitWorkflowFile 创建模块 workflow 骨架文件
// path: 目标文件路径（如 cmd/ckjr-cli/workflows/asset.yaml）
// moduleName: 模块名（用于 name 和 description 字段）
func InitWorkflowFile(path string, moduleName string) error
```

---

### Task 1: 实现 InitWorkflowFile 函数

**Files:**
- Create: `internal/yamlgen/workflow.go`
- Reference: `internal/yamlgen/generate.go:50-62` — CreateFile 模式
- Reference: `internal/workflow/workflow.go:25-38` — Workflow/Config struct

- [ ] **Step 1: 创建 workflow.go**

```go
package yamlgen

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/childelins/ckjr-cli/internal/workflow"
)

// InitWorkflowFile 创建模块 workflow 骨架文件
func InitWorkflowFile(path string, moduleName string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("文件已存在: %s", path)
	}

	cfg := &workflow.Config{
		Name:        moduleName,
		Description: moduleName,
		Workflows: map[string]workflow.Workflow{
			"workflow-name": {
				Description: "工作流描述",
				Triggers:    []string{},
				Inputs:      []workflow.Input{},
				Steps:       []workflow.Step{},
			},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("YAML 序列化失败: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 2: 验证编译通过**

Run: `go build ./internal/yamlgen/`
Expected: 无错误

---

### Task 2: 编写测试

**Files:**
- Create: `internal/yamlgen/workflow_test.go`
- Reference: `internal/yamlgen/generate_test.go:116-158` — TestCreateFile 模式

- [ ] **Step 1: 创建测试文件**

```go
package yamlgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/childelins/ckjr-cli/internal/workflow"
)

func TestInitWorkflowFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "asset.yaml")

	if err := InitWorkflowFile(path, "asset"); err != nil {
		t.Fatalf("InitWorkflowFile() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	cfg, err := workflow.Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if cfg.Name != "asset" {
		t.Errorf("Name = %q, want asset", cfg.Name)
	}
	if cfg.Description != "asset" {
		t.Errorf("Description = %q, want asset", cfg.Description)
	}
	if _, ok := cfg.Workflows["workflow-name"]; !ok {
		t.Fatal("workflow-name not found")
	}
	wf := cfg.Workflows["workflow-name"]
	if wf.Description != "工作流描述" {
		t.Errorf("Description = %q, want 工作流描述", wf.Description)
	}
	if len(wf.Triggers) != 0 {
		t.Errorf("Triggers = %v, want empty", wf.Triggers)
	}
	if len(wf.Inputs) != 0 {
		t.Errorf("Inputs = %v, want empty", wf.Inputs)
	}
	if len(wf.Steps) != 0 {
		t.Errorf("Steps = %v, want empty", wf.Steps)
	}
}

func TestInitWorkflowFile_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.yaml")
	os.WriteFile(path, []byte("test"), 0644)

	err := InitWorkflowFile(path, "test")
	if err == nil {
		t.Error("expected file exists error")
	}
}
```

- [ ] **Step 2: 运行测试**

Run: `go test ./internal/yamlgen/ -run TestInitWorkflowFile -v`
Expected: PASS

- [ ] **Step 3: 运行全部 yamlgen 测试确认无回归**

Run: `go test ./internal/yamlgen/ -v`
Expected: 全部 PASS

---

### Task 3: 提交

- [ ] **Step 1: 提交代码**

```bash
git add internal/yamlgen/workflow.go internal/yamlgen/workflow_test.go
git commit -m "feat(yamlgen): add InitWorkflowFile for quick workflow YAML skeleton creation"
```
