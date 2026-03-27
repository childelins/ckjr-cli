#!/bin/bash
set -e

# 配置变量 - Fork 用户可修改此处
REPO="childelins/ckjr-cli"
BINARY_NAME="ckjr-cli"
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

    # 确定归档文件名模式（带版本号）
    local archive_pattern
    if [ "$os" = "windows" ]; then
        archive_pattern="${BINARY_NAME}_.*_${os}_${arch}\.zip"
    else
        archive_pattern="${BINARY_NAME}_.*_${os}_${arch}\.tar\.gz"
    fi

    download_url=$(echo "$release_info" | grep "browser_download_url" | grep -E "$archive_pattern" | head -n1 | cut -d'"' -f4)

    # 从 URL 提取实际文件名
    archive_name=$(basename "$download_url")

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

    # 查找二进制文件（解压后的目录名包含版本号）
    local binary_path=$(find . -name "$BINARY_NAME*" -type f | head -n1)
    if [ -z "$binary_path" ]; then
        error "Binary not found in archive"
    fi

    # 安装（Windows 保留 .exe 后缀）
    mkdir -p "$INSTALL_DIR"
    local dest_name="$BINARY_NAME"
    if [ "$os" = "windows" ]; then
        dest_name="$BINARY_NAME.exe"
    fi
    mv "$binary_path" "$INSTALL_DIR/$dest_name"
    chmod +x "$INSTALL_DIR/$dest_name"

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

    info "Installed to $INSTALL_DIR/$dest_name"
}

# 主函数
main() {
    info "Installing $BINARY_NAME from $REPO"
    install_via_release
}

main "$@"
