# 安装指南

## 前置条件

| 项目 | 要求 |
|------|------|
| 操作系统 | Linux、macOS、Windows (WSL) |
| 架构 | amd64、arm64 |

安装位置：
- 一键脚本：`~/.local/bin/ckjr-cli`

## 方式一：一键安装脚本（推荐）

适用于非 Go 开发者。脚本自动检测操作系统和架构，下载预编译二进制文件，并自动检测已安装的 AI 编码平台（Claude Code、Gemini、OpenClaw、Codex）安装对应 skills。

```bash
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/master/install.sh | bash
```

脚本执行流程：检测 OS/架构 -> 获取最新 Release -> 下载二进制 -> 安装到 `~/.local/bin` -> 配置 PATH -> 检测已安装的 AI 平台并安装 skills

## 方式二：从源码构建（贡献者）

适用于需要修改代码的开发者。

```bash
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli
```

### 编译命令

| 命令 | 说明 | 环境 |
|------|------|------|
| `make build-local` | 当前平台编译（先跑测试） | production |
| `make build` | 多平台交叉编译（linux/darwin/windows, amd64/arm64） | production |
| `go install ./cmd/ckjr-cli` | 安装到 GOPATH/bin | development |
| `go build ./cmd/ckjr-cli` | 当前平台编译（无 ldflags） | development |

### 发布命令

```bash
make release VERSION=vX.Y.Z
```

完整流程：跑测试 -> 打 tag -> 推送到 GitHub -> 多平台编译 -> 创建 GitHub Release（附预编译产物）。

前提：已配置 `github` remote 且 `gh auth login` 已完成。

### 环境模式

两种环境仅影响日志行为：

| 环境 | 文件日志级别 | request/response body |
|------|------------|---------------------|
| development（源码默认） | DEBUG | 记录完整内容 |
| production（Makefile 构建） | ERROR | 不记录 |

`make build-local` 和 `make build` 通过 ldflags 注入 `-X main.Environment=production`，`go install` / `go build` 则使用源码默认值 `development`。

## 验证安装

```bash
ckjr-cli --version
```

## 常见问题

| 问题 | 解决方案 |
|------|---------|
| `command not found` | 检查 PATH 是否包含 `~/.local/bin` |
| Release 下载失败 | 检查网络连接，或切换到源码构建方式 |

## Fork 自定义

如果你 Fork 了仓库，修改 `install.sh` 顶部的 `REPO` 变量即可：

```bash
REPO="your-username/ckjr-cli"
```

发布新版本：`make release VERSION=vX.Y.Z`

---

[文档目录](HOME.md) | 下一步：[快速开始](quickstart.md)
