# CLI 重命名设计文档 (ckjr -> ckjr-cli)

> Created: 2026-03-26
> Status: Draft

## 概述

将 CLI 二进制名称从 `ckjr` 改为 `ckjr-cli`，同时将公司名称"创客匠人"体现在相关描述中。涉及入口文件、cobra 命令定义、安装脚本、CI/CD、技能文件、文档等全面更新。

配置目录 `~/.ckjr/` 和 Go module 路径 `github.com/childelins/ckjr-cli` 保持不变（配置目录短名更方便，module 路径已经是 ckjr-cli）。

## 变更范围

### 分类 A: 代码文件（影响编译和运行）

| 文件 | 变更内容 |
|------|----------|
| `cmd/ckjr/main.go` | 目录重命名为 `cmd/ckjr-cli/main.go`（入口路径变化） |
| `cmd/root.go` | `Use: "ckjr"` -> `Use: "ckjr-cli"`；`Short` 描述加入"创客匠人" |
| `cmd/root.go` | 错误提示 `请先执行 ckjr config init` -> `请先执行 ckjr-cli config init` |
| `cmd/config.go` | 错误提示 `请先执行 ckjr config init` -> `请先执行 ckjr-cli config init` |
| `cmd/root_test.go` | 断言 `rootCmd.Use` 改为 `"ckjr-cli"` |
| `main.go` | 根目录入口文件不变（import 路径不变） |

### 分类 B: 构建与发布

| 文件 | 变更内容 |
|------|----------|
| `.github/workflows/release.yml` | `BINARY_NAME=ckjr` -> `BINARY_NAME=ckjr-cli`；构建路径 `./cmd/ckjr` -> `./cmd/ckjr-cli`；dist 目录名更新 |
| `install.sh` | `BINARY_NAME="ckjr"` -> `BINARY_NAME="ckjr-cli"`；`go install` 路径更新；注释更新 |

### 分类 C: 技能文件

| 文件 | 变更内容 |
|------|----------|
| `skills/ckjr-agent/SKILL.md` | 所有 `ckjr ` 命令调用改为 `ckjr-cli `；安装命令路径更新 |
| `skills/ckjr-agent/README.md` | 所有 `ckjr ` 命令调用改为 `ckjr-cli `；描述更新 |

### 分类 D: 项目文档

| 文件 | 变更内容 |
|------|----------|
| `README.md` | 所有 `ckjr ` 命令调用改为 `ckjr-cli `；安装路径更新；公司名称加入描述 |

### 不变项

| 项目 | 原因 |
|------|------|
| Go module 路径 `github.com/childelins/ckjr-cli` | 已经是 ckjr-cli，无需变更 |
| 配置目录 `~/.ckjr/` | 短名更方便用户使用，且已有用户配置不应破坏 |
| 日志目录 `~/.ckjr/logs/` | 同上 |
| 仓库名 `ckjr-cli` | 已经正确 |
| `docs/` 下的历史文档 | 历史记录，不做回溯修改 |
| `progress.md`、`task_plan.md`、`findings.md` | 历史记录文件 |

## 关键决策

1. **`cmd/ckjr/` 目录重命名为 `cmd/ckjr-cli/`**: Go 的 `go install ./cmd/ckjr` 会产生名为 `ckjr` 的二进制。改为 `cmd/ckjr-cli` 后 `go install ./cmd/ckjr-cli` 自动产生 `ckjr-cli` 二进制。
2. **配置目录不改**: `~/.ckjr/` 作为配置目录已被使用，变更会破坏现有用户。且目录名不必与二进制名完全一致。
3. **公司名"创客匠人"**: 体现在 cobra root command 的 Short 描述中。
4. **历史文档不改**: `docs/superpowers/` 下的 plans/specs 是历史记录，不做回溯修改。

## 实现步骤

### Step 1: 重命名入口目录

```bash
git mv cmd/ckjr cmd/ckjr-cli
```

### Step 2: 更新代码文件

- `cmd/root.go`: `Use: "ckjr-cli"`, `Short: "创客匠人 CLI - 知识付费 SaaS 系统的命令行工具"`
- `cmd/root.go`: 错误提示中的 `ckjr` -> `ckjr-cli`
- `cmd/config.go`: 错误提示中的 `ckjr` -> `ckjr-cli`
- `cmd/root_test.go`: 断言更新

### Step 3: 更新构建配置

- `.github/workflows/release.yml`: BINARY_NAME 和构建路径
- `install.sh`: BINARY_NAME 和 go install 路径

### Step 4: 更新技能文件

- `skills/ckjr-agent/SKILL.md`: 所有命令调用
- `skills/ckjr-agent/README.md`: 所有命令调用

### Step 5: 更新 README

- `README.md`: 所有命令调用和安装路径

### Step 6: 验证

```bash
go test ./... -v
go build -o ckjr-cli ./cmd/ckjr-cli
./ckjr-cli --help
./ckjr-cli --version
```

## 错误处理

- 如果 `git mv` 失败（文件被修改未提交），先 stash 或 commit
- 确保 `go install` 路径更新后能正确编译

## 测试策略

1. **单元测试**: `go test ./...` 确保所有现有测试通过（重点是 `root_test.go` 中 `Use` 字段断言）
2. **构建测试**: `go build -o ckjr-cli ./cmd/ckjr-cli` 确保能编译
3. **功能测试**: `./ckjr-cli --help` 确认命令名显示正确
4. **安装测试**: `go install ./cmd/ckjr-cli` 确认安装到 `$GOBIN/ckjr-cli`

## 实现注意事项

- `cmd/ckjr-cli/main.go` 的 import 路径保持 `github.com/childelins/ckjr-cli/cmd`，不需要变更
- 根目录 `main.go` 可以保留也可以删除（它是早期入口，现在正式入口在 `cmd/ckjr-cli/main.go`）
- 全局替换时注意不要误改 `ckjr-cli`（仓库名/module名）为 `ckjr-cli-cli`
- 替换规则：命令调用场景的 `ckjr ` (后跟空格) 改为 `ckjr-cli `；`ckjr\b` 作为独立单词时才替换
