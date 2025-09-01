# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Summary
- Redesigned consumer-group architecture: subscription-level consumer groups support multiple independent consumers for the same event.
- Flexible per-subscription configuration: retry policies, concurrency, buffer sizes, and distributed-mode options.
- Dynamic management: add/remove consumer groups at runtime.
- Dual-mode design: high-performance in-memory mode (implemented) and planned distributed mode for cross-service communication.
- Observability & reliability: built-in metrics, retry mechanisms, graceful shutdown, and error handling.
- Examples and tests updated to validate the new architecture.

### Added
- Multiple consumer groups support (same event consumed independently by different groups).
- Subscription options: eventstream.WithConsumerGroup, WithConcurrency, WithRetryPolicy, WithBufferSize, etc.
- Tests and examples demonstrating multi-group consumption and isolation.

### API examples
```go
bus.On("user.registered", sendWelcomeEmail, eventstream.WithConsumerGroup("notification-service"))
bus.On("user.registered", grantWelcomePoints, eventstream.WithConsumerGroup("points-service"))
bus.On("user.registered", recordAnalytics, eventstream.WithConsumerGroup("analytics-service"))
```

### Problems solved
- Replaces global/fixed consumer group model with subscription-level groups so the same event can be consumed by multiple services independently.

### Performance (reported)
- Emit only: ~93.36 ns/op
- Emit + process: ~614.2 ns/op
- Multi-consumer-groups (5 groups): ~774.3 ns/op
- Throughput: ~1M+ events/second

### Documentation & examples
- docs/consumer-groups.md (merged)
- examples/memory/multiple_consumers.go
- README.md updated with usage and examples

### Breaking changes
- Consumer group identifier moved from global config to subscription options.
- Configuration structure updated to support per-subscription consumer groups.

### Roadmap (planned)
- Distributed mode (message queue based producer/consumer, persistence, replay, DLQ)
- Event filtering/routing, schema validation, batch processing, delayed events
- Monitoring dashboard, CLI, Docker images, Kubernetes operator