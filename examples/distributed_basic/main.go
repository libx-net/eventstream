package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"libx.net/eventstream"
)

// 简单的MQAdapter实现示例
type SimpleMQAdapter struct {
	messages map[string][]chan []byte
	mu       sync.RWMutex
}

func NewSimpleMQAdapter() *SimpleMQAdapter {
	return &SimpleMQAdapter{
		messages: make(map[string][]chan []byte),
	}
}

func (s *SimpleMQAdapter) Publish(ctx context.Context, topic string, payload []byte) error {
	s.mu.RLock()
	channels := s.messages[topic]
	s.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- payload:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 如果通道满，跳过
		}
	}
	return nil
}

func (s *SimpleMQAdapter) Subscribe(ctx context.Context, topic, groupID string) (<-chan eventstream.Message, func(), error) {
	s.mu.Lock()
	ch := make(chan []byte, 100)
	s.messages[topic] = append(s.messages[topic], ch)
	s.mu.Unlock()

	messageCh := make(chan eventstream.Message, 100)
	stopCh := make(chan struct{})

	// 转换消息格式
	go func() {
		defer close(messageCh)
		for {
			select {
			case msg := <-ch:
				messageCh <- &simpleMessage{data: msg}
			case <-stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return messageCh, func() { close(stopCh) }, nil
}

func (s *SimpleMQAdapter) Ack(ctx context.Context, groupID string, msg eventstream.Message) error {
	fmt.Printf("Acknowledging message for topic %s, group %s", msg.Topic(), groupID)
	return nil
}

func (s *SimpleMQAdapter) Close() error {
	return nil
}

type simpleMessage struct {
	data []byte
}

func (m *simpleMessage) Topic() string        { return "test.topic" } // 为示例硬编码主题名
func (m *simpleMessage) Value() []byte        { return m.data }
func (m *simpleMessage) Timestamp() time.Time { return time.Now() }
func (m *simpleMessage) Key() []byte          { return nil }

func main() {
	// 创建配置
	config := eventstream.DefaultConfig()
	config.Mode = eventstream.ModeDistributed
	config.Distributed = &eventstream.DistributedConfig{
		MQAdapter: NewSimpleMQAdapter(),
	}

	// 创建事件总线
	eventBus, err := eventstream.New(config)
	if err != nil {
		log.Fatalf("Failed to create event bus: %v", err)
	}
	defer eventBus.Close()

	// 订阅事件
	subscription, err := eventBus.On("test.topic", "test-group", func(ctx context.Context, event *eventstream.Event) error {
		fmt.Printf("Received event: %s - %v\n", event.Topic, event.Data)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	defer eventBus.Off(subscription)

	// 发布事件
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		err := eventBus.Emit(ctx, "test.topic", []byte(fmt.Sprintf("Hello World %d", i)))
		if err != nil {
			log.Printf("Failed to emit event: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 等待一段时间让事件处理完成
	time.Sleep(2 * time.Second)
	fmt.Println("Example completed")
}
