package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"libx.net/eventstream"
)

func main() {
	fmt.Println("Kafka Adapter Example")
	fmt.Println("=====================")

	// 创建 Kafka 配置
	kafkaConfig := eventstream.KafkaConfig{
		Brokers: []string{"localhost:9092"}, // Kafka broker 地址
		Producer: eventstream.KafkaProducerConfig{
			BatchSize:    100,
			BatchTimeout: 100 * time.Millisecond,
			Compression:  "gzip",
			RequiredAcks: -1, // 所有副本确认
		},
		Consumer: eventstream.KafkaConsumerConfig{
			StartOffset:    "earliest",
			CommitInterval: 1 * time.Second,
			MaxWait:        500 * time.Millisecond,
			MinBytes:       1,
			MaxBytes:       10e6, // 10MB
		},
	}

	// 创建 Kafka 适配器
	kafkaAdapter, err := eventstream.NewKafkaAdapter(kafkaConfig)
	if err != nil {
		log.Fatalf("Failed to create Kafka adapter: %v", err)
	}
	defer kafkaAdapter.Close()

	// 创建事件流配置
	config := eventstream.DefaultConfig()
	config.Mode = eventstream.ModeDistributed
	config.Distributed = &eventstream.DistributedConfig{
		MQAdapter: kafkaAdapter,
	}

	// 创建事件总线
	eventBus, err := eventstream.New(config)
	if err != nil {
		log.Fatalf("Failed to create event bus: %v", err)
	}
	defer eventBus.Close()

	// 订阅用户事件
	userSubscription, err := eventBus.On("user.created", "user-service", func(ctx context.Context, event *eventstream.Event) error {
		fmt.Printf("📧 Received user created event: %s\n", string(event.Data))
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to user.created: %v", err)
	}
	defer eventBus.Off(userSubscription)

	// 订阅订单事件
	orderSubscription, err := eventBus.On("order.placed", "order-service", func(ctx context.Context, event *eventstream.Event) error {
		fmt.Printf("📦 Received order placed event: %s\n", string(event.Data))
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to order.placed: %v", err)
	}
	defer eventBus.Off(orderSubscription)

	// 等待订阅者准备就绪
	time.Sleep(1 * time.Second)

	ctx := context.Background()

	// 发布用户创建事件
	fmt.Println("\nPublishing user events...")
	for i := 1; i <= 3; i++ {
		userData := fmt.Sprintf(`{"id": %d, "name": "User %d", "email": "user%d@example.com"}`, i, i, i)
		if err := eventBus.Emit(ctx, "user.created", []byte(userData)); err != nil {
			log.Printf("Failed to emit user.created event: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// 发布订单创建事件
	fmt.Println("\nPublishing order events...")
	for i := 1; i <= 2; i++ {
		orderData := fmt.Sprintf(`{"id": "order-%d", "userId": %d, "amount": %.2f}`, i, i, 100.0*float64(i))
		if err := eventBus.Emit(ctx, "order.placed", []byte(orderData)); err != nil {
			log.Printf("Failed to emit order.placed event: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// 等待事件处理完成
	time.Sleep(2 * time.Second)
	fmt.Println("\nExample completed! Check your Kafka topics for the messages.")
}

// 注意：运行此示例前需要：
// 1. 启动 Kafka 服务器（localhost:9092）
// 2. 创建 topics: user.created 和 order.placed
// 3. 或者使用 Kafka 的自动创建 topics 功能
