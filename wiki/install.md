# 安装指南

## 前置条件

| 项目 | 要求 |
|------|------|
| 操作系统 | Linux、macOS、Windows (WSL) |
| 架构 | amd64、arm64 |
| 认证 | GitHub Token 或 SSH Key（私有仓库） |

安装位置：
- 一键脚本：`~/.local/bin/ckjr-cli`
- go install：`$GOPATH/bin/ckjr-cli`

## 方式一：一键安装脚本（推荐）

适用于非 Go 开发者。脚本自动检测操作系统和架构，下载预编译二进制文件。

```bash
# 设置 GitHub Token（私有仓库访问）
export GITHUB_TOKEN=ghp_xxx

# 执行安装
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/main/install.sh | bash
```

脚本执行流程：检测 OS/架构 -> 获取最新 Release -> 下载二进制 -> 安装到 `~/.local/bin` -> 配置 PATH

## 方式二：go install（Go 开发者）

适用于已安装 Go 环境的开发者。

```bash
# 配置私有仓库访问
export GOPRIVATE=github.com/childelins/*
git config --global url."git@github.com:".insteadOf "https://github.com/"

# 安装
go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest
```

## 方式三：从源码构建（贡献者）

适用于需要修改代码的开发者。

```bash
# 克隆仓库
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli

# 生产构建（默认）- INFO 级别日志，省略 request/response body
go install ./cmd/ckjr-cli

# 开发构建 - DEBUG 级别日志，记录完整 request/response body
go install -ldflags="-X main.Environment=development" ./cmd/ckjr-cli
```

> 两种构建的区别仅在于日志行为：
>
> | 环境 | 日志级别 | request/response body |
> |------|---------|---------------------|
> | production（默认） | INFO | 不记录 |
> | development | DEBUG | 记录完整内容 |
>
> `--verbose` flag 在两种环境下行为一致，均输出日志到 stderr。

## 验证安装

```bash
ckjr-cli --version
```

## 常见问题

| 问题 | 解决方案 |
|------|---------|
| `command not found` | 检查 PATH 是否包含安装目录（`~/.local/bin` 或 `$GOPATH/bin`） |
| `go install` 失败 | 检查 `GOPRIVATE` 环境变量和 SSH Key 配置 |
| 私有仓库 403 | 设置 `GITHUB_TOKEN` 环境变量 |
| Release 下载失败 | 检查网络连接，或切换到 go install 方式 |

## Fork 自定义

如果你 Fork 了仓库，修改 `install.sh` 顶部的 `REPO` 变量即可：

```bash
REPO="your-username/ckjr-cli"
```

发布流程：推送 tag（如 `v1.0.0`）-> GitHub Actions 自动构建 -> 创建 Release 上传二进制。

---

[文档目录](HOME.md) | 下一步：[快速开始](quickstart.md)
