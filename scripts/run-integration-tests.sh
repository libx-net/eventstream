#!/bin/bash

# Event Stream 集成测试运行脚本
# 用于本地开发和CI环境

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查前置条件
check_prerequisites() {
    log_info "检查前置条件..."
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或不在 PATH 中"
        exit 1
    fi
    
    # 检查 Docker 是否运行
    if ! docker info &> /dev/null; then
        log_error "Docker 未运行，请启动 Docker"
        exit 1
    fi
    
    # 检查 Go
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装或不在 PATH 中"
        exit 1
    fi
    
    # 检查 Go 版本
    GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    REQUIRED_VERSION="1.21"
    if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
        log_error "Go 版本过低，需要 $REQUIRED_VERSION 或更高版本，当前版本: $GO_VERSION"
        exit 1
    fi
    
    log_success "前置条件检查通过"
}

# 清理函数
cleanup() {
    log_info "清理资源..."
    
    # 停止并删除测试容器
    docker ps -a --filter "label=org.testcontainers=true" --format "{{.ID}}" | xargs -r docker rm -f
    
    # 清理测试网络
    docker network ls --filter "label=org.testcontainers=true" --format "{{.ID}}" | xargs -r docker network rm
    
    # 清理测试卷
    docker volume ls --filter "label=org.testcontainers=true" --format "{{.Name}}" | xargs -r docker volume rm
    
    log_success "资源清理完成"
}

# 预拉取镜像
pull_images() {
    log_info "预拉取 Docker 镜像..."
    
    KAFKA_IMAGE="confluentinc/confluent-local:7.5.0"
    
    if docker image inspect "$KAFKA_IMAGE" &> /dev/null; then
        log_info "镜像 $KAFKA_IMAGE 已存在"
    else
        log_info "拉取镜像 $KAFKA_IMAGE..."
        docker pull "$KAFKA_IMAGE"
        log_success "镜像拉取完成"
    fi
}

# 安装依赖
install_dependencies() {
    log_info "安装依赖..."
    
    # 主项目依赖
    go mod download
    
    # 集成测试依赖
    cd tests/integration
    go mod download
    cd ../..
    
    log_success "依赖安装完成"
}

# 运行单元测试
run_unit_tests() {
    log_info "运行单元测试..."
    
    if go test -v ./...; then
        log_success "单元测试通过"
    else
        log_error "单元测试失败"
        return 1
    fi
}

# 运行集成测试
run_integration_tests() {
    log_info "运行集成测试..."
    
    cd tests/integration
    
    # 设置环境变量
    export TESTCONTAINERS_RYUK_DISABLED=true
    export DOCKER_HOST=${DOCKER_HOST:-unix:///var/run/docker.sock}
    
    if go test -tags=integration -v ./...; then
        log_success "集成测试通过"
        cd ../..
    else
        log_error "集成测试失败"
        cd ../..
        return 1
    fi
}

# 运行性能测试
run_performance_tests() {
    log_info "运行性能测试..."
    
    cd tests/integration
    
    export TESTCONTAINERS_RYUK_DISABLED=true
    
    if go test -tags=integration -run=Performance -v ./...; then
        log_success "性能测试完成"
        cd ../..
    else
        log_warning "性能测试失败或跳过"
        cd ../..
    fi
}

# 生成测试报告
generate_report() {
    log_info "生成测试报告..."
    
    # 生成覆盖率报告
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    
    log_success "测试报告已生成: coverage.html"
}

# 主函数
main() {
    echo "========================================"
    echo "Event Stream 集成测试运行器"
    echo "========================================"
    
    # 解析命令行参数
    SKIP_UNIT=false
    SKIP_INTEGRATION=false
    SKIP_PERFORMANCE=false
    CLEANUP_ONLY=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-unit)
                SKIP_UNIT=true
                shift
                ;;
            --skip-integration)
                SKIP_INTEGRATION=true
                shift
                ;;
            --skip-performance)
                SKIP_PERFORMANCE=true
                shift
                ;;
            --cleanup-only)
                CLEANUP_ONLY=true
                shift
                ;;
            --help|-h)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --skip-unit         跳过单元测试"
                echo "  --skip-integration  跳过集成测试"
                echo "  --skip-performance  跳过性能测试"
                echo "  --cleanup-only      只执行清理操作"
                echo "  --help, -h          显示此帮助信息"
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                exit 1
                ;;
        esac
    done
    
    # 设置清理陷阱
    trap cleanup EXIT
    
    if [ "$CLEANUP_ONLY" = true ]; then
        cleanup
        exit 0
    fi
    
    # 执行测试流程
    check_prerequisites
    pull_images
    install_dependencies
    
    if [ "$SKIP_UNIT" = false ]; then
        run_unit_tests
    fi
    
    if [ "$SKIP_INTEGRATION" = false ]; then
        run_integration_tests
    fi
    
    if [ "$SKIP_PERFORMANCE" = false ]; then
        run_performance_tests
    fi
    
    generate_report
    
    log_success "所有测试完成！"
}

# 运行主函数
main "$@"