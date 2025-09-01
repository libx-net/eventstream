package eventstream

import (
	"errors"
	"time"
)

// 模式常量
const (
	ModeMemory      = "memory"
	ModeDistributed = "distributed"
)

// 重试策略常量
const (
	BackoffFixed       = "fixed"
	BackoffExponential = "exponential"
)

// Config EventBus配置
type Config struct {
	// Mode 运行模式: "memory" 或 "distributed"
	Mode string `json:"mode"`

	// Pool 协程池配置
	Pool PoolConfig `json:"pool"`

	// Memory 内存模式配置
	Memory *MemoryConfig `json:"memory,omitempty"`

	// Distributed 分布式模式配置
	Distributed *DistributedConfig `json:"distributed,omitempty"`

	// Logger 日志配置
	Logger LoggerConfig `json:"logger"`
}

// PoolConfig 协程池配置
type PoolConfig struct {
	// Size 协程池大小
	Size int `json:"size"`

	// ExpiryDuration 协程过期时间
	ExpiryDuration time.Duration `json:"expiry_duration"`

	// PreAlloc 是否预分配协程
	PreAlloc bool `json:"pre_alloc"`
}

// MemoryConfig 内存模式配置
type MemoryConfig struct {
	// BufferSize 事件缓冲区大小
	BufferSize int `json:"buffer_size"`

	// EnableHistory 是否启用事件历史记录
	EnableHistory bool `json:"enable_history"`

	// MaxHistorySize 历史记录最大数量
	MaxHistorySize int `json:"max_history_size"`

	// EnableMetrics 是否启用指标收集
	EnableMetrics bool `json:"enable_metrics"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	// Level 日志级别: "debug", "info", "warn", "error"
	Level string `json:"level"`

	// Format 日志格式: "json", "text"
	Format string `json:"format"`

	// Output 输出目标: "stdout", "stderr", "file"
	Output string `json:"output"`

	// FilePath 文件路径 (当Output为"file"时)
	FilePath string `json:"file_path,omitempty"`
}

// DistributedConfig 分布式模式配置
type DistributedConfig struct {
	// MQAdapter 是要使用的消息队列适配器。这是必填字段。
	MQAdapter MQAdapter `json:"-"`

	// EnableMetrics 控制是否为此事件总线实例收集指标。
	EnableMetrics bool `json:"enable_metrics"`

	// Serializer 是用于编码和解码事件的序列化器。
	// 如果未提供，将使用默认的JSON序列化器。
	Serializer EventSerializer `json:"-"`
}

// SubscribeConfig 订阅配置
type SubscribeConfig struct {
	// ConsumerGroup 消费者组 (分布式模式必需，内存模式忽略)
	ConsumerGroup string `json:"consumer_group"`

	// Concurrency 并发处理数量
	Concurrency int `json:"concurrency"`

	// RetryPolicy 重试策略
	RetryPolicy *RetryPolicy `json:"retry_policy,omitempty"`

	// AutoCommit 是否自动提交offset (仅分布式模式)
	AutoCommit bool `json:"auto_commit"`

	// BufferSize 缓冲区大小
	BufferSize int `json:"buffer_size"`

	// StartOffset 起始偏移量 (仅分布式模式): "earliest", "latest"
	StartOffset string `json:"start_offset,omitempty"`

	// CommitInterval 提交间隔 (仅分布式模式)
	CommitInterval time.Duration `json:"commit_interval,omitempty"`

	// EnableDeadLetterQueue 是否启用死信队列 (仅分布式模式)
	EnableDeadLetterQueue bool `json:"enable_dead_letter_queue"`

	// DeadLetterTopic 死信队列主题名 (仅分布式模式)
	DeadLetterTopic string `json:"dead_letter_topic,omitempty"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries"`

	// BackoffStrategy 退避策略: "fixed", "exponential"
	BackoffStrategy string `json:"backoff_strategy"`

	// InitialDelay 初始延迟
	InitialDelay time.Duration `json:"initial_delay"`

	// MaxDelay 最大延迟
	MaxDelay time.Duration `json:"max_delay"`

	// Multiplier 指数退避乘数
	Multiplier float64 `json:"multiplier"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证模式
	if c.Mode != ModeMemory && c.Mode != ModeDistributed {
		return errors.New("mode must be 'memory' or 'distributed'")
	}

	// 验证协程池配置
	if c.Pool.Size <= 0 {
		return errors.New("pool size must be greater than 0")
	}

	// 验证分布式模式配置
	if c.Mode == ModeDistributed {
		if c.Distributed == nil || c.Distributed.MQAdapter == nil {
			return errors.New("MQAdapter is required for distributed mode")
		}
	}

	// 验证内存模式配置
	if c.Mode == ModeMemory && c.Memory != nil {
		if err := c.Memory.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate 验证内存配置
func (m *MemoryConfig) Validate() error {
	if m.BufferSize < 0 {
		return errors.New("buffer size cannot be negative")
	}

	if m.EnableHistory && m.MaxHistorySize <= 0 {
		return errors.New("max history size must be greater than 0 when history is enabled")
	}

	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Mode: ModeMemory,
		Pool: PoolConfig{
			Size:           1000,
			ExpiryDuration: 10 * time.Second,
			PreAlloc:       false,
		},
		Memory: &MemoryConfig{
			BufferSize:     1000,
			EnableHistory:  false,
			MaxHistorySize: 10000,
			EnableMetrics:  false,
		},
		Logger: LoggerConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}
}

// DefaultDistributedConfig 返回一个合理的分布式模式默认配置。
// 注意：MQAdapter 必须由用户单独提供。
func DefaultDistributedConfig() *Config {
	config := DefaultConfig()
	config.Mode = ModeDistributed
	config.Pool.Size = 2000
	config.Memory = nil // 分布式模式下不使用内存配置
	config.Distributed = &DistributedConfig{
		EnableMetrics: false,
		Serializer:    &DefaultEventSerializer{},
	}
	return config
}

// DefaultSubscribeConfig 返回默认订阅配置
func DefaultSubscribeConfig() *SubscribeConfig {
	return &SubscribeConfig{
		ConsumerGroup:         "default",
		Concurrency:           1,
		AutoCommit:            true,
		BufferSize:            100,
		StartOffset:           "latest",
		CommitInterval:        1 * time.Second,
		EnableDeadLetterQueue: false,
		DeadLetterTopic:       "",
		RetryPolicy: &RetryPolicy{
			MaxRetries:      3,
			BackoffStrategy: BackoffExponential,
			InitialDelay:    100 * time.Millisecond,
			MaxDelay:        5 * time.Second,
			Multiplier:      2.0,
		},
	}
}
