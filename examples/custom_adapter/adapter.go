package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"libx.net/eventstream"
)

// RabbitMQConfig holds RabbitMQ specific configuration.
type RabbitMQConfig struct {
	URL          string
	ExchangeName string
	ExchangeType string // "direct", "fanout", "topic", "headers"
}

// RabbitMQAdapter is an MQAdapter implementation for RabbitMQ.
type RabbitMQAdapter struct {
	config    RabbitMQConfig
	conn      *amqp.Connection
	publishCh *amqp.Channel
	mu        sync.Mutex
}

// NewRabbitMQAdapter creates a new RabbitMQ adapter.
func NewRabbitMQAdapter(config RabbitMQConfig) (*RabbitMQAdapter, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("rabbitmq URL cannot be empty")
	}
	if config.ExchangeName == "" {
		config.ExchangeName = "eventstream"
	}
	if config.ExchangeType == "" {
		config.ExchangeType = "topic"
	}

	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	publishCh, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	err = publishCh.ExchangeDeclare(
		config.ExchangeName, // name
		config.ExchangeType, // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		publishCh.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare an exchange: %w", err)
	}

	return &RabbitMQAdapter{
		config:    config,
		conn:      conn,
		publishCh: publishCh,
	}, nil
}

// Publish implements the MQAdapter interface.
func (a *RabbitMQAdapter) Publish(ctx context.Context, topic string, payload []byte) error {
	return a.publishCh.PublishWithContext(ctx,
		a.config.ExchangeName, // exchange
		topic,                 // routing key
		false,                 // mandatory
		false,                 // immediate
		amqp.Publishing{
			ContentType: "application/octet-stream",
			Body:        payload,
			Timestamp:   time.Now(),
		})
}

// Subscribe implements the MQAdapter interface.
func (a *RabbitMQAdapter) Subscribe(ctx context.Context, topic, groupID string) (<-chan eventstream.Message, func(), error) {
	ch, err := a.conn.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	queueName := fmt.Sprintf("%s-%s", topic, groupID)
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		return nil, nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	err = ch.QueueBind(
		q.Name,                // queue name
		topic,                 // routing key
		a.config.ExchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		return nil, nil, fmt.Errorf("failed to bind a queue: %w", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		ch.Close()
		return nil, nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	deliveries, err := ch.Consume(
		q.Name,  // queue
		groupID, // consumer
		false,   // auto-ack
		false,   // exclusive
		false,   // no-local
		false,   // no-wait
		nil,     // args
	)
	if err != nil {
		ch.Close()
		return nil, nil, fmt.Errorf("failed to register a consumer: %w", err)
	}

	msgChan := make(chan eventstream.Message)
	stopChan := make(chan struct{})

	go func() {
		defer close(msgChan)
		for {
			select {
			case <-ctx.Done():
				ch.Close()
				return
			case <-stopChan:
				ch.Close()
				return
			case d, ok := <-deliveries:
				if !ok {
					return
				}
				msgChan <- &rabbitmqMessage{delivery: d}
			}
		}
	}()

	closeFunc := func() {
		close(stopChan)
	}

	return msgChan, closeFunc, nil
}

// Ack implements the MQAdapter interface.
func (a *RabbitMQAdapter) Ack(ctx context.Context, groupID string, msg eventstream.Message) error {
	rmqMsg, ok := msg.(*rabbitmqMessage)
	if !ok {
		return fmt.Errorf("invalid message type for rabbitmq adapter")
	}
	return rmqMsg.delivery.Ack(false)
}

// Close implements the MQAdapter interface.
func (a *RabbitMQAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

// rabbitmqMessage is a wrapper for amqp.Delivery.
type rabbitmqMessage struct {
	delivery amqp.Delivery
}

func (m *rabbitmqMessage) Topic() string {
	return m.delivery.RoutingKey
}

func (m *rabbitmqMessage) Value() []byte {
	return m.delivery.Body
}

func (m *rabbitmqMessage) Timestamp() time.Time {
	return m.delivery.Timestamp
}

func (m *rabbitmqMessage) Key() []byte {
	return []byte(m.delivery.MessageId)
}