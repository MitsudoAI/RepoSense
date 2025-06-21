# RepoSense Makefile
# 用于管理构建、测试和发布任务

# 项目信息
BINARY_NAME = reposense
PACKAGE = ./cmd/reposense
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建标志
LDFLAGS = -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
STATIC_LDFLAGS = -ldflags "-s -w -extldflags '-static' -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 平台和架构
PLATFORMS = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
BUILD_DIR = build

# Go 相关设置
GO = go
GOFLAGS = -trimpath

# 默认目标
.PHONY: all
all: clean build

# 开发构建（当前平台）
.PHONY: build
build:
	@echo "🔨 构建 $(BINARY_NAME) for $(shell go env GOOS)/$(shell go env GOARCH)..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(STATIC_LDFLAGS) -o $(BINARY_NAME) $(PACKAGE)
	@echo "✅ 构建完成: $(BINARY_NAME)"

# 开发构建（带调试信息）
.PHONY: build-dev
build-dev:
	@echo "🔨 构建开发版本..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -race -o $(BINARY_NAME)-dev $(PACKAGE)
	@echo "✅ 开发版本构建完成: $(BINARY_NAME)-dev"

# 生产构建（高度优化）
.PHONY: build-prod
build-prod:
	@echo "🚀 构建生产版本..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(STATIC_LDFLAGS) -o $(BINARY_NAME) $(PACKAGE)
	@echo "✅ 生产版本构建完成: $(BINARY_NAME)"

# 多平台构建
.PHONY: build-all
build-all: clean-build
	@echo "🌍 构建所有平台版本..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		output=$(BUILD_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		if [ "$$GOOS" = "windows" ]; then \
			output=$$output.exe; \
		fi; \
		echo "构建 $$GOOS/$$GOARCH..."; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH $(GO) build $(GOFLAGS) $(STATIC_LDFLAGS) -o $$output $(PACKAGE); \
		if [ $$? -eq 0 ]; then \
			echo "✅ $$GOOS/$$GOARCH 构建成功"; \
		else \
			echo "❌ $$GOOS/$$GOARCH 构建失败"; \
			exit 1; \
		fi; \
	done
	@echo "🎉 所有平台构建完成！文件位于 $(BUILD_DIR)/"

# 安装到系统
.PHONY: install
install: build
	@echo "📦 安装 $(BINARY_NAME) 到系统..."
	@mkdir -p ~/bin
	@cp $(BINARY_NAME) ~/bin/$(BINARY_NAME)
	@chmod +x ~/bin/$(BINARY_NAME)
	@echo "✅ 已安装到 ~/bin/$(BINARY_NAME)"
	@echo "💡 确保 ~/bin 在你的 PATH 中"

# 安装到 /usr/local/bin (需要管理员权限)
.PHONY: install-system
install-system: build
	@echo "📦 安装 $(BINARY_NAME) 到系统路径..."
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "✅ 已安装到 /usr/local/bin/$(BINARY_NAME)"

# 运行测试
.PHONY: test
test:
	@echo "🧪 运行测试..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "✅ 测试完成"

# 运行测试并显示覆盖率
.PHONY: test-coverage
test-coverage: test
	@echo "📊 生成覆盖率报告..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告生成: coverage.html"

# 基准测试
.PHONY: bench
bench:
	@echo "⚡ 运行基准测试..."
	$(GO) test -bench=. -benchmem ./...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "🎨 格式化代码..."
	$(GO) fmt ./...
	@echo "✅ 代码格式化完成"

# 代码检查
.PHONY: lint
lint:
	@echo "🔍 运行代码检查..."
	@which golangci-lint > /dev/null || (echo "请安装 golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...
	@echo "✅ 代码检查完成"

# 依赖管理
.PHONY: deps
deps:
	@echo "📚 下载依赖..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✅ 依赖更新完成"

# 依赖升级
.PHONY: deps-upgrade
deps-upgrade:
	@echo "⬆️ 升级依赖..."
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "✅ 依赖升级完成"

# 清理构建文件
.PHONY: clean
clean:
	@echo "🧹 清理构建文件..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-dev
	@rm -f coverage.out coverage.html
	@echo "✅ 清理完成"

# 清理构建目录
.PHONY: clean-build
clean-build:
	@echo "🧹 清理构建目录..."
	@rm -rf $(BUILD_DIR)

# 完全清理
.PHONY: clean-all
clean-all: clean clean-build
	@echo "🧹 完全清理..."
	@$(GO) clean -cache -modcache -testcache
	@echo "✅ 完全清理完成"

# 创建发布包
.PHONY: package
package: build-all
	@echo "📦 创建发布包..."
	@mkdir -p $(BUILD_DIR)/packages
	@cd $(BUILD_DIR) && \
	for file in $(BINARY_NAME)-*; do \
		if [ -f "$$file" ]; then \
			platform=$$(echo $$file | sed 's/$(BINARY_NAME)-//'); \
			if echo "$$file" | grep -q "\.exe$$"; then \
				platform=$$(echo $$platform | sed 's/\.exe$$//'); \
				zip -q "packages/$(BINARY_NAME)-$(VERSION)-$$platform.zip" "$$file"; \
			else \
				tar -czf "packages/$(BINARY_NAME)-$(VERSION)-$$platform.tar.gz" "$$file"; \
			fi; \
			echo "✅ 创建包: $(BINARY_NAME)-$(VERSION)-$$platform"; \
		fi; \
	done
	@echo "🎉 所有发布包创建完成！位于 $(BUILD_DIR)/packages/"

# 快速测试构建
.PHONY: check
check: fmt build test
	@echo "🎯 快速检查完成"

# 完整验证
.PHONY: verify
verify: clean fmt lint test build
	@echo "✅ 完整验证通过"

# 开发工作流
.PHONY: dev
dev:
	@echo "🚀 启动开发模式..."
	@which air > /dev/null || (echo "请安装 air: go install github.com/cosmtrek/air@latest" && exit 1)
	air

# 显示构建信息
.PHONY: info
info:
	@echo "📋 构建信息:"
	@echo "  版本: $(VERSION)"
	@echo "  构建时间: $(BUILD_TIME)"
	@echo "  Git提交: $(GIT_COMMIT)"
	@echo "  Go版本: $(shell $(GO) version)"
	@echo "  平台: $(shell go env GOOS)/$(shell go env GOARCH)"

# 运行示例
.PHONY: demo
demo: build
	@echo "🎭 运行演示..."
	./$(BINARY_NAME) --help
	@echo ""
	@echo "尝试运行: ./$(BINARY_NAME) analyze . --disable-llm"

# 帮助信息
.PHONY: help
help:
	@echo "RepoSense Makefile 帮助"
	@echo ""
	@echo "构建命令:"
	@echo "  build          构建当前平台的二进制文件"
	@echo "  build-dev      构建开发版本（带调试信息）"
	@echo "  build-prod     构建生产版本（高度优化）"
	@echo "  build-all      构建所有平台版本"
	@echo ""
	@echo "安装命令:"
	@echo "  install        安装到 ~/bin"
	@echo "  install-system 安装到 /usr/local/bin（需要sudo）"
	@echo ""
	@echo "测试命令:"
	@echo "  test           运行测试"
	@echo "  test-coverage  运行测试并生成覆盖率报告"
	@echo "  bench          运行基准测试"
	@echo ""
	@echo "代码质量:"
	@echo "  fmt            格式化代码"
	@echo "  lint           代码检查"
	@echo "  check          快速检查（fmt + build + test）"
	@echo "  verify         完整验证（fmt + lint + test + build）"
	@echo ""
	@echo "依赖管理:"
	@echo "  deps           下载和整理依赖"
	@echo "  deps-upgrade   升级所有依赖"
	@echo ""
	@echo "清理命令:"
	@echo "  clean          清理构建文件"
	@echo "  clean-build    清理构建目录"
	@echo "  clean-all      完全清理"
	@echo ""
	@echo "发布命令:"
	@echo "  package        创建所有平台的发布包"
	@echo ""
	@echo "开发命令:"
	@echo "  dev            启动开发模式（需要air）"
	@echo "  demo           运行演示"
	@echo "  info           显示构建信息"
	@echo ""
	@echo "用法示例:"
	@echo "  make build              # 构建当前平台版本"
	@echo "  make build-all          # 构建所有平台版本"
	@echo "  make test               # 运行测试"
	@echo "  make install            # 安装到用户目录"
	@echo "  make package            # 创建发布包"
	@echo "  make verify             # 完整验证"