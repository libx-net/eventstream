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

func TestConsumerGroups_MultipleGroupsSameEvent(t *testing.T) {
	config := DefaultConfig()
	bus, err := New(config)
	require.NoError(t, err)
	defer bus.Close()

	// 用于收集不同消费者组的处理结果
	var mu sync.Mutex
	results := make(map[string][]string)

	// 消费者组1: 通知服务
	_, err = bus.On("user.registered", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		results["notification-service"] = append(results["notification-service"], event.ID)
		return nil
	}, WithConsumerGroup("notification-service"))
	require.NoError(t, err)

	// 消费者组2: 积分服务
	_, err = bus.On("user.registered", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		results["points-service"] = append(results["points-service"], event.ID)
		return nil
	}, WithConsumerGroup("points-service"))
	require.NoError(t, err)

	// 消费者组3: 分析服务
	_, err = bus.On("user.registered", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		results["analytics-service"] = append(results["analytics-service"], event.ID)
		return nil
	}, WithConsumerGroup("analytics-service"))
	require.NoError(t, err)

	// 发布事件
	ctx := context.Background()
	err = bus.Emit(ctx, "user.registered", map[string]interface{}{
		"user_id": "12345",
		"email":   "test@example.com",
	})
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
	defer bus.Close()

	var mu sync.Mutex
	processedEvents := make(map[string]int)

	// 高可靠性消费者组 - 多次重试
	_, err = bus.On("order.created", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		processedEvents["reliable-service"]++
		return nil
	},
		WithConsumerGroup("reliable-service"),
		WithRetryPolicy(&RetryPolicy{
			MaxRetries:      5,
			BackoffStrategy: "exponential",
			InitialDelay:    10 * time.Millisecond,
		}),
		WithConcurrency(2),
	)
	require.NoError(t, err)

	// 快速处理消费者组 - 少量重试
	_, err = bus.On("order.created", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		processedEvents["fast-service"]++
		return nil
	},
		WithConsumerGroup("fast-service"),
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
		err = bus.Emit(ctx, "order.created", map[string]interface{}{
			"order_id": i,
			"amount":   100.0,
		})
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
	defer bus.Close()

	var mu sync.Mutex
	successCount := 0
	failureCount := 0

	// 成功的消费者组
	_, err = bus.On("test.event", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		successCount++
		return nil
	}, WithConsumerGroup("success-service"))
	require.NoError(t, err)

	// 失败的消费者组
	_, err = bus.On("test.event", func(ctx context.Context, event *Event) error {
		mu.Lock()
		defer mu.Unlock()
		failureCount++
		return assert.AnError // 模拟处理失败
	},
		WithConsumerGroup("failure-service"),
		WithRetryPolicy(&RetryPolicy{
			MaxRetries:   1,
			InitialDelay: 10 * time.Millisecond,
		}),
	)
	require.NoError(t, err)

	// 发布事件
	ctx := context.Background()
	err = bus.Emit(ctx, "test.event", map[string]interface{}{
		"data": "test",
	})
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
	defer bus.Close()

	memBus, ok := bus.(MemoryEventBus)
	require.True(t, ok, "应该是内存模式EventBus")

	// 创建多个消费者组
	_, err = bus.On("stats.test", func(ctx context.Context, event *Event) error {
		return nil
	}, WithConsumerGroup("service-a"))
	require.NoError(t, err)

	_, err = bus.On("stats.test", func(ctx context.Context, event *Event) error {
		return nil
	}, WithConsumerGroup("service-b"))
	require.NoError(t, err)

	// 发布事件
	ctx := context.Background()
	err = bus.Emit(ctx, "stats.test", map[string]interface{}{"test": "data"})
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
	defer bus.Close()

	// 创建5个消费者组
	for i := 0; i < 5; i++ {
		groupName := fmt.Sprintf("service-%d", i)
		_, err = bus.On("benchmark.event", func(ctx context.Context, event *Event) error {
			// 模拟一些处理时间
			time.Sleep(time.Microsecond)
			return nil
		}, WithConsumerGroup(groupName))
		require.NoError(b, err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := bus.Emit(ctx, "benchmark.event", map[string]interface{}{
				"data": "benchmark",
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
