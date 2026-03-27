# ckjr-cli Makefile

BINARY_NAME := ckjr-cli
BUILD_DIR := bin
CMD_PATH := ./cmd/ckjr-cli
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Environment=production

# GitHub 配置（项目主仓库在 GitLab，Release 发布在 GitHub）
GITHUB_REPO := childelins/ckjr-cli
GITHUB_REMOTE := github

# 目标平台列表
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build build-local release clean version check-gh check-clean check-github-remote

# 打印版本号
version:
	@echo $(VERSION)

# 清理构建产物
clean:
	rm -rf $(BUILD_DIR)
