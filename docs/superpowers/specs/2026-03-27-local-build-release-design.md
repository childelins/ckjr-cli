# 本地多平台构建与 GitHub Release 发布设计文档

> Created: 2026-03-27
> Status: Draft
> Revised: 2026-03-27 (补充双仓库 remote 配置)

## 概述

为 ckjr-cli 提供本地多平台交叉编译和一键发布能力。通过 Makefile 管理构建流程，使用 `make release VERSION=v0.1.0` 一条命令完成：打 tag、多平台编译、创建 GitHub Release、上传产物。产物文件名与现有 install.sh 的下载格式保持一致。

## 背景

项目已有 `.github/workflows/release.yml` 实现 CI 自动构建发布，但有时需要在本地快速构建和发布（如 CI 不可用、调试构建问题、或快速迭代）。本方案补充本地构建能力，与 CI 方案共存。

### 双仓库架构

项目使用双仓库模式：

| 仓库 | 用途 | 地址 |
|------|------|------|
| GitLab（内部） | 主开发仓库，日常 push/pull | `git@src2103.myckjr.com:ckjr001/ckjr-cli.git` |
| GitHub（公开） | Release 发布、install.sh 下载 | `https://github.com/childelins/ckjr-cli` |

当前 git remote 配置中 `origin` 指向 GitLab。发布流程需要将 tag 推送到 GitHub，并在 GitHub 上创建 Release。因此 Makefile 需要：

1. 配置或验证 `github` remote 指向 GitHub 仓库
2. 将 tag 推送到 `github` remote（而非 `origin`）
3. `gh release create` 指定 `--repo childelins/ckjr-cli`

## 架构

```
Makefile
  │
  ├── make build          # 多平台交叉编译 → bin/
  ├── make build-local    # 仅当前平台编译 → bin/
  ├── make release        # 全流程：tag + build + gh release create + upload
  │                       #   tag 推送到 github remote，Release 创建在 GitHub
  └── make clean          # 清理 bin/

bin/
  ├── ckjr-cli_v0.1.0_linux_amd64.tar.gz
  ├── ckjr-cli_v0.1.0_linux_arm64.tar.gz
  ├── ckjr-cli_v0.1.0_darwin_amd64.tar.gz
  ├── ckjr-cli_v0.1.0_darwin_arm64.tar.gz
  └── ckjr-cli_v0.1.0_windows_amd64.zip
```

## 组件

### 1. Makefile

项目根目录的 Makefile，包含以下目标：

| 目标 | 用途 | 依赖 |
|------|------|------|
| `build` | 编译全部 5 个目标平台，输出压缩包到 bin/ | Go |
| `build-local` | 仅编译当前平台，输出二进制到 bin/ | Go |
| `release` | 全自动发布流程（tag 推 GitHub，Release 创建在 GitHub） | Go, gh CLI, git |
| `clean` | 删除 bin/ 目录 | 无 |
| `version` | 打印当前版本号 | git |

关键变量：

| 变量 | 默认值 | 用途 |
|------|--------|------|
| `GITHUB_REPO` | `childelins/ckjr-cli` | GitHub 仓库，用于 `gh release create --repo` |
| `GITHUB_REMOTE` | `github` | git remote 名称，用于推送 tag 到 GitHub |

### 2. 版本号管理

混合模式：

- **默认**：从最新 git tag 获取（`git describe --tags --abbrev=0`）
- **手动覆盖**：`make build VERSION=v0.1.0`
- 版本号通过 `-ldflags "-X main.Version=$(VERSION)"` 注入二进制

```makefile
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
```

### 3. 目标平台

与现有 CI 流水线和 install.sh 保持一致：

| GOOS | GOARCH | 产物格式 |
|------|--------|---------|
| linux | amd64 | .tar.gz |
| linux | arm64 | .tar.gz |
| darwin | amd64 | .tar.gz |
| darwin | arm64 | .tar.gz |
| windows | amd64 | .zip |

### 4. 产物命名

格式：`ckjr-cli_{version}_{os}_{arch}.tar.gz` (或 `.zip`)

示例：
- `ckjr-cli_v0.1.0_linux_amd64.tar.gz`
- `ckjr-cli_v0.1.0_windows_amd64.zip`

此格式与 install.sh 中的 `archive_pattern` 匹配，确保兼容。

## 数据流

### make release 全流程

```
make release VERSION=v0.1.0
    │
    ├─► 1. 检查前置条件
    │     ├─ gh CLI 已登录
    │     ├─ git 工作区干净（无未提交修改）
    │     ├─ tag 尚未存在
    │     └─ github remote 已配置（指向 GitHub 仓库）
    │
    ├─► 2. 创建并推送 git tag（到 GitHub）
    │     ├─ git tag v0.1.0
    │     └─ git push github v0.1.0   ← 推送到 github remote，非 origin
    │
    ├─► 3. 多平台编译
    │     ├─ CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ...
    │     ├─ CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ...
    │     ├─ CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ...
    │     ├─ CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ...
    │     └─ CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ...
    │
    ├─► 4. 打包压缩
    │     ├─ tar.gz (linux/darwin)
    │     └─ zip (windows)
    │
    └─► 5. 创建 Release 并上传（指定 GitHub 仓库）
          └─ gh release create v0.1.0 bin/*.tar.gz bin/*.zip \
               --repo childelins/ckjr-cli --generate-notes
```

注意：tag 推送到 GitHub 后，如果 GitHub 上配置了 `.github/workflows/release.yml`，会触发 CI 流水线。但 `softprops/action-gh-release` 不会覆盖已存在的 Release，所以本地发布和 CI 不会冲突。

### make build 单独构建

```
make build VERSION=v0.1.0
    │
    ├─► 清理 bin/ 目录
    ├─► 逐平台编译到临时目录
    ├─► 打包为 tar.gz / zip
    └─► 输出到 bin/
```

## Makefile 详细设计

```makefile
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

# 仅当前平台编译
build-local:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) ($(VERSION))"

# 多平台交叉编译
build: clean
	@echo "Building $(BINARY_NAME) $(VERSION) for all platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output_name=$(BINARY_NAME); \
		if [ "$$GOOS" = "windows" ]; then output_name=$(BINARY_NAME).exe; fi; \
		echo "  → $$GOOS/$$GOARCH"; \
		tmp_dir=$(BUILD_DIR)/$(BINARY_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}; \
		mkdir -p $$tmp_dir; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH \
			go build -ldflags="$(LDFLAGS)" -o $$tmp_dir/$$output_name $(CMD_PATH); \
		if [ "$$GOOS" = "windows" ]; then \
			(cd $(BUILD_DIR) && zip -r $(BINARY_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}.zip \
				$(BINARY_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}/); \
		else \
			(cd $(BUILD_DIR) && tar -czf $(BINARY_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}.tar.gz \
				$(BINARY_NAME)_$(VERSION)_$${GOOS}_$${GOARCH}/); \
		fi; \
		rm -rf $$tmp_dir; \
	done
	@echo "Build complete. Artifacts in $(BUILD_DIR)/"

# 前置检查：gh CLI 已登录
check-gh:
	@gh auth status >/dev/null 2>&1 || (echo "Error: gh CLI not authenticated. Run 'gh auth login'" && exit 1)

# 前置检查：工作区干净
check-clean:
	@test -z "$$(git status --porcelain)" || (echo "Error: working directory not clean. Commit or stash changes first." && exit 1)

# 前置检查：github remote 已配置
check-github-remote:
	@git remote get-url $(GITHUB_REMOTE) >/dev/null 2>&1 || \
		(echo "Error: git remote '$(GITHUB_REMOTE)' not found." && \
		 echo "Run: git remote add $(GITHUB_REMOTE) git@github.com:$(GITHUB_REPO).git" && \
		 exit 1)

# 全自动发布
release: check-gh check-clean check-github-remote
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "Error: VERSION is required. Usage: make release VERSION=v0.1.0"; \
		exit 1; \
	fi
	@if git rev-parse "$(VERSION)" >/dev/null 2>&1; then \
		echo "Error: tag $(VERSION) already exists"; \
		exit 1; \
	fi
	@echo "Releasing $(BINARY_NAME) $(VERSION) to GitHub ($(GITHUB_REPO))..."
	git tag $(VERSION)
	git push $(GITHUB_REMOTE) $(VERSION)
	$(MAKE) build VERSION=$(VERSION)
	gh release create $(VERSION) $(BUILD_DIR)/*.tar.gz $(BUILD_DIR)/*.zip \
		--repo $(GITHUB_REPO) --generate-notes
	@echo "Release $(VERSION) published to https://github.com/$(GITHUB_REPO)/releases/tag/$(VERSION)"

# 清理构建产物
clean:
	rm -rf $(BUILD_DIR)
```

## 错误处理

| 错误场景 | 处理策略 |
|---------|---------|
| gh CLI 未登录 | `check-gh` 目标检测，提示运行 `gh auth login` |
| 工作区有未提交修改 | `check-clean` 目标检测，提示先 commit 或 stash |
| github remote 未配置 | `check-github-remote` 目标检测，提示运行 `git remote add github git@github.com:childelins/ckjr-cli.git` |
| VERSION 未指定（release 时） | 检测到 "dev" 时报错，提示正确用法 |
| tag 已存在 | 检测后报错，避免覆盖已有 Release |
| 编译失败 | Make 自动中止，保留已构建的产物用于排查 |
| tag 推送到 GitHub 失败 | Make 中止，tag 仅存于本地，用户可手动 `git push github <tag>` 重试 |
| gh release create 失败 | tag 已推送到 GitHub，用户可手动 `gh release create --repo childelins/ckjr-cli` 补救 |

## 测试策略

### 手动验证清单

- [ ] `make version` 输出正确版本号
- [ ] `make build-local` 生成当前平台二进制到 bin/
- [ ] `make build VERSION=v0.0.3` 生成全部 5 个平台压缩包到 bin/
- [ ] 验证产物文件名格式与 install.sh 兼容
- [ ] 解压产物并运行 `./ckjr-cli --version` 显示正确版本
- [ ] `make clean` 清理 bin/ 目录
- [ ] `make release VERSION=v0.0.3` 完成全流程
- [ ] 在另一台机器上用 install.sh 安装刚发布的版本

### 兼容性验证

- [ ] install.sh 能正确匹配并下载新发布的产物
- [ ] CI 流水线（release.yml）仍正常工作，两套方案共存

## 实现注意事项

### 文件结构

```
ckjr-cli/
├── Makefile                              # 新增
├── .github/workflows/release.yml         # 保留不变
├── install.sh                            # 保留不变（已兼容）
├── bin/                                  # 构建输出（已在 .gitignore）
└── cmd/ckjr-cli/main.go                  # 构建入口（已有）
```

### 首次使用：配置 GitHub remote

项目 `origin` 指向 GitLab 内部仓库。首次使用前需添加 GitHub remote：

```bash
git remote add github git@github.com:childelins/ckjr-cli.git

# 验证
git remote -v
# origin   git@src2103.myckjr.com:ckjr001/ckjr-cli.git (fetch/push)
# github   git@github.com:childelins/ckjr-cli.git (fetch/push)
```

`make release` 的 `check-github-remote` 目标会自动检测，未配置时给出提示。

### 编译参数

- `CGO_ENABLED=0`：静态编译，无 C 依赖
- `-ldflags="-s -w"`：去除调试信息和符号表，减小二进制体积
- `-X main.Version=$(VERSION)`：注入版本号
- `-X main.Environment=production`：注入环境标识

### 与 CI 的关系

本方案与 `.github/workflows/release.yml` 共存：

- **CI 方案**：push tag 到 GitHub 后自动触发，适合常规发布
- **本地方案**：`make release` 手动执行，适合快速发布或调试

两者产物格式完全一致，install.sh 无需修改。

`make release` 会将 tag 推送到 GitHub（`github` remote），这也会触发 CI 流水线。但 `softprops/action-gh-release` 默认不覆盖已存在的 Release，所以本地创建的 Release 和 CI 不会冲突。

注意：`make release` 只将 tag 推送到 GitHub，不会推送到 GitLab（`origin`）。如需同步 tag 到 GitLab，可手动执行 `git push origin <tag>`。

### 依赖

- Go 1.24+（已安装）
- gh CLI（已安装 v2.88.1）
- git（已安装）
- zip 命令（用于 Windows 产物打包）
- GitHub remote 已配置（`git remote add github git@github.com:childelins/ckjr-cli.git`）

## 使用示例

```bash
# 首次使用：添加 GitHub remote（仅需一次）
git remote add github git@github.com:childelins/ckjr-cli.git

# 查看当前版本
make version

# 仅构建当前平台（开发调试用）
make build-local

# 构建全部平台（使用 git tag 版本）
make build

# 构建全部平台（指定版本）
make build VERSION=v0.1.0

# 一键发布到 GitHub（tag 推送到 github remote，Release 创建在 GitHub）
make release VERSION=v0.1.0

# 清理
make clean
```
