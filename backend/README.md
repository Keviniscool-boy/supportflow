# SupportFlow 后端

后端采用 Go 1.24+、Gin、PostgreSQL/pgx 和模块化单体结构。当前已提供配置校验、Migration、统一 HTTP 错误边界、脱敏日志、轻量 OpenTelemetry 接入边界，以及 Demo Session 与 Customer 服务端身份上下文；Agent、知识库和工单业务按任务清单逐步实现。

## 本地运行

```powershell
go test ./...
$env:SUPPORTFLOW_ENV = "development"
$env:SUPPORTFLOW_MODEL_MODE = "mock"
go run ./cmd/server
```

服务默认监听 `:8080`，提供 `/health`、`/ready` 和 `/api/v1/demo/session*`。Customer Cookie 为 HttpOnly，写请求需要同时提供 Session 响应中的 CSRF Token。

## 执行 Migration

```powershell
$env:SUPPORTFLOW_DATABASE_URL = "postgres://supportflow:password@localhost:5432/supportflow?sslmode=disable"
go run ./cmd/migrate
```

Migration 必须作为独立命令在应用启动前执行。数据库连接信息和模型密钥只能通过环境变量或 Secret Reference 注入，不能提交到仓库。
