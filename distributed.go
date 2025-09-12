package eventstream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/panjf2000/ants/v2"
)

// DistributedEventBus 分布式模式事件总线实现
type DistributedEventBus struct {
	config  *Config
	adapter MQAdapter
	pool    *ants.Pool
	subMgr  *distributedSubscriberManager
	metrics *distributedEventMetrics
	closed  int32
	closeCh chan struct{}
}

// NewDistributedEventBus 创建分布式模式事件总线
func NewDistributedEventBus(config *Config) (*DistributedEventBus, error) {
	if config.Distributed == nil || config.Distributed.MQAdapter == nil {
		return nil, fmt.Errorf("MQAdapter is required for distributed mode")
	}

	// 创建协程池
	poolOptions := []ants.Option{
		ants.WithExpiryDuration(config.Pool.ExpiryDuration),
		ants.WithPreAlloc(config.Pool.PreAlloc),
	}

	pool, err := ants.NewPool(config.Pool.Size, poolOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	eb := &DistributedEventBus{
		config:  config,
		adapter: config.Distributed.MQAdapter,
		pool:    pool,
		subMgr:  newDistributedSubscriberManager(),
		closeCh: make(chan struct{}),
	}

	// 初始化指标收集
	if config.Distributed.EnableMetrics {
		eb.metrics = newDistributedEventMetrics()
	}

	return eb, nil
}

// Emit 发布事件
func (eb *DistributedEventBus) Emit(ctx context.Context, event *Event) error {
	if eb.isClosed() {
		return fmt.Errorf("eventbus is closed")
	}

	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// 验证主题
	if err := validateTopic(event.Topic); err != nil {
		return err
	}

	// 序列化事件
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// 发布到消息队列
	if err := eb.adapter.Publish(ctx, event.Topic, eventData); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	// 更新指标
	if eb.metrics != nil {
		eb.metrics.incEmitted(event.Topic)
	}

	return nil
}

// On 订阅事件
func (eb *DistributedEventBus) On(topic string, group string, handler EventHandler, opts ...SubscribeOption) (Subscription, error) {
	if eb.isClosed() {
		return nil, fmt.Errorf("eventbus is closed")
	}

	// 验证参数
	if err := validateTopic(topic); err != nil {
		return nil, err
	}
	if group == "" {
		return nil, fmt.Errorf("consumer group cannot be empty")
	}
	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	// 合并订阅选项
	config := mergeSubscribeOptions(opts)
	config.ConsumerGroup = group

	// 创建订阅
	sub := newDistributedSubscription(topic, handler, config, eb.adapter, eb.pool)

	// 启动订阅处理
	ctx := context.Background()
	if err := sub.start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start subscription: %w", err)
	}

	// 添加到管理器
	eb.subMgr.add(sub)

	// 更新指标
	if eb.metrics != nil {
		eb.metrics.incSubscribed(topic)
	}

	return sub, nil
}

// Off 取消订阅
func (eb *DistributedEventBus) Off(subscription Subscription) error {
	if eb.isClosed() {
		return fmt.Errorf("eventbus is closed")
	}

	sub, ok := subscription.(*distributedSubscription)
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
func (eb *DistributedEventBus) Close() error {
	if !atomic.CompareAndSwapInt32(&eb.closed, 0, 1) {
		return nil // 已经关闭
	}

	close(eb.closeCh)

	// 停止所有订阅
	subscribers := eb.subMgr.getAllSubscribers()
	for _, sub := range subscribers {
		sub.stop()
	}

	// 关闭适配器
	if err := eb.adapter.Close(); err != nil {
		fmt.Printf("Error closing adapter: %v\n", err)
	}

	// 关闭协程池
	eb.pool.Release()

	return nil
}

// isClosed 检查是否已关闭
func (eb *DistributedEventBus) isClosed() bool {
	return atomic.LoadInt32(&eb.closed) == 1
}

// distributedSubscriberManager 分布式订阅者管理器
type distributedSubscriberManager struct {
	subscribers map[string][]*distributedSubscription
	mu          sync.RWMutex
}

func newDistributedSubscriberManager() *distributedSubscriberManager {
	return &distributedSubscriberManager{
		subscribers: make(map[string][]*distributedSubscription),
	}
}

func (sm *distributedSubscriberManager) add(sub *distributedSubscription) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.subscribers[sub.topic] = append(sm.subscribers[sub.topic], sub)
}

func (sm *distributedSubscriberManager) remove(sub *distributedSubscription) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	subs := sm.subscribers[sub.topic]
	for i, s := range subs {
		if s.id == sub.id {
			sm.subscribers[sub.topic] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	if len(sm.subscribers[sub.topic]) == 0 {
		delete(sm.subscribers, sub.topic)
	}
}

func (sm *distributedSubscriberManager) getAllSubscribers() []*distributedSubscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var all []*distributedSubscription
	for _, subs := range sm.subscribers {
		all = append(all, subs...)
	}
	return all
}

func (sm *distributedSubscriberManager) count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, subs := range sm.subscribers {
		count += len(subs)
	}
	return count
}

// distributedSubscription 分布式订阅实现
type distributedSubscription struct {
	id        string
	topic     string
	handler   EventHandler
	config    *SubscribeConfig
	adapter   MQAdapter
	pool      *ants.Pool
	msgChan   <-chan Message
	closeFunc func()
	closed    int32
}

func newDistributedSubscription(topic string, handler EventHandler, config *SubscribeConfig, adapter MQAdapter, pool *ants.Pool) *distributedSubscription {
	return &distributedSubscription{
		id:      generateSubscriptionID(),
		topic:   topic,
		handler: handler,
		config:  config,
		adapter: adapter,
		pool:    pool,
	}
}

func (s *distributedSubscription) start(ctx context.Context) error {
	msgChan, closeFunc, err := s.adapter.Subscribe(ctx, s.topic, s.config.ConsumerGroup)
	if err != nil {
		return err
	}

	s.msgChan = msgChan
	s.closeFunc = closeFunc

	// 启动消息处理协程
	go s.processMessages(ctx)

	return nil
}

func (s *distributedSubscription) stop() {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return
	}

	if s.closeFunc != nil {
		s.closeFunc()
	}
}

func (s *distributedSubscription) processMessages(ctx context.Context) {
	for msg := range s.msgChan {
		if s.isClosed() {
			break
		}

		// 反序列化事件
		var event Event
		if err := json.Unmarshal(msg.Value(), &event); err != nil {
			fmt.Printf("Failed to deserialize event: %v\n", err)
			continue
		}

		// 异步处理事件
		if err := s.pool.Submit(func() {
			s.handleEvent(ctx, &event, msg)
		}); err != nil {
			fmt.Printf("Failed to submit event processing task: %v\n", err)
		}
	}
}

func (s *distributedSubscription) handleEvent(ctx context.Context, event *Event, msg Message) {
	// 处理事件
	err := s.handler(ctx, event)

	// 如果启用自动提交，确认消息
	if s.config.AutoCommit && err == nil {
		if ackErr := s.adapter.Ack(ctx, s.config.ConsumerGroup, msg); ackErr != nil {
			fmt.Printf("Failed to ack message: %v\n", ackErr)
		}
	}
}

func (s *distributedSubscription) Topic() string {
	return s.topic
}

func (s *distributedSubscription) ID() string {
	return s.id
}

func (s *distributedSubscription) IsActive() bool {
	return !s.isClosed()
}

func (s *distributedSubscription) isClosed() bool {
	return atomic.LoadInt32(&s.closed) == 1
}

// distributedEventMetrics 分布式事件指标
type distributedEventMetrics struct {
	emitted    map[string]int64
	subscribed map[string]int64
	mu         sync.RWMutex
}

func newDistributedEventMetrics() *distributedEventMetrics {
	return &distributedEventMetrics{
		emitted:    make(map[string]int64),
		subscribed: make(map[string]int64),
	}
}

func (em *distributedEventMetrics) incEmitted(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.emitted[topic]++
}

func (em *distributedEventMetrics) incSubscribed(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.subscribed[topic]++
}

func (em *distributedEventMetrics) decSubscribed(topic string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	if em.subscribed[topic] > 0 {
		em.subscribed[topic]--
	}
}

func (em *distributedEventMetrics) getStats() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return map[string]interface{}{
		"emitted":    em.copyMap(em.emitted),
		"subscribed": em.copyMap(em.subscribed),
	}
}

func (em *distributedEventMetrics) copyMap(m map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
