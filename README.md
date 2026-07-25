# SupportFlow

[简体中文](README.zh-CN.md)

An open-source AI after-sales support and knowledge operations platform built around controlled Agent workflows.

> **Project status:** Design and MVP development. The v0.2 target is a self-hosted demonstration, not a production enterprise release.

## ✨ Features

- Explicit Agent Runtime state machine with bounded routes and actions.
- Knowledge retrieval with traceable citations.
- Controlled order lookup and ticket creation through typed Tools.
- Human handoff for high-risk, low-confidence, and unresolved cases.
- Business-level Agent Trace without exposing model chain-of-thought.
- NovaTech sandbox with Mock identity, orders, tickets, and model mode.
- Lightweight Docker Compose profile designed for a 2 vCPU and 2 GiB host.

## 📷 Screenshots

The interface is being designed. Screenshots will be added with the v0.1 Technical Preview.

## 🏗️ Architecture

```text
Customer UI / Operations UI
            │
       SupportFlow App
  Vue + Go API + Agent + Worker
            │
  PostgreSQL ─ Redis ─ Local Storage
            │
   External Model Provider
```

SupportFlow uses a Go modular monolith, an explicit Agent state machine, PostgreSQL with pgvector as the business source of truth, Redis/Asynq for short-lived coordination, and an external OpenAI-compatible model provider.

See the [technical architecture](docs/Architecture.md) for boundaries and trade-offs.

## 🚀 Quick Start

The runnable application has not been released yet. Development intentionally follows the sequence PRD → architecture → database → API → tasks → implementation → testing → deployment.

You can review the accepted product and architecture baselines today:

```text
docs/PRD.md
docs/Architecture.md
```

Docker Compose and one-command Demo initialization will arrive with v0.2.

## 📖 Documentation

| Document | Status |
| --- | --- |
| [Product Requirements](docs/PRD.md) | v1.1 accepted |
| [Technical Architecture](docs/Architecture.md) | v1.0 review baseline |
| Database Design | Planned |
| API Design | Planned |
| Development Guide | Planned |
| Deployment Guide | Planned |

## 🛣️ Roadmap

- **v0.1 — Technical Preview:** Chat, Knowledge, Citation, Agent Runtime, and basic Trace.
- **v0.2 — SupportFlow MVP:** Customer flow, Mock Order Tool, Ticket, Human Handoff, and NovaTech Demo.
- **v0.3 — Enterprise Foundation:** Workspace, RBAC, Audit, real Business Connectors, and basic Analytics.
- **v0.4:** Workflow.
- **v0.5:** Plugin foundation and governance.
- **v1.0:** Stable enterprise-ready release target.

Roadmap items may change as validated product and engineering evidence becomes available.

## 🤝 Contributing

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md), follow the [Code of Conduct](CODE_OF_CONDUCT.md), and report vulnerabilities according to [SECURITY.md](SECURITY.md).

The project uses Conventional Commits and Semantic Versioning while keeping the MVP pull request process lightweight.

## 📄 License

Licensed under the [Apache License 2.0](LICENSE).
