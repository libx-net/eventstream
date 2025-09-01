package eventstream

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"time"
)

// generateEventID 生成事件唯一ID
func generateEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// 如果随机数生成失败，使用时间戳作为后备方案
		return fmt.Sprintf("event_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// calculateBackoff 计算退避延迟
func calculateBackoff(policy *RetryPolicy, attempt int) time.Duration {
	if policy == nil {
		return time.Second
	}

	var delay time.Duration

	switch policy.BackoffStrategy {
	case BackoffFixed:
		delay = policy.InitialDelay
	case BackoffExponential:
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

// safeClose 安全关闭channel
func safeClose(ch chan struct{}) {
	select {
	case <-ch:
		// channel已经关闭
	default:
		close(ch)
	}
}

// copyHeaders 复制事件头信息
func copyHeaders(headers map[string]interface{}) map[string]interface{} {
	if headers == nil {
		return nil
	}

	copied := make(map[string]interface{}, len(headers))
	for k, v := range headers {
		copied[k] = v
	}
	return copied
}

// validateTopic 验证主题名称
func validateTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	// 可以添加更多的主题验证规则
	// 例如：长度限制、字符限制等
	if len(topic) > 255 {
		return fmt.Errorf("topic name too long (max 255 characters)")
	}

	return nil
}

// mergeSubscribeOptions 合并订阅选项
func mergeSubscribeOptions(opts []SubscribeOption) *SubscribeConfig {
	config := DefaultSubscribeConfig()

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// generateSubscriptionID 生成订阅唯一ID
func generateSubscriptionID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("sub_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
