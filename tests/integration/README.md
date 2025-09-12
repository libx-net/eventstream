# Event Stream 集成测试

本目录包含使用 Testcontainers 的集成测试，用于测试 KafkaAdapter 和 DistributedEventBus 与真实 Kafka 的集成。

## 前置要求

1. **容器运行时**: 必须安装并运行以下之一：
   - **Docker**: 推荐使用 Docker Desktop 或 Docker Engine
   - **Podman**: 支持 Podman 作为 Docker 的替代品
2. **Go 1.21+**: 确保 Go 版本兼容
3. **网络连接**: 首次运行时需要下载 Kafka 容器镜像

## 快速开始

### 1. 安装依赖

```bash
# 在项目根目录
make deps

# 或者手动安装
cd tests/integration
go mod download
```

### 2. 运行集成测试

```bash
# 使用 Makefile（推荐）
make test-integration

# 或者直接运行
cd tests/integration
go test -tags=integration -v ./...
```

### 3. 运行所有测试

```bash
# 运行单元测试 + 集成测试
make test-all
```

## 测试内容

### KafkaAdapter 集成测试

- **基本发布订阅**: 测试消息的发布和接收
- **多消息处理**: 测试批量消息处理
- **多消费者组**: 测试消息广播到不同消费者组
- **错误处理**: 测试连接失败等异常情况

### DistributedEventBus 集成测试

- **事件发布订阅**: 测试完整的事件流程
- **多消费者组**: 测试事件广播机制
- **事件序列化**: 测试复杂数据结构的序列化/反序列化
- **错误处理**: 测试事件处理失败的情况

## 测试配置

### Kafka 容器配置

```go
kafkaContainer, err := kafka.RunContainer(ctx,
    kafka.WithClusterID("test-cluster"),
    testcontainers.WithImage("confluentinc/confluent-local:7.5.0"),
)
```

### 适配器配置

```go
kafkaConfig := eventstream.KafkaConfig{
    Brokers: brokers, // 从容器获取
    Producer: eventstream.KafkaProducerConfig{
        BatchSize:    1,
        BatchTimeout: 10 * time.Millisecond,
        RequiredAcks: 1,
    },
    Consumer: eventstream.KafkaConsumerConfig{
        StartOffset:    "earliest",
        CommitInterval: 1 * time.Second,
        MaxWait:        1 * time.Second,
        MinBytes:       1,
        MaxBytes:       10e6,
    },
}
```

## CI/CD 集成

### GitHub Actions 示例

```yaml
name: Integration Tests
on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.23'
      
      - name: Start Docker
        run: |
          sudo systemctl start docker
          sudo chmod 666 /var/run/docker.sock
      
      - name: Run Integration Tests
        run: make test-integration
        env:
          TESTCONTAINERS_RYUK_DISABLED: true
```

### GitLab CI 示例

```yaml
integration-tests:
  image: golang:1.23
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2376
    DOCKER_TLS_CERTDIR: "/certs"
    TESTCONTAINERS_HOST_OVERRIDE: "docker"
  before_script:
    - until docker info; do sleep 1; done
  script:
    - make test-integration
```

## 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `TESTCONTAINERS_RYUK_DISABLED` | 禁用 Ryuk 容器清理 | `false` |
| `DOCKER_HOST` | 容器运行时主机地址 | 自动检测 |
| `TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE` | 覆盖 Docker socket 路径 | 自动检测 |
| `TESTCONTAINERS_PODMAN` | 启用 Podman 支持 | 自动设置 |

### 容器运行时自动检测

测试会自动检测可用的容器运行时：

1. **Docker**: 检查 `docker info` 命令
2. **Podman**: 检查 `podman info` 命令

对于 Podman，会自动尝试以下 socket 路径：
- `unix:///run/user/1000/podman/podman.sock` (用户模式)
- `unix:///run/podman/podman.sock` (系统模式)
- `unix:///var/run/podman/podman.sock` (备用路径)

## 故障排除

### 1. 容器运行时连接问题

#### Docker
```bash
# 检查 Docker 是否运行
docker info

# 检查权限
sudo chmod 666 /var/run/docker.sock
```

#### Podman
```bash
# 检查 Podman 是否运行
podman info

# 启动 Podman socket (用户模式)
systemctl --user start podman.socket

# 启动 Podman socket (系统模式)
sudo systemctl start podman.socket

# 检查 socket 是否存在
ls -la /run/user/$(id -u)/podman/podman.sock
ls -la /run/podman/podman.sock
```

### 2. 容器启动超时

```bash
# Docker
docker pull confluentinc/confluent-local:7.5.0

# Podman
podman pull confluentinc/confluent-local:7.5.0
```

### 3. Podman 特定问题

#### 权限问题
```bash
# 确保用户在正确的组中
sudo usermod -aG podman $USER
newgrp podman

# 或者使用 rootless 模式
podman system migrate
```

#### Socket 路径问题
```bash
# 手动设置环境变量
export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/run/user/$(id -u)/podman/podman.sock
```

### 4. 端口冲突

测试使用随机端口，通常不会有冲突。如果有问题，检查是否有其他 Kafka 实例运行。

### 5. 内存不足

Kafka 容器需要一定内存，确保容器运行时有足够资源：

```bash
# Docker
docker system df
docker system prune

# Podman
podman system df
podman system prune
```

### 6. Windows 上的 Podman

在 Windows 上使用 Podman Desktop：

```bash
# 检查 Podman Machine 状态
podman machine list
podman machine start

# 设置环境变量
$env:DOCKER_HOST = "npipe:////./pipe/podman-machine-default"
```

## 性能考虑

- **并行运行**: 测试可以并行运行，但会消耗更多资源
- **容器复用**: 同一测试中的子测试会复用容器
- **清理**: 测试结束后容器会自动清理

## 最佳实践

1. **本地开发**: 使用 `make test-unit` 进行快速反馈
2. **提交前**: 运行 `make test-all` 确保完整性
3. **CI 环境**: 分离单元测试和集成测试流水线
4. **调试**: 使用 `-v` 参数查看详细输出

## 扩展测试

要添加新的集成测试：

1. 在 `kafka_integration_test.go` 中添加新的测试函数
2. 使用现有的容器设置模式
3. 确保测试的隔离性（使用不同的 topic 和 group）
4. 添加适当的超时和错误处理

## 相关链接

- [Testcontainers Go 文档](https://golang.testcontainers.org/)
- [Kafka Go 客户端](https://github.com/segmentio/kafka-go)
- [项目主 README](../../README.md)