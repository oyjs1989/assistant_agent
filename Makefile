# Assistant Agent Makefile

# 变量定义
BINARY_NAME=assistant_agent
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# 默认目标
.PHONY: all
all: build

# 构建
.PHONY: build
build:
	@echo "Building ${BINARY_NAME}..."
	go build ${LDFLAGS} -o ${BINARY_NAME} main.go

# 构建所有平台
.PHONY: build-all
build-all: build-linux build-windows build-darwin

# 构建 Linux
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}_linux_amd64 main.go
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}_linux_arm64 main.go

# 构建 Windows
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}_windows_amd64.exe main.go
	GOOS=windows GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}_windows_arm64.exe main.go

# 构建 macOS
.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}_darwin_amd64 main.go
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}_darwin_arm64 main.go

# 测试
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# 测试覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 清理
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -f ${BINARY_NAME}
	rm -f ${BINARY_NAME}_*
	rm -f coverage.out coverage.html
	rm -rf dist/

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# 格式化代码
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# 运行
.PHONY: run
run: build
	@echo "Running ${BINARY_NAME}..."
	./${BINARY_NAME}

# 开发模式运行
.PHONY: dev
dev:
	@echo "Running in development mode..."
	go run main.go

# 创建发布包
.PHONY: release
release: build-all
	@echo "Creating release packages..."
	mkdir -p dist
	tar -czf dist/${BINARY_NAME}_linux_amd64.tar.gz ${BINARY_NAME}_linux_amd64 config.yaml README.md
	tar -czf dist/${BINARY_NAME}_linux_arm64.tar.gz ${BINARY_NAME}_linux_arm64 config.yaml README.md
	zip -j dist/${BINARY_NAME}_windows_amd64.zip ${BINARY_NAME}_windows_amd64.exe config.yaml README.md
	zip -j dist/${BINARY_NAME}_windows_arm64.zip ${BINARY_NAME}_windows_arm64.exe config.yaml README.md
	tar -czf dist/${BINARY_NAME}_darwin_amd64.tar.gz ${BINARY_NAME}_darwin_amd64 config.yaml README.md
	tar -czf dist/${BINARY_NAME}_darwin_arm64.tar.gz ${BINARY_NAME}_darwin_arm64 config.yaml README.md

# Docker 构建
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t assistant-agent:${VERSION} .
	docker tag assistant-agent:${VERSION} assistant-agent:latest

# Docker 运行
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -d --name assistant-agent \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD)/config.yaml:/app/config.yaml \
		assistant-agent:latest

# 帮助
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all platforms"
	@echo "  build-linux  - Build for Linux"
	@echo "  build-windows- Build for Windows"
	@echo "  build-darwin - Build for macOS"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  run          - Build and run"
	@echo "  dev          - Run in development mode"
	@echo "  release      - Create release packages"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help" 