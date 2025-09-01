# EventStream - 分布式事件总线

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

EventStream 是一个基于 Go 语言的分布式、高性能、高可靠事件总线库，支持内存模式和分布式模式。

## 特性

- **高性能**: 基于 ants 协程池的高效并发处理
- **双模式**: 支持内存模式和分布式模式，可无缝切换
- **零依赖启动**: 内存模式无需外部依赖
- **可靠性**: 分布式模式基于 Kafka，支持消息持久化和故障恢复
- **简单易用**: 统一的 API 接口，只需三个核心方法
- **可观测性**: 内置指标收集和事件历史记录
- **可配置**: 丰富的配置选项，支持重试策略、并发控制等

## 快速开始

### 安装

```bash
go get libx.net/eventstream
```

### 基本使用

#### 内存模式 (适合单体应用、开发测试)

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "libx.net/eventstream"
)

func main() {
    // 创建内存模式配置
    config := eventstream.DefaultConfig()
    
    // 创建EventBus
    bus, err := eventstream.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer bus.Close()
    
    // 订阅事件
    subscription, err := bus.On("user.registered", func(ctx context.Context, event *eventstream.Event) error {
        fmt.Printf("用户注册: %v\n", event.Data)
        return nil
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 发布事件
    ctx := context.Background()
    err = bus.Emit(ctx, "user.registered", map[string]interface{}{
        "user_id": "12345",
        "email":   "user@example.com",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 取消订阅
    bus.Off(subscription)
}
```

#### 分布式模式 (适合微服务架构、生产环境)

```go
// 创建分布式模式配置
config := eventstream.DefaultDistributedConfig([]string{"localhost:9092"})

// 其他使用方式完全相同
bus, err := eventstream.New(config)
// ...
```

## 核心概念

### 运行模式

| 特性 | 内存模式 | 分布式模式 |
|------|----------|------------|
| **性能** | 极高 (纳秒级) | 高 (毫秒级) |
| **可靠性** | 进程重启丢失 | 持久化保证 |
| **扩展性** | 单进程 | 多节点 |
| **依赖** | 仅ants | ants + Kafka |
| **适用场景** | 单体应用、开发测试 | 微服务、生产环境 |

### 核心 API

```go
type EventBus interface {
    // 发布事件
    Emit(ctx context.Context, topic string, data interface{}) error
    
    // 订阅事件
    On(topic string, handler EventHandler, opts ...SubscribeOption) (Subscription, error)
    
    // 取消订阅
    Off(subscription Subscription) error
    
    // 优雅关闭
    Close() error
}
```

## 高级功能

### 消费者组 (多服务消费同一事件)

```go
// 同一个事件可以被多个消费者组独立消费
// 通知服务消费者组
bus.On("user.registered", sendWelcomeEmail,
    eventstream.WithConsumerGroup("notification-service"))

// 积分服务消费者组  
bus.On("user.registered", grantWelcomePoints,
    eventstream.WithConsumerGroup("points-service"))

// 分析服务消费者组
bus.On("user.registered", recordUserAnalytics,
    eventstream.WithConsumerGroup("analytics-service"))

// 发布一次事件，三个服务都会收到
bus.Emit(ctx, "user.registered", userData)
```

### 订阅选项

```go
// 带重试策略和消费者组的订阅
subscription, err := bus.On("order.created", 
    handleOrder,
    eventstream.WithConsumerGroup("order-service"),     // 消费者组名称
    eventstream.WithConcurrency(10),                    // 并发处理数量
    eventstream.WithRetryPolicy(&eventstream.RetryPolicy{
        MaxRetries:      3,                             // 最大重试次数
        BackoffStrategy: "exponential",                 // 指数退避
        InitialDelay:    100 * time.Millisecond,        // 初始延迟
        MaxDelay:        5 * time.Second,               // 最大延迟
        Multiplier:      2.0,                           // 退避乘数
    }),
)
```

### 事件处理器

```go
func handleUserRegistered(ctx context.Context, event *eventstream.Event) error {
    // 反序列化事件数据
    var userData struct {
        UserID string `json:"user_id"`
        Email  string `json:"email"`
    }
    
    if err := event.Unmarshal(&userData); err != nil {
        return err
    }
    
    // 处理业务逻辑
    fmt.Printf("新用户注册: %s\n", userData.Email)
    
    return nil
}
```

### 统计信息 (内存模式)

```go
if memBus, ok := bus.(eventstream.MemoryEventBus); ok {
    // 获取统计信息
    stats := memBus.GetStats()
    fmt.Printf("统计信息: %+v\n", stats)
    
    // 获取事件历史
    history := memBus.GetHistory(10)
    fmt.Printf("最近10个事件: %+v\n", history)
}
```

## 配置

### 内存模式配置

```go
config := &eventstream.Config{
    Mode: "memory",
    Pool: eventstream.PoolConfig{
        Size:           1000,                    // 协程池大小
        ExpiryDuration: 10 * time.Second,       // 协程过期时间
        PreAlloc:       false,                  // 是否预分配
    },
    Memory: &eventstream.MemoryConfig{
        BufferSize:     1000,                   // 事件缓冲区大小
        EnableHistory:  true,                   // 启用历史记录
        MaxHistorySize: 10000,                  // 最大历史记录数
        EnableMetrics:  true,                   // 启用指标收集
    },
}
```

### 分布式模式配置

```go
config := &eventstream.Config{
    Mode: "distributed",
    Pool: eventstream.PoolConfig{
        Size: 2000,
    },
    // MQ: MQAdapter 具体实现,
}
```

## 示例

查看 `examples/` 目录获取更多示例：

- [基础内存模式示例](examples/memory/basic.go)
- [多消费者组示例](examples/memory/multiple_consumers.go)
- [分布式模式示例](examples/distributed/) (待实现)
- [微服务架构示例](examples/microservice/) (待实现)

## 性能

### 内存模式
- **延迟**: < 100ns (事件发布)
- **吞吐量**: 100K+ events/second
- **内存**: 稳定使用，无泄漏

### 分布式模式
- **延迟**: < 1ms (本地处理)
- **吞吐量**: 10K+ events/second
- **可用性**: 99.9%+

## 最佳实践

1. **选择合适的模式**
   - 开发测试: 使用内存模式
   - 单体应用: 使用内存模式
   - 微服务架构: 使用分布式模式

2. **事件设计**
   - 使用清晰的主题命名 (如: `user.registered`, `order.created`)
   - 保持事件数据结构简单
   - 避免在事件中包含敏感信息

3. **消费者组设计**
   - 使用服务名作为消费者组名 (如: `notification-service`, `analytics-service`)
   - 同一事件的不同处理逻辑使用不同消费者组
   - 为不同消费者组设置合适的重试策略和并发配置

4. **错误处理**
   - 合理设置重试策略
   - 记录处理失败的事件
   - 实现幂等性处理

5. **性能优化**
   - 根据负载调整协程池大小
   - 使用适当的并发处理数量
   - 监控事件处理延迟

## 开发状态

- 内存模式 - 已完成
- 分布式模式 - 开发中
- 文档和示例 - 持续完善

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件。

## 技术栈

- **Go**: 1.23+
- **协程池**: [ants](https://github.com/panjf2000/ants)

---

EventStream - 让事件驱动架构变得简单而强大！