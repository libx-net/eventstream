package eventstream

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryEventBus_Creation(t *testing.T) {
	config := DefaultConfig()
	bus, err := newMemoryEventBusImpl(config)
	if err != nil {
		t.Fatalf("Failed to create memory event bus: %v", err)
	}
	defer bus.Close()

	memBus, ok := bus.(*memoryEventBus)
	if !ok {
		t.Fatal("Expected *memoryEventBus type")
	}

	if memBus.config != config {
		t.Error("Config should be set correctly")
	}

	if memBus.pool == nil {
		t.Error("Pool should be initialized")
	}

	if memBus.subMgr == nil {
		t.Error("Subscriber manager should be initialized")
	}
}

func TestMemoryEventBus_WithHistory(t *testing.T) {
	config := DefaultConfig()
	config.Memory.EnableHistory = true
	config.Memory.MaxHistorySize = 5

	bus, err := newMemoryEventBusImpl(config)
	if err != nil {
		t.Fatalf("Failed to create memory event bus: %v", err)
	}
	defer bus.Close()

	memBus := bus.(*memoryEventBus)

	// 发布多个事件
	ctx := context.Background()
	for i := 0; i < 7; i++ {
		err := bus.Emit(ctx, NewEvent("test.topic", i))
		if err != nil {
			t.Errorf("Failed to emit event %d: %v", i, err)
		}
	}

	time.Sleep(50 * time.Millisecond)

	// 检查历史记录
	history := memBus.GetHistory(10)
	if len(history) != 5 { // 应该只保留最新的5个
		t.Errorf("Expected 5 events in history, got %d", len(history))
	}

	// 检查是否是最新的5个事件
	for i, event := range history {
		expectedData := i + 2 // 应该是2,3,4,5,6
		if event.Data != expectedData {
			t.Errorf("Expected event data %d, got %v", expectedData, event.Data)
		}
	}
}

func TestMemoryEventBus_WithMetrics(t *testing.T) {
	config := DefaultConfig()
	config.Memory.EnableMetrics = true

	bus, err := newMemoryEventBusImpl(config)
	if err != nil {
		t.Fatalf("Failed to create memory event bus: %v", err)
	}
	defer bus.Close()

	memBus := bus.(*memoryEventBus)

	// 订阅事件
	_, err = bus.On("test.metrics", "test-group", func(ctx context.Context, event *Event) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// 发布事件
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		err := bus.Emit(ctx, NewEvent("test.metrics", i))
		if err != nil {
			t.Errorf("Failed to emit event %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// 检查统计信息
	stats := memBus.GetStats()

	if stats["mode"] != "memory" {
		t.Errorf("Expected mode 'memory', got %v", stats["mode"])
	}

	if stats["subscribers_count"] != 1 {
		t.Errorf("Expected 1 subscriber, got %v", stats["subscribers_count"])
	}

	metrics, ok := stats["metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("Metrics should be present")
	}

	emitted, ok := metrics["emitted"].(map[string]int64)
	if !ok {
		t.Fatal("Emitted metrics should be present")
	}

	if emitted["test.metrics"] != 3 {
		t.Errorf("Expected 3 emitted events, got %d", emitted["test.metrics"])
	}
}

func TestMemoryEventBus_ErrorHandling(t *testing.T) {
	config := DefaultConfig()
	bus, err := newMemoryEventBusImpl(config)
	if err != nil {
		t.Fatalf("Failed to create memory event bus: %v", err)
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

	// 关闭bus后测试
	bus.Close()

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

func TestMemoryEventBus_ConcurrentAccess(t *testing.T) {
	config := DefaultConfig()
	config.Memory.EnableMetrics = true

	bus, err := newMemoryEventBusImpl(config)
	if err != nil {
		t.Fatalf("Failed to create memory event bus: %v", err)
	}
	defer bus.Close()

	var wg sync.WaitGroup
	var receivedCount int64
	var mu sync.Mutex

	// 创建多个订阅者
	for i := 0; i < 5; i++ {
		_, err := bus.On("concurrent.test", "test-group", func(ctx context.Context, event *Event) error {
			mu.Lock()
			receivedCount++
			mu.Unlock()
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
	}

	time.Sleep(10 * time.Millisecond)

	// 并发发布事件
	numEvents := 10
	wg.Add(numEvents)

	for i := 0; i < numEvents; i++ {
		go func(i int) {
			defer wg.Done()
			ctx := context.Background()
			err := bus.Emit(ctx, NewEvent("concurrent.test", i))
			if err != nil {
				t.Errorf("Failed to emit event %d: %v", i, err)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// 检查结果
	mu.Lock()
	expected := int64(numEvents * 5) // 10个事件 * 5个订阅者
	if receivedCount != expected {
		t.Errorf("Expected %d received events, got %d", expected, receivedCount)
	}
	mu.Unlock()
}

func TestMemorySubscription_Interface(t *testing.T) {
	config := DefaultConfig()
	bus, err := newMemoryEventBusImpl(config)
	if err != nil {
		t.Fatalf("Failed to create memory event bus: %v", err)
	}
	defer bus.Close()

	sub, err := bus.On("test.subscription", "test-group", func(ctx context.Context, event *Event) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// 测试接口方法
	if sub.Topic() != "test.subscription" {
		t.Errorf("Expected topic 'test.subscription', got '%s'", sub.Topic())
	}

	if sub.ID() == "" {
		t.Error("Subscription ID should not be empty")
	}

	if !sub.IsActive() {
		t.Error("Subscription should be active")
	}

	// 取消订阅
	err = bus.Off(sub)
	if err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}
}

func TestEventHistory_Operations(t *testing.T) {
	history := newEventHistory(3)

	// 添加事件
	for i := 0; i < 5; i++ {
		event := NewEvent("test.topic", i)
		history.add(event)
	}

	// 检查数量
	if history.count() != 3 {
		t.Errorf("Expected 3 events, got %d", history.count())
	}

	// 获取历史记录
	events := history.get(10)
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// 检查是否是最新的3个事件
	for i, event := range events {
		expectedData := i + 2 // 应该是2,3,4
		if event.Data != expectedData {
			t.Errorf("Expected event data %d, got %v", expectedData, event.Data)
		}
	}

	// 测试限制获取数量
	limitedEvents := history.get(2)
	if len(limitedEvents) != 2 {
		t.Errorf("Expected 2 events, got %d", len(limitedEvents))
	}
}

func TestEventMetrics_Operations(t *testing.T) {
	metrics := newEventMetrics()

	// 增加指标
	metrics.incEmitted("topic1")
	metrics.incEmitted("topic1")
	metrics.incEmitted("topic2")

	metrics.incDelivered("topic1", 2)
	metrics.incDelivered("topic2", 1)

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

	delivered := stats["delivered"].(map[string]int64)
	if delivered["topic1"] != 2 {
		t.Errorf("Expected 2 delivered for topic1, got %d", delivered["topic1"])
	}

	subscribed := stats["subscribed"].(map[string]int64)
	if subscribed["topic1"] != 1 {
		t.Errorf("Expected 1 subscribed for topic1, got %d", subscribed["topic1"])
	}
	if subscribed["topic2"] != 0 {
		t.Errorf("Expected 0 subscribed for topic2, got %d", subscribed["topic2"])
	}
}
