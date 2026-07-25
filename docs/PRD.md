# SupportFlow 产品需求文档（PRD）

> 面向企业售后场景、允许个人体验和自托管的开源 AI Agent 平台。

## 1. 文档信息

| 项目 | 内容 |
| --- | --- |
| 产品名称 | SupportFlow |
| 文档版本 | PRD v1.1 |
| 文档状态 | 需求冻结（Frozen） |
| 冻结日期 | 2026-07-24 |
| 最后更新 | 2026-07-24（Lite 参考环境补充） |
| 当前目标版本 | v0.2 SupportFlow MVP |
| 前置验证版本 | v0.1 Technical Preview |
| 企业基础版本 | v0.3 Enterprise Foundation |
| 开源许可证 | Apache License 2.0 |

### 1.1 文档目的

本文档定义 SupportFlow 的产品定位、用户、版本边界、核心流程、功能需求、业务规则、验收指标和非功能要求，作为后续技术架构、数据库、API、任务拆分、开发与测试的产品基线。

### 1.2 文档边界

本文档只定义“做什么”和“如何验收”，不决定具体技术架构、数据库表结构、API 协议或代码实现。页面权限矩阵、接口字段、存储结构、测试用例数量和自动化方案分别在后续设计阶段确定。

### 1.3 需求变更规则

文档冻结后，新增功能不得直接进入 v0.2。任何范围变更必须说明目标版本、用户价值、验收口径及对现有范围的影响，并更新 PRD 版本记录。

## 2. 产品背景与问题陈述

SupportFlow 主要解决以下问题。

### 2.1 客户无法快速获得可信、可追溯的售后信息

客户面对产品参数、售后政策、订单状态和保修规则等问题时，通常需要人工咨询或自行搜索，难以确认答案是否准确及其来源依据。

### 2.2 客服大量时间消耗在重复查询和标准处理流程中

客服需要反复回答常见问题，并手工查询订单、整理上下文和创建售后工单，服务效率低且处理结果容易不一致。

### 2.3 企业缺少从售后交互中持续优化知识库的闭环

企业难以系统发现高频问题、无答案问题、错误回答和知识缺口，知识维护依赖人工经验，无法衡量知识是否真正解决客户问题。

### 2.4 AI 客服从知识回答到业务执行之间存在断层

AI 客服能够完成对话和知识检索，但真实售后流程还需要受控的业务工具调用、权限控制、人工接管和执行轨迹追踪。

### 2.5 企业业务数据与 AI 服务之间缺少安全连接方式

客户、订单和工单数据通常分散在不同业务系统中。Agent 需要通过受控 Tool 和 Connector 获取必要信息，而不是直接访问或暴露企业业务数据库。

## 3. 产品定位

### 3.1 一句话定位

SupportFlow 是面向企业售后场景、同时允许个人免费体验和自托管的开源 AI Agent 平台。

### 3.2 核心定义

SupportFlow 不是单纯的 AI 聊天机器人，而是一个**受控的售后 Agent 工作流平台**：

```text
用户输入
  → 意图与风险判断
  → 身份、权限与业务规则检查
  → 知识检索或 Tool 选择
  → 受控执行
  → 结构化 Trace
  → 回复、澄清、创建工单或转人工
```

### 3.3 产品形态

- Community Edition：完整开源核心，可完成售后 Agent 闭环。
- NovaTech Demo：无需真实企业数据即可体验客户与运营流程的沙箱。
- 企业扩展部署：在后续版本增加多租户、治理、高可用和企业集成能力。

### 3.4 产品设计原则

1. **回答有依据**：需要外部事实依据的回答必须基于知识或业务数据，不得以模型猜测代替事实。
2. **动作受控制**：Tool 调用必须通过身份、权限、参数、业务规则和确认检查。
3. **人始终可介入**：高风险、低依据、执行失败或用户明确要求时必须支持人工接管。
4. **业务过程可追踪**：每次 Agent 运行记录结构化业务轨迹，但不保存或展示模型内部推理过程。
5. **最小权限与数据隔离优先**：默认拒绝跨主体、跨 Workspace 和超出业务所需范围的数据访问。
6. **受控自治而非无限自治**：Agent 只能在固定路由、步骤上限和重试上限内执行，不自主扩展任务。
7. **接口保持架构弹性**：Provider、Connector 和 Tool 抽象用于降低未来扩展成本，不代表 v0.2 实现插件系统或复杂扩展平台。

## 4. 目标用户与身份主体

### 4.1 目标用户

| 用户群体 | 定位 | 核心价值 |
| --- | --- | --- |
| 企业售后团队 | 核心业务用户 | 提升标准问题处理效率，连接知识、订单、工单与人工接管 |
| 开发者与 AI 爱好者 | 开源与生态用户 | 快速体验、部署、理解和扩展受控 Agent 工作流 |
| 个人体验用户 | Demo 用户 | 无需真实企业数据体验完整售后流程 |

普通消费者不是独立的消费级产品目标，而是在企业售后场景中以 Customer 身份使用客服能力。

### 4.2 身份主体

| 身份主体 | 定义 | 目标版本 |
| --- | --- | --- |
| Platform User | 平台运营与管理身份，默认无企业业务数据权限 | v0.3 |
| Workspace Member | 企业员工身份，可拥有客服、知识运营或 Workspace Admin 等角色 | v0.3 |
| Customer | 企业业务系统中的客户身份；v0.2 使用 Demo 身份 | v0.2 基础、v0.3 企业接入 |

Customer 与 Workspace Member 必须使用相互独立的身份和权限体系。

### 4.3 企业角色

| 角色 | 基础职责 | 目标版本 |
| --- | --- | --- |
| Customer | 咨询、选择订单、确认业务动作、查看会话和工单状态 | v0.2 基础 |
| 人工客服 | 查看并领取待接管队列，处理会话和工单 | v0.2 基础、v0.3 RBAC |
| 知识运营人员 | 管理知识，只查看脱敏后的知识反馈上下文 | v0.2 基础、v0.3 RBAC |
| Workspace Admin | 管理成员、配置与汇总数据；默认不查看具体业务数据 | v0.3 |

Workspace Member 可拥有多个角色。Workspace Admin 如需查看具体会话、订单或工单，必须额外拥有客服角色。

## 5. 产品目标与版本边界

### 5.1 产品目标

1. 基于企业知识提供可引用、可追溯的售后回答。
2. 通过订单查询和工单创建完成低风险售后闭环。
3. 在证据不足、高风险或用户要求时无缝转人工。
4. 让开发者通过 Demo 或自托管快速体验完整 Agent 工作流。

### 5.2 Roadmap

| 版本 | 定位 | 核心范围 |
| --- | --- | --- |
| v0.1 | Technical Preview | Chat、Knowledge、Citation、Agent Runtime、基础 Trace |
| v0.2 | SupportFlow MVP | 单一默认 Workspace、Demo 身份、Mock Business Data、Customer、Knowledge、Mock Order Tool、Ticket、Human Handoff、Trace、NovaTech Demo |
| v0.3 | Enterprise Foundation | Workspace、多租户、RBAC、Audit、可信 Customer 身份、真实 Business Connector、配置中心、基础 Analytics |
| v0.4 | Workflow | 可控业务流程编排和高级运营能力 |
| v0.5 | Plugin & Ecosystem | 插件治理、扩展生态和开发者能力 |
| v1.0 | Stable | 稳定接口、生产部署基线和长期兼容策略 |

v0.2 用于验证完整售后 Agent 闭环，不作为企业生产部署版本。此前确认的企业能力继续保留在本文档中，但明确标记为 v0.3 或以后，不属于 v0.2 验收条件。

### 5.3 v0.2 保留的演进基础

- 基础 Agent Trace。
- Mock Connector 接口抽象。
- 基础角色概念和必要关联字段。
- 基础指标采集字段。
- Provider、Connector、Tool 的扩展边界。

以上内容只用于保持后续兼容性，不在 v0.2 实现完整企业治理。

### 5.4 v0.2 Out of Scope

- 企业多租户、正式 RBAC、完整 Audit Log 和真实 Business Connector。
- 退款、取消订单、修改订单、删除用户数据等高风险业务执行。
- Customer 附件上传、扫描 PDF OCR、语音及外部客服渠道。
- SLA 自动计算、智能分单和复杂工单流程。
- 高级 Analytics、自定义报表和成本治理。
- Workflow 编排、Plugin 系统、第三方插件市场和多 Agent 协作。
- 企业生产级高可用、SSO、MFA、Kubernetes 和灾备方案。
- 从客户对话中自动学习、自动修改或自动发布知识。

上述能力进入 v0.3 及后续 Roadmap，未实现不视为 v0.2 缺陷。

## 6. MVP 核心业务场景

### 6.1 场景描述

Customer 购买商品后遇到产品使用或售后问题。Agent 基于企业知识回答，并根据订单信息决定是否进入售后工单流程；无法处理、缺乏可靠依据或涉及高风险操作时转人工。

### 6.2 典型案例：蓝牙耳机左耳无声音

1. Customer 在客服窗口输入“我的蓝牙耳机左耳没有声音了”。
2. Agent 将问题识别为低风险产品故障，并路由到知识问答。
3. 系统检索蓝牙耳机产品手册、FAQ、售后政策和维修指南。
4. Agent 提供重置、重新连接和固件检查等方案，并引用文档名称与章节或页码。
5. Customer 回复“我试过了，还是没有声音”。
6. Agent 确认问题未解决，进入订单查询流程。
7. v0.2 从 Demo 身份取得 Customer 标识，通过 `GetOrder` 展示该 Customer 的模拟订单列表。
8. Customer 选择对应订单和商品。
9. 售后资格由 Mock Business Service 返回，LLM 不自行判断保修或售后资格。
10. Agent 展示拟创建工单的信息并取得 Customer 明确确认。
11. Agent 使用 Idempotency Key 调用 `CreateTicket`。
12. 系统返回工单编号、状态和后续处理说明。

### 6.3 未来企业身份链路

v0.3 企业环境采用以下链路：

```text
企业业务系统登录
  → 签发可信 Customer Token
  → SupportFlow 校验身份和 Workspace
  → Agent 查询该 Customer 的订单
  → Customer 选择商品
  → Agent 执行业务流程
```

### 6.4 必须转人工的情况

1. **预定义高风险请求**：退款、取消订单、修改订单、删除数据等。
2. **缺乏可靠依据**：无知识命中、检索相关性不足、无法提供有效引用或知识相互冲突。
3. **Customer 明确要求人工**：例如“我不想和机器人聊天”。
4. **连续三次未解决**：同一会话、同一问题下，Agent 提供三轮实质性解决方案，Customer 均明确表示未解决。澄清问题不计次数；问题变化或确认解决后计数重置。
5. **Tool 最终失败**：达到允许的重试上限后仍无法得到确定结果。

## 7. 功能需求

### 7.1 客户对话体验

**v0.2 必须满足：**

- Customer 可新建会话，并在当前 Demo 身份或 Session 有效期内查看、继续自己的历史会话。
- 对话支持纯文本输入和流式回复。
- 明确区分 AI Agent 与人工客服身份。
- 只展示检索中、查询订单、创建工单、等待人工等业务处理状态，不展示模型 CoT。
- 知识型回答提供可点击引用，显示文档名称、章节；文本型 PDF 尽可能定位页码。
- 订单和工单结果使用结构化卡片展示。
- Customer 可标记“有帮助/无帮助”并补充文字反馈。
- 人工接管在同一会话中继续，避免 Customer 重复描述。
- Customer 只能访问当前 Demo 身份下的会话和业务对象。
- Customer 附件上传进入后续 Roadmap，不属于永久不支持。

### 7.2 知识库与知识运营

**v0.1/v0.2 必须满足：**

- 支持 Markdown 和文本型 PDF；v0.2 不承诺扫描件 OCR。
- 文档生命周期为 `DRAFT`、`PROCESSING`、`PENDING_REVIEW`、`PUBLISHED`、`FAILED`、`DISABLED`。
- 只有已发布版本参与检索。
- 基础元数据包含标题、产品、分类和版本。
- 展示文档解析和索引状态。
- 文档更新后重新解析和索引，旧版本不参与检索但保留版本记录。
- 引用定位到文档、章节或页码。
- 知识变更必须经过人工审核发布，不从对话自动学习或自动发布。
- 知识运营待办状态至少包含待补充、已处理和无需处理。

**知识运营待办触发规则：**

- Customer 提交“无帮助”反馈。
- Agent 无法找到可靠知识或无法形成有效引用。
- 同一问题多轮未解决。
- 人工接管时被标记为知识缺口。

**基础知识效果字段：**

- 引用次数。
- 无帮助反馈次数。
- 知识回答解决效果。

v0.2 只采集必要指标字段；统计视图、跨 Workspace 聚合和完整 Analytics 属于 v0.3。

### 7.3 Agent 路由

MVP 路由类别固定为以下五类，不得在验收期间动态新增：

| 枚举 | 含义 |
| --- | --- |
| `KNOWLEDGE_ANSWER` | 知识问答 |
| `ORDER_QUERY` | 订单查询 |
| `TICKET_CREATION` | 创建工单 |
| `CLARIFICATION` | 信息澄清或确认 |
| `HUMAN_HANDOFF` | 人工接管 |

每次 Customer 输入后的下一步决策必须落入其中一类。调用错误 Tool 时，即使 Tool 技术执行成功，该路由仍判定错误。

### 7.4 Agent Run 状态机

Agent Run 使用固定状态：

| 状态 | 含义 |
| --- | --- |
| `CREATED` | 已创建但尚未执行 |
| `RUNNING` | 正在进行路由、检索、模型或 Tool 处理 |
| `WAITING_USER` | 等待 Customer 补充必要信息 |
| `WAITING_CONFIRMATION` | 等待 Customer 确认有状态变更的业务动作 |
| `COMPLETED` | 本次运行已完成 |
| `ESCALATED` | 已进入人工接管 |
| `FAILED` | 达到恢复上限后失败 |
| `CANCELLED` | 被 Customer 或系统取消 |

业务规则：

- 同一会话同一时刻只允许一个 Agent Run。
- Runtime 只能执行固定路由，不自主扩展任务或启动无限后台任务。
- 每次运行设置最大步骤数和重试上限。
- 信息不足时进入 `CLARIFICATION`，不得猜测 Tool 参数。
- 每次运行最终进入完成、转人工、失败或取消状态。

### 7.5 Tool 与权限边界

**v0.2 仅支持：**

- `GetOrder`：只读订单查询。
- `CreateTicket`：创建售后工单。

**通用规则：**

- Customer 身份由系统提供；禁止通过任意 Customer ID 查询其他主体数据。
- v0.2 使用静态、基础 Tool Permission Registry；v0.3 扩展为 Workspace 级可配置权限注册表。
- Tool Registry 记录工具名称、风险级别、启用状态、权限要求、必要参数和是否需要 Customer 确认。
- Tool 分为普通工具和高风险工具。v0.2 不开放高风险工具执行。
- Tool 调用前检查身份、对象归属、参数、权限和确认状态。
- 售后资格由业务服务返回，不由 LLM 判断。
- 创建工单前必须完成订单选择、资格校验和 Customer 明确确认。
- 所有 Tool 选择、参数摘要、执行结果、耗时和错误写入 Trace。

**事务与幂等：**

- 每次 Tool 调用构成独立事务边界，不在多个 Tool 之间承诺分布式事务。
- `CreateTicket` 必须使用 Idempotency Key。
- 只有返回有效工单编号才视为成功。
- 执行结果未知时不得假定成功，应进行幂等重试、结果核对或转人工。

**重试规则：**

- `GetOrder` 在限流或暂时不可用时最多自动重试 2 次。
- `CreateTicket` 超时后只允许使用同一 Idempotency Key 重试 1 次。
- 身份/权限错误、参数错误、数据不存在和业务规则拒绝不自动重试。
- 达到上限后说明情况并转人工，禁止无限循环。

### 7.6 Business Connector

**v0.2：**

- 使用 Mock Business Connector 和 Mock Business Data。
- `GetOrder` 遵循标准化输入输出契约。
- 保留 Provider/Connector 抽象，不接入真实企业系统。

**v0.3：**

- 企业业务系统作为 Customer、订单等业务数据源。
- SupportFlow 只保存最小身份映射和售后上下文。
- Customer 身份必须通过企业签发的可信 Token 校验。
- Connector 配置、密钥和数据访问严格绑定 Workspace。
- 配置中心提供连接配置、连通性测试和启停控制。
- 不直接访问企业业务数据库，不全量同步客户和订单，不反向修改订单。

Connector 错误分类：身份/权限错误、参数错误、数据不存在、限流、暂时不可用、业务规则拒绝。仅限流和暂时不可用允许自动重试；业务拒绝属于正常业务结果，不计为技术执行失败。

### 7.7 售后工单

- Agent 和人工客服均可创建工单；Agent 必须遵守确认与资格规则。
- 工单与会话保持关联但职责分离：会话负责沟通，工单负责业务处理。
- 订单与商品关联允许为空，以支持非订单类售后问题。
- 基础字段包含编号、Customer、可选订单、可选商品、可选会话、问题类型、描述、优先级、处理人、来源、创建原因、状态和时间记录。
- 工单来源至少区分 Agent 与人工客服。
- 创建原因至少覆盖问题未解决、需要售后处理和人工创建。
- 同一 Customer、订单和问题存在活跃工单时，应引导继续原工单，避免重复创建。
- 工单不得物理删除，终止处理使用关闭或取消状态。

**状态定义：**

| 状态 | 含义 |
| --- | --- |
| `PENDING` | 已创建，尚未领取 |
| `IN_PROGRESS` | 客服正在处理 |
| `WAITING_CUSTOMER` | 等待 Customer 补充信息或确认 |
| `RESOLVED` | 客服已给出解决结果 |
| `CLOSED` | Customer 确认或客服正式结案 |
| `CANCELLED` | 重复、误建或无需继续处理 |

`RESOLVED` 可因 Customer 反馈未解决返回 `IN_PROGRESS`；`CLOSED` 和 `CANCELLED` 为终态。

**Ticket Activity Timeline：**

- 记录创建、领取、转派、状态变化、Customer 补充、公开回复和内部备注。
- 内部备注不向 Customer 展示。
- 所有关键状态和处理人变化保留记录。

### 7.8 人工接管

- 接管触发条件遵循第 6.4 节。
- 接管请求进入客服队列，v0.2 使用人工主动领取，不做智能分配。
- 状态为 `WAITING`、`IN_PROGRESS`、`ENDED`、`CANCELLED`。
- 客服接管后 Agent 停止自动回复，避免 AI 与人工同时响应。
- 客服可查看对话摘要、完整消息、相关订单、引用、Tool 记录、已尝试方案和接管原因。
- Trace 只展示业务执行轨迹，不展示模型内部推理。
- 接管请求与售后工单是不同对象；接管不自动创建工单。
- 预留请求时间、接管时间、结束时间、等待时长和 SLA 截止时间字段；v0.2 不执行 SLA 规则。

### 7.9 Agent Trace

Agent Trace 定位为业务可观测能力，不是模型调试平台。

**记录内容：**

- 脱敏后的业务输入摘要和关联消息引用。
- 固定路由类别、结构化决策和原因代码。
- 检索文档、章节/页码、相关性分数、索引版本等元数据。
- Tool 名称、脱敏参数摘要、技术执行结果、业务结果、耗时和错误代码。
- 人工接管、最终状态及各阶段耗时。

**禁止记录：**

- 模型 CoT 或内部推理过程。
- 原始密码、Token、密钥或其他认证凭证。
- 未经脱敏的敏感 Customer 输入。
- 完整检索 Chunk 内容。

**生命周期：**

- 状态为 `RUNNING`、`COMPLETED`、`FAILED`、`CANCELLED`。
- 记录开始时间、结束时间、阶段耗时、总耗时和过期时间。
- 使用固定枚举和原因代码，避免自由文本成为主要判断依据。

**可见性：**

- Customer 只查看业务处理状态和引用。
- v0.2 运营端查看与 Demo 业务相关的 Trace。
- v0.3 按 RBAC 控制 Trace 检索权限。

Agent Trace、Audit Log 和系统日志是三个不同对象：Trace 追踪 Agent 业务决策，Audit 记录主体操作，系统日志用于运行诊断。

### 7.10 用户反馈与知识运营闭环

- “有帮助/无帮助”反馈与对话消息分离保存。
- 无帮助、无可靠答案、多轮未解决和人工标记知识缺口时生成知识运营待办。
- 待办关联脱敏对话上下文、问题摘要、引用、路由和触发原因。
- 知识运营人员可处理待办，但不得从 Customer 对话自动生成并发布知识。
- 反馈处理结果进入基础知识效果统计。

### 7.11 通知、待办与 Customer 状态中心

- 通知、聊天消息和待办任务相互分离。
- 通知由固定业务事件触发，不直接由系统日志生成。
- Customer 通过售后状态中心查看会话、接管和工单状态。
- v0.2 支持基础站内状态提醒和未读状态。
- 通知记录至少包含接收主体、事件类型、标题摘要、目标对象类型与 ID、创建时间、已读状态和跳转信息。
- v0.3 增加 Workspace 归属、成员通知和业务影响异常通知。
- 管理员只接收影响业务使用的 Connector、配置或服务异常，不接收原始系统日志。
- 邮件仅用于成员邀请、密码重置等账号生命周期事件，不用于 v0.2 业务进度通知。

### 7.12 NovaTech Demo 沙箱

**示例企业：** NovaTech 电子商城。

**预置知识：**

- 蓝牙耳机产品手册。
- 售后政策。
- 常见问题 FAQ。

**预置业务数据：**

- 模拟 Customer。
- 模拟订单。
- 模拟工单。

**体验规则：**

- 访客无需注册，系统分配短期 Demo Session 和临时 Customer 身份。
- 可体验客户聊天、售后状态中心及受限运营后台。
- 使用单一 NovaTech 默认 Workspace 模板，并按 Session 进行逻辑数据隔离；这不等同于 v0.3 企业多租户能力。
- 开发者可使用可保留的测试账号；游客 Session 短期过期。
- Demo 使用固定 Mock 数据和允许的模型配置，不允许真实密钥或外部业务访问。
- 不开放 Platform Admin 能力。
- 限制上传文件大小、数量、请求额度、Token 配额和 Agent 执行并发。
- 支持手动恢复 NovaTech 初始数据，并对长期测试账号执行周期性重置。
- 数据记录创建、过期和清理时间；过期或重置后清除会话、工单、上传文档和 Trace。

### 7.13 核心页面与信息架构

客户端与运营端使用分离导航。

**v0.2 客户端：**

- 客户聊天页。
- 售后状态中心。

**v0.2 运营端：**

- 接管队列。
- 会话。
- 工单。
- 知识库。
- 知识反馈。
- Agent Trace。

**v0.3 运营端新增：**

- 基础运营看板。
- Workspace 成员与权限。
- Business Connector。
- Audit Log。
- Workspace 配置中心。

v0.2 不实现 Prompt 调试、模型参数实验、CoT 浏览或独立模型调试平台。页面权限矩阵在技术设计阶段定义。

## 8. v0.3 Enterprise Foundation 需求

本章内容属于后续企业基础版本，不纳入 v0.2 验收。

### 8.1 Workspace 与数据隔离

- 每个企业对应独立 Workspace。
- Workspace Member 与 Customer 分离。
- Customer、订单映射、会话、工单、知识、配置、通知、Trace 和 Tool 调用均绑定 Workspace。
- 所有 Tool 调用必须携带并校验 Workspace 上下文。
- Platform User 默认只能管理 Workspace 状态，无权查看企业业务数据。
- Customer 在不同 Workspace 下独立隔离，即使外部标识相同也不得串联数据。

### 8.2 Workspace 配置中心

- 配置企业名称与品牌、Agent 名称和欢迎语、知识范围、允许模型、Tool 启用状态、人工接管参数和基础检索阈值。
- 配置采用草稿、发布、版本记录和回滚机制；只有已发布版本生效。
- 平台安全规则和高风险限制不可被 Workspace 覆盖。
- Tool 权限区分普通工具与高风险工具。
- 接管规则采用平台强制规则与 Workspace 可调参数分层。
- Workspace 只能选择平台加入白名单的 Provider、端点和模型，不能配置任意地址。
- 密钥只能写入或替换，不能明文回显。
- 不包含可视化 Workflow 编排和插件安装。

### 8.3 模型 Provider

- v0.2 参考支持 Mock 与 OpenAI Compatible。
- 具体可用端点和模型由部署侧白名单控制，Workspace 只能选择已允许项。
- 保留 Provider 扩展接口，但不承诺任意模型兼容。

### 8.4 身份认证与 Session

- 明确 Platform User、Workspace Member、Customer 三类身份模型。
- Workspace Member 由管理员邀请，不开放员工自助注册。
- Customer 优先使用企业签发的可信 Token，不建设完整消费者账号体系。
- 登出、密码重置、账号停用、角色撤销和 Workspace 停用触发 Session/Token 失效。
- 登录成功、失败、锁定、退出和 Session 撤销进入 Audit Log。
- MVP 后续可增加 MFA、SSO 和第三方身份集成。

### 8.5 RBAC

- 所有权限由后端强制校验，不依赖前端隐藏。
- 客服可查看并领取待接管队列，并处理被授权的会话和工单。
- 知识运营人员只能查看脱敏反馈上下文和知识效果指标。
- Workspace Admin 默认管理成员、配置和汇总数据；查看详细业务数据需要额外客服角色。
- Demo 用户只访问临时沙箱。
- 临时授权和限时提权作为后续扩展；任何平台临时访问必须经过授权并产生审计记录。

### 8.6 Audit Log

- 采用 Append Only 模式，业务后台不可修改或删除。
- 记录成员邀请、登录、角色变更、知识审核发布、配置发布、密钥替换、Tool 权限变更、人工接管、工单状态变化和访问拒绝等事件。
- 字段至少包含 Workspace、操作者、动作、目标对象、`source`、脱敏变更摘要、时间和结果。
- 敏感字段只记录操作类型和结果，不记录原值。
- 支持按时间、操作者、事件类型、对象、来源和结果筛选。
- Workspace Admin 只能查看本 Workspace 审计记录。
- Platform User 只能查看平台级管理记录；临时访问企业数据的授权、访问和结束全过程必须审计。

### 8.7 基础运营看板

- 会话量。
- 五类 Agent 路由分布。
- 自动闭环解决率。
- 人工接管率。
- 工单状态分布。
- Tool 技术执行成功率和业务结果分布。
- Agent 各阶段响应耗时。
- 输入 Token、输出 Token 和估算成本。
- 知识引用次数、无帮助反馈和知识解决效果。
- 支持按 Workspace、时间和产品筛选。
- 知识运营人员只查看知识相关指标。
- 自定义报表、数据导出和高级成本治理进入更后续版本。

## 9. 核心业务规则

### 9.1 决策优先级

1. 平台强制安全规则和高风险规则。
2. 身份、对象归属和 Tool 权限。
3. Business Service 返回的资格与业务结果。
4. 已发布的 Workspace 配置与知识。
5. Agent 在固定路由内的决策。

低优先级信息不得覆盖高优先级约束；知识文档和 Customer 输入均不能提升 Tool 权限或改变平台安全规则。

### 9.2 事实依据

- 产品参数、售后政策、订单状态和保修规则属于需要外部事实依据的问题。
- 知识型回答必须引用可靠知识。
- 订单状态和保修结果必须来自 Tool/Business Service，不由 LLM 生成。
- 无可靠依据时进入澄清或人工接管，不提供确定性答案。

### 9.3 确认与状态变更

- 只读查询不要求额外业务确认，但必须校验 Customer 归属。
- 创建工单等状态变更必须在执行前展示关键信息并取得 Customer 明确确认。
- 高风险状态变更在 v0.2 禁止执行，直接转人工。

### 9.4 Trace 与隐私

- Trace 采用允许字段清单。
- 密码、Token、密钥始终移除。
- 邮箱、手机号、地址、订单号等按字段掩码或仅保存必要引用。
- Tool 参数与结果只保留业务追踪所需的脱敏摘要。

## 10. 成功指标

所有离线验收指标均限定在发布前冻结、版本化并经人工标注的 MVP 标准验收测试集内。生产数据采用持续监控和趋势指标，不将测试集内的 `100%` 表述为生产环境绝对承诺。

| 指标 | 目标 | 定义 |
| --- | --- | --- |
| 知识型回答引用率 | 100% | 需要知识支撑的回答必须提供有效引用；闲聊、礼貌回复、澄清和纯 Tool 结果不计入 |
| 预定义高风险场景转人工率 | 100% | 冻结测试集中退款、取消订单、修改订单、删除数据等场景全部进入人工接管 |
| 事实性错误率 | ≤5% | 仅统计产品参数、售后政策、订单状态、保修规则；一条回答含任一无依据、冲突或虚构事实即记为错误 |
| Agent 路由准确率 | ≥90% | 按五类固定路由评估每次 Customer 输入后的下一步决策；Tool 选择准确性包含在内 |
| Tool Execution Success Rate | ≥99% | Tool 成功接收参数、完成执行并返回可解析结果；不代表 Tool 选择正确 |
| 自动闭环解决率 | ≥40% | 见第 10.1 节；用于验证闭环，不以替代人工客服为目标 |
| Agent Trace 完整率 | 100% | 冻结测试集内关键业务阶段、路由、引用、Tool、状态和原因代码完整，不要求 CoT |
| Demo 首次闭环时间 | ≤5 分钟 | 首次用户完成咨询、查询订单和创建工单 |
| 默认本地启动时间 | ≤10 分钟 | 使用官方默认配置完成数据库、Migration、Demo 数据初始化并进入 Demo |

### 10.1 自动闭环解决率定义

- 分母：已结束、属于 v0.2 支持范围且为低风险的会话。
- 分子：未由人工介入并完成预期结果的会话，包括 Customer 确认知识方案有效、成功返回订单结果，或按确认流程成功创建工单。
- Customer 中途离开、外部 Provider 明确不可用和超范围问题不计入。

### 10.2 Tool 指标拆分

- **Tool Selection Accuracy**：属于 Agent 路由准确率；调用错误 Tool 即为路由错误。
- **Tool Execution Success Rate**：只衡量技术执行。
- **Business Result**：成功、数据不存在、业务拒绝等业务结果单独统计；正常业务拒绝不计为技术失败。

### 10.3 Token 与成本

v0.2 记录输入 Token、输出 Token、模型、Agent Run 和估算成本等基础字段；v0.3 在运营看板按 Workspace、模型和时间聚合。

## 11. 非功能需求

### 11.1 v0.2 Lite 参考环境

v0.2 以以下轻量服务器作为官方最低参考环境和验收基线：

| 项目 | 规格 |
| --- | --- |
| CPU | 2 vCPU |
| 内存 | 2 GiB |
| 系统盘 | 40 GiB |
| 峰值公网带宽 | 200 Mbps |
| 公网地址 | 1 个固定 IPv4 |

该环境只用于单机、低并发、外部模型模式的 Community Demo，不代表 v0.3 企业生产部署能力。

**运行约束：**

- Chat Model 与 Embedding Model 均使用外部 Provider，不在服务器运行本地模型。
- 不部署 OCR、MinIO、Prometheus、Grafana 或 OpenTelemetry Collector。
- Object Storage 使用本地持久化 Volume，并保留未来替换 S3-Compatible 实现的接口。
- Agent 默认执行并发为 1，最大为 2；文档解析并发为 1。
- 单个文本型 PDF 或 Markdown 文件不得超过 10 MiB。
- 默认知识规模不超过 10,000 个 Chunk。
- 前端、Go 二进制和容器镜像在本地或 CI 构建，参考服务器只负责拉取镜像、Migration 和运行服务。
- 建议配置 1–2 GiB Swap 作为峰值保护，但日常运行不得依赖 Swap 承载负载。
- Docker 日志必须轮转，旧镜像、过期 Demo 数据和临时文件必须可清理。

**长驻组件资源预算：**

| 组件 | 建议内存上限 |
| --- | ---: |
| SupportFlow App | 512 MiB |
| PostgreSQL + pgvector | 512 MiB |
| Redis + Asynq | 96 MiB |
| HTTPS Reverse Proxy（可选） | 64 MiB |

剩余内存保留给操作系统、Docker 和短时峰值。具体容器参数在 `docs/Architecture.md` 与 `docs/Deployment.md` 中定义。

### 11.2 性能与响应体验

- 普通 API（不包含 Agent、RAG、LLM 或 Tool 链路）P95 响应时间不超过 500ms。
- 参考数据规模不超过 10,000 个 Chunk 时，知识检索 P95 不超过 1 秒。
- Mock Tool P95 不超过 500ms。
- Agent 在 1 秒内展示业务处理状态；该指标不要求 1 秒内完成回答。
- Agent 回复支持流式输出。
- LLM Provider 首字和总延迟单独统计，不作为 SupportFlow 平台自身性能缺陷。
- Demo 至少支持 20 个同时在线体验会话。
- Agent 执行并发使用独立、可配置额度；Lite 环境默认 1、最大 2，超限时进入等待或返回明确繁忙提示。
- Trace 分别记录路由、检索、模型、Tool 和总耗时。

### 11.3 服务降级与错误处理

- Provider 不可用时不得伪造模型结果，应提示暂时不可用或切换 Mock 体验能力。
- 知识检索不可用时不得形成无依据知识回答。
- Tool 不可用时按错误分类和重试规则处理，最终失败进入人工接管。
- 所有失败向 Customer 提供可理解的业务提示，向 Trace 提供结构化错误代码。

### 11.4 数据安全与隐私

- v0.2 仅使用 Demo 和 Mock 数据，不鼓励接入真实 Customer 数据。
- 密钥不得写入代码、Trace 或系统日志。
- Trace 和日志对敏感字段进行脱敏。
- 知识库文档上传限制类型、大小和数量；Lite 环境单文件上限为 10 MiB。
- 所有接口执行身份、对象归属和输入校验。
- 知识文档被视为不可信输入，不得覆盖系统规则、改变 Tool 权限或绕过确认。
- Demo 数据执行创建、过期、重置和清理生命周期管理。
- v0.2 只提供基础安全能力，不宣称任何行业合规认证。

### 11.5 Prompt Injection 基础防护

- 区分平台指令、业务配置、知识内容和 Customer 输入的信任层级。
- 知识或 Customer 指令不能修改固定路由、Tool Permission Registry 或高风险规则。
- Tool 参数必须经过结构化校验和权限检查。
- 越权和绕过确认的请求必须拒绝并写入 Trace；v0.3 同步进入 Audit Log。

### 11.6 可靠性与故障恢复

- 创建工单等关键写操作支持 Idempotency Key。
- Agent 和 Tool 重试次数受限。
- Trace 记录失败和中断状态。
- 知识解析和索引明确 `PROCESSING`、`FAILED` 等状态，不产生不可识别的半完成状态。
- 服务重启后已保存的会话、工单和知识状态可恢复。
- 异步任务的权威状态保存在 PostgreSQL；Redis 队列丢失后必须能够根据非终态业务记录重新生成任务。
- 提供健康检查和基础手动备份/恢复文档。
- v0.2 不实现自动企业备份、高可用、故障切换或灾备。

### 11.7 语言与国际化

- 核心 UI 文案使用 i18n 资源管理，支持中文和英文。
- README 以英文为主并提供 `README.zh-CN.md`，不要求全部文档完全双语。
- API 返回稳定错误码和参数，不硬编码用户语言。
- 日期、时间、数字和时区使用国际化格式。
- Agent 跟随 Customer 输入语言回答，但不自动翻译知识库；引用保持原文。

### 11.8 浏览器与可访问性

- 客户聊天页支持桌面和移动端响应式布局，移动端重点保证聊天、状态和流式回复体验。
- 运营后台以桌面端为主。
- 现代浏览器最近两个主要版本作为官方测试范围，不构成绝对兼容承诺。
- 不支持 Internet Explorer，不开发原生 App。
- 提供键盘操作、可见焦点、基础屏幕阅读标签和颜色对比能力。
- v0.2 不引入完整 WCAG 认证要求。

## 12. v0.2 验收口径

### 12.1 验收集管理

- 验收集必须版本化，并在发布前冻结。
- 案例由人工标注意图、预期路由、事实依据、引用、Tool、业务结果和是否转人工。
- 具体案例数量、数据构造方法和自动化方式在测试设计阶段确定。

### 12.2 必须覆盖的路由

- `KNOWLEDGE_ANSWER`
- `ORDER_QUERY`
- `TICKET_CREATION`
- `CLARIFICATION`
- `HUMAN_HANDOFF`

### 12.3 必须覆盖的事实范围

- 产品参数。
- 售后政策。
- 订单状态。
- 保修规则。

闲聊、礼貌回复和流程引导不计入事实性错误率。

### 12.4 必须覆盖的异常场景

- 知识缺失、相关性不足和知识冲突。
- Tool 超时、失败、非法参数和未知执行结果。
- 越权查询、跨 Customer 访问和高风险请求。
- 基础 Prompt Injection。
- 连续三次未解决。
- Customer 主动要求人工。

### 12.5 黄金路径

v0.2 必须完整通过以下端到端路径：

```text
Customer 提问
  → Knowledge 检索与 Citation
  → 方案未解决
  → Mock GetOrder
  → Customer 选择订单
  → Mock 资格校验
  → Customer 确认
  → CreateTicket
  → 返回工单
  → 必要时 Human Handoff
```

Trace 必须完整还原业务执行轨迹，但不保存或展示模型 CoT。

## 13. 开源、部署与协作

### 13.1 开源边界

SupportFlow 使用 Apache License 2.0。Community Edition 不做核心闭环功能阉割，开源内容包括：

- Agent Runtime。
- Knowledge 与 Citation。
- Tool 与 Mock Connector。
- Ticket。
- Human Handoff。
- 基础 Agent Trace 和 Audit 能力。
- NovaTech Demo。

未来企业扩展聚焦 SSO、高可用、高级 Analytics、Workflow、插件治理和商业支持。基础 Trace/Audit 保持开源，高级分析可作为扩展能力。

### 13.2 官方本地部署

- Docker Compose 是 v0.2 官方部署方式。
- `lite` 是 v0.2 默认部署 Profile，以第 11.1 节服务器规格作为最低参考环境。
- Lite 模式将 API、异步 Worker 和 Vue 静态资源合并到同一个 SupportFlow App 运行单元；模块职责保持分离。
- Lite 模式的长驻依赖仅包含 PostgreSQL + pgvector 与 Redis；公开 Demo 可增加轻量 HTTPS Reverse Proxy。
- 原始 Markdown/PDF 使用本地 Volume 保存，不启动独立 Object Storage 服务。
- 默认 Mock 模式可完成完整体验。
- 模型 Provider 支持 Mock 与外部 OpenAI Compatible；不得在 Lite 服务器运行本地 LLM 或 Embedding Model。
- 初始化只负责数据库创建、顺序 Migration 和 NovaTech Demo 数据。
- 区分开发和生产配置。
- 升级采用简单、顺序化 Migration 流程。
- 官方发布流程应提供预构建镜像，服务器通过 `docker compose pull` 和 `docker compose up` 完成升级，不在服务器执行源码构建。
- PostgreSQL 数据与文档 Volume 需要进入基础备份说明；Redis 不作为备份事实源。
- Kubernetes、高可用和自动扩缩容不属于 v0.2。

### 13.3 文档体系

计划采用以下仓库结构基线：

```text
supportflow/
├── .github/
│   ├── ISSUE_TEMPLATE/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── workflows/
├── docs/
├── backend/
├── frontend/
├── deploy/
├── examples/
├── scripts/
├── LICENSE
├── README.md
├── CONTRIBUTING.md
├── CHANGELOG.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
└── .gitignore
```

计划维护以下核心文档：

- `docs/PRD.md`
- `docs/Architecture.md`
- `docs/Database.md`
- `docs/API.md`
- `docs/Roadmap.md`
- `docs/Development.md`
- `docs/Deployment.md`

根目录 `CONTRIBUTING.md` 是唯一贡献入口，避免维护重复的 `docs/Contributing.md`。

README 包含：产品介绍、Features、Screenshots、Architecture、Quick Start、Documentation、Roadmap、Contributing 和 License。README 展示简版 Roadmap，`docs/Roadmap.md` 维护详细规划。

### 13.4 社区治理

- 使用 Conventional Commits，例如 `feat(auth): implement JWT login`、`docs: finish PRD v1`。
- 使用 Semantic Versioning 和持续维护的 `CHANGELOG.md`。
- 保留 `CODE_OF_CONDUCT.md`、`SECURITY.md`、`CONTRIBUTING.md`。
- PR 流程保持轻量，以关联 Issue、变更说明和必要测试为核心。
- Issue 编号由 GitHub 实际创建后生成，不在规划阶段预设编号。
- MVP 阶段不引入复杂审批和重型流程。
- 贡献规范细节和品牌使用策略在后续版本补充。

## 14. 关键产品风险

| 风险 | 影响 | 控制方向 |
| --- | --- | --- |
| 事实性错误 | 错误产品或政策信息损害可信度 | 引用、拒答、事实性错误率、人工验收集 |
| Agent 信任边界与 Prompt Injection | Customer 或知识内容诱导越权 | 固定路由、指令分层、参数校验、Tool 权限和数据隔离 |
| Provider 差异 | 路由、引用和响应表现不稳定 | 参考 Provider、兼容范围声明、Provider 延迟与错误独立统计 |
| Demo 资源滥用 | Token 与基础设施成本失控 | Session 过期、请求/文件/Token 配额、并发限制和数据重置 |
| v0.2 被误用于生产 | 缺少企业治理导致安全或运维风险 | 产品和文档醒目标识，不承诺生产可用 |
| 版本范围膨胀 | 延迟闭环验证 | 冻结 v0.1/v0.2/v0.3 边界，新增需求进入 Roadmap |
| Tool 越权或错误调用 | 访问错误数据或产生错误业务动作 | 最小权限、归属校验、固定 Registry、确认和 Idempotency Key |
| 知识库质量 | 过期、冲突知识导致低质量回答 | 人工审核发布、版本管理、反馈待办和效果指标 |
| 开源维护压力 | Issue、兼容性和贡献质量不可持续 | 轻量贡献流程、清晰范围、版本策略和安全报告流程 |
| Token 成本 | Demo 或企业成本不可预测 | Token/成本字段、配额、模型白名单和后续成本看板 |

## 15. 后续设计阶段输入

PRD 冻结后，按以下顺序继续：

1. 技术架构设计：定义 Agent Runtime、Knowledge、Tool、Ticket、Handoff、Trace 的组件边界和运行流程。
2. 数据库设计：把本文档中的业务对象、状态和关联映射为数据模型。
3. API 设计：定义 Customer、Knowledge、Agent Run、Tool、Ticket、Handoff、Trace 和 Demo 接口。
4. 开发任务拆分：按版本和端到端闭环拆分 Issue/Task。
5. 编码实现。
6. 测试。
7. 部署。

后续设计必须遵守本文档的关键约束：不展示 CoT、业务资格不由 LLM 判断、Tool 权限后端强制校验、关键写操作幂等、v0.2 不扩展为企业生产平台。

## 16. 术语表

| 术语 | 定义 |
| --- | --- |
| Agent Runtime | 在固定路由和安全边界内协调知识、模型、Tool 与人工接管的运行能力 |
| Agent Run | 一次由 Customer 输入触发并进入明确终态的 Agent 执行实例 |
| Agent Trace | Agent Run 的结构化业务执行轨迹，不包含模型 CoT |
| CoT | 模型内部思维或推理过程，不对用户保存或展示 |
| Tool | Agent 可在权限和规则约束下调用的标准业务能力 |
| Business Connector | 连接企业业务系统并实现标准 Tool 契约的适配层 |
| Workspace | v0.3 中企业数据、成员、配置和工具权限的隔离边界；v0.2 仅保留默认概念 |
| Customer | 发起售后咨询的企业客户；v0.2 使用 Demo 身份 |
| Workspace Member | 企业内部客服、知识运营或管理员 |
| Human Handoff | 将会话从 Agent 转交人工客服处理的过程 |
| Ticket | 用于跟踪售后业务处理的工单对象，与会话职责分离 |
| Citation | 知识回答所依据的文档、章节或页码引用 |
| Mock Business Data | NovaTech Demo 使用的模拟 Customer、订单和工单数据 |

---

**冻结结论：** v0.2 的唯一目标是验证 `Customer → Knowledge → Order Tool → Ticket → Human Handoff` 的受控售后 Agent 闭环。任何不服务于该闭环的新增能力默认进入 v0.3 或后续 Roadmap。
