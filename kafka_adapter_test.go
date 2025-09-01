package eventstream

import (
	"testing"
	"time"
)

func TestKafkaConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  KafkaConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: KafkaConfig{
				Brokers: []string{"localhost:9092"},
				Producer: KafkaProducerConfig{
					BatchSize:    100,
					BatchTimeout: time.Second,
					Compression:  "gzip",
					RequiredAcks: 1,
				},
				Consumer: KafkaConsumerConfig{
					StartOffset:    "latest",
					CommitInterval: time.Second,
					MaxWait:        time.Second,
					MinBytes:       1,
					MaxBytes:       1024,
				},
			},
			wantErr: false,
		},
		{
			name: "empty brokers",
			config: KafkaConfig{
				Brokers: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewKafkaAdapter(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKafkaAdapter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetKafkaCompression(t *testing.T) {
	tests := []struct {
		input    string
		expected int // 使用int因为kafka.Compression是int类型
	}{
		{"gzip", 1},
		{"snappy", 2},
		{"lz4", 3},
		{"zstd", 4},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getKafkaCompression(tt.input)
			if int(result) != tt.expected {
				t.Errorf("getKafkaCompression(%s) = %d, expected %d", tt.input, int(result), tt.expected)
			}
		})
	}
}

func TestGetKafkaStartOffset(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"earliest", -2}, // kafka.FirstOffset
		{"latest", -1},   // kafka.LastOffset
		{"unknown", -1},  // 默认为LastOffset
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getKafkaStartOffset(tt.input)
			if result != tt.expected {
				t.Errorf("getKafkaStartOffset(%s) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}
