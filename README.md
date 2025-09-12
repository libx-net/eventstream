警告：此项目仍在开发中，禁止用于生产环境

# EventStream — 轻量级事件总线库（Go）

[![GitHub Actions](https://img.shields.io/github/actions/workflow/status/libx-net/eventstream/test.yml?branch=main&label=Build&logo=github)](https://github.com/libx-net/eventstream/actions)
[![codecov](https://codecov.io/gh/libx-net/eventstream/branch/main/graph/badge.svg)](https://codecov.io/gh/libx-net/eventstream)
[![Go Report Card](https://goreportcard.com/badge/github.com/libx-net/eventstream)](https://goreportcard.com/report/github.com/libx-net/eventstream)
[![golangci-lint](https://img.shields.io/badge/golangci--lint-passing-brightgreen)](https://golangci-lint.run/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/libx-net/eventstream)](https://golang.org/)
[![License](https://img.shields.io/github/license/libx-net/eventstream)](LICENSE)


## 简介
EventStream 是一个用于构建内存或分布式事件传递的轻量级 Go 库。该项目仍在积极开发中，API 可能发生变化，当前版本不适合在生产环境中使用。请在生产环境使用前自行评估稳定性并进行充分测试。

## 主要功能
- 可插拔的 MQ 适配器接口（支持本地内存模式与外部 MQ）
- 事件序列化与反序列化支持
- 订阅者分组（consumer group）能力与并发控制
- 可选历史记录与统计指标
- 完整的单元测试覆盖和示例代码

## 安装
在 Go 模块项目中直接引用本模块：
```
go get libx.net/eventstream
```

## 快速开始

### 分布式模式 - 使用内置 KafkaAdapter

1. 创建 Kafka 配置并初始化适配器：
```go
kafkaCfg := eventstream.KafkaConfig{
    Brokers: []string{"localhost:9092"},
    // 其他 Kafka 配置选项
}
kafkaAdapter, err := eventstream.NewKafkaAdapter(kafkaCfg)
if err != nil {
    // 处理错误
}
defer kafkaAdapter.Close()
```

2. 创建分布式事件总线配置：
```go
cfg := eventstream.DefaultDistributedConfig()
cfg.Distributed.MQAdapter = kafkaAdapter
```

3. 创建事件总线并订阅事件：
```go
bus, err := eventstream.New(cfg)
if err != nil {
    // 处理错误
}
defer bus.Close()

// 订阅主题
sub, err := bus.On("user.registered", "user-service", func(ctx context.Context, e *eventstream.Event) error {
    var userData map[string]interface{}
    if err := e.Unmarshal(&userData); err != nil {
        return err
    }
    fmt.Printf("处理用户注册事件: %v\n", userData)
    return nil
})
if err != nil {
    // 处理错误
}
defer bus.Off(sub)
```

4. 发布事件：
```go
err = bus.Emit(context.Background(), "user.registered", map[string]interface{}{
    "user_id": "u1",
    "email": "user@example.com",
    "name": "Test User"
})
if err != nil {
    // 处理错误
}
```

### 内存模式（用于本地测试和开发）
- 适用于单机环境，事件在进程内传递
- 无需外部消息队列依赖，适合测试和开发
- 消费者组（group）参数主要用于接口兼容性，在单机环境中实际意义有限
- 详见 `examples/memory_basic` 中的基础示例

## 示例
`examples/` 目录下提供了几个示例：
- `memory_basic`：演示了内存模式的基础用法，适用于单机测试和开发。
- `kafka_adapter`：演示了如何使用内置的 Kafka 适配器来构建分布式事件流。
- `custom_adapter`：演示了如何实现一个自定义的 MQ 适配器（以 RabbitMQ 为例），并将其集成到 `event-stream` 中。

要了解如何实现自定义的分布式适配器，请重点参考 `examples/custom_adapter` 示例。

贡献指南
- 使用 `gofmt` / `go vet` 保持代码风格一致
- 新增功能请补充对应单元测试，目标覆盖率不低于 50%
- 提交信息遵循 Angular commit message 规范（英文）

许可
本项目遵循 [MIT](LICENSE) 开源许可协议。