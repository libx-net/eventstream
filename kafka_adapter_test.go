package eventstream

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKafkaWriter 是 KafkaWriter 的模拟实现。
type MockKafkaWriter struct {
	mu              sync.Mutex
	WriteMessagesFn func(ctx context.Context, msgs ...kafka.Message) error
	CloseFn         func() error
	WrittenMessages []kafka.Message
}

func (m *MockKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WrittenMessages = append(m.WrittenMessages, msgs...)
	if m.WriteMessagesFn != nil {
		return m.WriteMessagesFn(ctx, msgs...)
	}
	return nil
}

func (m *MockKafkaWriter) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

// MockKafkaReader 是 KafkaReader 的模拟实现。
type MockKafkaReader struct {
	FetchMessageFn   func(ctx context.Context) (kafka.Message, error)
	CommitMessagesFn func(ctx context.Context, msgs ...kafka.Message) error
	CloseFn          func() error
}

func (m *MockKafkaReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if m.FetchMessageFn != nil {
		return m.FetchMessageFn(ctx)
	}
	return kafka.Message{}, errors.New("FetchMessageFn not implemented")
}

func (m *MockKafkaReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	if m.CommitMessagesFn != nil {
		return m.CommitMessagesFn(ctx, msgs...)
	}
	return nil
}

func (m *MockKafkaReader) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

func TestKafkaAdapter_Publish(t *testing.T) {
	mockWriter := &MockKafkaWriter{}
	adapter := &KafkaAdapter{
		producer: mockWriter,
	}

	topic := "test-topic"
	message := []byte("hello world")

	err := adapter.Publish(context.Background(), topic, message)
	require.NoError(t, err)

	mockWriter.mu.Lock()
	defer mockWriter.mu.Unlock()

	require.Len(t, mockWriter.WrittenMessages, 1)
	assert.Equal(t, topic, mockWriter.WrittenMessages[0].Topic)
	assert.Equal(t, message, mockWriter.WrittenMessages[0].Value)
}

func TestKafkaAdapter_Subscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockReader := &MockKafkaReader{}
	adapter := &KafkaAdapter{
		readers: make(map[string]KafkaReader),
	}
	adapter.newReaderFunc = func(cfg kafka.ReaderConfig) KafkaReader {
		return mockReader
	}

	topic := "test-topic"
	groupID := "test-group"
	expectedMsg := kafka.Message{
		Topic: topic,
		Value: []byte("hello from kafka"),
		Time:  time.Now(),
	}

	// 设置模拟，使其返回一条消息，然后阻塞
	fetchCalled := make(chan struct{})
	mockReader.FetchMessageFn = func(ctx context.Context) (kafka.Message, error) {
		select {
		case <-fetchCalled:
			// 在后续调用中永久阻塞，以防止在测试中出现繁忙循环
			<-ctx.Done()
			return kafka.Message{}, ctx.Err()
		default:
			close(fetchCalled)
			return expectedMsg, nil
		}
	}

	msgChan, closeFunc, err := adapter.Subscribe(ctx, topic, groupID)
	require.NoError(t, err)
	defer closeFunc()

	select {
	case receivedMsg := <-msgChan:
		assert.Equal(t, expectedMsg.Topic, receivedMsg.Topic())
		assert.Equal(t, expectedMsg.Value, receivedMsg.Value())
		assert.Equal(t, expectedMsg.Time.Unix(), receivedMsg.Timestamp().Unix())
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestKafkaAdapter_Ack(t *testing.T) {
	mockReader := &MockKafkaReader{}
	adapter := &KafkaAdapter{
		readers: make(map[string]KafkaReader),
	}

	topic := "test-topic"
	groupID := "test-group"
	readerKey := topic + "-" + groupID
	adapter.readers[readerKey] = mockReader

	originalMsg := kafka.Message{
		Topic:     topic,
		Partition: 1,
		Offset:    10,
	}
	msgToAck := &kafkaMessage{msg: originalMsg}

	var committedMsgs []kafka.Message
	commitCalled := make(chan struct{})
	mockReader.CommitMessagesFn = func(ctx context.Context, msgs ...kafka.Message) error {
		committedMsgs = msgs
		close(commitCalled)
		return nil
	}

	err := adapter.Ack(context.Background(), groupID, msgToAck)
	require.NoError(t, err)

	select {
	case <-commitCalled:
		require.Len(t, committedMsgs, 1)
		assert.Equal(t, originalMsg, committedMsgs[0])
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for CommitMessages to be called")
	}
}

func TestKafkaAdapter_Close(t *testing.T) {
	mockWriter := &MockKafkaWriter{}
	mockReader1 := &MockKafkaReader{}
	mockReader2 := &MockKafkaReader{}

	adapter := &KafkaAdapter{
		producer: mockWriter,
		readers: map[string]KafkaReader{
			"r1": mockReader1,
			"r2": mockReader2,
		},
	}

	var writerClosed, reader1Closed, reader2Closed bool
	mockWriter.CloseFn = func() error {
		writerClosed = true
		return nil
	}
	mockReader1.CloseFn = func() error {
		reader1Closed = true
		return nil
	}
	mockReader2.CloseFn = func() error {
		reader2Closed = true
		return nil
	}

	err := adapter.Close()
	require.NoError(t, err)

	assert.True(t, writerClosed, "writer should be closed")
	assert.True(t, reader1Closed, "reader1 should be closed")
	assert.True(t, reader2Closed, "reader2 should be closed")
}
