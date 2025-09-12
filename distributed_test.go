package eventstream

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDistributedEventBus_Creation(t *testing.T) {
	config := DefaultDistributedConfig()
	config.Distributed.MQAdapter = &mockDistributedMQAdapter{}

	bus, err := NewDistributedEventBus(config)
	if err != nil {
		t.Fatalf("Failed to create distributed event bus: %v", err)
	}
	defer bus.Close()

	if bus.config != config {
		t.Error("Config should be set correctly")
	}

	if bus.adapter == nil {
		t.Error("Adapter should be set")
	}

	if bus.serializer == nil {
		t.Error("Serializer should be set")
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
	defer bus.Close()

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
		err := bus.Emit(ctx, "test.distributed", data)
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
	err = bus.Emit(ctx, "", "data")
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
	bus.Close()

	err = bus.Emit(ctx, "test.topic", "data")
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

func (m *mockDistributedMQAdapter) Publish(ctx context.Context, topic string, payload []byte) error {
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

func (m *mockDistributedMQAdapter) Subscribe(ctx context.Context, topic, groupID string) (<-chan Message, func(), error) {
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

func (m *mockDistributedMQAdapter) Ack(ctx context.Context, groupID string, msg Message) error {
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
