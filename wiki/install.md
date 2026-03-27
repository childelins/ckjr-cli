# 安装指南

## 前置条件

| 项目 | 要求 |
|------|------|
| 操作系统 | Linux、macOS、Windows (WSL) |
| 架构 | amd64、arm64 |

安装位置：
- 一键脚本：`~/.local/bin/ckjr-cli`

## 方式一：一键安装脚本（推荐）

适用于非 Go 开发者。脚本自动检测操作系统和架构，下载预编译二进制文件。

```bash
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/master/install.sh | bash
```

脚本执行流程：检测 OS/架构 -> 获取最新 Release -> 下载二进制 -> 安装到 `~/.local/bin` -> 配置 PATH

## 方式二：从源码构建（贡献者）

适用于需要修改代码的开发者。

```bash
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli
make build-local
```

> 两种构建的区别仅在于日志行为：
>
> | 环境 | 日志级别 | request/response body |
> |------|---------|---------------------|
> | production（默认） | INFO | 不记录 |
> | development | DEBUG | 记录完整内容 |
>
> 开发构建：`make build-local` 默认使用 production 模式，如需 development 模式：
> ```bash
> go build -ldflags="-X main.Environment=development" -o bin/ckjr-cli ./cmd/ckjr-cli
> ```

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
