# CLI 重命名 (ckjr -> ckjr-cli) 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 CLI 二进制名称从 `ckjr` 改为 `ckjr-cli`，公司名称"创客匠人"体现在描述中

**Architecture:** 纯重命名/文本替换任务，涉及入口目录、cobra 命令定义、构建配置、安装脚本、技能文件、项目文档。配置目录 `~/.ckjr/` 和 Go module 路径不变。

**Tech Stack:** Go, cobra, GitHub Actions

**Spec:** `docs/superpowers/specs/2026-03-26-cli-rename-design.md`

---

## 文件结构

| 文件 | 操作 | 说明 |
|------|------|------|
| `cmd/ckjr/main.go` | 重命名目录为 `cmd/ckjr-cli/main.go` | 入口文件，`go install` 以目录名生成二进制 |
| `cmd/root.go` | 修改 | `Use` 字段、`Short` 描述、错误提示 |
| `cmd/config.go` | 修改 | 错误提示 |
| `cmd/root_test.go` | 修改 | `Use` 字段断言 |
| `.github/workflows/release.yml` | 修改 | `BINARY_NAME`、构建路径、dist 目录名 |
| `install.sh` | 修改 | `BINARY_NAME`、`go install` 路径 |
| `skills/ckjr-agent/SKILL.md` | 修改 | 所有命令调用 |
| `skills/ckjr-agent/README.md` | 修改 | 所有命令调用 |
| `README.md` | 修改 | 所有命令调用、安装路径、描述 |

**不变项:** `~/.ckjr/` 配置目录、Go module 路径 `github.com/childelins/ckjr-cli`、`main.go`（根目录）、`docs/` 历史文档

---

### Task 1: 更新测试断言 (TDD - 先改测试)

**Files:**
- Modify: `cmd/root_test.go:13`

- [ ] **Step 1: 更新 root_test.go 中 Use 字段断言**

```go
// cmd/root_test.go:13 行
// 旧:
if rootCmd.Use != "ckjr" {
    t.Errorf("rootCmd.Use = %s, want ckjr", rootCmd.Use)
}
// 新:
if rootCmd.Use != "ckjr-cli" {
    t.Errorf("rootCmd.Use = %s, want ckjr-cli", rootCmd.Use)
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `go test ./cmd/ -run TestRootCmdExists -v`
Expected: FAIL，`rootCmd.Use = ckjr, want ckjr-cli`

---

### Task 2: 更新 cobra 命令定义

**Files:**
- Modify: `cmd/root.go:27-28`
- Modify: `cmd/root.go:111`

- [ ] **Step 1: 更新 rootCmd Use 和 Short 字段**

```go
// cmd/root.go:26-30
// 旧:
var rootCmd = &cobra.Command{
	Use:     "ckjr",
	Short:   "Claude Code 与公司 SaaS 平台的桥梁",
	Version: Version,
}
// 新:
var rootCmd = &cobra.Command{
	Use:     "ckjr-cli",
	Short:   "创客匠人 CLI - 知识付费 SaaS 系统的命令行工具",
	Version: Version,
}
```

- [ ] **Step 2: 更新 createClient 错误提示**

```go
// cmd/root.go:111
// 旧:
return nil, fmt.Errorf("未找到配置文件，请先执行 ckjr config init")
// 新:
return nil, fmt.Errorf("未找到配置文件，请先执行 ckjr-cli config init")
```

- [ ] **Step 3: 运行测试，确认通过**

Run: `go test ./cmd/ -run TestRootCmdExists -v`
Expected: PASS

---

### Task 3: 更新 config.go 错误提示

**Files:**
- Modify: `cmd/config.go:110`

- [ ] **Step 1: 更新 runConfigShow 错误提示**

```go
// cmd/config.go:110
// 旧:
fmt.Fprintf(os.Stderr, "读取配置失败: %v\n请先执行 ckjr config init\n", err)
// 新:
fmt.Fprintf(os.Stderr, "读取配置失败: %v\n请先执行 ckjr-cli config init\n", err)
```

- [ ] **Step 2: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 3: 提交代码变更**

```bash
git add cmd/root.go cmd/config.go cmd/root_test.go
git commit -m "refactor(cmd): rename CLI from ckjr to ckjr-cli

- Update cobra Use field to ckjr-cli
- Update Short description with 创客匠人 branding
- Update error messages to reference ckjr-cli
- Update test assertions"
```

---

### Task 4: 重命名入口目录

**Files:**
- Rename: `cmd/ckjr/` -> `cmd/ckjr-cli/`

- [ ] **Step 1: 使用 git mv 重命名目录**

```bash
git mv cmd/ckjr cmd/ckjr-cli
```

- [ ] **Step 2: 验证构建**

```bash
go build -o /tmp/ckjr-cli ./cmd/ckjr-cli
/tmp/ckjr-cli --help
/tmp/ckjr-cli --version
```

Expected: 帮助信息中显示 `ckjr-cli`，版本号显示 `dev`

- [ ] **Step 3: 运行全量测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/ckjr-cli/
git commit -m "refactor: rename cmd/ckjr to cmd/ckjr-cli

go install ./cmd/ckjr-cli now produces ckjr-cli binary"
```

---

### Task 5: 更新构建与发布配置

**Files:**
- Modify: `.github/workflows/release.yml:41,45-46`
- Modify: `install.sh:6,61`

- [ ] **Step 1: 更新 release.yml**

```yaml
# .github/workflows/release.yml
# Build binary step 中:
# 旧:
          BINARY_NAME=ckjr
          if [ "$GOOS" = "windows" ]; then
            BINARY_NAME=ckjr.exe
          fi
          mkdir -p dist/ckjr_${VERSION}_${GOOS}_${GOARCH}
          go build -ldflags="-s -w -X main.Version=${VERSION}" -o dist/ckjr_${VERSION}_${GOOS}_${GOARCH}/${BINARY_NAME} ./cmd/ckjr
# 新:
          BINARY_NAME=ckjr-cli
          if [ "$GOOS" = "windows" ]; then
            BINARY_NAME=ckjr-cli.exe
          fi
          mkdir -p dist/ckjr-cli_${VERSION}_${GOOS}_${GOARCH}
          go build -ldflags="-s -w -X main.Version=${VERSION}" -o dist/ckjr-cli_${VERSION}_${GOOS}_${GOARCH}/${BINARY_NAME} ./cmd/ckjr-cli
```

- [ ] **Step 2: 更新 install.sh**

```bash
# install.sh:6
# 旧:
BINARY_NAME="ckjr"
# 新:
BINARY_NAME="ckjr-cli"

# install.sh:61
# 旧:
go install "github.com/${REPO}/cmd/ckjr@latest"
# 新:
go install "github.com/${REPO}/cmd/ckjr-cli@latest"
```

- [ ] **Step 3: 提交**

```bash
git add .github/workflows/release.yml install.sh
git commit -m "build: update binary name to ckjr-cli in release and install scripts"
```

---

### Task 6: 更新技能文件

**Files:**
- Modify: `skills/ckjr-agent/SKILL.md`
- Modify: `skills/ckjr-agent/README.md`

- [ ] **Step 1: 更新 SKILL.md**

将所有命令调用中独立出现的 `ckjr ` 替换为 `ckjr-cli `，将 `go install github.com/childelins/ckjr-cli@latest` 改为 `go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest`。

具体替换（共约 20 处 `ckjr` -> `ckjr-cli`）:
- 第 19 行: `go install github.com/childelins/ckjr-cli@latest` -> `go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest`
- 第 25 行: `ckjr config init` -> `ckjr-cli config init`
- 第 33-34 行: `ckjr --help` / `ckjr agent --help` -> `ckjr-cli --help` / `ckjr-cli agent --help`
- 第 41,44,47 行: `ckjr agent list` -> `ckjr-cli agent list`
- 第 53 行: `ckjr agent get` -> `ckjr-cli agent get`
- 第 59,62 行: `ckjr agent create` -> `ckjr-cli agent create`
- 第 69,70 行: `ckjr agent update` -> `ckjr-cli agent update`
- 第 76 行: `ckjr agent delete` -> `ckjr-cli agent delete`
- 第 90 行: `ckjr config init` -> `ckjr-cli config init`

- [ ] **Step 2: 更新 README.md (技能)**

将所有命令调用中的 `ckjr ` 替换为 `ckjr-cli `。同时更新第 39 行描述: `Claude 会自动调用 ckjr 命令完成操作。` -> `Claude 会自动调用 ckjr-cli 命令完成操作。`

具体替换:
- 第 39 行: `ckjr 命令` -> `ckjr-cli 命令`
- 第 49-53 行: 表格中所有 `ckjr agent` -> `ckjr-cli agent`
- 第 58 行: `ckjr agent create --template` -> `ckjr-cli agent create --template`

- [ ] **Step 3: 提交**

```bash
git add skills/ckjr-agent/SKILL.md skills/ckjr-agent/README.md
git commit -m "docs(skill): update command references from ckjr to ckjr-cli"
```

---

### Task 7: 更新项目 README.md

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 更新 README.md**

将所有命令调用中独立出现的 `ckjr ` 替换为 `ckjr-cli `。更新以下关键位置:

1. 第 2 行描述: `基于 Go 的 CLI 工具，作为 Claude Code Skills 与公司 SaaS 平台之间的桥梁。` -> `创客匠人 CLI - 知识付费 SaaS 系统的命令行工具。`
2. 第 41 行: `go install github.com/childelins/ckjr-cli/cmd/ckjr@latest` -> `go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest`
3. 第 49 行: `go install ./cmd/ckjr` -> `go install ./cmd/ckjr-cli`
4. 第 69-156 行: 所有 `ckjr ` 命令调用改为 `ckjr-cli `（约 25 处）
5. 第 185 行: `ckjr example list` -> `ckjr-cli example list`
6. 第 241 行: `Claude 会自动调用 ckjr 命令完成操作。` -> `Claude 会自动调用 ckjr-cli 命令完成操作。`

- [ ] **Step 2: 提交**

```bash
git add README.md
git commit -m "docs: update README with ckjr-cli naming and 创客匠人 branding"
```

---

### Task 8: 最终验证

- [ ] **Step 1: 运行全量测试**

```bash
go test ./... -v
```

Expected: ALL PASS

- [ ] **Step 2: 构建并验证**

```bash
go build -o /tmp/ckjr-cli ./cmd/ckjr-cli
/tmp/ckjr-cli --help
```

Expected: 显示 `创客匠人 CLI - 知识付费 SaaS 系统的命令行工具`，命令名为 `ckjr-cli`

- [ ] **Step 3: 全局搜索遗漏**

搜索项目中所有非历史文档文件，确认没有遗漏的 `ckjr ` (后跟空格的独立命令引用)。排除 `docs/superpowers/`、`progress.md`、`task_plan.md`、`findings.md`。

```bash
grep -rn 'ckjr ' --include='*.go' --include='*.yaml' --include='*.yml' --include='*.sh' --include='*.md' . | grep -v 'ckjr-cli' | grep -v 'docs/superpowers/' | grep -v 'progress.md' | grep -v 'task_plan.md' | grep -v 'findings.md' | grep -v '.spf-'
```

Expected: 仅剩 `~/.ckjr/` 配置目录引用（这些保持不变）

- [ ] **Step 4: 清理临时构建产物**

```bash
rm -f /tmp/ckjr-cli
```
