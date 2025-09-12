package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"libx.net/eventstream"
)

func main() {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}

	rabbitMQConfig := RabbitMQConfig{
		URL: rabbitmqURL,
	}

	adapter, err := NewRabbitMQAdapter(rabbitMQConfig)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ adapter: %v", err)
	}
	defer adapter.Close()

	config := &eventstream.Config{
		Distributed: &eventstream.DistributedConfig{
			MQAdapter: adapter,
		},
	}

	bus, err := eventstream.NewDistributedEventBus(config)
	if err != nil {
		log.Fatalf("Failed to create distributed event bus: %v", err)
	}
	defer bus.Close()

	handler := func(ctx context.Context, event *eventstream.Event) error {
		fmt.Printf("Received event: Topic=%s, Data=%v\n", event.Topic, event.Data)
		return nil
	}

	sub, err := bus.On("user.created", "user-service-group", handler)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	defer bus.Off(sub)

	fmt.Println("Subscribed to 'user.created' with group 'user-service-group'")

	go func() {
		for i := 0; i < 5; i++ {
			data := map[string]interface{}{"userId": i, "name": fmt.Sprintf("user-%d", i)}
			if err := bus.Emit(context.Background(), "user.created", data); err != nil {
				log.Printf("Failed to emit event: %v", err)
			} else {
				fmt.Printf("Emitted event for user %d\n", i)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down...")
}
