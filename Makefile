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

# 清理构建产物
clean:
	rm -rf $(BUILD_DIR)
