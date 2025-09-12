package main

import (
	"context"
	"fmt"
	"log"
	"sync"
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

	fmt.Println("EventStream Multiple Consumers Example")
	fmt.Println("=====================================")

	var wg sync.WaitGroup

	// 第一个消费者组：通知服务
	// 处理用户注册事件，发送欢迎邮件
	wg.Add(1)
	subscription1, err := bus.On("user.registered",
		"notification-service",
		func(ctx context.Context, event *eventstream.Event) error {
			defer wg.Done()
			var userData struct {
				UserID string `json:"user_id"`
				Email  string `json:"email"`
				Name   string `json:"name"`
			}

			if err := event.Unmarshal(&userData); err != nil {
				return err
			}

			fmt.Printf("📧 [通知服务] 发送欢迎邮件给: %s (%s)\n", userData.Name, userData.Email)
			time.Sleep(50 * time.Millisecond) // 模拟邮件发送
			return nil
		},
		eventstream.WithConcurrency(1),
	)
	if err != nil {
		log.Fatal("Failed to subscribe notification service:", err)
	}

	// 第二个消费者组：分析服务
	// 处理同样的用户注册事件，进行数据分析
	wg.Add(1)
	subscription2, err := bus.On("user.registered",
		"analytics-service",
		func(ctx context.Context, event *eventstream.Event) error {
			defer wg.Done()
			var userData struct {
				UserID string `json:"user_id"`
				Email  string `json:"email"`
				Name   string `json:"name"`
			}

			if err := event.Unmarshal(&userData); err != nil {
				return err
			}

			fmt.Printf("📊 [分析服务] 记录用户注册数据: %s\n", userData.UserID)
			time.Sleep(30 * time.Millisecond) // 模拟数据处理
			return nil
		},
		eventstream.WithConcurrency(2),
	)
	if err != nil {
		log.Fatal("Failed to subscribe analytics service:", err)
	}

	// 第三个消费者组：积分服务
	// 处理同样的用户注册事件，给新用户发放积分
	wg.Add(1)
	subscription3, err := bus.On("user.registered",
		"points-service",
		func(ctx context.Context, event *eventstream.Event) error {
			defer wg.Done()
			var userData struct {
				UserID string `json:"user_id"`
				Email  string `json:"email"`
				Name   string `json:"name"`
			}

			if err := event.Unmarshal(&userData); err != nil {
				return err
			}

			fmt.Printf("🎁 [积分服务] 为新用户 %s 发放100积分\n", userData.Name)
			time.Sleep(20 * time.Millisecond) // 模拟积分发放
			return nil
		},
		eventstream.WithConcurrency(1),
		eventstream.WithRetryPolicy(&eventstream.RetryPolicy{
			MaxRetries:      2,
			BackoffStrategy: "fixed",
			InitialDelay:    100 * time.Millisecond,
			MaxDelay:        1 * time.Second,
			Multiplier:      1.0,
		}),
	)
	if err != nil {
		log.Fatal("Failed to subscribe points service:", err)
	}

	// 等待订阅者准备好
	time.Sleep(100 * time.Millisecond)

	// 发布用户注册事件
	ctx := context.Background()
	userData := map[string]interface{}{
		"user_id": "user_12345",
		"email":   "john.doe@example.com",
		"name":    "John Doe",
	}

	fmt.Println("\n🚀 发布用户注册事件...")
	if err := bus.Emit(ctx, "user.registered", userData); err != nil {
		log.Printf("Failed to emit user.registered: %v", err)
	}

	// 等待所有消费者处理完成
	wg.Wait()

	// 显示统计信息
	if memBus, ok := bus.(eventstream.MemoryEventBus); ok {
		stats := memBus.GetStats()
		fmt.Println("\n📈 EventBus统计信息:")
		fmt.Printf("  模式: %v\n", stats["mode"])
		fmt.Printf("  订阅者数量: %v\n", stats["subscribers_count"])
		fmt.Printf("  协程池状态: 运行中=%v, 空闲=%v, 容量=%v\n",
			stats["pool_running"], stats["pool_free"], stats["pool_capacity"])

		if metrics, ok := stats["metrics"].(map[string]interface{}); ok {
			fmt.Printf("  事件指标: %+v\n", metrics)
		}
	}

	// 演示多个消费者组处理不同事件
	fmt.Println("\n🔄 演示不同服务处理不同事件...")

	// 订单服务只关心订单事件
	var orderWg sync.WaitGroup
	orderWg.Add(1)
	orderSub, err := bus.On("order.created",
		"order-service",
		func(ctx context.Context, event *eventstream.Event) error {
			defer orderWg.Done()
			fmt.Printf("📦 [订单服务] 处理订单创建: %v\n", event.Data)
			return nil
		},
	)
	if err != nil {
		log.Fatal("Failed to subscribe order service:", err)
	}

	// 库存服务也关心订单事件
	orderWg.Add(1)
	inventorySub, err := bus.On("order.created",
		"inventory-service",
		func(ctx context.Context, event *eventstream.Event) error {
			defer orderWg.Done()
			fmt.Printf("📋 [库存服务] 更新库存: %v\n", event.Data)
			return nil
		},
	)
	if err != nil {
		log.Fatal("Failed to subscribe inventory service:", err)
	}

	time.Sleep(50 * time.Millisecond)

	// 发布订单创建事件
	orderData := map[string]interface{}{
		"order_id": "order_67890",
		"user_id":  "user_12345",
		"amount":   299.99,
	}

	if err := bus.Emit(ctx, "order.created", orderData); err != nil {
		log.Printf("Failed to emit order.created: %v", err)
	}

	orderWg.Wait()

	// 清理订阅
	fmt.Println("\n🧹 清理订阅...")
	subscriptions := []eventstream.Subscription{
		subscription1, subscription2, subscription3, orderSub, inventorySub,
	}

	for i, sub := range subscriptions {
		if err := bus.Off(sub); err != nil {
			log.Printf("Failed to unsubscribe %d: %v", i, err)
		}
	}

	fmt.Println("✅ 示例完成！")
	fmt.Println("\n💡 关键点:")
	fmt.Println("  1. 同一个事件可以被多个消费者组消费")
	fmt.Println("  2. 每个消费者组可以有不同的处理逻辑和配置")
	fmt.Println("  3. 消费者组之间完全独立，互不影响")
	fmt.Println("  4. 可以动态添加和移除消费者组")
}
