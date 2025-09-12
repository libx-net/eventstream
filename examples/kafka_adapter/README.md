# Kafka Adapter Example

这是一个使用内置 KafkaAdapter 的分布式事件流示例。

## 运行前提

1. 安装并启动 Kafka 服务器
2. 确保 Kafka 在 `localhost:9092` 可访问
3. 创建 topics（或启用自动创建）：
   - `user.created`
   - `order.placed`

## 运行说明

1. 进入项目目录：
```bash
cd examples/kafka_adapter
```

2. 下载依赖：
```bash
go mod tidy
```

3. 运行示例：
```bash
go run main.go
```

## 配置说明

示例使用了以下 Kafka 配置：
- Broker: localhost:9092
- 生产者：批量大小 100，Gzip 压缩，需要所有副本确认
- 消费者：从最早偏移量开始，1秒提交间隔

## 代码结构

- `main.go` - 主程序文件
- `go.mod` - 模块定义文件