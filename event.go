package eventstream

import (
	"encoding/json"
	"time"
)

// Event 事件数据结构
type Event struct {
	// ID 事件唯一标识
	ID string `json:"id"`

	// Topic 事件主题
	Topic string `json:"topic"`

	// Data 事件数据
	Data interface{} `json:"data"`

	// Timestamp 事件时间戳
	Timestamp time.Time `json:"timestamp"`

	// Headers 事件头信息
	Headers map[string]interface{} `json:"headers,omitempty"`

	// Source 事件源标识
	Source string `json:"source,omitempty"`

	// Version 事件版本
	Version string `json:"version,omitempty"`
}

// NewEvent 创建新事件
func NewEvent(topic string, data interface{}) *Event {
	return &Event{
		ID:        generateEventID(),
		Topic:     topic,
		Data:      data,
		Timestamp: time.Now(),
		Headers:   make(map[string]interface{}),
		Version:   "1.0",
	}
}

// WithHeader 添加头信息
func (e *Event) WithHeader(key string, value interface{}) *Event {
	if e.Headers == nil {
		e.Headers = make(map[string]interface{})
	}
	e.Headers[key] = value
	return e
}

// GetHeader 获取头信息
func (e *Event) GetHeader(key string) (interface{}, bool) {
	if e.Headers == nil {
		return nil, false
	}
	value, exists := e.Headers[key]
	return value, exists
}

// WithSource 设置事件源
func (e *Event) WithSource(source string) *Event {
	e.Source = source
	return e
}

// WithVersion 设置事件版本
func (e *Event) WithVersion(version string) *Event {
	e.Version = version
	return e
}

// Unmarshal 将事件数据反序列化到指定结构
func (e *Event) Unmarshal(v interface{}) error {
	data, err := json.Marshal(e.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Marshal 将事件序列化为字节数组
func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalEvent 从字节数组反序列化事件
func UnmarshalEvent(data []byte) (*Event, error) {
	var event Event
	err := json.Unmarshal(data, &event)
	return &event, err
}
