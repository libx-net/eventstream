# Event Stream 项目 Makefile

.PHONY: test test-unit test-integration test-all clean deps help

# 默认目标
help:
	@echo "Available targets:"
	@echo "  test-unit        - 运行单元测试"
	@echo "  test-integration - 运行集成测试 (需要Docker)"
	@echo "  test-all         - 运行所有测试"
	@echo "  deps             - 安装依赖"
	@echo "  clean            - 清理测试缓存"
	@echo "  help             - 显示此帮助信息"

# 安装依赖
deps:
	go mod download
	cd tests/integration && go mod download

# 运行单元测试
test-unit:
	@echo "Running unit tests..."
	go test -v ./...

# 运行集成测试
test-integration:
	@echo "Running integration tests..."
	@echo "Note: This requires Docker or Podman to be running"
	@if command -v docker >/dev/null 2>&1; then \
		echo "Found Docker runtime"; \
	elif command -v podman >/dev/null 2>&1; then \
		echo "Found Podman runtime"; \
	else \
		echo "Error: Neither Docker nor Podman found. Please install one of them."; \
		exit 1; \
	fi
	cd tests/integration && go test -tags=integration -v ./...

# 运行所有测试
test-all: test-unit test-integration

# 清理测试缓存
clean:
	go clean -testcache
	cd tests/integration && go clean -testcache

# 运行基准测试
benchmark:
	go test -bench=. -benchmem ./...

# 生成测试覆盖率报告（排除examples目录）
coverage:
	go test -coverprofile=coverage.out $(shell go list ./... | grep -v /examples/)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 运行竞态检测
race:
	go test -race ./...

# 运行短测试（跳过性能测试）
test-short:
	go test -short ./...

# 检查代码格式
fmt:
	go fmt ./...

# 运行代码检查
vet:
	go vet ./...

# 运行所有检查
check: fmt vet race

# CI 流水线目标
ci: deps check test-unit
	@echo "CI pipeline completed successfully"

# 完整的本地测试（包括集成测试）
local-test: deps check test-all
	@echo "Local testing completed successfully"