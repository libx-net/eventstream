# EventStream Examples

本目录包含 EventStream 库的各种使用示例，每个示例都是独立的 Go module 项目。

## 示例项目列表

### 1. Memory Basic (`memory_basic/`)
- **类型**: 内存模式示例
- **描述**: 演示事件流的核心功能，包括事件历史、统计信息和重试策略
- **运行**: `cd memory_basic && go run main.go`



### 3. Kafka Adapter (`kafka_adapter/`)
- **类型**: 内置适配器示例
- **描述**: 使用内置的 KafkaAdapter 与 Apache Kafka 集成
- **依赖**: Kafka 服务器 (localhost:9092)
- **运行**: `cd kafka_adapter && go mod tidy && go run main.go`

### 3. Custom Adapter (`custom_adapter/`)
- **类型**: 自定义适配器示例
- **描述**: 演示如何实现自定义 MQAdapter（以 RabbitMQ 为例），展示适配器扩展能力
- **依赖**: RabbitMQ 服务器 (优先使用 RABBITMQ_URL 环境变量，默认 amqp://guest:guest@localhost:5672/)
- **运行**: `cd custom_adapter && go mod tidy && go run main.go`

## 项目结构

每个示例项目都包含：
- `go.mod` - 独立的模块定义文件
- `main.go` - 示例主程序
- `README.md` - 项目特定的说明文档

## 运行说明

1. **独立运行**: 每个示例都可以独立运行，进入对应目录执行 `go run main.go`
2. **依赖管理**: 需要外部服务的示例（Kafka、Redis）需要先启动相应服务
3. **模块替换**: 所有示例都使用 `replace` 指令指向主项目，确保使用最新代码

## 开发说明

- 每个示例都是独立的 Go module，便于理解和测试特定功能
- 自定义适配器示例展示了如何实现 `eventstream.MQAdapter` 接口
- 内置适配器示例展示了库提供的现成解决方案