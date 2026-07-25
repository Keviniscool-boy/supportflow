# SupportFlow 贡献指南

感谢你参与建设 SupportFlow。项目在 MVP 阶段保持轻量协作流程，同时坚持需求、设计、实现和测试边界清晰。

## 开始之前

- 提交新 Issue 前先搜索是否已有相同问题。
- 涉及较大功能、架构或行为变更时，先创建 Issue 讨论范围。
- 安全问题请按照[安全政策](SECURITY.md)私下报告，不要创建公开 Issue。
- 每次变更只处理一个关注点；无关清理应放到独立 Pull Request。

## 开发流程

1. Fork 仓库，并从 `main` 创建分支。
2. 完成一个职责明确的变更，同时补充必要文档和测试。
3. 使用 Conventional Commits。
4. 创建 Pull Request，说明问题、解决方案、验证结果和兼容性影响。

推荐分支名：`feat/agent-state-machine`、`fix/ticket-idempotency`、`docs/database-design`。

## Commit 信息

使用以下格式：

```text
<type>(<scope>): <description>
```

常用类型：`feat`、`fix`、`docs`、`refactor`、`test`、`chore`。

示例：

```text
feat(agent): implement run state machine
fix(ticket): prevent duplicate creation
docs(api): define SSE event contract
test(tool): cover permission failures
```

## Pull Request 要求

Pull Request 应当：

- 在存在关联 Issue 时添加引用。
- 说明用户可见影响和架构影响。
- 包含测试，或解释测试不适用的原因。
- 在需要时同步更新文档和变更日志。
- 不包含密钥、个人数据、运行时生成数据或无关文件。

维护者可以要求把过大的 Pull Request 拆成能够独立评审的变更。

## 当前阶段

仓库采用设计优先流程。PRD、技术架构和开发任务拆分已经完成，数据库与 API 设计仍以计划稿为准；当前进入编码阶段，后续按 Task → 编码 → 测试 → 部署顺序推进。

参与项目即表示你同意遵守[社区行为准则](CODE_OF_CONDUCT.md)。
