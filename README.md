# go-jaeger-demo

这是一个用于演示如何在两台简单的 Go 服务之间使用 Jaeger 进行分布式追踪的示例项目。

项目结构

- docker-compose.yml         # 可选的 docker-compose 配置（可能包含 jaeger 和两个服务）
- server-A/
  - main.go                  # 服务 A 的入口
- server-B/
  - main.go                  # 服务 B 的入口

目标

- 演示如何在服务间传递追踪上下文并在 Jaeger UI 中查看调用链。

先决条件

- Go (建议 >= 1.18)
- Docker 与 Docker Compose（如果使用 docker-compose 启动）

快速开始 — 本地运行

1. 在两个不同的终端窗口中分别运行服务：

```bash
# 在项目根目录
cd /path/to/go-jaeger-demo
# 终端 1: 启动 server-A
go run ./server-A/main.go
# 终端 2: 启动 server-B
go run ./server-B/main.go
```

2. 访问产生调用的端点（取决于代码中定义的端点），观察服务日志和追踪输出（如果服务配置为发送到本地 Jaeger）。

使用 Docker Compose（如果仓库包含可用的 `docker-compose.yml`）

```bash
# 在项目根目录
docker-compose up --build
```

Compose 文件通常会同时启动 Jaeger UI（通常在 http://localhost:16686）、以及两个示例服务。启动后，打开 Jaeger UI 查看追踪。

构建二进制并运行（可选）

```bash
# 构建
go build -o bin/server-A ./server-A
go build -o bin/server-B ./server-B
# 运行
./bin/server-A &
./bin/server-B &

# 发送测试请求 （如果不带 Trace-Id ，接口内部自动生成一个 并返回）
curl -v -H "Trace-Id: 1111" http://localhost:8080/callb 
curl -v -H  http://localhost:8080/callb 
```

常见问题与排查建议

- 未看到追踪数据：确保服务中配置的 Jaeger 采样和发送地址正确，且 Jaeger Collector/Agent 已运行。
- 端口冲突：确认服务和 Jaeger 使用的端口（默认 Jaeger UI 16686，Agent UDP 6831/6832）没有被占用。
- Go 模块/依赖问题：若构建失败，先运行 `go mod tidy` 以同步依赖。

扩展与下步建议

- 在服务中添加更多的 span/标签以提高可观测性。
- 使用 OpenTelemetry 替代单独的 Jaeger 客户端以便更好地兼容其他后端。

许可证

请根据需要自行添加 LICENSE 文件或使用合适的许可证。# go-jaeger-demo
