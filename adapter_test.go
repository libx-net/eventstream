package eventstream

import (
	"testing"
	"time"
)

func TestEvent_MarshalJSON(t *testing.T) {
	event := &Event{
		ID:        "test-id",
		Topic:     "test.topic",
		Type:      "test.event",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"source": "test"},
	}

	data, err := event.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	if len(data) == 0 {
		t.Error("Marshaled data should not be empty")
	}
}

func TestEvent_UnmarshalEvent(t *testing.T) {
	jsonData := `{
		"id": "test-id",
		"topic": "test.topic",
		"type": "test.event",
		"data": {"key": "value"},
		"timestamp": "2023-01-01T00:00:00Z",
		"metadata": {"source": "test"}
	}`

	event, err := UnmarshalEvent([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if event.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", event.ID)
	}

	if event.Topic != "test.topic" {
		t.Errorf("Expected Topic 'test.topic', got '%s'", event.Topic)
	}
}

func TestDefaultEventSerializer_Serialize(t *testing.T) {
	serializer := &DefaultEventSerializer{}
	event := &Event{
		ID:        "test-id",
		Topic:     "test.topic",
		Type:      "test.event",
		Data:      "test data",
		Timestamp: time.Now(),
	}

	data, err := serializer.Serialize(event)
	if err != nil {
		t.Fatalf("Failed to serialize event: %v", err)
	}

	if len(data) == 0 {
		t.Error("Serialized data should not be empty")
	}
}

func TestDefaultEventSerializer_Deserialize(t *testing.T) {
	serializer := &DefaultEventSerializer{}

	// 先序列化一个事件
	originalEvent := &Event{
		ID:        "test-id",
		Topic:     "test.topic",
		Type:      "test.event",
		Data:      "test data",
		Timestamp: time.Now(),
	}

	data, err := serializer.Serialize(originalEvent)
	if err != nil {
		t.Fatalf("Failed to serialize event: %v", err)
	}

	// 然后反序列化
	deserializedEvent, err := serializer.Deserialize(data)
	if err != nil {
		t.Fatalf("Failed to deserialize event: %v", err)
	}

	if deserializedEvent.ID != originalEvent.ID {
		t.Errorf("Expected ID '%s', got '%s'", originalEvent.ID, deserializedEvent.ID)
	}

	if deserializedEvent.Topic != originalEvent.Topic {
		t.Errorf("Expected Topic '%s', got '%s'", originalEvent.Topic, deserializedEvent.Topic)
	}
}

func TestEvent_WithHeader(t *testing.T) {
	event := NewEvent("test.topic", "test data")

	event.WithHeader("key1", "value1").WithHeader("key2", "value2")

	if val, exists := event.GetHeader("key1"); !exists || val != "value1" {
		t.Errorf("Expected header 'key1' to be 'value1', got %v", val)
	}

	if val, exists := event.GetHeader("key2"); !exists || val != "value2" {
		t.Errorf("Expected header 'key2' to be 'value2', got %v", val)
	}
}

func TestEvent_WithSource(t *testing.T) {
	event := NewEvent("test.topic", "test data")
	event.WithSource("test-service")

	if event.Source != "test-service" {
		t.Errorf("Expected source 'test-service', got '%s'", event.Source)
	}
}

func TestEvent_WithVersion(t *testing.T) {
	event := NewEvent("test.topic", "test data")
	event.WithVersion("2.0")

	if event.Version != "2.0" {
		t.Errorf("Expected version '2.0', got '%s'", event.Version)
	}
}

func TestEvent_Unmarshal(t *testing.T) {
	type TestData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	testData := TestData{Name: "John", Age: 30}
	event := NewEvent("test.topic", testData)

	var result TestData
	err := event.Unmarshal(&result)
	if err != nil {
		t.Fatalf("Failed to unmarshal event data: %v", err)
	}

	if result.Name != testData.Name {
		t.Errorf("Expected name '%s', got '%s'", testData.Name, result.Name)
	}

	if result.Age != testData.Age {
		t.Errorf("Expected age %d, got %d", testData.Age, result.Age)
	}
}
