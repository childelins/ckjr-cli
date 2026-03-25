# 私有仓库安装分发方案设计文档

> Created: 2026-03-25
> Status: Draft

## 概述

为 ckjr-cli 设计私有 GitHub 仓库的安装分发方案，支持混合用户群体（Go 开发者和无 Go 环境的技术用户），同时支持 PAT 和 SSH Key 两种认证方式。

## 目标用户

| 用户类型 | 特点 | 推荐安装方式 |
|---------|------|-------------|
| Go 开发者 | 有 Go 环境，熟悉 go install | GOPRIVATE + go install |
| 技术用户 | 无 Go 环境，有命令行经验 | 下载预编译二进制 |
| 复刻用户 | Fork 后自定义使用 | 源码编译或自定义 Release |

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                    私有 GitHub 仓库                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Source    │  │   Skills    │  │  GitHub Releases    │  │
│  │   Code      │  │   Files     │  │  (预编译二进制)      │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
└─────────┼────────────────┼───────────────────┼─────────────┘
          │                │                   │
          ▼                ▼                   ▼
   ┌──────────────┐ ┌──────────────┐  ┌──────────────────┐
   │  go install  │ │ git clone +  │  │  install.sh      │
   │  (Go 开发者) │ │ 本地安装     │  │  (自动检测环境)   │
   └──────────────┘ └──────────────┘  └──────────────────┘
```

## 组件

### 1. GitHub Actions 发布流水线

创建 `.github/workflows/release.yml`，在 tag 推送时自动构建多平台二进制并发布到 GitHub Releases。

**目标平台**：
- Linux: amd64, arm64
- macOS: amd64 (intel), arm64 (apple silicon)
- Windows: amd64

**触发条件**：推送 `v*` 格式的 tag

### 2. 安装脚本 (install.sh)

提供一键安装脚本，自动处理：
- 检测本地 Go 环境
- 选择最优安装方式
- 处理 PAT 认证
- 配置 PATH 环境变量

### 3. Skills 安装方案

两种方式：
1. **本地文件方式**：克隆仓库后指向本地路径
2. **远程 URL 方式**：通过 GitHub API + PAT 访问

### 4. 用户文档

更新 README.md，添加私有仓库安装指南。

## 数据流

### 安装流程（无 Go 环境）

```
用户执行 install.sh
    │
    ├─► 检测 Go 环境 ─────► 有 Go ──► go install 方式
    │
    └─► 无 Go
         │
         ├─► 检测操作系统和架构
         │
         ├─► 检测认证方式 (PAT/SSH)
         │
         ├─► 下载对应 Release 二进制
         │
         └─► 安装到 ~/.local/bin 或 /usr/local/bin
```

### Skills 安装流程

```
用户安装 Skills
    │
    ├─► 方式1: 本地文件
    │    └─► git clone 仓库
    │    └─► claude skills add ./ckjr-cli/skills/ckjr-agent
    │
    └─► 方式2: GitHub API (需 PAT)
         └─► 设置 GITHUB_TOKEN 环境变量
         └─► claude skills add https://github.com/xxx/ckjr-cli --skill ckjr-agent
```

## 错误处理

| 错误场景 | 处理策略 |
|---------|---------|
| 无访问权限 | 提示配置 PAT 或 SSH Key |
| 下载失败 | 重试机制 + 错误信息提示 |
| 二进制已存在 | 提示覆盖或跳过 |
| PATH 未配置 | 自动添加到 shell 配置文件 |
| Go 版本过低 | 提示升级 Go 版本 |

## 测试策略

### 单元测试

- 安装脚本的各检测函数
- 版本解析逻辑

### 集成测试

- 在干净环境中测试完整安装流程
- 测试 PAT 和 SSH 两种认证方式
- 测试不同操作系统和架构

### 手动测试清单

- [ ] 有 Go 环境 + PAT 认证
- [ ] 有 Go 环境 + SSH 认证
- [ ] 无 Go 环境 + PAT 认证
- [ ] 无 Go 环境 + SSH 认证
- [ ] Skills 本地安装
- [ ] Skills 远程安装
- [ ] Fork 后自定义使用

## 实现注意事项

### 文件结构

```
ckjr-cli/
├── .github/
│   └── workflows/
│       └── release.yml      # 发布流水线
├── install.sh               # 一键安装脚本
├── skills/
│   └── ckjr-agent/
│       └── SKILL.md
└── README.md                # 更新安装文档
```

### 版本管理

- 使用 Git tag 管理版本（如 v1.0.0）
- 二进制文件名包含版本号和平台信息
- 提供 `latest` 指向最新版本

### 安全考虑

- PAT 不记录到日志
- 验证下载的二进制完整性（可选 checksum）
- 最小权限原则（PAT 只需 repo 权限）

### Fork 友好设计

- 安装脚本使用变量配置仓库地址
- GitHub Actions 支持在 Fork 中复用
- 文档说明 Fork 后的自定义步骤

## 使用示例

### Go 开发者安装

```bash
# 配置私有仓库访问
export GOPRIVATE=github.com/your-org/*
git config --global url."git@github.com:".insteadOf "https://github.com/"

# 安装
go install github.com/your-org/ckjr-cli@latest
```

### 无 Go 环境安装

```bash
# PAT 方式
export GITHUB_TOKEN=ghp_xxx
curl -fsSL https://raw.githubusercontent.com/your-org/ckjr-cli/main/install.sh | bash

# 或 SSH 方式（需先配置 SSH Key）
curl -fsSL https://raw.githubusercontent.com/your-org/ckjr-cli/main/install.sh | bash
```

### Skills 安装

```bash
# 方式1: 本地文件
git clone git@github.com:your-org/ckjr-cli.git
claude skills add ./ckjr-cli/skills/ckjr-agent

# 方式2: 远程 URL（需 PAT）
export GITHUB_TOKEN=ghp_xxx
claude skills add https://github.com/your-org/ckjr-cli --skill ckjr-agent
```

### Fork 用户自定义

```bash
# 1. Fork 仓库
# 2. 修改 install.sh 中的 REPO 变量
REPO="your-username/ckjr-cli"

# 3. 推送 tag 触发 Release
git tag v1.0.0
git push origin v1.0.0
```

## 待实现清单

1. [ ] 创建 `.github/workflows/release.yml`
2. [ ] 创建 `install.sh` 安装脚本
3. [ ] 更新 `README.md` 添加私有仓库安装指南
4. [ ] 创建 `skills/ckjr-agent/README.md` 说明 Skill 安装方式
5. [ ] 测试完整流程
