package eventstream

import (
	"context"
	"testing"
)

// mockMQAdapter 用于测试的简单模拟适配器
type mockMQAdapter struct{}

func (m *mockMQAdapter) Publish(ctx context.Context, topic string, message []byte) error {
	return nil
}

func (m *mockMQAdapter) Subscribe(ctx context.Context, topic, groupID string) (<-chan Message, func(), error) {
	ch := make(chan Message)
	close(ch)
	return ch, func() {}, nil
}

func (m *mockMQAdapter) Ack(ctx context.Context, groupID string, msg Message) error {
	return nil
}

func (m *mockMQAdapter) Close() error {
	return nil
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid memory config",
			config: &Config{
				Mode: ModeMemory,
				Pool: PoolConfig{Size: 100},
				Memory: &MemoryConfig{
					BufferSize:     1000,
					EnableHistory:  true,
					MaxHistorySize: 100,
				},
			},
			wantErr: false,
		},
		{
			name: "valid distributed config",
			config: &Config{
				Mode: ModeDistributed,
				Pool: PoolConfig{Size: 100},
				Distributed: &DistributedConfig{
					MQAdapter: &mockMQAdapter{},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: &Config{
				Mode: "invalid",
				Pool: PoolConfig{Size: 100},
			},
			wantErr: true,
		},
		{
			name: "zero pool size",
			config: &Config{
				Mode: ModeMemory,
				Pool: PoolConfig{Size: 0},
			},
			wantErr: true,
		},
		{
			name: "distributed without adapter",
			config: &Config{
				Mode: ModeDistributed,
				Pool: PoolConfig{Size: 100},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *MemoryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &MemoryConfig{
				BufferSize:     1000,
				EnableHistory:  true,
				MaxHistorySize: 100,
			},
			wantErr: false,
		},
		{
			name: "negative buffer size",
			config: &MemoryConfig{
				BufferSize: -1,
			},
			wantErr: true,
		},
		{
			name: "history enabled but zero max size",
			config: &MemoryConfig{
				EnableHistory:  true,
				MaxHistorySize: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("MemoryConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Mode != ModeMemory {
		t.Errorf("Expected mode %s, got %s", ModeMemory, config.Mode)
	}

	if config.Pool.Size <= 0 {
		t.Error("Pool size should be greater than 0")
	}

	if config.Memory == nil {
		t.Error("Memory config should not be nil")
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}
}

func TestDefaultDistributedConfig(t *testing.T) {
	config := DefaultDistributedConfig()

	if config.Mode != ModeDistributed {
		t.Errorf("Expected mode %s, got %s", ModeDistributed, config.Mode)
	}

	if config.Distributed == nil {
		t.Error("Distributed config should not be nil")
	}

	if config.Distributed.Serializer == nil {
		t.Error("Serializer should not be nil")
	}

	if config.Memory != nil {
		t.Error("Memory config should be nil for distributed mode")
	}
}

func TestDefaultSubscribeConfig(t *testing.T) {
	config := DefaultSubscribeConfig()

	if config.ConsumerGroup == "" {
		t.Error("Consumer group should not be empty")
	}

	if config.Concurrency <= 0 {
		t.Error("Concurrency should be greater than 0")
	}

	if config.RetryPolicy == nil {
		t.Error("Retry policy should not be nil")
	}

	if config.RetryPolicy.MaxRetries <= 0 {
		t.Error("Max retries should be greater than 0")
	}
}
