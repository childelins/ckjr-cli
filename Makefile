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

.PHONY: test build build-local release clean version check-gh check-clean check-github-remote

# 打印版本号
version:
	@echo $(VERSION)

# 运行测试
test:
	go test ./...

# 仅当前平台编译
build-local: test
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) ($(VERSION))"

# 多平台交叉编译
build: test clean
	@echo "Building $(BINARY_NAME) $(VERSION) for all platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output_name=$(BINARY_NAME); \
		if [ "$$GOOS" = "windows" ]; then output_name=$(BINARY_NAME).exe; fi; \
		echo "  -> $$GOOS/$$GOARCH"; \
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
release: test check-gh check-clean check-github-remote
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
