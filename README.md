警告：此项目仍在开发中，禁止用于生产环境

# EventStream — 轻量级事件总线库（Go）

[![GitHub Actions](https://img.shields.io/github/actions/workflow/status/libx-net/eventstream/test.yml?branch=main&label=Build&logo=github)](https://github.com/libx-net/eventstream/actions)
[![codecov](https://codecov.io/gh/libx-net/eventstream/branch/main/graph/badge.svg)](https://codecov.io/gh/libx-net/eventstream)
[![Go Report Card](https://goreportcard.com/badge/github.com/libx-net/eventstream)](https://goreportcard.com/report/github.com/libx-net/eventstream)
[![golangci-lint](https://img.shields.io/badge/golangci--lint-passing-brightgreen)](https://golangci-lint.run/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/libx-net/eventstream)](https://golang.org/)
[![License](https://img.shields.io/github/license/libx-net/eventstream)](LICENSE)


简介
EventStream 是一个用于构建内存或分布式事件传递的轻量级 Go 库。该项目仍在积极开发中，API 可能发生变化，当前版本不适合在生产环境中使用。请在生产环境使用前自行评估稳定性并进行充分测试。

主要功能
- 可插拔的 MQ 适配器接口（支持本地内存模式与外部 MQ）
- 事件序列化与反序列化支持
- 订阅者分组（consumer group）能力与并发控制
- 可选历史记录与统计指标
- 完整的单元测试覆盖和示例代码

安装
在 Go 模块项目中直接引用本模块：
```
go get libx.net/eventstream
```

快速开始（内存模式）

1. 使用默认配置创建事件总线：
```go
cfg := eventstream.DefaultConfig()
bus, err := eventstream.New(cfg)
if err != nil {
    // 处理错误
}
defer bus.Close()
```

2. 订阅主题并处理事件：
```go
sub, err := bus.On("user.registered", func(ctx context.Context, e *eventstream.Event) error {
    // 处理事件
    return nil
})
if err == nil {
    defer bus.Off(sub)
}
```

3. 发布事件：
```go
_ = bus.Emit(context.Background(), "user.registered", map[string]interface{}{"user_id": "u1"})
```

分布式模式（概览）
- 提供 `DistributedConfig`，可以注入自定义 `MQAdapter`（需实现 Publish/Subscribe/Ack/Close）
- 支持不同消费者组（consumer group）隔离消费
- 详见 `docs/distributed_usage.md` 中的配置和示例

示例
`examples/` 下提供三个示例：
- `memory_basic`：内存模式的基本示例
- `memory_multiple`：多个消费者组示例
- `distributed_basic`：分布式适配器示例（包含演示用简单 MQAdapter）

贡献指南
- 使用 `gofmt` / `go vet` 保持代码风格一致
- 新增功能请补充对应单元测试，目标覆盖率不低于 50%
- 提交信息遵循 Angular commit message 规范（英文）

许可
本项目遵循 [MIT](LICENSE) 开源许可协议。