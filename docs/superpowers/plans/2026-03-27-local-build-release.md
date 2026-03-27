# 本地多平台构建与 GitHub Release 发布实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 创建 Makefile 实现本地多平台交叉编译和一键发布到 GitHub Release

**Architecture:** 单个 Makefile 包含 build/build-local/release/clean/version 目标。双仓库模式：origin 指向 GitLab（开发），github remote 指向 GitHub（发布）。`make release VERSION=v0.1.0` 一条命令完成 tag + 构建 + 创建 Release + 上传。

**Tech Stack:** Go 1.24+, Make, gh CLI, git

---

### Task 1: 创建 Makefile 基础框架

**Files:**
- Create: `Makefile`

- [ ] **Step 1: 创建 Makefile 包含变量定义和辅助目标**

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

# 清理构建产物
clean:
	rm -rf $(BUILD_DIR)
```

- [ ] **Step 2: 验证基础框架**

Run: `make version`
Expected: 输出当前 git tag 版本号（如 `v0.0.2`）或 `dev`

Run: `make clean`
Expected: 无错误输出

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat: add Makefile with version and clean targets"
```

### Task 2: 添加 build-local 目标（当前平台编译）

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: 在 Makefile 中添加 build-local 目标**

在 `clean` 目标之前添加：

```makefile
# 仅当前平台编译
build-local:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) ($(VERSION))"
```

- [ ] **Step 2: 验证 build-local**

Run: `make build-local`
Expected: 输出 `Built bin/ckjr-cli (v0.0.2)` 并在 bin/ 下生成二进制文件

Run: `bin/ckjr-cli --help`
Expected: 显示 CLI 帮助信息

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat: add build-local target for current platform compilation"
```

### Task 3: 添加 build 目标（多平台交叉编译）

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: 在 Makefile 中添加 build 目标**

在 `build-local` 之后添加：

```makefile
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
```

- [ ] **Step 2: 验证多平台构建**

Run: `make build VERSION=v0.0.2`
Expected:
- 输出 5 个平台的编译进度
- bin/ 下生成 5 个压缩包：
  - `ckjr-cli_v0.0.2_linux_amd64.tar.gz`
  - `ckjr-cli_v0.0.2_linux_arm64.tar.gz`
  - `ckjr-cli_v0.0.2_darwin_amd64.tar.gz`
  - `ckjr-cli_v0.0.2_darwin_arm64.tar.gz`
  - `ckjr-cli_v0.0.2_windows_amd64.zip`

Run: `ls -lh bin/`
Expected: 5 个压缩包文件

- [ ] **Step 3: 验证产物文件名与 install.sh 兼容**

install.sh 使用的模式是 `${BINARY_NAME}_.*_${os}_${arch}\.tar\.gz`（正则匹配）。

Run: `echo "ckjr-cli_v0.0.2_linux_amd64.tar.gz" | grep -E "ckjr-cli_.*_linux_amd64\.tar\.gz"`
Expected: 匹配成功

- [ ] **Step 4: 验证解压后二进制可用**

Run: `mkdir -p /tmp/test-extract && tar -xzf bin/ckjr-cli_v0.0.2_linux_amd64.tar.gz -C /tmp/test-extract && /tmp/test-extract/ckjr-cli_v0.0.2_linux_amd64/ckjr-cli --help && rm -rf /tmp/test-extract`
Expected: 显示 CLI 帮助信息

- [ ] **Step 5: Commit**

```bash
git add Makefile
git commit -m "feat: add build target for multi-platform cross-compilation"
```

### Task 4: 添加前置检查和 release 目标

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: 在 Makefile 中添加前置检查目标**

在 `build` 目标之后添加：

```makefile
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
```

- [ ] **Step 2: 验证前置检查**

Run: `make check-gh`
Expected: 无输出（gh 已登录）或提示登录

Run: `make check-github-remote`
Expected: 报错提示添加 github remote（如果尚未配置）

- [ ] **Step 3: 添加 release 目标**

在前置检查之后添加：

```makefile
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
```

- [ ] **Step 4: 验证 release 目标的防护检查**

Run: `make release`（不带 VERSION）
Expected: 报错 `Error: VERSION is required. Usage: make release VERSION=v0.1.0`

Run: `make release VERSION=v0.0.2`（已存在的 tag）
Expected: 报错 `Error: tag v0.0.2 already exists`

- [ ] **Step 5: Commit**

```bash
git add Makefile
git commit -m "feat: add release target with pre-flight checks and GitHub upload"
```

### Task 5: 配置 GitHub remote 并验证完整流程

**Files:**
- 无代码修改，仅环境配置和端到端验证

- [ ] **Step 1: 添加 GitHub remote（如果尚未配置）**

Run: `git remote get-url github 2>/dev/null || git remote add github git@github.com:childelins/ckjr-cli.git`

Run: `git remote -v`
Expected:
```
github  git@github.com:childelins/ckjr-cli.git (fetch)
github  git@github.com:childelins/ckjr-cli.git (push)
origin  git@src2103.myckjr.com:ckjr001/ckjr-cli.git (fetch)
origin  git@src2103.myckjr.com:ckjr001/ckjr-cli.git (push)
```

- [ ] **Step 2: 验证 check-github-remote 通过**

Run: `make check-github-remote`
Expected: 无输出（检查通过）

- [ ] **Step 3: 端到端验证 make build**

Run: `make clean && make build VERSION=v0.0.3-test`
Expected: 5 个压缩包生成在 bin/ 下

Run: `make clean`（清理测试产物）

- [ ] **Step 4: 记录使用说明**

验证完成后，完整使用流程：
```bash
# 日常开发构建
make build-local

# 多平台构建（不发布）
make build VERSION=v0.1.0

# 一键发布
make release VERSION=v0.1.0
```

注意：实际 `make release` 会创建真实的 tag 和 Release，在验证阶段不要执行。Task 4 Step 4 的防护检查验证已确认 release 逻辑正确。
