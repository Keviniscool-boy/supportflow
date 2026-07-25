# SupportFlow

SupportFlow 是一款围绕受控 Agent 工作流构建的开源 AI 售后客服与知识运营平台。

> **项目状态：** 已完成 PRD、技术架构和开发任务拆分；数据库与 API 设计文档仍处于计划稿阶段。编码阶段已经开始，v0.2 的目标是可自托管 Demo，不是企业生产版本。

## ✨ 功能特性

- 使用显式状态机控制 Agent 路由与执行边界。
- 基于企业知识库回答问题并提供可追溯引用。
- 通过类型化 Tool 受控查询订单和创建工单。
- 对高风险、低置信度和连续未解决场景转人工。
- 提供不包含模型思维链的业务级 Agent Trace。
- 提供 NovaTech 沙箱、Mock 身份、订单、工单和模型模式。
- 提供面向 2 vCPU、2 GiB 主机的轻量 Docker Compose 部署方案。

## 📷 界面截图

界面仍在设计中，v0.1 Technical Preview 发布时补充截图。

## 🏗️ 技术架构

```text
客户界面 / 运营界面
        │
   SupportFlow App
Vue + Go API + Agent + Worker
        │
PostgreSQL ─ Redis ─ 本地文件存储
        │
     外部模型服务
```

系统采用 Go 模块化单体、显式 Agent 状态机、PostgreSQL + pgvector、Redis + Asynq，以及外部 OpenAI-Compatible 模型服务。

完整边界与技术取舍参见[技术架构设计](docs/Architecture.md)。

## 🚀 快速开始

当前尚未发布完整产品，但后端工程骨架已经可以本地运行。项目严格按照 PRD → 技术架构 → 数据库 → API → Task → 编码 → 测试 → 部署的顺序推进。

目前可以阅读已经完成的设计基线：

```text
docs/PRD.md
docs/Architecture.md
docs/Database.md
docs/API.md
```

后端启动方式参见[后端开发说明](backend/README.md)。Docker Compose 和一键 Demo 初始化将在 v0.2 实现阶段提供。

## 📖 项目文档

| 文档 | 状态 |
| --- | --- |
| [产品需求文档](docs/PRD.md) | v1.1 已确认 |
| [技术架构设计](docs/Architecture.md) | v1.0 评审基线 |
| [数据库设计](docs/Database.md) | 计划中（草案） |
| [API 设计](docs/API.md) | 计划中（草案） |
| 开发任务拆分 | v1.0 评审版 |
| 部署指南 | 计划中 |

## 🛣️ 版本路线图

- **v0.1 — Technical Preview：** Chat、Knowledge、Citation、Agent Runtime、基础 Trace。
- **v0.2 — SupportFlow MVP：** Customer、Mock Order Tool、Ticket、Human Handoff、NovaTech Demo。
- **v0.3 — Enterprise Foundation：** Workspace、RBAC、Audit、真实 Business Connector、基础 Analytics。
- **v0.4：** Workflow。
- **v0.5：** Plugin 基础与治理。
- **v1.0：** 稳定企业版本目标。

路线图会根据真实产品验证和工程证据调整。

## 🤝 参与贡献

欢迎参与贡献。请阅读[贡献指南](CONTRIBUTING.md)，遵守[社区行为准则](CODE_OF_CONDUCT.md)，安全问题请按照[安全政策](SECURITY.md)私下报告。

项目采用 Conventional Commits 和 Semantic Versioning，MVP 阶段保持轻量 PR 流程。

## 📄 开源许可证

本项目采用 [Apache License 2.0](LICENSE)。
