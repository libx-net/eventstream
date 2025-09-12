# 分布式模式使用说明

## 概述

分布式模式允许将外部消息队列（MQ）作为事件传递的底层载体，从而实现跨进程或跨主机的事件流转。库通过 MQAdapter 接口抽象具体 MQ 实现，用户可注入自定义适配器（例如 Kafka、RabbitMQ 等）。

> 注意：本项目仍在开发中，API 可能会有调整。示例供参考，请在生产环境使用前评估稳定性并充分测试。

---

## 核心概念

- EventBus：事件总线，负责发布与订阅事件。
- MQAdapter：外部消息队列适配器，负责将事件发布到 MQ 以及从 MQ 订阅消息。
- Subscription（订阅）：通过事件主题（topic）接收消息的注册句柄，可指定消费者组、并发数、重试策略等。
- Serializer：事件序列化/反序列化器（默认 JSON）。

---

## MQAdapter 接口（说明）

MQAdapter 为外部 MQ 的抽象契约，需实现下列方法（伪签名）：

- Publish(ctx context.Context, topic string, message []byte) error  
  将消息发布到指定主题。

- Subscribe(ctx context.Context, topic, groupID string) (<-chan Message, func(), error)  
  订阅指定主题并返回消息通道与取消订阅的回调函数。

- Ack(ctx context.Context, groupID string, msg Message) error  
  确认消息（可选用于需要手动 Ack 的 MQ）。

- Close() error  
  关闭适配器，释放资源。

Message 接口代表 MQ 的消息，需提供至少以下方法：Topic(), Value(), Timestamp(), Key()（具体可根据项目实现调整）。

---

## 配置要点（DistributedConfig）

在构造 DistributedConfig 时，必须提供有效的 MQAdapter。常见字段：

- Distributed.MQAdapter：MQAdapter 实例（必须）
- Distributed.Serializer：序列化实现（可选，默认 JSON）
- Pool / 性能相关参数：并发池大小等
- Subscribe 默认选项：可通过订阅时覆盖

示例（伪代码）：
```go
cfg := eventstream.DefaultDistributedConfig()
cfg.Distributed.MQAdapter = myAdapter       // 必须
cfg.Distributed.Serializer = mySerializer   // 可选，默认 JSON
```

---

## 创建分布式 EventBus

示例：
```go
cfg := eventstream.DefaultDistributedConfig()
cfg.Distributed.MQAdapter = myAdapter

bus, err := eventstream.New(cfg)
if err != nil {
    // 处理错误
}
defer bus.Close()
```

New 会根据配置创建分布式的 EventBus 实例。

---

## 订阅与发布

订阅示例：
```go
sub, err := bus.On(
    "topic.name",
    "group-1", // 消费组现在是必需参数
    func(ctx context.Context, e *eventstream.Event) error {
        // 处理消息
        var payload MyStruct
        if err := e.Unmarshal(&payload); err != nil {
            return err
        }
        // 业务处理
        return nil
    },
    eventstream.WithConcurrency(4),
    eventstream.WithRetryPolicy(&eventstream.RetryPolicy{
        MaxRetries:      3,
        BackoffStrategy: eventstream.BackoffExponential,
        InitialDelay:    100 * time.Millisecond,
        MaxDelay:        2 * time.Second,
        Multiplier:      2.0,
    }),
)
if err != nil {
    // 处理错误
}
defer bus.Off(sub)
```

发布示例：
```go
err := bus.Emit(context.Background(), "topic.name", payload)
if err != nil {
    // 处理错误
}
```

说明：
- `On` 方法现在要求提供一个消费组（`group`）作为必需参数。
- `On` 方法返回的 `Subscription` 可用于取消订阅（`bus.Off(sub)`）。
- 订阅时可通过选项指定并发数、缓冲大小与重试策略等。
- 回调签名为 `func(ctx context.Context, e *eventstream.Event) error`。

---

## 事件数据与序列化

- Event.Data 字段可能为序列化后的字节数组、字符串或已通过 Serializer 反序列化的对象，具体类型取决于所用 Serializer 与 Adapter。
- 推荐使用 Event.Unmarshal(dst) 将事件解码到目标结构，避免手动类型断言。

示例：
```go
var p MyPayload
if err := e.Unmarshal(&p); err != nil {
    return err
}
```

---

## 内置 Kafka 适配器（使用说明）

仓库包含一个 Kafka 适配器示例，可直接参考或使用。使用步骤：

1. 构建 Kafka 配置并创建 Adapter：
```go
kCfg := kafka.Config{
    Brokers: []string{"localhost:9092"},
    // 其它配置项（压缩、分区、offset 等）
}
kAdapter, err := kafka.NewAdapter(kCfg)
if err != nil {
    // 处理错误
}
defer kAdapter.Close()
```

2. 将 Adapter 注入 DistributedConfig 并创建 EventBus：
```go
cfg := eventstream.DefaultDistributedConfig()
cfg.Distributed.MQAdapter = kAdapter
bus, _ := eventstream.New(cfg)
defer bus.Close()
```

注意：
- Kafka Adapter 会在内部对消息做序列化，确保发布与订阅端使用相同的 Serializer。
- Kafka 的消息 Key/Timestamp 等字段可由 Adapter 映射到 Message 接口。

---

## 示例：自定义简单 MQAdapter（伪代码）

以下为简化示例，供参考，不适用于生产环境：
```go
type SimpleMQAdapter struct {
    // 内部消息通道等
}

func (s *SimpleMQAdapter) Publish(ctx context.Context, topic string, message []byte) error {
    // 将 message 推到内部通道或外部 MQ
    return nil
}

func (s *SimpleMQAdapter) Subscribe(ctx context.Context, topic, groupID string) (<-chan eventstream.Message, func(), error) {
    ch := make(chan eventstream.Message)
    // 启动 goroutine 从底层 MQ 转换为 eventstream.Message 并发送到 ch
    return ch, func() { /* 取消订阅并关闭 ch */ }, nil
}

func (s *SimpleMQAdapter) Ack(ctx context.Context, groupID string, msg eventstream.Message) error {
    // 可选实现
    return nil
}

func (s *SimpleMQAdapter) Close() error {
    // 关闭资源
    return nil
}
```

---

## 注意事项与最佳实践

- 订阅回调应尽量幂等，方便重试与重复投递处理。
- 确保 MQAdapter 在 Close 时正确关闭通道和后台 goroutine，避免泄漏。
- 在高并发场景下，请根据负载调整并发数、缓冲区与池大小。
- 对于关键业务，请在生产前充分测试序列化格式、重试策略与消息确认行为。
- 若使用 Kafka 等有 exactly-once 或 at-least-once 要求的 MQ，请结合业务设计合适的幂等或去重策略。

---

## 故障排查

- 示例运行中出现类型断言错误（例如将 string 断言为 []byte），说明发布端和订阅端序列化格式不一致；检查使用的 Serializer 与实际传递的数据类型。
- 若出现 goroutine 泄漏，多半是 Subscribe 返回的通道或取消函数未正确关闭/调用。
- 若消息丢失或重复，检查 MQAdapter 的 Ack/提交策略与序列化一致性。

---

## 参考示例

见仓库 examples/distributed_basic、examples/memory_basic 和 examples/memory_multiple，分别演示分布式模式与内存模式的典型用法。

--- 

如需我将本文件中某个代码片段改为可直接复制运行的完整示例或添加 Kafka 配置模板，请告诉我具体需求。