# SupportFlow 部署说明

## 1. Lite 参考环境

官方 Lite 部署以以下单机环境为基线：

- 2 vCPU
- 2 GiB 内存
- 40 GiB 系统盘
- 200 Mbps 峰值公网带宽
- 1 个固定 IPv4

该环境用于 v0.2 Demo 和开发验证，不是企业生产高可用方案。默认模型模式为 `mock`，不会自动访问外部模型服务。

## 2. 启动方式

在仓库根目录执行：

```powershell
Copy-Item deploy/.env.example deploy/.env
```

然后修改 `deploy/.env` 中的 `POSTGRES_PASSWORD`，再启动：

```powershell
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --build
docker compose --env-file deploy/.env -f deploy/docker-compose.yml ps
```

Migration 是独立服务。只有 Migration 成功、PostgreSQL 和 Redis 健康后，App 才会启动。

## 3. 网络与资源边界

- 默认只把 App 的 `127.0.0.1:8080` 映射到宿主机。
- PostgreSQL、Redis 和内部 Migration 不映射宿主机端口。
- 公网部署应在前面增加 HTTPS 反向代理，不直接暴露 App、PostgreSQL 或 Redis。
- PostgreSQL 限制约 512 MiB，Redis 限制约 96 MiB，App 限制约 512 MiB。
- Docker 日志按容器轮转，每个文件 10 MiB、最多保留 3 个。

## 4. 停止与数据

```powershell
docker compose --env-file deploy/.env -f deploy/docker-compose.yml down
```

默认业务数据保存在 Docker Volume：`postgres_data` 和 `object_data`。删除 Volume 会删除数据库和知识原文件，除非已经完成备份，不得使用 `down -v`。

Redis 只承担队列和短期协调数据；Redis 丢失后，后续 Reconciler 根据 PostgreSQL 状态恢复任务。PostgreSQL 与 `object_data` 必须纳入同一批次的基础备份。

## 5. 当前限制

本阶段 Compose 只提供后端骨架、健康检查和 Migration 验证。Customer、Knowledge、Order Tool、Ticket 和 Human Handoff 仍按 `docs/Development.md` 的 Task 顺序开发；不要把当前 Compose 状态当作 v0.2 完整 Demo。
