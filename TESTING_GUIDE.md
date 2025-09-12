# Event Stream 集成测试指南

## 📋 概述

本项目现在包含了完整的集成测试套件，使用 Testcontainers 与真实的 Kafka 容器进行测试，支持 Docker 和 Podman 两种容器运行时。

## 🏗️ 测试架构

### 测试层次
```
┌─────────────────────────────────────┐
│           单元测试                    │
│     (Mock 外部依赖)                  │
├─────────────────────────────────────┤
│          集成测试                     │
│   (真实 Kafka 容器)                  │
├─────────────────────────────────────┤
│         端到端测试                    │
│    (完整系统验证)                     │
└─────────────────────────────────────┘
```

### 文件结构
```
event-stream/
├── *_test.go                          # 单元测试 (Mock)
├── tests/integration/
│   ├── kafka_integration_test.go      # 集成测试 (真实容器)
│   ├── go.mod                         # 独立依赖管理
│   └── README.md                      # 详细文档
├── .github/workflows/test.yml         # CI/CD 配置
├── Makefile                           # 构建命令
└── scripts/run-integration-tests.sh   # 测试脚本
```

## 🔧 容器运行时支持

### 自动检测机制
测试会自动检测并配置可用的容器运行时：

1. **Docker 检测**
   ```bash
   docker info  # 检查 Docker 是否可用
   ```

2. **Podman 检测**
   ```bash
   podman info  # 检查 Podman 是否可用
   ```

### Podman 特殊配置
- 自动设置 socket 路径
- 禁用 Ryuk 容器清理器
- 支持用户模式和系统模式

## 🧪 测试内容

### KafkaAdapter 集成测试
- ✅ **基本发布订阅**: 验证消息发布和接收
- ✅ **批量消息处理**: 测试高吞吐量场景
- ✅ **多消费者组**: 验证消息广播机制
- ✅ **错误处理**: 测试连接失败和重试

### DistributedEventBus 集成测试
- ✅ **端到端事件流**: 完整的事件处理流程
- ✅ **多消费者组**: 分布式事件分发
- ✅ **数据序列化**: JSON 序列化/反序列化
- ✅ **并发处理**: 多线程安全性验证

## 🚀 运行方式

### 1. 使用 Makefile (推荐)
```bash
# 运行单元测试
make test-unit

# 运行集成测试
make test-integration

# 运行所有测试
make test-all

# 本地完整测试
make local-test
```

### 2. 直接运行
```bash
# 进入集成测试目录
cd tests/integration

# 运行集成测试
go test -tags=integration -v ./...
```

### 3. 使用脚本
```bash
# 完整测试流程
./scripts/run-integration-tests.sh

# 跳过单元测试
./scripts/run-integration-tests.sh --skip-unit
```

## 🔍 故障排除

### 容器运行时问题

#### Docker
```bash
# 检查状态
docker info

# 权限问题 (Linux)
sudo chmod 666 /var/run/docker.sock
```

#### Podman
```bash
# 检查状态
podman info

# 启动 socket (用户模式)
systemctl --user start podman.socket

# 启动 socket (系统模式)
sudo systemctl start podman.socket

# 检查 socket 路径
ls -la /run/user/$(id -u)/podman/podman.sock
```

### 常见错误

1. **权限被拒绝**
   - 确保用户在 docker/podman 组中
   - 检查 socket 文件权限

2. **容器启动超时**
   - 预先拉取镜像: `docker/podman pull confluentinc/confluent-local:7.5.0`
   - 检查网络连接

3. **端口冲突**
   - 测试使用随机端口，通常不会冲突
   - 检查是否有其他 Kafka 实例运行

## 📊 CI/CD 集成

### GitHub Actions
```yaml
# 分层测试策略
jobs:
  unit-tests:      # 快速反馈
  integration:     # 完整验证
  code-quality:    # 代码质量检查
```

### 支持的环境
- ✅ **Linux**: 完全支持 (Docker/Podman)
- ⚠️ **Windows**: 支持 Docker Desktop 和 Podman Desktop
- ✅ **macOS**: 支持 Docker Desktop

## 🎯 最佳实践

### 开发流程
1. **日常开发**: 主要运行单元测试 (快速)
2. **提交前**: 运行完整测试套件
3. **CI 环境**: 分离快速测试和完整测试
4. **发布前**: 集成测试作为最后验证

### 性能考虑
- **单元测试**: ~2秒 (快速反馈)
- **集成测试**: ~30-60秒 (首次需下载镜像)
- **并行执行**: CI 中并行运行不同测试类型

## 📈 测试覆盖

### 补充了单元测试的不足
- ❌ **过度 Mock** → ✅ **真实集成**
- ❌ **无法验证网络** → ✅ **真实网络通信**
- ❌ **缺少序列化测试** → ✅ **完整数据流**
- ❌ **无并发验证** → ✅ **真实并发场景**

### 测试金字塔
```
        /\
       /  \     E2E Tests (少量)
      /____\
     /      \   Integration Tests (适量)
    /________\
   /          \  Unit Tests (大量)
  /__________\
```

## 🔮 未来扩展

### 计划中的改进
1. **性能基准测试**: 吞吐量和延迟测试
2. **故障注入测试**: 网络中断、节点故障
3. **多版本兼容性**: 测试不同 Kafka 版本
4. **安全性测试**: SSL/SASL 认证测试

这套集成测试为 event-stream 项目提供了可靠的质量保证！🎉