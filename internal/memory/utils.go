package memory

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"libx.net/eventstream"
)

// generateSubscriptionID 生成订阅唯一ID
func generateSubscriptionID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("sub_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// calculateBackoff 计算退避延迟
func calculateBackoff(policy *eventstream.RetryPolicy, attempt int) time.Duration {
	if policy == nil {
		return time.Second
	}

	var delay time.Duration

	switch policy.BackoffStrategy {
	case "fixed":
		delay = policy.InitialDelay
	case "exponential":
		delay = time.Duration(float64(policy.InitialDelay) * math.Pow(policy.Multiplier, float64(attempt)))
	default:
		delay = policy.InitialDelay
	}

	// 限制最大延迟
	if delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}

	return delay
}
