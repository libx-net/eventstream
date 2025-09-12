//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkaTC "github.com/testcontainers/testcontainers-go/modules/kafka"

	eventstream "libx.net/eventstream"
)

// isContainerRuntimeAvailable 检查容器运行时是否可用
func isContainerRuntimeAvailable() bool {
	// 检查Docker
	if cmd := exec.Command("docker", "info"); cmd.Run() == nil {
		return true
	}
	// 检查Podman
	if cmd := exec.Command("podman", "info"); cmd.Run() == nil {
		return true
	}
	return false
}

// createTopicWithKafkaGo 使用kafka-go直接创建topic
func createTopicWithKafkaGo(brokerAddr, topicName string) error {
	conn, err := kafka.Dial("tcp", brokerAddr)
	if err != nil {
		return fmt.Errorf("failed to dial kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to dial controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topicName,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	return controllerConn.CreateTopics(topicConfigs...)
}

func TestSimpleKafkaIntegration(t *testing.T) {
	if !isContainerRuntimeAvailable() {
		t.Skip("Container runtime not available")
	}

	// 设置环境变量禁用Ryuk
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx := context.Background()

	// 启动Kafka容器
	kafkaContainer, err := kafkaTC.RunContainer(ctx,
		kafkaTC.WithClusterID("test-cluster"),
		testcontainers.WithImage("confluentinc/confluent-local:7.5.0"),
	)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, kafkaContainer.Terminate(ctx))
	}()

	// 获取broker地址
	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers)

	brokerAddr := brokers[0]
	t.Logf("Kafka broker: %s", brokerAddr)

	// 等待Kafka启动
	time.Sleep(15 * time.Second)

	// 创建测试topic
	topicName := "integration-test-topic"
	err = createTopicWithKafkaGo(brokerAddr, topicName)
	if err != nil {
		t.Logf("Failed to create topic: %v", err)
		// 继续测试，可能Kafka会自动创建topic
	}

	// 等待topic创建
	time.Sleep(5 * time.Second)

	t.Run("TestKafkaAdapter", func(t *testing.T) {
		// 创建KafkaAdapter配置
		config := eventstream.KafkaConfig{
			Brokers: brokers,
			Producer: eventstream.KafkaProducerConfig{
				BatchSize:    1,
				BatchTimeout: 100 * time.Millisecond,
				RequiredAcks: 1,
			},
			Consumer: eventstream.KafkaConsumerConfig{
				StartOffset:    "earliest",
				CommitInterval: 1 * time.Second,
				MaxWait:        1 * time.Second,
				MinBytes:       1,
				MaxBytes:       10e6,
			},
		}

		adapter, err := eventstream.NewKafkaAdapter(config)
		require.NoError(t, err)
		defer adapter.Close()

		// 测试基本的发布订阅
		testData := map[string]interface{}{
			"message":   "Hello Kafka Integration Test",
			"timestamp": time.Now().Unix(),
		}

		event := &eventstream.Event{
			ID:    "test-event-1",
			Topic: topicName,
			Data:  testData,
		}

		// 先订阅消息
		msgChan, closeFunc, err := adapter.Subscribe(ctx, topicName, "test-group")
		require.NoError(t, err)
		defer closeFunc()

		// 等待订阅生效
		time.Sleep(3 * time.Second)

		// 序列化事件数据
		eventData, err := json.Marshal(event)
		require.NoError(t, err)

		// 发布消息
		err = adapter.Publish(ctx, topicName, eventData)
		require.NoError(t, err)

		// 等待接收消息
		select {
		case msg := <-msgChan:
			t.Logf("Successfully received message: %s", string(msg.Value()))

			// 验证消息内容
			var receivedEvent eventstream.Event
			err = json.Unmarshal(msg.Value(), &receivedEvent)
			require.NoError(t, err)

			assert.Equal(t, event.ID, receivedEvent.ID)
			assert.Equal(t, event.Topic, receivedEvent.Topic)
		case <-time.After(20 * time.Second):
			t.Fatal("Timeout waiting for message")
		}
	})

	t.Run("TestDistributedEventBus", func(t *testing.T) {
		// 由于需要实际的KafkaAdapter实例，我们先跳过这个测试
		t.Skip("DistributedEventBus test requires proper adapter setup")
	})
}
