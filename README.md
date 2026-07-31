# SupportFlow

SupportFlow 是一款围绕“受控 Agent 工作流”设计的开源 AI 售后客服与知识运营平台。  
使用显式状态机 + 类型化 Tools + 企业知识库，目标是构建可审计、可追溯且可自托管的客服自动化系统，适合企业试验智能客服自动化与知识运营流程。

关键卖点
- 可审计的 Agent 路由：显式状态机为 Agent 行为设边界，便于治理与审计。
- 类型化业务 Tools：安全接入订单查询、工单创建等真实业务操作。
- 低成本自托管：轻量 Docker Compose 配置，便于快速搭建 Demo 环境。

快速体验（最小可行 demo）
```bash
# 克隆仓库并在本机以 mock 模式运行后端（需要 Go 1.24+）
git clone https://github.com/Keviniscool-boy/supportflow.git
cd supportflow
# 在 Windows PowerShell / macOS/Linux（设置环境变量后）
export SUPPORTFLOW_ENV=development
export SUPPORTFLOW_MODEL_MODE=mock
go run ./cmd/server
# 打开 http://localhost:8080 查看 demo 接口 /health /api/v1/demo/session*
```

当前状态
- PRD、技术架构与工程任务拆分已完成，后端骨架支持本地运行。v0.2 目标：自托管 Demo 与一键初始化脚本。

文档与贡献
- 详见 docs/ 目录与 backend/README.md。欢迎阅读 CONTRIBUTING.md 并参与贡献.
