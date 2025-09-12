# 自定义 Adapter 示例 (RabbitMQ 实现)

这是一个演示如何为 `event-stream` 库实现并集成一个自定义 `MQAdapter` 的示例。我们使用 RabbitMQ 作为具体的消息代理，来展示扩展 `event-stream` 以支持其他消息中间件的完整步骤。

**重要提示**: 此示例的核心目的不是演示 RabbitMQ 的用法，而是作为一个开发指南，向您展示如何为您选择的任何消息系统（如 NATS, Pulsar 等）构建自己的适配器。

## 运行前提

1.  安装并启动 RabbitMQ 服务器。
2.  确保 RabbitMQ 服务可访问。程序会优先读取 `RABBITMQ_URL` 环境变量作为连接地址。如果该环境变量未设置，则会使用默认地址 `amqp://guest:guest@localhost:5672/`。
3.  示例代码会自动声明所需的 exchange 和 queue，您无需手动创建它们。

## 运行说明

1.  进入示例目录：
    ```bash
    cd examples/custom_adapter
    ```

2.  下载依赖：
    ```bash
    go mod tidy
    ```

3.  运行示例：
    ```bash
    go run .
    ```

## 代码结构

-   `main.go`: 主程序文件。它演示了如何初始化和注册自定义的 `RabbitMQAdapter`，并用它来发布和消费事件。
-   `adapter.go`: `RabbitMQAdapter` 的具体实现。这是理解如何构建自定义适配器的关键文件，包含了连接、发布和消费逻辑。
-   `go.mod`: Go 模块定义文件。