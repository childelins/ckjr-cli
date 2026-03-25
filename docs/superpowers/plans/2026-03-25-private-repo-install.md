# 私有仓库安装分发方案实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 ckjr-cli 创建私有 GitHub 仓库的安装分发方案，支持 Go 开发者和无 Go 环境用户，同时支持 PAT 和 SSH 认证。

**Architecture:** GitHub Actions 自动构建多平台二进制发布到 Releases，install.sh 脚本自动检测环境选择最优安装方式，Skills 支持本地和远程两种安装模式。

**Tech Stack:** GitHub Actions (goreleaser/cross-compile), Bash (install.sh), Markdown (文档)

---

## 文件结构

```
ckjr-cli/
├── .github/
│   └── workflows/
│       └── release.yml          # 新建：GitHub Actions 发布流水线
├── install.sh                   # 新建：一键安装脚本
├── skills/
│   └── ckjr-agent/
│       ├── SKILL.md             # 已存在
│       └── README.md            # 新建：Skill 安装说明
└── README.md                    # 修改：添加私有仓库安装指南
```

---

### Task 1: 创建 GitHub Actions Release 流水线

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: 创建 .github/workflows 目录**

```bash
mkdir -p /home/childelins/code/ckjr-cli/.github/workflows
```

- [ ] **Step 2: 创建 release.yml 工作流文件**

创建 `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        goos: [linux, darwin, windows]
        goarch: [amd64, arm64]
        exclude:
          - goos: windows
            goarch: arm64
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build binary
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 0
        run: |
          BINARY_NAME=ckjr
          if [ "$GOOS" = "windows" ]; then
            BINARY_NAME=ckjr.exe
          fi
          go build -ldflags="-s -w" -o dist/${BINARY_NAME}_${GOOS}_${GOARCH}/${BINARY_NAME} .

      - name: Create archive
        run: |
          cd dist
          for dir in */; do
            dirname=${dir%/}
            if [ "${dirname##*_}" = "windows" ]; then
              zip -r ${dirname}.zip ${dirname}
            else
              tar -czvf ${dirname}.tar.gz ${dirname}
            fi
          done

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ matrix.goos }}-${{ matrix.goarch }}
          path: dist/*.tar.gz
          retention-days: 1

  release:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Download all artifacts
        uses: actions/download-artifact@v4
        with:
          path: dist
          pattern: binaries-*
          merge-multiple: true

      - name: List artifacts
        run: ls -la dist/

      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          files: dist/*
          generate_release_notes: true
```

- [ ] **Step 3: 验证 YAML 语法**

```bash
# 可选：如果有 yq 工具
cat /home/childelins/code/ckjr-cli/.github/workflows/release.yml
```

---

### Task 2: 创建 install.sh 一键安装脚本

**Files:**
- Create: `install.sh`

- [ ] **Step 1: 创建 install.sh**

创建 `install.sh`:

```bash
#!/bin/bash
set -e

# 配置变量 - Fork 用户可修改此处
REPO="childelins/ckjr-cli"
BINARY_NAME="ckjr"
INSTALL_DIR="$HOME/.local/bin"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# 检测操作系统
detect_os() {
    case "$(uname -s)" in
        Linux*) echo "linux" ;;
        Darwin*) echo "darwin" ;;
        CYGWIN*|MINGW*|MSYS*) echo "windows" ;;
        *) error "Unsupported OS: $(uname -s)" ;;
    esac
}

# 检测架构
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# 检测 Go 环境
has_go() {
    command -v go &> /dev/null
}

# Go install 方式安装
install_via_go() {
    info "Installing via go install..."

    # 配置私有仓库
    export GOPRIVATE="github.com/${REPO%%/*}/*"

    # 检测认证方式
    if [ -n "$GITHUB_TOKEN" ]; then
        info "Using GITHUB_TOKEN for authentication"
        git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
    elif [ -f "$HOME/.ssh/id_rsa" ] || [ -f "$HOME/.ssh/id_ed25519" ]; then
        info "Using SSH key for authentication"
        git config --global url."git@github.com:".insteadOf "https://github.com/"
    else
        warn "No authentication configured. Private repo access may fail."
    fi

    go install "github.com/${REPO}@latest"
    info "Installed via go install"
}

# 下载 Release 二进制
install_via_release() {
    local os=$(detect_os)
    local arch=$(detect_arch)

    info "Detected OS: $os, Arch: $arch"

    # 获取最新版本
    local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
    local download_url
    local archive_name

    if [ -n "$GITHUB_TOKEN" ]; then
        latest_url="${latest_url}?access_token=${GITHUB_TOKEN}"
    fi

    # 获取下载 URL
    local release_info
    if command -v curl &> /dev/null; then
        release_info=$(curl -fsSL ${GITHUB_TOKEN:+-H "Authorization: token $GITHUB_TOKEN"} "$latest_url")
    elif command -v wget &> /dev/null; then
        release_info=$(wget -qO- ${GITHUB_TOKEN:+--header="Authorization: token $GITHUB_TOKEN"} "$latest_url")
    else
        error "curl or wget required"
    fi

    # 确定归档文件名
    if [ "$os" = "windows" ]; then
        archive_name="${BINARY_NAME}_${os}_${arch}.zip"
    else
        archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
    fi

    download_url=$(echo "$release_info" | grep "browser_download_url" | grep "$archive_name" | head -n1 | cut -d'"' -f4)

    if [ -z "$download_url" ]; then
        error "Binary not found for ${os}/${arch}. Please check releases: https://github.com/${REPO}/releases"
    fi

    # 创建临时目录
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    info "Downloading $archive_name..."

    # 下载
    if command -v curl &> /dev/null; then
        curl -fsSL ${GITHUB_TOKEN:+-H "Authorization: token $GITHUB_TOKEN"} -o "$tmp_dir/$archive_name" "$download_url"
    else
        wget -q ${GITHUB_TOKEN:+--header="Authorization: token $GITHUB_TOKEN"} -O "$tmp_dir/$archive_name" "$download_url"
    fi

    # 解压
    info "Extracting..."
    cd "$tmp_dir"
    if [ "$os" = "windows" ]; then
        unzip -o "$archive_name"
    else
        tar -xzf "$archive_name"
    fi

    # 安装
    mkdir -p "$INSTALL_DIR"
    mv "$BINARY_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    # 配置 PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        info "Adding $INSTALL_DIR to PATH..."
        local shell_rc=""
        if [ -n "$ZSH_VERSION" ]; then
            shell_rc="$HOME/.zshrc"
        elif [ -n "$BASH_VERSION" ]; then
            shell_rc="$HOME/.bashrc"
        fi

        if [ -n "$shell_rc" ]; then
            echo "" >> "$shell_rc"
            echo "# Added by ckjr-cli installer" >> "$shell_rc"
            echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$shell_rc"
            info "Added to $shell_rc. Run 'source $shell_rc' or restart your shell."
        fi
    fi

    info "Installed to $INSTALL_DIR/$BINARY_NAME"
}

# 主函数
main() {
    info "Installing $BINARY_NAME from $REPO"

    # 优先使用 go install（如果有 Go 环境）
    if has_go; then
        info "Go environment detected"
        read -p "Use 'go install' method? (y/n, default: y): " use_go
        if [ -z "$use_go" ] || [ "$use_go" = "y" ]; then
            install_via_go
            exit 0
        fi
    fi

    # 下载预编译二进制
    install_via_release
}

main "$@"
```

- [ ] **Step 2: 添加执行权限**

```bash
chmod +x /home/childelins/code/ckjr-cli/install.sh
```

---

### Task 3: 创建 Skills 安装说明

**Files:**
- Create: `skills/ckjr-agent/README.md`

- [ ] **Step 1: 创建 Skill README**

创建 `skills/ckjr-agent/README.md`:

```markdown
# ckjr-agent Skill

Claude Code Skill，用于通过自然语言操作公司 SaaS 平台的 AI 智能体。

## 安装前提

首先安装 ckjr CLI 二进制文件，详见 [主项目 README](../../README.md)。

## 安装方式

### 方式 1: 本地文件安装（推荐）

适用于已克隆仓库的用户：

```bash
# 克隆仓库
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli

# 安装 Skill
claude skills add ./skills/ckjr-agent
```

### 方式 2: 远程 URL 安装

适用于私有仓库，需要 GitHub Personal Access Token：

```bash
# 设置 PAT 环境变量
export GITHUB_TOKEN=ghp_xxx

# 安装 Skill
claude skills add https://github.com/childelins/ckjr-cli --skill ckjr-agent
```

## 使用

安装后，在 Claude Code 对话中直接描述需求：

```
帮我创建一个销售助手智能体
```

```
查看所有智能体列表
```

```
删除 ID 为 xxx 的智能体
```

Claude 会自动调用 ckjr 命令完成操作。

## Fork 自定义

如果 Fork 了此仓库，需要修改 `SKILL.md` 中的命令说明以匹配你的使用场景。

## 可用命令

| 命令 | 说明 |
|------|------|
| `ckjr agent list` | 获取智能体列表 |
| `ckjr agent get '<json>'` | 获取智能体详情 |
| `ckjr agent create '<json>'` | 创建智能体 |
| `ckjr agent update '<json>'` | 更新智能体 |
| `ckjr agent delete '<json>'` | 删除智能体 |

使用 `--template` 查看参数模板：

```bash
ckjr agent create --template
```
```

---

### Task 4: 更新 README.md 添加私有仓库安装指南

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 更新安装章节**

将 `README.md` 的安装章节（第 7-11 行）替换为：

```markdown
## 安装

### 方式 1: 一键安装脚本（推荐）

适用于无 Go 环境的用户：

```bash
# 设置 GitHub Token（私有仓库需要）
export GITHUB_TOKEN=ghp_xxx

# 执行安装脚本
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/main/install.sh | bash
```

安装脚本会自动：
- 检测操作系统和架构
- 下载对应的预编译二进制
- 配置 PATH 环境变量

### 方式 2: go install（Go 开发者）

适用于有 Go 环境的开发者：

```bash
# 配置私有仓库访问
export GOPRIVATE=github.com/childelins/*

# 使用 SSH（推荐）
git config --global url."git@github.com:".insteadOf "https://github.com/"

# 或使用 PAT
# git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

# 安装
go install github.com/childelins/ckjr-cli@latest
```

### 方式 3: 从源码构建

```bash
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli
go build -o ckjr .
```

### Fork 自定义

如果 Fork 了此仓库，安装时需要：

1. 修改 `install.sh` 中的 `REPO` 变量为你的仓库地址
2. 推送 tag 触发 Release：
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
3. 使用你的仓库地址执行安装脚本
```

- [ ] **Step 2: 更新 Claude Code Skill 安装章节**

将 `README.md` 的 Claude Code Skill 安装章节（第 163-188 行）替换为：

```markdown
## Claude Code Skill 安装

如果你使用 Claude Code，可以安装 ckjr-agent skill 来通过自然语言操作智能体。

### 安装 Skill

**方式 1: 本地文件（推荐）**

```bash
git clone git@github.com:childelins/ckjr-cli.git
claude skills add ./ckjr-cli/skills/ckjr-agent
```

**方式 2: 远程 URL（需 PAT）**

```bash
export GITHUB_TOKEN=ghp_xxx
claude skills add https://github.com/childelins/ckjr-cli --skill ckjr-agent
```

详细说明见 [skills/ckjr-agent/README.md](skills/ckjr-agent/README.md)。

### 使用

在 Claude Code 对话中直接描述需求：

- "帮我创建一个销售助手智能体"
- "查看所有智能体列表"
- "删除 ID 为 xxx 的智能体"

Claude 会自动调用 ckjr 命令完成操作。
```

---

### Task 5: 提交变更

**Files:**
- All modified files

- [ ] **Step 1: 检查变更**

```bash
cd /home/childelins/code/ckjr-cli && git status
```

- [ ] **Step 2: 提交变更**

```bash
git add .github/workflows/release.yml install.sh skills/ckjr-agent/README.md README.md
git commit -m "$(cat <<'EOF'
feat: add private repo install and distribution support

- Add GitHub Actions release workflow for multi-platform builds
- Add install.sh for one-click installation (auto-detect environment)
- Add Skills README with local/remote install methods
- Update README with private repo installation guide

Supports:
- Go developers: go install with GOPRIVATE config
- Non-Go users: pre-built binaries from GitHub Releases
- PAT and SSH authentication
- Fork-friendly customization

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## 测试清单

- [ ] 推送 tag 测试 GitHub Actions（`git tag v0.1.0 && git push origin v0.1.0`）
- [ ] 在无 Go 环境机器测试 `install.sh`
- [ ] 测试 PAT 认证方式
- [ ] 测试 SSH 认证方式
- [ ] 测试 Skills 本地安装
- [ ] 测试 Skills 远程安装（需 PAT）
