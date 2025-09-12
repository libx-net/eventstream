package eventstream

import (
	"testing"
	"time"
)

func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	id2 := generateEventID()

	if id1 == "" {
		t.Error("Generated ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if len(id1) != 32 { // hex encoded 16 bytes = 32 characters
		t.Errorf("Expected ID length 32, got %d", len(id1))
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name     string
		policy   *RetryPolicy
		attempt  int
		expected time.Duration
	}{
		{
			name: "fixed backoff",
			policy: &RetryPolicy{
				BackoffStrategy: BackoffFixed,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        time.Second,
			},
			attempt:  1,
			expected: 100 * time.Millisecond,
		},
		{
			name: "exponential backoff",
			policy: &RetryPolicy{
				BackoffStrategy: BackoffExponential,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        time.Second,
				Multiplier:      2.0,
			},
			attempt:  1,
			expected: 200 * time.Millisecond,
		},
		{
			name: "exponential backoff with max delay",
			policy: &RetryPolicy{
				BackoffStrategy: BackoffExponential,
				InitialDelay:    100 * time.Millisecond,
				MaxDelay:        300 * time.Millisecond,
				Multiplier:      2.0,
			},
			attempt:  3,
			expected: 300 * time.Millisecond,
		},
		{
			name:     "nil policy",
			policy:   nil,
			attempt:  1,
			expected: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBackoff(tt.policy, tt.attempt)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSafeClose(t *testing.T) {
	// Test closing an open channel
	ch1 := make(chan struct{})
	safeClose(ch1)

	// Test closing an already closed channel (should not panic)
	safeClose(ch1)
}

func TestCopyHeaders(t *testing.T) {
	// Test nil headers
	result := copyHeaders(nil)
	if result != nil {
		t.Error("Expected nil result for nil input")
	}

	// Test copying headers
	original := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	copied := copyHeaders(original)

	if len(copied) != len(original) {
		t.Errorf("Expected length %d, got %d", len(original), len(copied))
	}

	for k, v := range original {
		if copied[k] != v {
			t.Errorf("Expected %v for key %s, got %v", v, k, copied[k])
		}
	}

	// Modify original to ensure it's a deep copy
	original["key1"] = "modified"
	if copied["key1"] == "modified" {
		t.Error("Headers should be deep copied")
	}
}

func TestValidateTopic(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		wantErr bool
	}{
		{
			name:    "valid topic",
			topic:   "valid.topic",
			wantErr: false,
		},
		{
			name:    "empty topic",
			topic:   "",
			wantErr: true,
		},
		{
			name:    "too long topic",
			topic:   string(make([]byte, 256)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTopic(tt.topic)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTopic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMergeSubscribeOptions(t *testing.T) {
	opts := []SubscribeOption{
		WithConcurrency(5),
		WithBufferSize(200),
	}

	config := mergeSubscribeOptions(opts)

	if config.Concurrency != 5 {
		t.Errorf("Expected concurrency 5, got %d", config.Concurrency)
	}

	if config.BufferSize != 200 {
		t.Errorf("Expected buffer size 200, got %d", config.BufferSize)
	}
}

func TestGenerateSubscriptionID(t *testing.T) {
	id1 := generateSubscriptionID()
	id2 := generateSubscriptionID()

	if id1 == "" {
		t.Error("Generated subscription ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Generated subscription IDs should be unique")
	}

	if len(id1) != 16 { // hex encoded 8 bytes = 16 characters
		t.Errorf("Expected subscription ID length 16, got %d", len(id1))
	}
}

func TestMinMax(t *testing.T) {
	if min(5, 3) != 3 {
		t.Error("min(5, 3) should return 3")
	}

	if min(2, 7) != 2 {
		t.Error("min(2, 7) should return 2")
	}

	if max(5, 3) != 5 {
		t.Error("max(5, 3) should return 5")
	}

	if max(2, 7) != 7 {
		t.Error("max(2, 7) should return 7")
	}
}
