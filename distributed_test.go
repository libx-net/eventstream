package eventstream

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributedEventBus_Creation(t *testing.T) {
	config := DefaultDistributedConfig()
	config.Distributed.MQAdapter = &mockDistributedMQAdapter{}

	bus, err := NewDistributedEventBus(config)
	if err != nil {
		t.Fatalf("Failed to create distributed event bus: %v", err)
	}
	defer func() {
		_ = bus.Close()
	}()

	if bus.config != config {
		t.Error("Config should be set correctly")
	}

	if bus.adapter == nil {
		t.Error("Adapter should be set")
	}

}

func TestDistributedEventBus_CreationErrors(t *testing.T) {
	// 测试没有适配器的情况
	config := DefaultDistributedConfig()
	config.Distributed.MQAdapter = nil

	_, err := NewDistributedEventBus(config)
	if err == nil {
		t.Error("Expected error when MQAdapter is nil")
	}

	// 测试没有Distributed配置的情况
	config2 := DefaultDistributedConfig()
	config2.Distributed = nil

	_, err = NewDistributedEventBus(config2)
	if err == nil {
		t.Error("Expected error when Distributed config is nil")
	}
}

func TestDistributedEventBus_EmitAndSubscribe(t *testing.T) {
	adapter := &mockDistributedMQAdapter{}
	config := DefaultDistributedConfig()
	config.Distributed.MQAdapter = adapter

	bus, err := NewDistributedEventBus(config)
	if err != nil {
		t.Fatalf("Failed to create distributed event bus: %v", err)
	}
	defer func() {
		_ = bus.Close()
	}()

	var receivedEvents []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 订阅事件
	_, err = bus.On("test.distributed", "test-group", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents = append(receivedEvents, event.Data.(string))
		wg.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// 发布事件
	ctx := context.Background()
	testData := []string{"event1", "event2", "event3"}

	wg.Add(len(testData))
	for _, data := range testData {
		err := bus.Emit(ctx, NewEvent("test.distributed", data))
		if err != nil {
			t.Errorf("Failed to emit event: %v", err)
		}
	}

	wg.Wait()

	// 验证结果
	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != len(testData) {
		t.Errorf("Expected %d events, got %d", len(testData), len(receivedEvents))
	}
}

func TestDistributedEventBus_ErrorHandling(t *testing.T) {
	config := DefaultDistributedConfig()
	config.Distributed.MQAdapter = &mockDistributedMQAdapter{}

	bus, err := NewDistributedEventBus(config)
	if err != nil {
		t.Fatalf("Failed to create distributed event bus: %v", err)
	}

	// 测试空主题
	ctx := context.Background()
	err = bus.Emit(ctx, NewEvent("", "data"))
	if err == nil {
		t.Error("Expected error for empty topic")
	}

	// 测试nil handler
	_, err = bus.On("test.topic", "test-group", nil)
	if err == nil {
		t.Error("Expected error for nil handler")
	}

	// 测试空 group
	_, err = bus.On("test.topic", "", func(ctx context.Context, event *Event) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error for empty group")
	}

	// 关闭bus后测试
	_ = bus.Close()

	err = bus.Emit(ctx, NewEvent("test.topic", "data"))
	if err == nil {
		t.Error("Expected error when emitting to closed bus")
	}

	_, err = bus.On("test.topic", "test-group", func(ctx context.Context, event *Event) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error when subscribing to closed bus")
	}
}

func TestDistributedSubscriberManager(t *testing.T) {
	mgr := newDistributedSubscriberManager()

	// 创建测试订阅
	sub1 := &distributedSubscription{id: "sub1", topic: "topic1"}
	sub2 := &distributedSubscription{id: "sub2", topic: "topic1"}
	sub3 := &distributedSubscription{id: "sub3", topic: "topic2"}

	// 添加订阅
	mgr.add(sub1)
	mgr.add(sub2)
	mgr.add(sub3)

	// 检查数量
	if mgr.count() != 3 {
		t.Errorf("Expected 3 subscribers, got %d", mgr.count())
	}

	// 获取所有订阅者
	all := mgr.getAllSubscribers()
	if len(all) != 3 {
		t.Errorf("Expected 3 subscribers, got %d", len(all))
	}

	// 移除订阅
	mgr.remove(sub2)
	if mgr.count() != 2 {
		t.Errorf("Expected 2 subscribers after removal, got %d", mgr.count())
	}

	// 移除topic1的最后一个订阅者
	mgr.remove(sub1)
	if mgr.count() != 1 {
		t.Errorf("Expected 1 subscriber after removal, got %d", mgr.count())
	}
}

func TestDistributedSubscription_Interface(t *testing.T) {
	sub := &distributedSubscription{
		id:    "test-id",
		topic: "test.topic",
	}

	if sub.ID() != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", sub.ID())
	}

	if sub.Topic() != "test.topic" {
		t.Errorf("Expected topic 'test.topic', got '%s'", sub.Topic())
	}

	if !sub.IsActive() {
		t.Error("Subscription should be active initially")
	}

	sub.stop()

	if sub.IsActive() {
		t.Error("Subscription should not be active after stop")
	}
}

func TestDistributedEventMetrics(t *testing.T) {
	metrics := newDistributedEventMetrics()

	// 增加指标
	metrics.incEmitted("topic1")
	metrics.incEmitted("topic1")
	metrics.incEmitted("topic2")

	metrics.incSubscribed("topic1")
	metrics.incSubscribed("topic2")
	metrics.decSubscribed("topic2")

	// 获取统计信息
	stats := metrics.getStats()

	emitted := stats["emitted"].(map[string]int64)
	if emitted["topic1"] != 2 {
		t.Errorf("Expected 2 emitted for topic1, got %d", emitted["topic1"])
	}
	if emitted["topic2"] != 1 {
		t.Errorf("Expected 1 emitted for topic2, got %d", emitted["topic2"])
	}

	subscribed := stats["subscribed"].(map[string]int64)
	if subscribed["topic1"] != 1 {
		t.Errorf("Expected 1 subscribed for topic1, got %d", subscribed["topic1"])
	}
	if subscribed["topic2"] != 0 {
		t.Errorf("Expected 0 subscribed for topic2, got %d", subscribed["topic2"])
	}
}

// 扩展的mockDistributedMQAdapter，支持实际的消息传递
type mockDistributedMQAdapter struct {
	messages map[string][]Message
	channels map[string][]chan Message
	mu       sync.RWMutex
}

func (m *mockDistributedMQAdapter) Publish(_ context.Context, topic string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.messages == nil {
		m.messages = make(map[string][]Message)
	}

	msg := &mockMessage{
		topic:     topic,
		value:     payload,
		timestamp: time.Now(),
	}

	m.messages[topic] = append(m.messages[topic], msg)

	// 发送到所有订阅的通道
	if channels, exists := m.channels[topic]; exists {
		for _, ch := range channels {
			select {
			case ch <- msg:
			default:
				// 通道满了，跳过
			}
		}
	}

	return nil
}

func (m *mockDistributedMQAdapter) Subscribe(_ context.Context, topic, _ string) (<-chan Message, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.channels == nil {
		m.channels = make(map[string][]chan Message)
	}

	ch := make(chan Message, 100)
	m.channels[topic] = append(m.channels[topic], ch)

	closeFunc := func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		// 从channels中移除
		channels := m.channels[topic]
		for i, c := range channels {
			if c == ch {
				m.channels[topic] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		close(ch)
	}

	return ch, closeFunc, nil
}

func (m *mockDistributedMQAdapter) Ack(_ context.Context, _ string, _ Message) error {
	return nil
}

func (m *mockDistributedMQAdapter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭所有通道
	for _, channels := range m.channels {
		for _, ch := range channels {
			close(ch)
		}
	}

	m.channels = nil
	m.messages = nil

	return nil
}

type mockMessage struct {
	topic     string
	value     []byte
	timestamp time.Time
	key       []byte
}

func (m *mockMessage) Topic() string        { return m.topic }
func (m *mockMessage) Value() []byte        { return m.value }
func (m *mockMessage) Timestamp() time.Time { return m.timestamp }
func (m *mockMessage) Key() []byte          { return m.key }

func TestConsumerGroups_MultipleGroupsSameEvent(t *testing.T) {
	config := DefaultConfig()
	bus, err := New(config)
	require.NoError(t, err)
	defer func() {
		_ = bus.Close()
	}()

	// 用于收集不同消费者组的处理结果
	var mu sync.Mutex
	results := make(map[string][]string)

	// 消费者组1: 通知服务
	_, err = bus.On("user.registered", "notification-service", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		results["notification-service"] = append(results["notification-service"], event.ID)
		return nil
	})
	require.NoError(t, err)

	// 消费者组2: 积分服务
	_, err = bus.On("user.registered", "points-service", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		results["points-service"] = append(results["points-service"], event.ID)
		return nil
	})
	require.NoError(t, err)

	// 消费者组3: 分析服务
	_, err = bus.On("user.registered", "analytics-service", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		results["analytics-service"] = append(results["analytics-service"], event.ID)
		return nil
	})
	require.NoError(t, err)

	// 发布事件
	ctx := context.Background()
	err = bus.Emit(ctx, NewEvent("user.registered", map[string]interface{}{
		"user_id": "12345",
		"email":   "test@example.com",
	}))
	require.NoError(t, err)

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)

	// 验证每个消费者组都收到了事件
	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, results["notification-service"], 1, "通知服务应该收到1个事件")
	assert.Len(t, results["points-service"], 1, "积分服务应该收到1个事件")
	assert.Len(t, results["analytics-service"], 1, "分析服务应该收到1个事件")

	// 验证所有消费者组收到的是同一个事件
	eventID := results["notification-service"][0]
	assert.Equal(t, eventID, results["points-service"][0])
	assert.Equal(t, eventID, results["analytics-service"][0])
}

func TestConsumerGroups_DifferentConfigurations(t *testing.T) {
	config := DefaultConfig()
	bus, err := New(config)
	require.NoError(t, err)
	defer func() {
		_ = bus.Close()
	}()

	var mu sync.Mutex
	processedEvents := make(map[string]int)

	// 高可靠性消费者组 - 多次重试
	_, err = bus.On("order.created", "reliable-service", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		processedEvents["reliable-service"]++
		return nil
	},
		WithRetryPolicy(&RetryPolicy{
			MaxRetries:      5,
			BackoffStrategy: "exponential",
			InitialDelay:    10 * time.Millisecond,
		}),
		WithConcurrency(2),
	)
	require.NoError(t, err)

	// 快速处理消费者组 - 少量重试
	_, err = bus.On("order.created", "fast-service", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		processedEvents["fast-service"]++
		return nil
	},
		WithRetryPolicy(&RetryPolicy{
			MaxRetries:      1,
			BackoffStrategy: "fixed",
			InitialDelay:    5 * time.Millisecond,
		}),
		WithConcurrency(10),
	)
	require.NoError(t, err)

	// 发布多个事件
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		err = bus.Emit(ctx, NewEvent("order.created", map[string]interface{}{
			"order_id": i,
			"amount":   100.0,
		}))
		require.NoError(t, err)
	}

	// 等待处理完成
	time.Sleep(200 * time.Millisecond)

	// 验证两个消费者组都处理了所有事件
	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, 5, processedEvents["reliable-service"], "可靠服务应该处理5个事件")
	assert.Equal(t, 5, processedEvents["fast-service"], "快速服务应该处理5个事件")
}

func TestConsumerGroups_IndependentFailures(t *testing.T) {
	config := DefaultConfig()
	bus, err := New(config)
	require.NoError(t, err)
	defer func() {
		_ = bus.Close()
	}()

	var mu sync.Mutex
	successCount := 0
	failureCount := 0

	// 成功的消费者组
	_, err = bus.On("test.event", "success-service", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		successCount++
		return nil
	})
	require.NoError(t, err)

	// 失败的消费者组
	_, err = bus.On("test.event", "failure-service", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		failureCount++
		return assert.AnError // 模拟处理失败
	},
		WithRetryPolicy(&RetryPolicy{
			MaxRetries:   1,
			InitialDelay: 10 * time.Millisecond,
		}),
	)
	require.NoError(t, err)

	// 发布事件
	ctx := context.Background()
	err = bus.Emit(ctx, NewEvent("test.event", map[string]interface{}{
		"data": "test",
	}))
	require.NoError(t, err)

	// 等待处理完成（包括重试）
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// 成功的消费者组应该处理成功
	assert.Equal(t, 1, successCount, "成功服务应该处理1个事件")

	// 失败的消费者组应该尝试处理（原始尝试 + 1次重试 = 2次）
	assert.Equal(t, 2, failureCount, "失败服务应该尝试2次（1次原始 + 1次重试）")
}

func TestConsumerGroups_Statistics(t *testing.T) {
	config := DefaultConfig()
	bus, err := New(config)
	require.NoError(t, err)
	defer func() {
		_ = bus.Close()
	}()

	memBus, ok := bus.(MemoryEventBus)
	require.True(t, ok, "应该是内存模式EventBus")

	// 创建多个消费者组
	_, err = bus.On("stats.test", "service-a", func(ctx context.Context, event *Event) error {
		return nil
	})
	require.NoError(t, err)

	_, err = bus.On("stats.test", "service-b", func(ctx context.Context, event *Event) error {
		return nil
	})
	require.NoError(t, err)

	// 发布事件
	ctx := context.Background()
	err = bus.Emit(ctx, NewEvent("stats.test", map[string]interface{}{"test": "data"}))
	require.NoError(t, err)

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)

	// 获取统计信息
	stats := memBus.GetStats()

	// 验证统计信息
	assert.Equal(t, 2, stats["subscribers_count"], "应该有2个订阅者")

	// 验证指标
	if metrics, ok := stats["metrics"].(map[string]interface{}); ok {
		if emitted, ok := metrics["emitted"].(map[string]int); ok {
			assert.Equal(t, 1, emitted["stats.test"], "应该发布了1个事件")
		}
		if delivered, ok := metrics["delivered"].(map[string]int); ok {
			assert.Equal(t, 2, delivered["stats.test"], "应该投递了2个事件（每个消费者组1个）")
		}
	}
}

func BenchmarkConsumerGroups_MultipleGroups(b *testing.B) {
	config := DefaultConfig()
	bus, err := New(config)
	require.NoError(b, err)
	defer func() {
		_ = bus.Close()
	}()

	// 创建5个消费者组
	for i := 0; i < 5; i++ {
		groupName := fmt.Sprintf("service-%d", i)
		_, err = bus.On("benchmark.event", groupName, func(ctx context.Context, event *Event) error {
			// 模拟一些处理时间
			time.Sleep(time.Microsecond)
			return nil
		})
		require.NoError(b, err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := bus.Emit(ctx, NewEvent("benchmark.event", map[string]interface{}{
				"data": "benchmark",
			}))
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
