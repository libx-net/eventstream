package eventstream

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// EventBus 事件总线接口
type EventBus interface {
	// Emit 发布事件
	Emit(ctx context.Context, topic string, data interface{}) error

	// On 订阅事件
	On(topic string, group string, handler EventHandler, opts ...SubscribeOption) (Subscription, error)

	// Off 取消订阅
	Off(subscription Subscription) error

	// Close 优雅关闭
	Close() error
}

// EventHandler 事件处理函数
type EventHandler func(ctx context.Context, event *Event) error

// Subscription 订阅句柄
type Subscription interface {
	// Topic 获取订阅的主题
	Topic() string

	// ID 获取订阅ID
	ID() string

	// IsActive 检查订阅是否活跃
	IsActive() bool
}

// SubscribeOption 订阅选项
type SubscribeOption func(*SubscribeConfig)

// PoolSubmitter 协程池提交接口
type PoolSubmitter interface {
	Submit(task func()) error
}

// MemoryEventBus 内存模式事件总线接口
type MemoryEventBus interface {
	EventBus
	GetStats() map[string]interface{}
	GetHistory(limit int) []*Event
}

// New 创建EventBus实例
func New(config *Config) (EventBus, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	switch config.Mode {
	case ModeMemory:
		return newMemoryEventBus(config)
	case ModeDistributed:
		if config.Distributed == nil || config.Distributed.MQAdapter == nil {
			return nil, errors.New("MQAdapter required for distributed mode")
		}
		return newDistributedEventBus(config)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", config.Mode)
	}
}

// WithConcurrency 设置并发处理数量
func WithConcurrency(concurrency int) SubscribeOption {
	return func(config *SubscribeConfig) {
		config.Concurrency = concurrency
	}
}

// WithRetryPolicy 设置重试策略
func WithRetryPolicy(policy *RetryPolicy) SubscribeOption {
	return func(config *SubscribeConfig) {
		config.RetryPolicy = policy
	}
}

// WithAutoCommit 设置是否自动提交offset
func WithAutoCommit(autoCommit bool) SubscribeOption {
	return func(config *SubscribeConfig) {
		config.AutoCommit = autoCommit
	}
}

// WithStartOffset 设置起始偏移量
func WithStartOffset(offset string) SubscribeOption {
	return func(config *SubscribeConfig) {
		config.StartOffset = offset
	}
}

// WithCommitInterval 设置提交间隔
func WithCommitInterval(interval time.Duration) SubscribeOption {
	return func(config *SubscribeConfig) {
		config.CommitInterval = interval
	}
}

// WithBufferSize 设置缓冲区大小
func WithBufferSize(size int) SubscribeOption {
	return func(config *SubscribeConfig) {
		config.BufferSize = size
	}
}

// WithDeadLetterQueue 启用死信队列
func WithDeadLetterQueue(topic string) SubscribeOption {
	return func(config *SubscribeConfig) {
		config.EnableDeadLetterQueue = true
		config.DeadLetterTopic = topic
	}
}

// newMemoryEventBus 创建内存模式事件总线
func newMemoryEventBus(config *Config) (EventBus, error) {
	return newMemoryEventBusImpl(config)
}

// newDistributedEventBus 创建分布式模式事件总线
func newDistributedEventBus(config *Config) (EventBus, error) {
	return NewDistributedEventBus(config)
}
