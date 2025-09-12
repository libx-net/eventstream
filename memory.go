package eventstream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/panjf2000/ants/v2"
)

// memoryEventBus 内存模式事件总线实现
type memoryEventBus struct {
	config  *Config
	pool    *ants.Pool
	subMgr  *subscriberManager
	history *eventHistory
	metrics *eventMetrics
	closed  int32
	closeCh chan struct{}
}

// newMemoryEventBusImpl 创建内存模式事件总线实现
func newMemoryEventBusImpl(config *Config) (EventBus, error) {
	// 创建协程池
	poolOptions := []ants.Option{
		ants.WithExpiryDuration(config.Pool.ExpiryDuration),
		ants.WithPreAlloc(config.Pool.PreAlloc),
	}

	pool, err := ants.NewPool(config.Pool.Size, poolOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	eb := &memoryEventBus{
		config:  config,
		pool:    pool,
		subMgr:  newSubscriberManager(),
		closeCh: make(chan struct{}),
	}

	// 初始化历史记录
	if config.Memory != nil && config.Memory.EnableHistory {
		eb.history = newEventHistory(config.Memory.MaxHistorySize)
	}

	// 初始化指标收集
	if config.Memory != nil && config.Memory.EnableMetrics {
		eb.metrics = newEventMetrics()
	}

	return eb, nil
}

// Emit 发布事件
func (eb *memoryEventBus) Emit(ctx context.Context, topic string, data interface{}) error {
	if eb.isClosed() {
		return fmt.Errorf("eventbus is closed")
	}

	// 验证主题
	if err := validateTopic(topic); err != nil {
		return err
	}

	// 创建事件
	event := NewEvent(topic, data)

	// 添加到历史记录
	if eb.history != nil {
		eb.history.add(event)
	}

	// 更新指标
	if eb.metrics != nil {
		eb.metrics.incEmitted(topic)
	}

	// 获取订阅者
	subscribers := eb.subMgr.getSubscribers(topic)
	if len(subscribers) == 0 {
		return nil // 没有订阅者
	}

	// 异步投递事件到所有订阅者
	if err := eb.pool.Submit(func() {
		eb.deliverEvent(event, subscribers)
	}); err != nil {
		return fmt.Errorf("failed to submit event delivery task: %w", err)
	}
	return nil
}

// On 订阅事件
func (eb *memoryEventBus) On(topic string, group string, handler EventHandler, opts ...SubscribeOption) (Subscription, error) {
	if eb.isClosed() {
		return nil, fmt.Errorf("eventbus is closed")
	}

	// 验证参数
	if err := validateTopic(topic); err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	// 合并订阅选项
	config := mergeSubscribeOptions(opts)

	// 创建订阅
	sub := newMemorySubscription(topic, handler, config)

	// 启动订阅处理
	ctx := context.Background() // 使用背景上下文
	sub.start(ctx, eb.pool)

	// 添加到管理器
	eb.subMgr.add(sub)

	// 更新指标
	if eb.metrics != nil {
		eb.metrics.incSubscribed(topic)
	}

	return sub, nil
}

// Off 取消订阅
func (eb *memoryEventBus) Off(subscription Subscription) error {
	if eb.isClosed() {
		return fmt.Errorf("eventbus is closed")
	}

	sub, ok := subscription.(*memorySubscription)
	if !ok {
		return fmt.Errorf("invalid subscription type")
	}

	// 停止订阅
	sub.stop()

	// 从管理器中移除
	eb.subMgr.remove(sub)

	// 更新指标
	if eb.metrics != nil {
		eb.metrics.decSubscribed(sub.topic)
	}

	return nil
}

// Close 优雅关闭
func (eb *memoryEventBus) Close() error {
	if !atomic.CompareAndSwapInt32(&eb.closed, 0, 1) {
		return nil // 已经关闭
	}

	close(eb.closeCh)

	// 停止所有订阅
	subscribers := eb.subMgr.getAllSubscribers()
	for _, sub := range subscribers {
		sub.stop()
	}

	// 关闭协程池
	eb.pool.Release()

	return nil
}

// GetStats 获取统计信息
func (eb *memoryEventBus) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"mode":              "memory",
		"pool_running":      eb.pool.Running(),
		"pool_free":         eb.pool.Free(),
		"pool_capacity":     eb.pool.Cap(),
		"subscribers_count": eb.subMgr.count(),
		"closed":            eb.isClosed(),
	}

	if eb.metrics != nil {
		stats["metrics"] = eb.metrics.getStats()
	}

	if eb.history != nil {
		stats["history_count"] = eb.history.count()
	}

	return stats
}

// GetHistory 获取事件历史
func (eb *memoryEventBus) GetHistory(limit int) []*Event {
	if eb.history == nil {
		return nil
	}
	return eb.history.get(limit)
}

// isClosed 检查是否已关闭
func (eb *memoryEventBus) isClosed() bool {
	return atomic.LoadInt32(&eb.closed) == 1
}

// deliverEvent 投递事件到订阅者
func (eb *memoryEventBus) deliverEvent(event *Event, subscribers []*memorySubscription) {
	delivered := 0
	for _, sub := range subscribers {
		if sub.deliver(event) {
			delivered++
		}
	}

	// 更新指标
	if eb.metrics != nil {
		eb.metrics.incDelivered(event.Topic, delivered)
	}
}
