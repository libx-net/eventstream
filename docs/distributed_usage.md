# 分布式事件总线使用指南

## 概述

分布式事件总线基于一个可插拔的适配器模型，允许使用任何消息队列系统作为底层传输层。您只需要实现 `adapter.MQAdapter` 接口，就可以将事件总线与您选择的消息队列集成。

## 核心接口 (`adapter` 包)

所有适配器都必须实现 `adapter.MQAdapter` 接口。

```go
package adapter

// MQAdapter 定义了与消息队列交互所需的方法。
type MQAdapter interface {
    Publish(ctx context.Context, topic string, message []byte) error
    Subscribe(ctx context.Context, topic, groupID string) (<-chan Message, func(), error)
    Ack(ctx context.Context, groupID string, msg Message) error // 确认消息
    Close() error
}

// Message 代表从队列中读取的单条消息。
type Message interface {
    // ... 方法定义
}
```

## 快速开始

### 1. 实现你自己的 MQAdapter

```go
package mymq

import "libx.net/eventstream/adapter"

type MyMQAdapter struct {
    // ... 你的 MQ 客户端
}

// 实现 adapter.MQAdapter 的所有方法...
func (a *MyMQAdapter) Publish(...) error { /* ... */ }
func (a *MyMQAdapter) Subscribe(...) (<-chan adapter.Message, func(), error) { /* ... */ }
func (a *MyMQAdapter) Ack(...) error { /* ... */ }
func (a *MyMQAdapter) Close() error { /* ... */ }
```

### 2. 配置并创建事件总线

```go
import (
    "libx.net/eventstream"
    "path/to/your/mymq"
)

func main() {
    // 1. 创建你的适配器实例
    myAdapter := &mymq.MyMQAdapter{}

    // 2. 创建一个分布式配置
    config := eventstream.DefaultDistributedConfig()
    config.Distributed.MQAdapter = myAdapter

    // 3. 创建事件总线实例
    eventBus, err := eventstream.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer eventBus.Close()

    // 4. 开始发布和订阅事件
    // ...
}
```

## 内置 Kafka 适配器 (`kafka` 包)

本项目提供了一个官方的 Kafka 适配器作为开箱即用的实现。

### 使用方法

```go
import (
    "libx.net/eventstream"
    "libx.net/eventstream/kafka"
)

// 1. 创建 Kafka 适配器的配置
kafkaConfig := kafka.Config{
    Brokers: []string{"localhost:9092"},
    // ... 其他生产者和消费者配置
}

// 2. 创建 KafkaAdapter 实例
kafkaAdapter, err := kafka.NewAdapter(kafkaConfig)
if err != nil {
    log.Fatal(err)
}

// 3. 将适配器传入事件总线配置
busConfig := eventstream.DefaultDistributedConfig()
busConfig.Distributed.MQAdapter = kafkaAdapter

// 4. 创建事件总线
eventBus, err := eventstream.New(busConfig)
// ...
```

## 总结

通过将接口（`adapter`）与实现（`kafka` 或您自己的适配器）分离，现在的架构更加清晰、灵活和可扩展。内部实现细节被妥善地封装在 `internal/distributed` 包中。