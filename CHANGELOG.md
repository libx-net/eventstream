# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Core Features
- **Event Bus Architecture**: Unified EventBus interface supporting both memory-based and distributed event streaming
- **Memory Event Bus**: High-performance in-memory event processing with consumer groups support
- **Distributed Event Bus**: Scalable distributed event streaming with pluggable MQ adapter architecture
- **Event Model**: Rich event structure with metadata support (ID, Type, Source, Headers, Version, Timestamp)
- **Consumer Groups**: Independent consumer group processing with configurable retry policies and concurrency
- **Subscription Management**: Flexible subscription lifecycle management with automatic cleanup

### MQ Adapters
- **Kafka Adapter**: Production-ready Kafka integration with producer/consumer capabilities
- **Pluggable Architecture**: Extensible MQAdapter interface for custom message queue implementations
- **Message Abstraction**: Unified Message interface abstracting different MQ systems

### Configuration & Options
- **Flexible Configuration**: Comprehensive configuration system for both memory and distributed modes
- **Subscription Options**: Rich subscription options including retry policies, concurrency control, and error handling
- **Default Configurations**: Sensible defaults for quick setup and development

### Testing & Quality
- **Comprehensive Test Suite**: Extensive unit and integration tests with 78.6% coverage
- **Integration Testing**: Testcontainers-based integration tests for Kafka adapter
- **Benchmarks**: Performance benchmarks for critical paths
- **CI/CD Pipeline**: GitHub Actions workflow for automated testing

### Documentation & Examples
- **Complete Documentation**: Detailed usage guides for memory and distributed modes
- **Working Examples**: Ready-to-run examples for memory, Kafka, and custom adapter implementations
- **API Documentation**: Comprehensive API documentation with code examples

### Development Tools
- **Makefile**: Development automation with test, build, and integration commands
- **Scripts**: Helper scripts for testing and development workflow
- **Docker Support**: Docker Compose setup for local Kafka testing

### Architecture Highlights
- **JSON Serialization**: Unified JSON-based event serialization across all components
- **Concurrent Processing**: Thread-safe design with efficient concurrent event processing
- **Error Handling**: Robust error handling with configurable retry mechanisms
- **Metrics Support**: Built-in metrics collection for monitoring and observability
- **Clean Package Structure**: Well-organized codebase with clear separation of concerns

---