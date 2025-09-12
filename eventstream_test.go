package eventstream

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryEventBus_Basic(t *testing.T) {
	// 创建内存模式EventBus
	config := DefaultConfig()
	bus, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create eventbus: %v", err)
	}
	defer bus.Close()

	// 测试数据
	var receivedEvents []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 订阅事件
	subscription, err := bus.On("test.event", "test-group", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents = append(receivedEvents, event.Data.(string))
		wg.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// 等待订阅准备好
	time.Sleep(10 * time.Millisecond)

	// 发布事件
	ctx := context.Background()
	testData := []string{"event1", "event2", "event3"}

	wg.Add(len(testData))
	for _, data := range testData {
		if err := bus.Emit(ctx, NewEvent("test.event", data)); err != nil {
			t.Errorf("Failed to emit event: %v", err)
		}
	}

	// 等待所有事件处理完成
	wg.Wait()

	// 验证结果 - 只检查数量，不检查顺序（因为是异步处理）
	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != len(testData) {
		t.Errorf("Expected %d events, got %d", len(testData), len(receivedEvents))
	}

	// 验证所有期望的事件都被接收到
	expectedMap := make(map[string]bool)
	for _, data := range testData {
		expectedMap[data] = true
	}

	for _, received := range receivedEvents {
		if !expectedMap[received] {
			t.Errorf("Unexpected event received: %s", received)
		}
		delete(expectedMap, received)
	}

	if len(expectedMap) > 0 {
		t.Errorf("Some expected events were not received: %v", expectedMap)
	}

	// 取消订阅
	if err := bus.Off(subscription); err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}
}

func TestMemoryEventBus_MultipleSubscribers(t *testing.T) {
	config := DefaultConfig()
	bus, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create eventbus: %v", err)
	}
	defer bus.Close()

	var received1, received2 []string
	var mu1, mu2 sync.Mutex

	// 第一个订阅者
	sub1, err := bus.On("test.multi", "group1", func(ctx context.Context, event *Event) error {
		mu1.Lock()
		defer mu1.Unlock()
		received1 = append(received1, event.Data.(string))
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe 1: %v", err)
	}

	// 第二个订阅者
	sub2, err := bus.On("test.multi", "group2", func(ctx context.Context, event *Event) error {
		mu2.Lock()
		defer mu2.Unlock()
		received2 = append(received2, event.Data.(string))
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe 2: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// 发布事件
	ctx := context.Background()
	if err := bus.Emit(ctx, NewEvent("test.multi", "shared_event")); err != nil {
		t.Errorf("Failed to emit event: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// 验证两个订阅者都收到了事件
	mu1.Lock()
	mu2.Lock()
	defer mu1.Unlock()
	defer mu2.Unlock()

	if len(received1) != 1 || received1[0] != "shared_event" {
		t.Errorf("Subscriber 1 did not receive event correctly: %v", received1)
	}

	if len(received2) != 1 || received2[0] != "shared_event" {
		t.Errorf("Subscriber 2 did not receive event correctly: %v", received2)
	}

	// 清理
	if err := bus.Off(sub1); err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}
	if err := bus.Off(sub2); err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}
}

func TestMemoryEventBus_WithRetry(t *testing.T) {
	config := DefaultConfig()
	bus, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create eventbus: %v", err)
	}
	defer bus.Close()

	var attempts int
	var mu sync.Mutex

	// 订阅事件，前两次失败，第三次成功
	subscription, err := bus.On("test.retry", "retry-group", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		attempts++
		if attempts < 3 {
			return &testError{"simulated error"}
		}
		return nil
	}, WithRetryPolicy(&RetryPolicy{
		MaxRetries:      3,
		BackoffStrategy: BackoffFixed,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		Multiplier:      1.0,
	}))
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// 发布事件
	ctx := context.Background()
	if err := bus.Emit(ctx, NewEvent("test.retry", "retry_test")); err != nil {
		t.Errorf("Failed to emit event: %v", err)
	}

	// 等待重试完成
	time.Sleep(200 * time.Millisecond)

	// 验证重试次数
	mu.Lock()
	defer mu.Unlock()

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if err := bus.Off(subscription); err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}
}

func TestMemoryEventBus_Stats(t *testing.T) {
	config := DefaultConfig()
	config.Memory.EnableMetrics = true
	config.Memory.EnableHistory = true

	bus, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create eventbus: %v", err)
	}
	defer bus.Close()

	// 类型断言到MemoryEventBus
	memBus, ok := bus.(MemoryEventBus)
	if !ok {
		t.Fatalf("Expected MemoryEventBus, got %T", bus)
	}

	// 订阅事件
	subscription, err := bus.On("test.stats", "stats-group", func(ctx context.Context, event *Event) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// 发布几个事件
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if err := bus.Emit(ctx, NewEvent("test.stats", i)); err != nil {
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

	// 检查历史记录
	history := memBus.GetHistory(10)
	if len(history) != 3 {
		t.Errorf("Expected 3 events in history, got %d", len(history))
	}

	if err := bus.Off(subscription); err != nil {
		t.Errorf("Failed to unsubscribe: %v", err)
	}
}

func TestMemoryEventBus_Close(t *testing.T) {
	config := DefaultConfig()
	bus, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create eventbus: %v", err)
	}

	// 订阅事件
	subscription, err := bus.On("test.close", "close-group", func(ctx context.Context, event *Event) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// 关闭EventBus
	if err := bus.Close(); err != nil {
		t.Errorf("Failed to close eventbus: %v", err)
	}

	// 尝试发布事件应该失败
	ctx := context.Background()
	if err := bus.Emit(ctx, NewEvent("test.close", "data")); err == nil {
		t.Error("Expected error when emitting to closed eventbus")
	}

	// 尝试订阅应该失败
	_, err = bus.On("test.close2", "close-group2", func(ctx context.Context, event *Event) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error when subscribing to closed eventbus")
	}

	// 取消订阅应该失败
	if err := bus.Off(subscription); err == nil {
		t.Error("Expected error when unsubscribing from closed eventbus")
	}
}

// testError 测试用的错误类型
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func BenchmarkMemoryEventBus_Emit(b *testing.B) {
	config := DefaultConfig()
	config.Memory.EnableHistory = false
	config.Memory.EnableMetrics = false

	bus, err := New(config)
	if err != nil {
		b.Fatalf("Failed to create eventbus: %v", err)
	}
	defer bus.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := bus.Emit(ctx, NewEvent("bench.test", "data")); err != nil {
				b.Errorf("Emit failed: %v", err)
			}
		}
	})
}

func BenchmarkMemoryEventBus_EmitAndProcess(b *testing.B) {
	config := DefaultConfig()
	config.Memory.EnableHistory = false
	config.Memory.EnableMetrics = false

	bus, err := New(config)
	if err != nil {
		b.Fatalf("Failed to create eventbus: %v", err)
	}
	defer bus.Close()

	// 订阅事件
	var processed int64
	subscription, err := bus.On("bench.process", "bench-group", func(ctx context.Context, event *Event) error {
		// 模拟一些处理
		_ = event.Data
		return nil
	})
	if err != nil {
		b.Fatalf("On failed: %v", err)
	}
	defer func() {
		if err := bus.Off(subscription); err != nil {
			b.Errorf("Off failed: %v", err)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := bus.Emit(ctx, NewEvent("bench.process", "data")); err != nil {
				b.Errorf("Emit failed: %v", err)
			}
			processed++
		}
	})
}
