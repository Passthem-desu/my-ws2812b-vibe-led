# ==============================================================================
# 项目配置变量
# ==============================================================================

# 目标设备配置：OrangePi Zero 2W (ARM64 Linux)
TARGET_OS := linux
TARGET_ARCH := arm64
TARGET_CC := aarch64-linux-gnu-gcc
AR := aarch64-linux-gnu-ar

CGO_ENABLED := 1

GO_APP_NAME := led-shader-api
GO_SOURCE := main.go

BUILD_DIR := build
C_BUILD_DIR := $(BUILD_DIR)/cgo
OUTPUT_PATH := $(BUILD_DIR)/$(GO_APP_NAME)-$(TARGET_ARCH)

C_DRIVER_DIR := c-driver
C_DRIVER_SOURCES := $(wildcard $(C_DRIVER_DIR)/*.c)
C_DRIVER_OBJECTS := $(patsubst $(C_DRIVER_DIR)/%.c, $(C_BUILD_DIR)/%.o, $(C_DRIVER_SOURCES))
C_DRIVER_LIB_NAME := $(C_BUILD_DIR)/libdriver.a

# ==============================================================================
# 导出环境变量 (用于 Go 和 CGO 交叉编译)
# ==============================================================================
export GOOS := $(TARGET_OS)
export GOARCH := $(TARGET_ARCH)
export CGO_ENABLED := $(CGO_ENABLED)
export CC := $(TARGET_CC)
export AR := $(AR)

export CGO_LDFLAGS := -L$(C_BUILD_DIR) -ldriver
export CGO_CFLAGS := -I$(C_DRIVER_DIR)

.PHONY: all driver go clean deploy

all: $(OUTPUT_PATH)

# ----------------- C Driver 编译目标 -----------------
$(C_DRIVER_LIB_NAME): $(C_DRIVER_OBJECTS)
	@echo "-> [CGO] 链接静态库 $@"
	@mkdir -p $(C_BUILD_DIR)
	$(AR) rcs $@ $^

$(C_BUILD_DIR)/%.o: $(C_DRIVER_DIR)/%.c
	@echo "-> [CGO] 交叉编译 C 文件 $<"
	@mkdir -p $(C_BUILD_DIR)
	$(CC) -c -o $@ $< -I$(C_DRIVER_DIR) -Wall -Werror

# ----------------- Go 主程序编译目标 -----------------
$(OUTPUT_PATH): $(C_DRIVER_LIB_NAME) $(GO_SOURCE)
	@mkdir -p $(BUILD_DIR)
	@echo "-> [Go] 交叉编译 Go 主程序 $@"
	go build -o $@ .

# ----------------- 清理目标 -----------------
clean:
	@echo "-> [CLEAN] 清理构建文件..."
	rm -rf $(BUILD_DIR)

# ----------------- 部署目标 -----------------
DEPLOY_HOST := orangepi
DEPLOY_PATH := /home/kagami/led-server
DEPLOY_APP_NAME := $(notdir $(OUTPUT_PATH))

deploy: all
	@echo "-> [DEPLOY] 正在验证 SSH 连接和目标目录..."
	@ssh -q $(DEPLOY_HOST) exit || { echo "错误: 无法连接到 $(DEPLOY_HOST). 请检查 SSH 配置或网络."; exit 1; }

	@echo "-> [DEPLOY] 远程创建目录 $(DEPLOY_PATH)..."
	@ssh $(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH) && echo '远程目录创建/验证成功.'"

	@echo "-> [DEPLOY] 正在传输可执行文件 $(OUTPUT_PATH) 到 $(DEPLOY_HOST):$(DEPLOY_PATH)..."
	@scp $(OUTPUT_PATH) $(DEPLOY_HOST):$(DEPLOY_PATH)/$(DEPLOY_APP_NAME)
	@echo "-> [DEPLOY] 文件传输成功!"

	@echo "---"
	@echo "✅ 部署完成! 可通过 SSH 运行:"
	@echo "ssh $(DEPLOY_HOST) '$(DEPLOY_PATH)/$(DEPLOY_APP_NAME)'"
