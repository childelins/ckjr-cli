# Version Flag ldflags 注入修复实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 --version flag 的 ldflags 注入问题，将 Version/Environment 变量移到 main 包保持短路径

**Architecture:** 在 main 包定义 Version/Environment 变量供 ldflags 注入，通过 setter 方法传递给 cmd 包。release.yml 无需修改。

**Tech Stack:** Go 1.24, Cobra

---

### Task 1: 重构 cmd 包的 Version/Environment 为私有变量 + setter

**Files:**
- Modify: `cmd/root.go:21-26`
- Modify: `cmd/root_test.go:112-116`
- Test: `cmd/root_test.go`

- [ ] **Step 1: 先写失败测试 - SetVersion 和 SetEnvironment**

在 `cmd/root_test.go` 中新增测试：

```go
func TestSetVersion(t *testing.T) {
	SetVersion("v1.2.3")
	if rootCmd.Version != "v1.2.3" {
		t.Errorf("rootCmd.Version = %s, want v1.2.3", rootCmd.Version)
	}
}

func TestSetEnvironment(t *testing.T) {
	SetEnvironment("development")
	if environment != "development" {
		t.Errorf("environment = %s, want development", environment)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./cmd/ -run "TestSetVersion|TestSetEnvironment" -v`
Expected: FAIL（SetVersion/SetEnvironment 未定义）

- [ ] **Step 3: 修改 cmd/root.go**

将 `cmd/root.go` 的变量声明从：

```go
var (
	Version      = "dev"
	Environment = "production"
)
```

改为：

```go
var (
	version      = "dev"
	environment  = "production"
)

// SetVersion 由 main 包调用，通过 ldflags 注入版本号
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// SetEnvironment 由 main 包调用，通过 ldflags 注入环境模式
func SetEnvironment(e string) {
	environment = e
}
```

同时更新 `initLogging` 中对 `Environment` 的引用为 `environment`：

```go
env := logging.ParseEnvironment(environment)
```

注意：`rootCmd` 的 `Version` 字段在声明时写死为 `Version` 的值（Cobra 初始化），改为 `version` 后需要手动同步。在 `SetVersion` 中已处理 `rootCmd.Version = v`，但还需要处理未调用 setter 时的默认值 —— 在 `rootCmd` 声明中改为 `Version: version`（Go 允许这样写，因为 `version` 在同一个包内可见）。

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./cmd/ -run "TestSetVersion|TestSetEnvironment" -v`
Expected: PASS

- [ ] **Step 5: 运行全部现有测试确保无回归**

Run: `go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add cmd/root.go cmd/root_test.go
git commit -m "refactor: move Version/Environment to private vars with setters"
```

### Task 2: 在 main 包定义 Version/Environment 供 ldflags 注入

**Files:**
- Modify: `cmd/ckjr-cli/main.go`
- Test: 手动构建验证

- [ ] **Step 1: 修改 cmd/ckjr-cli/main.go**

将文件内容从：

```go
package main

import "github.com/childelins/ckjr-cli/cmd"

func main() {
	cmd.Execute()
}
```

改为：

```go
package main

import "github.com/childelins/ckjr-cli/cmd"

var (
	// Version 版本号，通过 -ldflags "-X main.Version=x.x.x" 注入
	Version = "dev"
	// Environment 环境模式，通过 -ldflags "-X main.Environment=production" 注入
	Environment = "production"
)

func init() {
	cmd.SetVersion(Version)
	cmd.SetEnvironment(Environment)
}

func main() {
	cmd.Execute()
}
```

- [ ] **Step 2: 运行全部测试确保无回归**

Run: `go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 3: 不带 ldflags 构建验证默认值**

Run: `go build -o /tmp/ckjr-cli-test ./cmd/ckjr-cli && /tmp/ckjr-cli-test --version`
Expected: `ckjr-cli version dev`

- [ ] **Step 4: 带 ldflags 构建验证注入**

Run: `go build -ldflags="-X main.Version=v9.9.9 -X main.Environment=production" -o /tmp/ckjr-cli-test ./cmd/ckjr-cli && /tmp/ckjr-cli-test --version`
Expected: `ckjr-cli version v9.9.9`

- [ ] **Step 5: 提交**

```bash
git add cmd/ckjr-cli/main.go
git commit -m "feat: define Version/Environment in main package for ldflags injection"
```

### Task 3: 补充版本默认值测试

**Files:**
- Modify: `cmd/root_test.go`

- [ ] **Step 1: 新增默认值测试**

在 `cmd/root_test.go` 中新增：

```go
func TestDefaultVersion(t *testing.T) {
	// 未调用 SetVersion 时，version 应为默认值 "dev"
	if version != "dev" {
		t.Errorf("default version = %s, want dev", version)
	}
}

func TestDefaultEnvironment(t *testing.T) {
	if environment != "production" {
		t.Errorf("default environment = %s, want production", environment)
	}
}
```

- [ ] **Step 2: 运行测试确认通过**

Run: `go test ./cmd/ -run "TestDefault" -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add cmd/root_test.go
git commit -m "test: add default value tests for version and environment"
```
