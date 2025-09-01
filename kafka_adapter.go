package eventstream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaConfig 保存所有 Kafka 特定的配置。
type KafkaConfig struct {
	Brokers  []string
	Producer KafkaProducerConfig
	Consumer KafkaConsumerConfig
}

// KafkaProducerConfig 保存 Kafka 生产者特定的配置。
type KafkaProducerConfig struct {
	BatchSize    int
	BatchTimeout time.Duration
	Compression  string // "gzip", "snappy", "lz4", "zstd"
	RequiredAcks int
}

// KafkaConsumerConfig 保存 Kafka 消费者特定的配置。
type KafkaConsumerConfig struct {
	StartOffset    string // "earliest", "latest"
	CommitInterval time.Duration
	MaxWait        time.Duration
	MinBytes       int
	MaxBytes       int
}

// KafkaAdapter 是用于 Apache Kafka 的 MQAdapter 实现。
type KafkaAdapter struct {
	config   KafkaConfig
	producer *kafka.Writer
	readers  map[string]*kafka.Reader // 每个消费者组一个 reader
	mu       sync.Mutex
}

// NewKafkaAdapter 创建一个新的 Kafka 适配器。
func NewKafkaAdapter(config KafkaConfig) (*KafkaAdapter, error) {
	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers cannot be empty")
	}

	producer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		BatchSize:    config.Producer.BatchSize,
		BatchTimeout: config.Producer.BatchTimeout,
		Compression:  getKafkaCompression(config.Producer.Compression),
		RequiredAcks: kafka.RequiredAcks(config.Producer.RequiredAcks),
	}

	return &KafkaAdapter{
		config:   config,
		producer: producer,
		readers:  make(map[string]*kafka.Reader),
	}, nil
}

// Publish 实现 MQAdapter 接口。
func (a *KafkaAdapter) Publish(ctx context.Context, topic string, message []byte) error {
	return a.producer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: message,
	})
}

// Subscribe 实现 MQAdapter 接口。
func (a *KafkaAdapter) Subscribe(ctx context.Context, topic, groupID string) (<-chan Message, func(), error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	readerKey := fmt.Sprintf("%s-%s", topic, groupID)
	reader, exists := a.readers[readerKey]
	if !exists {
		reader = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        a.config.Brokers,
			Topic:          topic,
			GroupID:        groupID,
			StartOffset:    getKafkaStartOffset(a.config.Consumer.StartOffset),
			CommitInterval: a.config.Consumer.CommitInterval,
			MaxWait:        a.config.Consumer.MaxWait,
			MinBytes:       a.config.Consumer.MinBytes,
			MaxBytes:       a.config.Consumer.MaxBytes,
		})
		a.readers[readerKey] = reader
	}

	msgChan := make(chan Message, 100)
	stopChan := make(chan struct{})

	go func() {
		defer close(msgChan)
		for {
			select {
			case <-stopChan:
				return
			default:
				msg, err := reader.FetchMessage(ctx)
				if err != nil {
					return
				}
				msgChan <- &kafkaMessage{msg: msg}
			}
		}
	}()

	closeFunc := func() {
		close(stopChan)
	}

	return msgChan, closeFunc, nil
}

// Ack 实现 MQAdapter 接口。
func (a *KafkaAdapter) Ack(ctx context.Context, groupID string, msg Message) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	readerKey := fmt.Sprintf("%s-%s", msg.Topic(), groupID)
	reader, exists := a.readers[readerKey]
	if !exists {
		return fmt.Errorf("no reader found for topic %s and group %s", msg.Topic(), groupID)
	}

	kmsg, ok := msg.(*kafkaMessage)
	if !ok {
		return fmt.Errorf("invalid message type for kafka adapter")
	}

	return reader.CommitMessages(ctx, kmsg.msg)
}

// Close 实现 MQAdapter 接口。
func (a *KafkaAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.producer.Close(); err != nil {
		return err
	}
	for _, reader := range a.readers {
		if err := reader.Close(); err != nil {
			fmt.Printf("Error closing Kafka reader: %v\n", err)
		}
	}
	return nil
}

// kafkaMessage 是 kafka.Message 的包装器。
type kafkaMessage struct {
	msg kafka.Message
}

func (m *kafkaMessage) Topic() string        { return m.msg.Topic }
func (m *kafkaMessage) Value() []byte        { return m.msg.Value }
func (m *kafkaMessage) Timestamp() time.Time { return m.msg.Time }
func (m *kafkaMessage) Key() []byte          { return m.msg.Key }

func getKafkaCompression(c string) kafka.Compression {
	switch c {
	case "gzip":
		return kafka.Gzip
	case "snappy":
		return kafka.Snappy
	case "lz4":
		return kafka.Lz4
	case "zstd":
		return kafka.Zstd
	default:
		return 0 // kafka.None 的值是 0
	}
}

func getKafkaStartOffset(offset string) int64 {
	if offset == "earliest" {
		return kafka.FirstOffset
	}
	return kafka.LastOffset
}
