# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Distributed event streaming support with Kafka integration
- Kafka adapter implementation with producer/consumer capabilities  
- Comprehensive integration tests using Testcontainers
- GitHub Actions workflow for CI testing
- Makefile and test scripts for development
- Documentation for distributed usage and testing
- Refactored adapter, kafka, and internal packages into main package to eliminate circular imports
- Added extensive unit tests for core modules (config, utils, memory, distributed, adapter, kafka)
- Implemented distributed event bus main logic and subscription management in distributed.go
- Added kafka_adapter.go with corresponding unit tests
- Enhanced README and examples

### Changed
- Unified Event type definitions and serialization logic
- Simplified project package structure and import paths
- Improved test coverage to 78.6%

### Fixed
- Fixed go test failures caused by refactoring (circular imports, type mismatches)
- Fixed type conversion and field access errors in examples

### Changed
- Consolidated packages: moved adapter, kafka, and internal code into main package to eliminate circular imports
- Removed internal/memory/ directory structure

### Security
- No security vulnerabilities identified in this release

---