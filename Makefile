# 媒体管理程序 Makefile
# 支持多平台编译

# 项目名称
PROJECT_NAME = media-manager

# 编译参数
GOFLAGS = -ldflags "-s -w"

# 输出目录
BUILD_DIR = build

# 平台定义
PLATFORMS := linux windows darwin
ARCH := amd64

# 默认目标
.DEFAULT_GOAL := help

# 帮助信息
.PHONY: help
help:
	@echo "媒体管理程序编译工具"
	@echo "可用命令:"
	@echo "  make all          - 编译所有平台版本"
	@echo "  make linux        - 编译Linux版本"
	@echo "  make windows      - 编译Windows版本"
	@echo "  make macos        - 编译macOS版本"
	@echo "  make clean        - 清理编译结果"
	@echo "  make help         - 显示帮助信息"

# 创建输出目录
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# 编译所有平台
.PHONY: all
all: linux windows macos
	@echo "所有平台编译完成，输出目录: $(BUILD_DIR)"

# 编译Linux版本
.PHONY: linux
linux: $(BUILD_DIR)
	@echo "编译Linux版本..."
	GOOS=linux GOARCH=$(ARCH) go build $(GOFLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-linux-$(ARCH) .

# 编译Windows版本
.PHONY: windows
windows: $(BUILD_DIR)
	@echo "编译Windows版本..."
	GOOS=windows GOARCH=$(ARCH) go build $(GOFLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-windows-$(ARCH).exe .

# 编译macOS版本
.PHONY: macos
macos: $(BUILD_DIR)
	@echo "编译macOS版本..."
	GOOS=darwin GOARCH=$(ARCH) go build $(GOFLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME)-darwin-$(ARCH)

# 清理编译结果
.PHONY: clean
clean:
	@echo "清理编译结果..."
	@rm -rf $(BUILD_DIR)
	@echo "清理完成"
