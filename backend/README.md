# SupportFlow 后端

后端采用 Go 1.24+、Gin、PostgreSQL/pgx 和模块化单体结构。当前第一批代码只提供应用骨架、配置校验、统一 HTTP 错误边界和顺序 Migration Runner，不包含 Agent、知识库或工单业务实现。

## 本地运行

```powershell
go test ./...
$env:SUPPORTFLOW_ENV = "development"
$env:SUPPORTFLOW_MODEL_MODE = "mock"
go run ./cmd/server
```

服务默认监听 `:8080`，提供 `/health` 和 `/ready`。

## 执行 Migration

```powershell
$env:SUPPORTFLOW_DATABASE_URL = "postgres://supportflow:password@localhost:5432/supportflow?sslmode=disable"
go run ./cmd/migrate
```

Migration 必须作为独立命令在应用启动前执行。数据库连接信息和模型密钥只能通过环境变量或 Secret Reference 注入，不能提交到仓库。
