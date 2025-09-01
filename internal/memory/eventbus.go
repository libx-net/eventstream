package memory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/panjf2000/ants/v2"
	"libx.net/eventstream"
)

// EventBus 内存模式事件总线实现
type EventBus struct {
	config  *eventstream.Config
	pool    *ants.Pool
	subMgr  *subscriberManager
	history *eventHistory
	metrics *eventMetrics
	closed  int32
	closeCh chan struct{}
	mu      sync.RWMutex
}

// NewEventBus 创建内存模式事件总线
func NewEventBus(config *eventstream.Config) (*EventBus, error) {
	// 创建协程池
	poolOptions := []ants.Option{
		ants.WithExpiryDuration(config.Pool.ExpiryDuration),
		ants.WithPreAlloc(config.Pool.PreAlloc),
	}

	pool, err := ants.NewPool(config.Pool.Size, poolOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	eb := &EventBus{
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
func (eb *EventBus) Emit(ctx context.Context, topic string, data interface{}) error {
	if eb.isClosed() {
		return fmt.Errorf("eventbus is closed")
	}

	// 验证主题
	if err := eb.validateTopic(topic); err != nil {
		return err
	}

	// 创建事件
	event := eventstream.NewEvent(topic, data)

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
	return eb.pool.Submit(func() {
		eb.deliverEvent(event, subscribers)
	})
}

// On 订阅事件
func (eb *EventBus) On(topic string, handler eventstream.EventHandler, opts ...eventstream.SubscribeOption) (eventstream.Subscription, error) {
	if eb.isClosed() {
		return nil, fmt.Errorf("eventbus is closed")
	}

	// 验证参数
	if err := eb.validateTopic(topic); err != nil {
		return nil, err
	}
	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	// 合并订阅选项
	config := eb.mergeSubscribeOptions(opts)

	// 创建订阅
	sub := newSubscription(topic, handler, config)

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
func (eb *EventBus) Off(sub eventstream.Subscription) error {
	if eb.isClosed() {
		return fmt.Errorf("eventbus is closed")
	}

	_sub, ok := sub.(*subscription)
	if !ok {
		return fmt.Errorf("invalid subscription type")
	}

	// 停止订阅
	_sub.stop()

	// 从管理器中移除
	eb.subMgr.remove(_sub)

	// 更新指标
	if eb.metrics != nil {
		eb.metrics.decSubscribed(_sub.topic)
	}

	return nil
}

// Close 优雅关闭
func (eb *EventBus) Close() error {
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

// isClosed 检查是否已关闭
func (eb *EventBus) isClosed() bool {
	return atomic.LoadInt32(&eb.closed) == 1
}

// validateTopic 验证主题
func (eb *EventBus) validateTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	if len(topic) > 255 {
		return fmt.Errorf("topic name too long (max 255 characters)")
	}
	return nil
}

// mergeSubscribeOptions 合并订阅选项
func (eb *EventBus) mergeSubscribeOptions(opts []eventstream.SubscribeOption) *eventstream.SubscribeConfig {
	config := eventstream.DefaultSubscribeConfig()

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// deliverEvent 投递事件到订阅者
func (eb *EventBus) deliverEvent(event *eventstream.Event, subscribers []*subscription) {
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

// GetStats 获取统计信息
func (eb *EventBus) GetStats() map[string]interface{} {
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
func (eb *EventBus) GetHistory(limit int) []*eventstream.Event {
	if eb.history == nil {
		return nil
	}
	return eb.history.get(limit)
}

// eventHistory 事件历史记录
type eventHistory struct {
	events  []*eventstream.Event
	maxSize int
	mu      sync.RWMutex
}

// newEventHistory 创建事件历史记录
func newEventHistory(maxSize int) *eventHistory {
	return &eventHistory{
		events:  make([]*eventstream.Event, 0, maxSize),
		maxSize: maxSize,
	}
}

// add 添加事件到历史记录
func (eh *eventHistory) add(event *eventstream.Event) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	// 如果达到最大容量，移除最旧的事件
	if len(eh.events) >= eh.maxSize {
		eh.events = eh.events[1:]
	}

	eh.events = append(eh.events, event)
}

// get 获取历史事件
func (eh *eventHistory) get(limit int) []*eventstream.Event {
	eh.mu.RLock()
	defer eh.mu.RUnlock()

	if limit <= 0 || limit > len(eh.events) {
		limit = len(eh.events)
	}

	// 返回最新的limit个事件
	start := len(eh.events) - limit
	result := make([]*eventstream.Event, limit)
	copy(result, eh.events[start:])

	return result
}

// count 获取历史事件数量
func (eh *eventHistory) count() int {
	eh.mu.RLock()
	defer eh.mu.RUnlock()
	return len(eh.events)
}

// eventMetrics 事件指标
type eventMetrics struct {
	emitted    map[string]int64
	delivered  map[string]int64
	subscribed map[string]int64
	mu         sync.RWMutex
}

// newEventMetrics 创建事件指标
func newEventMetrics() *eventMetrics {
	return &eventMetrics{
		emitted:    make(map[string]int64),
		delivered:  make(map[string]int64),
		subscribed: make(map[string]int64),
	}
}

// incEmitted 增加发布计数
func (em *eventMetrics) incEmitted(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.emitted[topic]++
}

// incDelivered 增加投递计数
func (em *eventMetrics) incDelivered(topic string, count int) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.delivered[topic] += int64(count)
}

// incSubscribed 增加订阅计数
func (em *eventMetrics) incSubscribed(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.subscribed[topic]++
}

// decSubscribed 减少订阅计数
func (em *eventMetrics) decSubscribed(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	if em.subscribed[topic] > 0 {
		em.subscribed[topic]--
	}
}

// getStats 获取统计信息
func (em *eventMetrics) getStats() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return map[string]interface{}{
		"emitted":    em.copyMap(em.emitted),
		"delivered":  em.copyMap(em.delivered),
		"subscribed": em.copyMap(em.subscribed),
	}
}

// copyMap 复制map
func (em *eventMetrics) copyMap(m map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
