package eventstream

import (
	"context"
	"encoding/json"
	"time"
)

// MQAdapter 是一个接口，定义了分布式事件总线所需的消息队列功能。
type MQAdapter interface {
	// Publish 将消息的原始载荷（payload）发布到指定主题。
	// ctx: 用于控制请求的上下文，例如超时或取消。
	// topic: 消息要发布到的主题名称。
	// payload: 要发布的消息的原始载荷（字节切片）。
	Publish(ctx context.Context, topic string, payload []byte) error

	// Subscribe 订阅一个主题以消费消息。
	// ctx: 用于控制订阅生命周期的上下文。
	// topic: 要订阅的主题名称。
	// groupID: 消费者组的ID，用于负载均衡和状态管理。
	// 返回一个只读的消息通道，一个用于关闭订阅的函数，以及可能发生的错误。
	Subscribe(ctx context.Context, topic, groupID string) (<-chan Message, func(), error)

	// Ack 用于在手动模式下确认消息已被成功处理。
	// ctx: 用于控制确认操作的上下文。
	// groupID: 执行确认操作的消费者组ID。
	// msg: 需要被确认的消息。
	Ack(ctx context.Context, groupID string, msg Message) error

	// Close 关闭与消息队列的所有连接。
	Close() error
}

// Message 代表从消息队列中消费的单条消息。
type Message interface {
	// Topic 返回消息所属的主题。
	Topic() string

	// Value 返回消息的原始负载。
	Value() []byte

	// Timestamp 返回消息的时间戳。
	Timestamp() time.Time

	// Key 返回与消息关联的键。
	Key() []byte
}

// EventSerializer 定义了在事件对象和字节切片之间进行序列化和反序列化的接口。
type EventSerializer interface {
	// Serialize 将事件对象编码为字节切片。
	Serialize(event *Event) ([]byte, error)

	// Deserialize 将字节切片解码为事件对象。
	Deserialize(data []byte) (*Event, error)
}

// DefaultEventSerializer 是一个使用标准JSON进行序列化和反序列化的默认实现。
type DefaultEventSerializer struct{}

// Serialize 使用JSON编码事件。
func (s *DefaultEventSerializer) Serialize(event *Event) ([]byte, error) {
	return json.Marshal(event)
}

// Deserialize 使用JSON解码事件。
func (s *DefaultEventSerializer) Deserialize(data []byte) (*Event, error) {
	event := &Event{}
	err := json.Unmarshal(data, event)
	return event, err
}
