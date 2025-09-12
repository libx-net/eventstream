package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"libx.net/eventstream"
)

func main() {
	// 创建内存模式配置
	config := eventstream.DefaultConfig()
	config.Memory.EnableHistory = true
	config.Memory.EnableMetrics = true

	// 创建EventBus
	bus, err := eventstream.New(config)
	if err != nil {
		log.Fatal("Failed to create eventbus:", err)
	}
	defer bus.Close()

	fmt.Println("EventStream Memory Mode Example")
	fmt.Println("===============================")

	// 订阅用户注册事件
	subscription1, err := bus.On("user.registered", "user-group", handleUserRegistered)
	if err != nil {
		log.Fatal("Failed to subscribe to user.registered:", err)
	}

	// 订阅订单创建事件，带重试策略
	subscription2, err := bus.On("order.created",
		"order-group",
		handleOrderCreated,
		eventstream.WithConcurrency(2),
		eventstream.WithRetryPolicy(&eventstream.RetryPolicy{
			MaxRetries:      3,
			BackoffStrategy: "exponential",
			InitialDelay:    100 * time.Millisecond,
			MaxDelay:        2 * time.Second,
			Multiplier:      2.0,
		}),
	)
	if err != nil {
		log.Fatal("Failed to subscribe to order.created:", err)
	}

	// 等待一下让订阅者准备好
	time.Sleep(100 * time.Millisecond)

	// 发布一些事件
	ctx := context.Background()

	// 发布用户注册事件
	for i := 1; i <= 3; i++ {
		userData := map[string]interface{}{
			"user_id": fmt.Sprintf("user_%d", i),
			"email":   fmt.Sprintf("user%d@example.com", i),
			"name":    fmt.Sprintf("User %d", i),
		}

		if err := bus.Emit(ctx, eventstream.NewEvent("user.registered", userData)); err != nil {
			log.Printf("Failed to emit user.registered: %v", err)
		}
	}

	// 发布订单创建事件
	for i := 1; i <= 2; i++ {
		orderData := map[string]interface{}{
			"order_id": fmt.Sprintf("order_%d", i),
			"user_id":  fmt.Sprintf("user_%d", i),
			"amount":   100.0 * float64(i),
			"items":    []string{"item1", "item2"},
		}

		if err := bus.Emit(ctx, eventstream.NewEvent("order.created", orderData)); err != nil {
			log.Printf("Failed to emit order.created: %v", err)
		}
	}

	// 等待事件处理完成
	time.Sleep(1 * time.Second)

	// 显示统计信息
	if memBus, ok := bus.(eventstream.MemoryEventBus); ok {
		stats := memBus.GetStats()
		fmt.Println("\nEventBus Statistics:")
		for key, value := range stats {
			fmt.Printf("  %s: %v\n", key, value)
		}

		// 显示事件历史
		history := memBus.GetHistory(10)
		fmt.Printf("\nEvent History (%d events):\n", len(history))
		for i, event := range history {
			fmt.Printf("  %d. [%s] %s at %s\n",
				i+1, event.ID[:8], event.Topic, event.Timestamp.Format("15:04:05"))
		}
	}

	// 取消订阅
	fmt.Println("\nUnsubscribing...")
	if err := bus.Off(subscription1); err != nil {
		log.Printf("Failed to unsubscribe: %v", err)
	}
	if err := bus.Off(subscription2); err != nil {
		log.Printf("Failed to unsubscribe: %v", err)
	}

	fmt.Println("Example completed!")
}

// handleUserRegistered 处理用户注册事件
func handleUserRegistered(ctx context.Context, event *eventstream.Event) error {
	var userData struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Name   string `json:"name"`
	}

	if err := event.Unmarshal(&userData); err != nil {
		return fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	fmt.Printf("✅ User registered: %s (%s)\n", userData.Name, userData.Email)

	// 模拟一些处理时间
	time.Sleep(50 * time.Millisecond)

	return nil
}

// handleOrderCreated 处理订单创建事件
func handleOrderCreated(ctx context.Context, event *eventstream.Event) error {
	var orderData struct {
		OrderID string   `json:"order_id"`
		UserID  string   `json:"user_id"`
		Amount  float64  `json:"amount"`
		Items   []string `json:"items"`
	}

	if err := event.Unmarshal(&orderData); err != nil {
		return fmt.Errorf("failed to unmarshal order data: %w", err)
	}

	fmt.Printf("📦 Order created: %s for user %s, amount: $%.2f\n",
		orderData.OrderID, orderData.UserID, orderData.Amount)

	// 模拟一些处理时间
	time.Sleep(100 * time.Millisecond)

	return nil
}
