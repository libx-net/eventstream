# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- 将 adapter、kafka、internal 包重构并合并到主包，消除循环导入问题。
- 为核心模块补充大量单元测试（config、utils、memory、distributed、adapter、kafka），新增测试文件。
- 新增 distributed.go，实现分布式事件总线的主逻辑与订阅管理。
- 新增 kafka_adapter.go 以及对应的单元测试。
- 新增 README、示例（examples/*）的修正与完善。

### Changed
- 统一 Event 类型定义与序列化逻辑，修复跨包类型冲突。
- 统一项目包结构，简化导入路径和依赖关系。
- 提升测试覆盖率至 78.6%。

### Fixed
- 修复重构后导致的 go test 失败问题（循环导入、类型不匹配等）。
- 修复 examples 中的类型转换和字段访问错误，使示例可直接运行。

---