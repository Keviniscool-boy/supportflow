# SupportFlow API 设计

## 1. 文档信息

| 项目 | 内容 |
| --- | --- |
| 文档版本 | v1.0 |
| 文档状态 | 计划中（草案） |
| 对应产品版本 | v0.2 SupportFlow MVP |
| 编写日期 | 2026-07-24 |
| 上游文档 | [PRD](./PRD.md)、[Architecture](./Architecture.md)、[Database](./Database.md) |
| API 版本 | `v1` |

### 1.1 文档目的

本文档冻结 SupportFlow v0.2 的外部 REST、Agent SSE、Demo 身份、运营端资源、错误码、幂等、并发控制以及内部 Tool/Business Connector 契约，作为 OpenAPI、后端实现、前端接入、Task 拆分和测试设计的共同输入。

### 1.2 文档边界

本文档只定义接口契约，不包含：

- Gin Handler、Go 类型、Repository、SQL 或 Migration 实现。
- Vue 页面、TDesign 组件或 Pinia Store 实现。
- OpenAPI 3.1 文件和自动生成 SDK；它们在开发任务阶段由本文档派生。
- v0.3 的企业可信 Customer Token、SSO、完整 RBAC、Workspace 配置、真实 Connector 管理与 Audit API。
- 退款、取消订单、修改订单、删除 Customer 数据等高风险 Tool。
- WebSocket、GraphQL、Webhook、插件或多 Agent API。

## 2. API 目标与强制原则

1. Customer、Operations 和内部 Tool 接口边界清晰，任何浏览器端都不能直接执行 Tool。
2. `tenant_id`、Customer 身份、Member 身份和权限只来自服务端验证上下文，不接受请求 Payload 覆盖。
3. 普通查询使用 REST + JSON；Agent 流式执行使用 Fetch POST + SSE，不引入 WebSocket。
4. API 返回稳定错误码和参数，不把中文或英文后端文案作为机器判断依据。
5. Customer 对象访问同时校验 Tenant、Session 和对象归属，避免 IDOR 和跨 Customer 查询。
6. 创建工单等关键写操作具备幂等保护；状态变更具备乐观并发保护。
7. SSE 只发送业务处理状态和安全结果，不发送 Prompt、CoT、凭证、完整 Chunk 或未脱敏内容。
8. 流式 Delta 是瞬时数据；最终 Message、Agent Run、Ticket 和 Handoff 以 PostgreSQL 查询结果为准。
9. v0.2 是单 Workspace Demo API，不宣称企业生产身份或隔离能力。

## 3. 路由空间与版本

### 3.1 Base Path

| 路由空间 | Base Path | 用途 |
| --- | --- | --- |
| Demo Bootstrap | `/api/v1/demo` | 创建、查询、重置 Demo Session 和取得沙箱运营身份 |
| Customer API | `/api/v1/customer` | 客户会话、Agent、反馈、状态中心、工单、接管和通知 |
| Operations API | `/api/v1/operations` | 客服、知识运营和 Demo Admin 工作台 |
| Health | `/health` | 容器 Liveness 与 Readiness |

内部 Tool、Connector 和 Domain Service 使用进程内 Port，不注册外部 HTTP 路由。禁止提供通用 `/tools/execute`、`/internal/sql`、`/prompt` 或模型代理端点。

### 3.2 版本策略

- 主版本写入路径 `/api/v1`；破坏性变更进入 `/api/v2`。
- 在不改变既有语义时，可以新增可选字段、枚举以外的扩展元数据或新端点。
- Response Consumer 必须忽略未知字段；Command Request 默认拒绝未知字段，避免拼写错误静默生效。
- 固定状态机枚举不能在 v0.2 验收期间动态扩展。
- SSE Payload 独立包含 `schema_version: 1`；破坏事件结构时增加 Schema Version。

## 4. 传输与序列化约定

### 4.1 HTTP

- 公网 Demo 只通过 HTTPS 提供；本地开发可使用 `http://localhost`。
- JSON Request/Response 使用 `application/json; charset=utf-8`。
- 文件上传使用 `multipart/form-data`。
- SSE Response 使用 `text/event-stream; charset=utf-8`。
- 默认请求超时 15 秒；Agent SSE 和文件上传使用各自超时规则。
- 不使用 HTTP 302 把 API 错误重定向到登录页。

### 4.2 JSON

| 数据 | 表示方式 |
| --- | --- |
| 字段名 | `snake_case` |
| ID | UUID 字符串；业务编号使用单独字段 |
| 时间 | RFC 3339 UTC，例如 `2026-07-24T18:30:00Z` |
| 枚举 | 大写蛇形，例如 `WAITING_CONFIRMATION` |
| 金额 | `{ "amount_minor": 19900, "currency": "CNY" }` |
| 空值 | 仅在“已知但当前不存在”时返回 `null`；列表始终返回数组 |
| 布尔 | JSON `true` / `false`，不用 `0` / `1` |
| 大整数 | Token、金额和耗时在 JSON 中使用安全整数范围；超出时改为字符串 |

Request Body 最大 256 KiB；Customer 单条文本最大 8,000 Unicode 字符。文本先执行 UTF-8、控制字符、长度和必要脱敏校验。Customer 附件字段不属于 v0.2，请求中出现时返回 `UNKNOWN_FIELD`。

### 4.3 通用请求头

| Header | 必需范围 | 说明 |
| --- | --- | --- |
| `Accept` | 推荐 | JSON 或 `text/event-stream` 内容协商 |
| `Content-Type` | 有 Body 时 | JSON、Multipart |
| `X-Request-ID` | 可选 | 客户端 UUID；不合法时服务端替换 |
| `X-CSRF-Token` | Cookie 鉴权的 Command | 与 Demo Session 绑定的 CSRF Token |
| `Idempotency-Key` | 指定关键写操作 | 8–128 个可打印 ASCII 字符，推荐 UUID |
| `If-Match` | 受并发保护的状态变更 | 使用资源 ETag，例如 `"rv-7"` |
| `Accept-Language` | 可选 | UI 首选语言；错误仍以代码为准 |

所有 Response 返回 `X-Request-ID`。公网代理不得记录 Cookie、CSRF Token、Idempotency Key 原值或 Authorization 凭证。

### 4.4 语言与国际化

- API 枚举、错误码、`message_key` 和字段名不随语言变化。
- `Accept-Language` 和 Customer Message `locale` 只表达语言偏好，不改变权限或业务规则。
- Agent 跟随 Customer 输入语言回答；不自动翻译知识文档，Citation 标题、章节和摘录保持来源原文。
- 日期时间始终传输 UTC RFC 3339，由前端按 Locale/时区格式化。
- Notification 使用 `title_code + title_params`，Backend 不持久化必须展示的固定语言文案。

## 5. 身份、Session 与权限

### 5.1 v0.2 Demo 身份

Customer 浏览器调用 `POST /api/v1/demo/sessions` 后，Backend 设置：

- `sf_demo_customer`：高熵、HttpOnly、SameSite=Lax Cookie；公网环境设置 Secure。
- Response Body 中的 `csrf_token`：只在当前页面内存保存，用于 Command Header。

数据库只保存 Session Token 哈希。Cookie 绑定 Demo Session、默认 Tenant、临时 Customer、数据代次和过期时间；客户端不得提交 `tenant_id` 或任意 `customer_id`。

运营体验通过 `POST /api/v1/demo/operator-access` 获取单独的 `sf_demo_member` HttpOnly Cookie。该身份绑定当前 Demo Session 和允许角色，不是 Platform Admin，也不能跨沙箱查看数据。

### 5.2 Session 状态

| 状态 | API 行为 |
| --- | --- |
| `ACTIVE` | 允许配额内的 Customer/Operations 请求 |
| `EXPIRED`、`REVOKED` | 返回 `401 SESSION_EXPIRED`，清除 Cookie |
| `RESETTING` | 只允许查询 Session 和重置状态，其余返回 `409 DEMO_RESET_IN_PROGRESS` |
| `RESET` | 允许查询结果和重新创建 Session，不允许读取旧业务数据 |

Session 到期不意味着正在执行的 Tool 可以继续无限运行；Backend 按已持久化 Run 状态安全终止或完成已提交事务。

### 5.3 v0.2 角色矩阵

| 能力 | Customer | Support Agent | Knowledge Operator | Demo Admin |
| --- | --- | --- | --- | --- |
| 自己的会话、状态、通知 | 读写 | 只按接管范围读 | 否 | 沙箱内读 |
| 人工接管队列和领取 | 否 | 是 | 否 | 是 |
| 工单处理与内部备注 | 只读公开内容/补充 | 是 | 否 | 是 |
| 知识文档和版本 | 只通过引用读取 | 否 | 是 | 是 |
| 知识运营待办 | 否 | 可标记知识缺口 | 是，只有脱敏上下文 | 是 |
| Agent Trace | 否 | 沙箱内安全 Trace | 否 | 沙箱内安全 Trace |
| Platform Admin | 否 | 否 | 否 | 否 |

权限必须由 Backend Handler 和 Application Service 双重校验。前端隐藏按钮不是权限控制。

### 5.4 对象归属与枚举防护

- Customer 使用不属于自己的 Conversation、Message、Run、Ticket、Handoff、Citation 或 Notification ID 时统一返回 `404 RESOURCE_NOT_FOUND`。
- Operations 身份已认证但角色不足时返回 `403 PERMISSION_DENIED`。
- 列表查询自动加入当前 Tenant、Demo Session 和主体范围，不接受 URL 参数扩大范围。
- Mock Order 查询始终从当前 Customer 上下文取得主体；请求不得传入另一个 Customer ID。
- Public Demo 的 NovaTech 基线知识只读；用户上传知识只在当前 Demo Session 内可见和可管理。

### 5.5 CSRF 与 CORS

- Cookie 鉴权的 `POST`、`PUT`、`PATCH`、`DELETE` 必须校验 `X-CSRF-Token`。
- Public Demo 默认同源部署，不开放通配符 Credentialed CORS。
- Development 可配置明确 Origin 白名单；不接受请求参数动态设置 Origin。
- SSE 使用 Fetch POST，沿用 Cookie 和 CSRF 校验。

## 6. 通用响应与错误模型

### 6.1 成功响应

单资源响应：

```json
{
  "data": {
    "id": "019b..."
  },
  "meta": {
    "request_id": "019b..."
  }
}
```

异步 Command 使用 `202 Accepted`，同时返回业务资源和 Durable Task 引用。真正成功以资源/任务终态为准，不能把“已入队”表达为“处理完成”。

### 6.2 列表响应

```json
{
  "data": [],
  "meta": {
    "request_id": "019b...",
    "next_cursor": null,
    "has_more": false
  }
}
```

### 6.3 错误响应

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message_key": "errors.invalid_argument",
    "params": {
      "field": "message.text"
    },
    "retryable": false,
    "details": [
      {
        "field": "message.text",
        "code": "REQUIRED"
      }
    ],
    "request_id": "019b..."
  }
}
```

- `message_key` 和 `params` 供前端 i18n；后端不返回必须展示的固定中文/英文文案。
- `details` 只用于安全的字段级校验，不暴露 SQL、堆栈、Provider 原始响应或策略内部信息。
- `retryable` 表示技术上是否允许重试，不代表业务操作一定应由客户端自动重试。
- Tool 业务拒绝和技术失败必须使用不同代码与字段。

### 6.4 HTTP Status

| Status | 使用场景 |
| --- | --- |
| `200 OK` | 查询、幂等重放、同步 Command 成功 |
| `201 Created` | 同步创建资源 |
| `202 Accepted` | 已持久化异步任务或 Run，尚未完成 |
| `204 No Content` | 无 Body 的成功更新 |
| `400 Bad Request` | JSON、Header、Cursor 或基础参数格式错误 |
| `401 Unauthorized` | Session 缺失、无效、过期 |
| `403 Forbidden` | Operations 角色不足、CSRF 失败 |
| `404 Not Found` | 资源不存在或 Customer 无权知道其存在 |
| `409 Conflict` | 活动 Run、状态机、重复工单或 Idempotency 冲突 |
| `412 Precondition Failed` | `If-Match` 的 `row_version` 已过期 |
| `413 Payload Too Large` | JSON 或文件超过限制 |
| `415 Unsupported Media Type` | 文件/Content-Type 不支持 |
| `422 Unprocessable Content` | 结构合法但业务字段校验失败 |
| `429 Too Many Requests` | 配额、速率或并发限制 |
| `500 Internal Server Error` | 未分类服务错误，只返回稳定错误码和 Request ID |
| `503 Service Unavailable` | PostgreSQL/Redis/Provider 等依赖导致服务不可用 |
| `504 Gateway Timeout` | 上游在明确超时内未返回且结果可确定为未完成 |

### 6.5 v0.2 错误码目录

| Code | HTTP | Retryable | 含义 |
| --- | --- | --- | --- |
| `INVALID_JSON` | 400 | 否 | JSON 无法解析 |
| `INVALID_ARGUMENT` | 422 | 否 | 字段不符合业务约束 |
| `UNKNOWN_FIELD` | 400 | 否 | Command 含未定义字段 |
| `INVALID_CURSOR` | 400 | 否 | Cursor 无效、过期或过滤条件不匹配 |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | 否 | Content-Type 或文件类型不支持 |
| `PAYLOAD_TOO_LARGE` | 413 | 否 | Body 或文件超限 |
| `UNAUTHENTICATED` | 401 | 否 | 缺少有效身份 |
| `SESSION_EXPIRED` | 401 | 否 | Demo Session 过期或撤销 |
| `CSRF_VALIDATION_FAILED` | 403 | 否 | CSRF 校验失败 |
| `PERMISSION_DENIED` | 403 | 否 | Operations 角色不足 |
| `RESOURCE_NOT_FOUND` | 404 | 否 | 不存在或不可见 |
| `RATE_LIMITED` | 429 | 是 | 请求速率超限，遵循 `Retry-After` |
| `QUOTA_EXCEEDED` | 429 | 否 | Demo 请求、Token 或文件配额用尽 |
| `VERSION_CONFLICT` | 412 | 是 | ETag/row_version 过期，需重新读取 |
| `IDEMPOTENCY_CONFLICT` | 409 | 否 | 相同 Key 对应不同请求指纹 |
| `ACTIVE_RUN_EXISTS` | 409 | 否 | 会话已有另一个活动 Run |
| `INVALID_STATE_TRANSITION` | 409 | 否 | 当前状态不允许该 Command |
| `AGENT_BUSY` | 429 | 是 | Agent 全局并发额度已满 |
| `PROVIDER_UNAVAILABLE` | 503 | 是 | Model Provider 暂时不可用 |
| `KNOWLEDGE_UNAVAILABLE` | 503 | 是 | 检索基础设施不可用 |
| `TOOL_TEMPORARILY_UNAVAILABLE` | 503 | 是 | Tool 限流或临时不可用 |
| `TOOL_RESULT_UNKNOWN` | 503 | 否 | 写 Tool 结果未知，需同 Key 核对 |
| `ORDER_NOT_FOUND` | 404 | 否 | 当前 Customer 范围内无订单 |
| `TICKET_INELIGIBLE` | 409 | 否 | Business Service 判定不具备售后资格 |
| `TICKET_DUPLICATE_ACTIVE` | 409 | 否 | 已存在匹配的活动工单，并返回该工单引用 |
| `HANDOFF_ALREADY_ACTIVE` | 409 | 否 | 会话已有活动接管 |
| `HANDOFF_ALREADY_CLAIMED` | 409 | 否 | 接管已被其他客服领取 |
| `DOCUMENT_NOT_REVIEWABLE` | 409 | 否 | Version 未达到待审核状态 |
| `DOCUMENT_PROCESSING_FAILED` | 409 | 可选 | Version 处理失败，可人工重试 |
| `FILE_TYPE_NOT_ALLOWED` | 415 | 否 | 非 Markdown 或文本型 PDF |
| `DEMO_RESET_IN_PROGRESS` | 409 | 是 | 沙箱正在重置 |
| `TASK_FAILED` | 409 | 可选 | Durable Task 已失败 |
| `SERVICE_UNAVAILABLE` | 503 | 是 | 核心依赖不可用 |
| `INTERNAL_ERROR` | 500 | 可选 | 未分类错误，仅返回 Request ID |

错误码可以增加，但不能复用既有 Code 表达不同语义。

## 7. 分页、过滤、并发与幂等

### 7.1 Keyset Pagination

- 列表参数：`limit` 默认 20，最小 1，最大 100；`cursor` 为不透明字符串。
- Cursor 绑定 Tenant、主体、排序字段和过滤条件；修改过滤条件后不能复用旧 Cursor。
- Cursor 不是授权凭证，每页仍重新做对象归属校验。
- 服务端使用带完整性保护的 Base64URL Cursor，不暴露可篡改 SQL Offset。
- 时间线资源优先使用稳定序号：`after_sequence` 默认 0，`limit` 最大 200。

### 7.2 过滤与排序

- 每个列表端点只接受端点矩阵中定义的过滤字段，未知过滤字段返回 `INVALID_ARGUMENT`。
- 排序使用固定白名单，不接受任意数据库列名。
- 默认排序：Conversation/Notification 按时间倒序，Ticket/Handoff 队列按优先级降序和创建时间升序，Timeline 按 Sequence 升序。
- 时间范围使用 `created_after`、`created_before` RFC 3339 UTC；最大查询窗口由部署配置限制。

### 7.3 ETag 与 If-Match

可变资源 Response 返回 `ETag: "rv-{row_version}"`。以下 Command 强制 `If-Match`：

- Handoff 领取、结束、取消。
- Ticket 领取、转派、状态变更。
- Customer Run 补充信息、确认和取消。
- Customer/Operations 追加 Conversation Message 或 Ticket Activity。
- Knowledge Document 元数据/禁用。
- Knowledge Version 重试/发布。
- Knowledge Todo 领取和处理。

版本不匹配返回 `412 VERSION_CONFLICT` 和当前资源 ETag，不自动覆盖。

### 7.4 Idempotency-Key

- `CreateTicket`、手工创建 Ticket、Demo Reset 等关键写操作使用 `Idempotency-Key`。
- Key 作用域至少包含 Tenant、Customer/Member 主体和 Operation。
- 相同 Key + 相同请求指纹返回原结果；相同 Key + 不同指纹返回 `409 IDEMPOTENCY_CONFLICT`。
- 正在执行返回 `202` 和同一资源/操作引用，不并发执行第二次。
- 结果未知时客户端只能用同一 Key 核对，不能换 Key 重新创建。
- 原始 Key 不写入 Trace、日志或数据库，只保存哈希。

### 7.5 Client Request ID

首次创建 Agent Run 的 Command Body 必须包含 `client_request_id` UUID。其作用域为 Conversation：

- 相同 ID 重试返回已创建的 Message/Run 状态，不创建第二个 Run。
- 相同 ID 不能对应不同 Message 内容，否则返回 `IDEMPOTENCY_CONFLICT`。
- 不同 ID 且已有活动 Run 时返回 `ACTIVE_RUN_EXISTS`。
- 同一 Run 的后续补充、确认和取消使用资源 ETag/`If-Match`，不创建第二个 Run Request ID。

## 8. 通用资源表示

### 8.1 `Conversation`

| 字段 | 类型 | 可见范围 | 说明 |
| --- | --- | --- | --- |
| `id` | UUID | Customer/Operations | 内部 ID |
| `conversation_number` | string | Customer/Operations | 展示编号 |
| `status` | enum | Customer/Operations | `OPEN`、`CLOSED` |
| `response_owner` | enum | Customer/Operations | `AGENT`、`HUMAN` |
| `subject` | string/null | Customer/Operations | 脱敏标题 |
| `active_run` | `AgentRunSummary`/null | Customer/Operations | 当前活动 Run |
| `active_handoff` | `HandoffSummary`/null | Customer/Operations | 当前接管 |
| `last_message_at` | timestamp/null | Customer/Operations | 列表排序 |
| `created_at`、`updated_at` | timestamp | Customer/Operations | 生命周期 |
| `row_version` | integer | Operations | Customer 不依赖该字段 |

### 8.2 `Message`

```json
{
  "id": "019b...",
  "conversation_id": "019b...",
  "sequence_no": 12,
  "actor": {
    "type": "AGENT",
    "display_name": "Nova"
  },
  "content": {
    "type": "TEXT",
    "text": "建议先尝试重置耳机。"
  },
  "citations": [],
  "created_at": "2026-07-24T18:30:00Z"
}
```

`actor.type` 为 `CUSTOMER`、`AGENT`、`MEMBER`、`SYSTEM`。`content.type` 为 `TEXT`、`ORDER_CARD`、`TICKET_CARD`、`HANDOFF_STATUS`、`SYSTEM_STATUS`。Operations 内部备注不作为 Conversation Message 返回。

### 8.3 `AgentRunSummary`

| 字段 | Customer | Operations | 说明 |
| --- | --- | --- | --- |
| `id`、`conversation_id` | 是 | 是 | 关联 |
| `status` | 是 | 是 | 固定 Run 状态 |
| `business_status` | 是 | 是 | 可展示业务状态 |
| `current_route` | 否 | 是 | 固定五类路由 |
| `route_reason_code` | 否 | 是 | 结构化原因 |
| `pending_action` | 是，安全摘要 | 是 | 等待确认时返回 |
| `failure` | 安全错误 | 结构化错误 | 不含内部堆栈 |
| `started_at`、`finished_at` | 是 | 是 | 生命周期 |

Customer 的 `business_status` 只允许：`QUEUED`、`ANALYZING`、`RETRIEVING_KNOWLEDGE`、`QUERYING_ORDER`、`WAITING_FOR_INFORMATION`、`WAITING_FOR_CONFIRMATION`、`CREATING_TICKET`、`ESCALATING_TO_HUMAN`、`COMPOSING_RESPONSE`、`COMPLETED`、`FAILED`。

### 8.4 `Citation`

```json
{
  "id": "019b...",
  "rank": 1,
  "source_title": "蓝牙耳机常见问题 FAQ",
  "section_title": "3. 单耳无声音",
  "page_number": null,
  "quote_excerpt": "请将两只耳机放回充电盒后执行重置。"
}
```

Customer Citation 不返回 Chunk ID、Index Version、向量分数或内部 Object Key。Operations Trace 可以返回安全的文档/版本/分数元数据，但仍不返回完整 Chunk。

### 8.5 `TicketSummary`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id`、`ticket_number` | UUID/string | 内部 ID 与展示编号 |
| `status` | enum | 固定 Ticket 状态 |
| `priority` | enum | `LOW`、`NORMAL`、`HIGH`、`URGENT` |
| `problem_type` | string | 受控代码 |
| `problem_summary` | string | 脱敏摘要 |
| `order_reference` | string/null | 可选掩码订单引用 |
| `product_reference` | string/null | 可选商品快照 |
| `source`、`creation_reason` | string | 创建来源与原因 |
| `assignee` | object/null | Customer 仅返回显示名，不返回内部身份信息 |
| `created_at`、`updated_at` | timestamp | 生命周期 |

### 8.6 `HandoffSummary`

包含 `id`、`handoff_number`、`conversation_id`、`status`、`reason_code`、`priority`、`assigned_member`、`requested_at`、`accepted_at`、`ended_at` 和可选 `sla_due_at`。Customer 不返回内部上下文摘要或知识缺口标记。

### 8.7 `Notification`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | UUID | 通知 ID |
| `event_type` | string | 固定业务事件代码 |
| `title_code` | string | 前端 i18n Key |
| `title_params` | object | 已脱敏插值参数 |
| `target` | object | `type`、`id` 和安全业务编号 |
| `navigation_path` | string | 服务端允许的站内相对路径 |
| `read_at` | timestamp/null | 空表示未读 |
| `created_at` | timestamp | 业务事件时间 |

Notification 与 Conversation Message、Knowledge Todo 和系统日志相互独立。

### 8.8 `DurableTaskSummary`

包含 `id`、`task_type`、`status`、`attempt_count`、`max_attempts`、`progress`（可空）、`last_error`（安全错误）、`created_at`、`started_at`、`finished_at`。不得返回 Asynq Task ID、Lease Owner 或内部 Payload。

## 9. Demo Session API

### 9.1 端点矩阵

| Method | Path | 身份 | 成功 | 说明 |
| --- | --- | --- | --- | --- |
| `POST` | `/api/v1/demo/sessions` | 匿名/已有 Demo | 201/200 | 创建或返回当前有效 Session |
| `GET` | `/api/v1/demo/session` | Demo Cookie | 200 | 查询 Session、Customer、配额和重置状态 |
| `DELETE` | `/api/v1/demo/session` | Customer + CSRF | 202 | 撤销并异步清理当前 Demo 数据 |
| `POST` | `/api/v1/demo/session/reset` | Customer + CSRF + Idempotency | 202 | 请求恢复 NovaTech 初始数据 |
| `POST` | `/api/v1/demo/operator-access` | Customer + CSRF | 200 | 取得当前沙箱运营体验身份 |

### 9.2 创建 Session

`POST /api/v1/demo/sessions`

Request：

```json
{
  "locale": "zh-CN"
}
```

Response `201`：

```json
{
  "data": {
    "session": {
      "status": "ACTIVE",
      "expires_at": "2026-07-25T18:30:00Z",
      "data_generation": 1,
      "quota": {
        "requests_remaining": 20,
        "tokens_remaining": 50000,
        "upload_files_remaining": 3
      }
    },
    "customer": {
      "id": "019b...",
      "display_name": "NovaTech Visitor",
      "locale": "zh-CN"
    },
    "csrf_token": "opaque-csrf-token"
  },
  "meta": {
    "request_id": "019b..."
  }
}
```

已有有效 Cookie 时返回 `200` 和同一 Session，不重复创建 Customer。原始 Session Token 不出现在 JSON。

### 9.3 Reset

`POST /api/v1/demo/session/reset` 必须携带 `Idempotency-Key`。Response 返回 `reset_task: DurableTaskSummary`。重置期间只有 `GET /demo/session` 可用；完成后前端重新调用 Session 创建端点取得新代次。

### 9.4 Operator Access

Request：

```json
{
  "role": "SUPPORT_AGENT"
}
```

允许角色为 `SUPPORT_AGENT`、`KNOWLEDGE_OPERATOR`、`DEMO_ADMIN`。Public Demo 可通过部署配置禁用 `DEMO_ADMIN`。Response 设置单独 Member Cookie 并返回角色、CSRF Token 和过期时间；不返回密码、平台权限或其他 Session 数据。

## 10. Customer Conversation 与 Agent Run API

### 10.1 端点矩阵

| Method | Path | 成功 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/customer/conversations` | 201 | 创建空会话 |
| `GET` | `/api/v1/customer/conversations` | 200 | 当前 Customer 会话列表 |
| `GET` | `/api/v1/customer/conversations/{conversation_id}` | 200 | 会话详情 |
| `GET` | `/api/v1/customer/conversations/{conversation_id}/messages` | 200 | 按 Sequence 读取消息 |
| `POST` | `/api/v1/customer/conversations/{conversation_id}/agent-runs` | 200 SSE / 202 JSON | 写入首条消息并创建 Run |
| `GET` | `/api/v1/customer/agent-runs/{run_id}` | 200 | 查询持久化 Run 状态 |
| `POST` | `/api/v1/customer/agent-runs/{run_id}/messages` | 200 SSE / 202 JSON | `WAITING_USER` 时补充信息 |
| `POST` | `/api/v1/customer/agent-runs/{run_id}/confirmation` | 200 SSE / 202 JSON | 确认或拒绝待执行动作 |
| `POST` | `/api/v1/customer/agent-runs/{run_id}/cancel` | 200 | 取消可取消 Run |
| `POST` | `/api/v1/customer/conversations/{conversation_id}/handoff-requests` | 201/200 | Customer 明确请求人工 |

所有端点都要求有效 Customer Cookie；所有 Command 要求 CSRF。

### 10.2 创建 Conversation

Request 可选提供脱敏主题：

```json
{
  "subject": "蓝牙耳机使用问题"
}
```

Response 返回 `Conversation`。服务器不得接受 Customer ID、Tenant ID 或 `response_owner`。

### 10.3 消息恢复

`GET /conversations/{id}/messages?after_sequence=0&limit=100`

- 默认正序返回，保证前端可确定性拼接。
- `after_sequence` 表示只返回更大序号。
- Response 包含 `next_after_sequence` 和 `has_more`。
- Citation 可随 Agent Message 内嵌；也可通过 Citation 端点重新查询。

### 10.4 创建 Run

Request：

```json
{
  "client_request_id": "019b...",
  "message": {
    "text": "我的蓝牙耳机左耳没有声音了",
    "locale": "zh-CN"
  }
}
```

处理必须原子创建 Customer Message、Agent Run、Trace 根和首 Trace Event。`Accept: text/event-stream` 时进入 SSE；`Accept: application/json` 时返回 `202` 和 `message`、`agent_run`，客户端轮询 Run 和 Messages。

如果 1 秒内无法开始模型处理，也必须返回 `run.accepted` 或 `business.status: QUEUED`，不能保持无反馈空连接。

### 10.5 补充信息

`POST /agent-runs/{run_id}/messages` 仅允许 Run 为 `WAITING_USER`，并要求当前 Run 的 `If-Match`。Body 只包含 `message`：

```json
{
  "message": {
    "text": "订单号是 SF20260001",
    "locale": "zh-CN"
  }
}
```

Message 写入和 Run 转回 `RUNNING` 必须处于同一事务。网络重试先重新读取 Run；旧 ETag 返回 `VERSION_CONFLICT`，不会追加第二条 Message。Run 已终态时返回 `INVALID_STATE_TRANSITION`，前端应创建新 Run。

### 10.6 确认业务动作

Request：

```json
{
  "decision": "CONFIRM"
}
```

`decision` 仅为 `CONFIRM`、`REJECT`，并要求当前 Run 的 `If-Match`。服务端使用 Run 已保存的 `pending_action`，不允许客户端重新提交 Tool 名、Customer ID、价格、资格结果或任意 Tool 参数。

- `CONFIRM`：写入确认 Message/事实后，继续同一 Run。
- `REJECT`：Run 转为 `CANCELLED` 或安全完成，不执行写 Tool。
- 待确认内容过期时返回 `INVALID_STATE_TRANSITION`，不得按旧参数执行。

### 10.7 取消 Run

取消要求当前 Run 的 `If-Match`，且只改变状态，不删除 Run、Message 或 Trace。已经进入不可回滚的 Ticket 事务时，API 返回当前持久化结果，不声称取消已提交 Ticket。

### 10.8 Customer 请求人工

Request 固定为：

```json
{
  "reason": "CUSTOMER_REQUEST"
}
```

如果已有活动 Handoff，幂等返回该 Handoff。创建成功后 Agent 停止发起新的自动回复，Conversation 状态由 Handoff 规则控制。

## 11. Agent SSE 协议

### 11.1 Response Headers

```text
Content-Type: text/event-stream; charset=utf-8
Cache-Control: no-cache, no-transform
Connection: keep-alive
X-Accel-Buffering: no
X-Request-ID: <uuid>
```

反向代理必须关闭 SSE Buffering。服务端每 15 秒发送注释 Heartbeat；Heartbeat 不计入业务事件序号。

### 11.2 帧格式

```text
id: 4
event: business.status
data: {"schema_version":1,"stream_sequence":4,"request_id":"...","conversation_id":"...","run_id":"...","occurred_at":"2026-07-24T18:30:00Z","status":"RETRIEVING_KNOWLEDGE"}
```

所有 Data Payload 包含：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `schema_version` | integer | v0.2 固定 1 |
| `stream_sequence` | integer | 当前连接内从 1 递增 |
| `request_id` | UUID | HTTP Request 关联 |
| `conversation_id` | UUID | 会话 |
| `run_id` | UUID | Run |
| `occurred_at` | timestamp | 事件时间 |

### 11.3 事件目录

| Event | 持久性 | Customer 内容 |
| --- | --- | --- |
| `run.accepted` | Run 已持久化 | Run ID、状态、是否排队 |
| `run.status` | Run 状态持久化 | `CREATED`、`RUNNING`、等待或终态 |
| `business.status` | Trace/Run 可恢复摘要 | 可展示业务状态，不含内部推理 |
| `message.delta` | 瞬时 | Assistant 文本增量和增量序号 |
| `citation.added` | Citation 已持久化 | Customer `Citation` |
| `tool.started` | Trace/Tool Call 已持久化 | `ORDER_QUERY` 或 `TICKET_CREATION` 安全类型 |
| `tool.completed` | Tool Result 已持久化 | 技术状态、业务结果和安全卡片 |
| `order.presented` | Message/Tool Result 可恢复 | 当前 Customer 的订单卡片数组 |
| `ticket.created` | Ticket 已持久化 | `TicketSummary` |
| `handoff.updated` | Handoff 已持久化 | `HandoffSummary` |
| `message.completed` | Message 已持久化 | 完整最终 Agent Message |
| `run.completed` | Run 终态 | 最终 Run 摘要 |
| `run.escalated` | Run 终态 | `ESCALATED` 和 Handoff 引用 |
| `run.cancelled` | Run 终态 | `CANCELLED` 和安全原因代码 |
| `run.failed` | Run/Trace 失败已持久化 | 安全错误对象 |
| `stream.end` | 连接控制 | 当前持久化状态和恢复 URL |

不提供 `model.thought`、`chain_of_thought`、完整 Prompt、原始 Tool 参数、完整 Chunk 或 Provider 原始响应事件。

### 11.4 关键事件示例

`run.accepted`：

```json
{
  "schema_version": 1,
  "stream_sequence": 1,
  "request_id": "019b...",
  "conversation_id": "019b...",
  "run_id": "019b...",
  "occurred_at": "2026-07-24T18:30:00Z",
  "status": "CREATED",
  "queued": false
}
```

`message.delta`：

```json
{
  "schema_version": 1,
  "stream_sequence": 5,
  "request_id": "019b...",
  "conversation_id": "019b...",
  "run_id": "019b...",
  "occurred_at": "2026-07-24T18:30:01Z",
  "message_id": "019b...",
  "delta_sequence": 1,
  "delta": "建议您先尝试"
}
```

`tool.completed`：

```json
{
  "schema_version": 1,
  "stream_sequence": 8,
  "request_id": "019b...",
  "conversation_id": "019b...",
  "run_id": "019b...",
  "occurred_at": "2026-07-24T18:30:02Z",
  "tool_kind": "ORDER_QUERY",
  "execution_status": "SUCCEEDED",
  "business_result": "SUCCESS",
  "safe_output": {
    "order_count": 1
  }
}
```

### 11.5 错误与连接关闭

- 在 Response Headers 发出前失败：返回普通 JSON Error 和对应 HTTP Status。
- Stream 开始后失败：发送 `run.failed`，随后 `stream.end` 并关闭连接；不得在 SSE 中混入 JSON Error Envelope。
- Browser 主动断开不等于取消 Run；Backend 在执行窗口内继续并持久化终态。
- `stream.end` 之后不得再发送业务事件。
- Agent 总执行超过配置上限时进入已持久化 `FAILED` 或安全 Handoff，不让连接无限保持。

### 11.6 断线恢复

v0.2 不保证通过 `Last-Event-ID` 重放 Token Delta，也不提供跨实例 SSE Fan-out。断线后客户端：

1. 调用 `GET /customer/agent-runs/{run_id}` 读取权威状态。
2. 调用 Message Sequence 接口读取最终持久化消息。
3. Run 仍执行时按退避轮询，不重复创建不同 `client_request_id` 的 Run。

### 11.7 事件顺序与持久化

- 首个业务事件必须是 `run.accepted`；之后才能发送 Route 对应的业务状态。
- `message.delta.message_id` 是预分配 ID，在 `message.completed` 前不代表 Message 已持久化。
- 最终 Message 和 Citation 在同一业务完成事务中持久化后，才能发送 `citation.added` 和 `message.completed`。
- `COMPLETED`、`ESCALATED`、`FAILED`、`CANCELLED` 分别发送对应终态事件，再发送 `stream.end`。
- `WAITING_USER`、`WAITING_CONFIRMATION` 是可暂停状态：发送 `run.status` 和 `stream.end`；后续 Command 建立新 SSE 连接，`stream_sequence` 从 1 重新开始。
- `stream.end` 不改变 Run 状态，只表示当前 HTTP Stream 已结束。

## 12. Customer Feedback、Citation 与状态中心

### 12.1 端点矩阵

| Method | Path | 成功 | 说明 |
| --- | --- | --- | --- |
| `PUT` | `/api/v1/customer/messages/{message_id}/feedback` | 200 | 幂等创建/更新反馈 |
| `GET` | `/api/v1/customer/messages/{message_id}/citations` | 200 | 读取可展示引用 |
| `GET` | `/api/v1/customer/status-center` | 200 | 聚合售后状态投影 |
| `GET` | `/api/v1/customer/tickets` | 200 | 当前 Customer 工单 |
| `GET` | `/api/v1/customer/tickets/{ticket_id}` | 200 | 工单详情 |
| `GET` | `/api/v1/customer/tickets/{ticket_id}/activities` | 200 | 只返回 Public Timeline |
| `POST` | `/api/v1/customer/tickets/{ticket_id}/activities` | 201 | Customer 补充内容 |
| `GET` | `/api/v1/customer/handoff-requests/{handoff_id}` | 200 | 接管状态 |
| `GET` | `/api/v1/customer/notifications` | 200 | 站内通知 |
| `PATCH` | `/api/v1/customer/notifications/{notification_id}` | 200 | 更新已读状态 |

### 12.2 Feedback

Request：

```json
{
  "value": "NOT_HELPFUL",
  "reason_code": "SOLUTION_DID_NOT_WORK",
  "comment": "重置后仍然只有右耳有声音"
}
```

- 只允许评价当前 Customer 会话中的 Agent Message。
- `comment` 最大 500 字符，入库前脱敏。
- `NOT_HELPFUL` 必须触发或复用 Knowledge Operations Todo。
- Response 不暴露内部 Todo ID 给 Customer。

### 12.3 Status Center

`GET /status-center?cursor=...&limit=20` 返回按更新时间倒序的只读投影：

```json
{
  "data": [
    {
      "type": "TICKET",
      "target": {
        "id": "019b...",
        "number": "TK20260001",
        "status": "PENDING"
      },
      "updated_at": "2026-07-24T18:30:00Z",
      "unread": true
    }
  ],
  "meta": {
    "request_id": "019b...",
    "next_cursor": null,
    "has_more": false
  }
}
```

`type` 为 `CONVERSATION`、`TICKET`、`HANDOFF`。该端点不创建独立业务事实；数据来自现有资源和 Notification 投影。

### 12.4 Ticket Activity

Customer `GET` 只返回 `visibility = PUBLIC`。Customer `POST` 固定创建 `CUSTOMER_UPDATE`，Request：

```json
{
  "body": "我已经补充了设备序列号。"
}
```

该 Command 要求当前 Ticket 的 `If-Match`；Activity 写入会递增 Ticket 版本。不能通过 Customer API 创建内部备注、改变 Assignee 或任意设置 Ticket 状态。

### 12.5 Notification Read State

`PATCH /notifications/{id}` Request 只允许：

```json
{
  "read": true
}
```

重复标记已读返回当前资源。v0.2 不提供业务邮件通知 API。

## 13. Operations Conversation 与 Handoff API

### 13.1 端点矩阵

| Method | Path | 角色 | 成功 | 说明 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/v1/operations/conversations` | Support/Demo Admin | 200 | 会话列表 |
| `GET` | `/api/v1/operations/conversations/{id}` | Support/Demo Admin | 200 | 会话、Run、Handoff 摘要 |
| `GET` | `/api/v1/operations/conversations/{id}/messages` | Support/Demo Admin | 200 | 完整已脱敏会话消息 |
| `POST` | `/api/v1/operations/conversations/{id}/messages` | Assigned Support/Demo Admin | 201 | 人工公开回复 |
| `GET` | `/api/v1/operations/handoff-requests` | Support/Demo Admin | 200 | 接管队列 |
| `GET` | `/api/v1/operations/handoff-requests/{id}` | Support/Demo Admin | 200 | 接管详情与安全上下文 |
| `POST` | `/api/v1/operations/handoff-requests/{id}/claim` | Support/Demo Admin | 200 | 领取 |
| `POST` | `/api/v1/operations/handoff-requests/{id}/end` | Assigned Support/Demo Admin | 200 | 结束 |
| `POST` | `/api/v1/operations/handoff-requests/{id}/cancel` | Support/Demo Admin | 200 | 取消等待请求 |

### 13.2 Conversation Filters

允许 `status`、`response_owner`、`handoff_status`、`updated_after` 和 Cursor。Knowledge Operator 无权调用 Conversation/Message API；其知识待办只通过脱敏摘要读取。

### 13.3 人工消息

Request：

```json
{
  "text": "您好，我来继续协助您处理。"
}
```

该 Command 要求 Conversation 的 `If-Match`。只有 Conversation `response_owner = HUMAN` 且当前 Member 已领取 Handoff 时允许发送。消息成为 Conversation 的 `MEMBER` Message，不成为 Ticket Internal Note。

### 13.4 Claim

`POST /handoff-requests/{id}/claim` 无需 Body，但必须携带最新 `If-Match`。只有 `WAITING` 可领取；并发输家收到 `HANDOFF_ALREADY_CLAIMED` 和当前 Assignee 安全摘要。

### 13.5 End

Request：

```json
{
  "outcome_code": "CUSTOMER_ASSISTED",
  "conversation_action": "RESUME_AGENT",
  "knowledge_gap": false
}
```

`conversation_action` 为 `RESUME_AGENT`、`CLOSE_CONVERSATION`。`knowledge_gap = true` 时必须创建/复用 Knowledge Todo。结束 Handoff 不自动创建 Ticket。

### 13.6 Cancel

只允许未领取 `WAITING` 请求取消。已进入 `IN_PROGRESS` 时必须使用 End 并给出 Outcome，不得伪造为未发生。

## 14. Operations Ticket API

### 14.1 端点矩阵

| Method | Path | 成功 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/operations/tickets` | 200 | 工单队列/列表 |
| `POST` | `/api/v1/operations/tickets` | 201/200 | 人工幂等创建工单 |
| `GET` | `/api/v1/operations/tickets/{id}` | 200 | 工单详情 |
| `GET` | `/api/v1/operations/tickets/{id}/activities` | 200 | 完整 Timeline |
| `POST` | `/api/v1/operations/tickets/{id}/claim` | 200 | 领取工单 |
| `POST` | `/api/v1/operations/tickets/{id}/assignment` | 200 | 转派 |
| `POST` | `/api/v1/operations/tickets/{id}/status-transitions` | 200 | 状态变更 |
| `POST` | `/api/v1/operations/tickets/{id}/activities` | 201 | 公开回复或内部备注 |

Support Agent 和 Demo Admin 可调用。Ticket 无 DELETE Endpoint。

### 14.2 Ticket List

允许过滤：`status`、`priority`、`assignee_id`、`unassigned=true`、`customer_id`、`created_after`、`created_before`。`sort` 只允许 `queue`（优先级降序、创建时间升序）和 `updated_desc`。

### 14.3 人工创建 Ticket

必须携带 `Idempotency-Key`。Request：

```json
{
  "customer_id": "019b...",
  "conversation_id": "019b...",
  "order_id": null,
  "order_item_id": null,
  "problem_type": "PRODUCT_FAILURE",
  "problem_summary": "左耳无声音，已完成重置仍未恢复",
  "priority": "NORMAL",
  "creation_reason": "HUMAN_CREATED"
}
```

Tenant 由身份上下文注入。所有可选对象必须属于同一 Tenant/Customer。命中活动重复工单时返回 `200`、已有 `TicketSummary` 和 `meta.deduplicated = true`；不得创建第二张。

### 14.4 Claim 与 Assignment

- Claim 需要 `If-Match`，仅 `PENDING` 且无 Assignee 时成功。
- Assignment Request 为 `{ "member_id": "..." }`，目标必须为当前沙箱活动 Support Agent。
- 每次领取/转派在同一事务追加 Ticket Activity。

### 14.5 Status Transition

Request：

```json
{
  "to_status": "WAITING_CUSTOMER",
  "reason_code": "NEED_MORE_INFORMATION",
  "public_note": "请补充设备序列号。"
}
```

Backend 按固定状态机校验。`CLOSED`、`CANCELLED` 为终态；`RESOLVED -> IN_PROGRESS` 只用于 Customer 明确反馈未解决。状态变更和 Timeline 必须同事务提交。

### 14.6 Activity

Request：

```json
{
  "type": "INTERNAL_NOTE",
  "body": "已核对保修期，等待用户补充序列号。"
}
```

允许 `PUBLIC_REPLY`、`INTERNAL_NOTE`，并要求 Ticket 的 `If-Match`。Customer API 永远不返回 Internal Activity。Activity 只追加，不提供修改或删除端点。

## 15. Operations Knowledge API

### 15.1 端点矩阵

| Method | Path | 成功 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/operations/knowledge/documents` | 200 | 文档列表 |
| `POST` | `/api/v1/operations/knowledge/documents` | 202 | 上传并创建 Document/Version |
| `GET` | `/api/v1/operations/knowledge/documents/{id}` | 200 | 文档详情 |
| `PATCH` | `/api/v1/operations/knowledge/documents/{id}` | 200 | 更新标题/产品/分类 |
| `POST` | `/api/v1/operations/knowledge/documents/{id}/versions` | 202 | 上传新版本 |
| `GET` | `/api/v1/operations/knowledge/documents/{id}/versions` | 200 | 版本历史 |
| `GET` | `/api/v1/operations/knowledge/versions/{id}` | 200 | 版本处理状态 |
| `POST` | `/api/v1/operations/knowledge/versions/{id}/retry` | 202 | 重试失败处理 |
| `POST` | `/api/v1/operations/knowledge/versions/{id}/publish` | 200 | 人工审核并原子发布 |
| `POST` | `/api/v1/operations/knowledge/documents/{id}/disable` | 200 | 禁用文档 |
| `GET` | `/api/v1/operations/tasks/{task_id}` | 200 | 查询当前沙箱异步任务 |

Knowledge Operator 和 Demo Admin 可调用。Support Agent 不能修改知识。

### 15.2 Document 表示

包含 `id`、`document_key`、`title`、`product_key`、`category`、`is_disabled`、`latest_version`、`current_published_version`、`effect_summary`、`created_at`、`updated_at`、`row_version`。版本分别包含状态，Document 列表不能用最新版本失败错误地表示旧 Published Version 已不可检索。

`effect_summary` 只向 Knowledge Operator 和 Demo Admin 返回当前沙箱的 `citation_count`、`not_helpful_feedback_count` 和 `knowledge_resolved_count`；不包含跨 Workspace Analytics、Customer 身份或完整会话上下文。

### 15.3 首次上传

`POST /operations/knowledge/documents` 使用 Multipart：

| Part | 类型 | 必需 | 说明 |
| --- | --- | --- | --- |
| `file` | binary | 是 | `.md`、`.markdown`、文本型 `.pdf`，最大 10 MiB |
| `metadata` | JSON | 是 | `title`、可选 `product_key`、`category` |

Header 必须有 `Idempotency-Key`。成功 `202`：

```json
{
  "data": {
    "document": {
      "id": "019b...",
      "title": "蓝牙耳机常见问题"
    },
    "version": {
      "id": "019b...",
      "version_no": 1,
      "status": "PROCESSING"
    },
    "task": {
      "id": "019b...",
      "task_type": "KNOWLEDGE_PROCESS",
      "status": "PENDING"
    }
  },
  "meta": {
    "request_id": "019b..."
  }
}
```

文件扩展名、MIME、Magic Bytes、大小和文本型 PDF 校验必须全部通过。扫描 PDF 返回 `FILE_TYPE_NOT_ALLOWED`，v0.2 不尝试 OCR。

### 15.4 新版本与处理状态

新版本上传只接受 File，Document 元数据通过 PATCH 更新。处理状态按 `DRAFT -> PROCESSING -> PENDING_REVIEW | FAILED` 返回；Worker 失败时 Response 提供安全 `failure.code`，不返回文件路径、Provider 响应或堆栈。

### 15.5 Retry

只允许 `FAILED` Version，要求 `If-Match`。重试复用相同 Version 和新的 Durable Task 代次，不创建重复 Chunk/Index Build；超过任务重试上限后必须由人工再次触发。

### 15.6 Publish

`POST /operations/knowledge/versions/{id}/publish`

```json
{
  "review_note": "内容和引用位置已核对"
}
```

要求 Version 为 `PENDING_REVIEW`、目标 Index Build 为 `READY`、Index Version 为 `ACTIVE`。`If-Match` 防止并发发布。成功后旧 Published Version 仍可用于历史 Citation，但不参与新 Retrieval。

### 15.7 Disable

禁用要求 `If-Match` 和可选受控原因：

```json
{
  "reason_code": "CONTENT_OUTDATED"
}
```

禁用后文档立即退出 Retrieval。v0.2 不提供“恢复旧内容”的快捷操作；重新启用需要创建、审核并发布新 Version，避免过期知识无审核复活。

## 16. Knowledge Operations Todo API

### 16.1 端点矩阵

| Method | Path | 成功 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/operations/knowledge-todos` | 200 | 待办列表 |
| `GET` | `/api/v1/operations/knowledge-todos/{id}` | 200 | 脱敏待办详情 |
| `POST` | `/api/v1/operations/knowledge-todos/{id}/claim` | 200 | 领取待办 |
| `POST` | `/api/v1/operations/knowledge-todos/{id}/resolution` | 200 | 处理结论 |

Knowledge Operator 和 Demo Admin 可调用。

### 16.2 List 与 Detail

允许过滤 `status`、`trigger_reason`、`assigned_to_me`、`created_after`。Detail 只返回：

- 脱敏 `question_summary`。
- 触发原因、固定 Route 和时间。
- 可展示 Citation。
- 关联知识效果字段。
- 处理状态和安全 Resolution Note。

不得返回完整 Conversation、Customer 联系方式、原始 Tool 参数或 Trace CoT。Knowledge Operator 不能通过 Todo ID 反向调用 Operations Conversation API。

### 16.3 Claim 与 Resolution

Claim 要求 `If-Match`。Resolution Request：

```json
{
  "status": "PROCESSED",
  "resolution_code": "KNOWLEDGE_UPDATED",
  "resolution_note": "已发布 FAQ v2"
}
```

`status` 只允许 `PROCESSED`、`NO_ACTION_REQUIRED`。该接口不自动创建、修改或发布知识文档。

## 17. Operations Agent Trace API

### 17.1 端点矩阵

| Method | Path | 成功 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/operations/agent-traces` | 200 | Trace 列表 |
| `GET` | `/api/v1/operations/agent-traces/{id}` | 200 | Trace Summary |
| `GET` | `/api/v1/operations/agent-traces/{id}/events` | 200 | 按 Sequence 读取事件 |
| `GET` | `/api/v1/operations/agent-runs/{run_id}/tool-calls` | 200 | Tool 安全执行记录 |

Support Agent 和 Demo Admin 可访问当前沙箱 Trace；Knowledge Operator 不可访问。

### 17.2 Trace List

允许过滤：`status`、`route`、`reason_code`、`conversation_id`、`run_id`、`created_after`、`created_before`。默认按创建时间倒序。

### 17.3 Trace Summary

返回 `id`、`agent_run_id`、`status`、固定 Route、原因代码、事件数量、路由/检索/模型/Tool/总耗时、Token 汇总、开始/结束/过期时间和关联业务对象引用。

### 17.4 Trace Event

`GET /events?after_sequence=0&limit=200` 返回：

- `sequence_no`、`stage`、`event_type`、`event_status`。
- 固定 Route、Reason Code、Error Code。
- Message/Citation/Tool Call 引用。
- 分数、模型名、索引版本和阶段耗时等允许元数据。
- 脱敏业务摘要。

禁止返回：

- 完整 Prompt、System Instruction 或模型 CoT。
- 密码、Session Token、Provider Key 或 Connector Secret。
- 未脱敏 Customer 输入。
- 完整检索 Chunk 或完整 Tool 请求/响应。
- slog、OpenTelemetry Span 或未来 Audit Event。

Trace API 不提供修改、补写和删除端点；Retention 由系统任务处理。

### 17.5 Tool Calls

每条记录分离：

- `execution_status`：技术成功、失败、未知或取消。
- `business_result`：成功、未找到、资格不符或业务拒绝。
- Tool 名称、版本、尝试次数、耗时和错误码。
- 允许字段的输入/输出摘要。

API 不返回 Idempotency Key 原值、身份上下文或可重放的写请求。

## 18. 内部 Tool Contract

### 18.1 边界

内部 Tool 是 Go 模块间的 Transport-Independent Contract，不是公网 HTTP API。只有 Agent Runtime 的 Tool Executor 在通过 Registry、Policy、Schema、归属、确认和幂等检查后才能调用。

### 18.2 `ToolSpec`

v0.2 Tool Registry 在代码内静态注册，Spec 至少包含：

| 字段 | `GetOrder` | `CreateTicket` |
| --- | --- | --- |
| `name` | `GetOrder` | `CreateTicket` |
| `version` | `1.0` | `1.0` |
| `operation_type` | `READ` | `WRITE` |
| `risk_level` | `NORMAL` | `NORMAL` |
| `required_permission` | `ORDER_READ_SELF` | `TICKET_CREATE_SELF` |
| `requires_confirmation` | `false` | `true` |
| `supports_idempotency` | `false` | `true` |
| `timeout_ms` | 部署白名单值 | 部署白名单值 |
| `max_attempts` | 3 | 2 |
| `input_schema_version` | 1 | 1 |
| `output_schema_version` | 1 | 1 |

退款、取消订单、修改订单和删除数据不注册为 Tool。模型输出不能新增 Tool、降低风险等级、扩大权限或覆盖 Spec。

### 18.3 `ToolCallContext`

由系统注入：

```json
{
  "tenant_id": "019b...",
  "customer_id": "019b...",
  "conversation_id": "019b...",
  "run_id": "019b...",
  "trace_id": "019b...",
  "tool_call_id": "019b...",
  "confirmation_message_id": null,
  "idempotency_key": null
}
```

该 Context 不来自 LLM 输出。Context 中的敏感字段不得写入 Model Prompt、Customer SSE 或日志。

### 18.4 通用 `ToolResult`

```json
{
  "execution_status": "SUCCEEDED",
  "business_result": "SUCCESS",
  "safe_output": {},
  "error": null,
  "trace_metadata": {
    "tool_name": "GetOrder",
    "tool_version": "1.0",
    "attempt_no": 1,
    "duration_ms": 42
  }
}
```

`execution_status` 为 `SUCCEEDED`、`FAILED`、`UNKNOWN`、`CANCELLED`；`business_result` 为 `SUCCESS`、`NOT_FOUND`、`INELIGIBLE`、`REJECTED`、`NOT_APPLICABLE`、`UNKNOWN`。业务拒绝必须使用 `execution_status = SUCCEEDED` 加对应 Business Result，不伪装成技术失败。

### 18.5 `GetOrder`

类型：Read Tool；不需要 Customer 额外确认。

Input：

```json
{
  "selector": {
    "order_id": null
  },
  "include_items": true,
  "limit": 20
}
```

- `order_id = null` 表示列出当前 Customer 最近订单。
- Input 不包含 Tenant ID、Customer ID 或任意外部查询主体。
- v0.2 `limit` 最大 20，不允许任意搜索其他 Customer。

Safe Output：

```json
{
  "orders": [
    {
      "id": "019b...",
      "order_number": "SF20260001",
      "status": "COMPLETED",
      "purchased_at": "2026-07-01T10:00:00Z",
      "warranty": {
        "status": "VALID",
        "valid_until": "2027-07-01T10:00:00Z"
      },
      "items": [
        {
          "id": "019b...",
          "product_key": "novatech-buds-1",
          "sku": "NB-100-BLK",
          "name": "NovaTech 蓝牙耳机",
          "quantity": 1
        }
      ]
    }
  ]
}
```

订单和保修事实来自 Mock Business Service，不由 LLM生成。

### 18.6 `CreateTicket`

类型：Write Tool；必须完成订单/问题确认、业务资格校验、Customer 明确确认和 Idempotency。

Input：

```json
{
  "order_id": "019b...",
  "order_item_id": "019b...",
  "problem_type": "PRODUCT_FAILURE",
  "problem_summary": "左耳无声音，完成重置仍未恢复",
  "priority": "NORMAL",
  "creation_reason": "AFTER_SALES_REQUIRED"
}
```

`order_id`、`order_item_id` 可为空以支持非订单售后；存在时必须属于 Context Customer。Input 不允许指定 Assignee、Ticket Status、资格结果或 Source。

成功 Safe Output：

```json
{
  "ticket": {
    "id": "019b...",
    "ticket_number": "TK20260001",
    "status": "PENDING",
    "priority": "NORMAL",
    "created_at": "2026-07-24T18:30:00Z"
  },
  "deduplicated": false
}
```

同一 Idempotency Key 重试返回相同 Ticket。命中重复活动 Ticket 时返回已有 Ticket 和 `deduplicated = true`；只有获得有效 Ticket ID/Number 才能报告成功。

### 18.7 事务与重试规则

每次 Tool Call 是独立事务边界，不在 `GetOrder` 与 `CreateTicket` 之间建立分布式事务。状态变更 Tool 执行前必须已成功写入关键 Trace；无法记录时不执行 Tool。

| Tool | 自动重试 | 允许错误 |
| --- | --- | --- |
| `GetOrder` | 最多 2 次重试，共 3 次尝试 | `RATE_LIMITED`、`TEMPORARILY_UNAVAILABLE` |
| `CreateTicket` | 超时后同 Key 最多 1 次重试，共 2 次尝试 | 仅结果未知/暂时错误，必须先核对 Idempotency |

身份、权限、参数、数据不存在、资格不符和业务拒绝不自动重试。达到上限后进入安全回复或 Human Handoff，禁止无限循环。

## 19. Business Connector Contract

### 19.1 v0.2 边界

Mock Business Connector 是进程内 Adapter。它实现标准 Contract，但不开放企业配置、外部 Endpoint、任意 URL 或 Secret API。

### 19.2 GetOrder Contract

Connector 接收服务端 `BusinessContext` 和 `OrderSelector`：

```json
{
  "context": {
    "tenant_id": "system-injected",
    "customer_ref": "system-injected",
    "request_id": "019b..."
  },
  "selector": {
    "order_id": null,
    "limit": 20
  }
}
```

返回标准化订单，不泄漏 Mock 数据库字段。Connector 必须在数据源查询中使用 Customer 范围，不能先全量查询再内存过滤。

### 19.3 Connector 错误分类

| Connector Code | Tool Execution | Business Result | 自动重试 |
| --- | --- | --- | --- |
| `AUTHENTICATION_FAILED` | `FAILED` | `UNKNOWN` | 否 |
| `PERMISSION_DENIED` | `FAILED` | `UNKNOWN` | 否 |
| `INVALID_ARGUMENT` | `FAILED` | `UNKNOWN` | 否 |
| `NOT_FOUND` | `SUCCEEDED` | `NOT_FOUND` | 否 |
| `RATE_LIMITED` | `FAILED` | `UNKNOWN` | 是，遵循 Retry-After |
| `TEMPORARILY_UNAVAILABLE` | `FAILED` | `UNKNOWN` | 是 |
| `BUSINESS_REJECTED` | `SUCCEEDED` | `REJECTED` | 否 |

v0.3 真实 Connector 仍须映射到该分类；不得把 Provider 原始错误直接传给 LLM 或 Customer。

## 20. Durable Task 查询契约

`GET /api/v1/operations/tasks/{task_id}` 只允许查看当前 Demo Session 产生的任务。状态为 `PENDING`、`ENQUEUED`、`RUNNING`、`SUCCEEDED`、`FAILED`、`CANCELLED`。

- `FAILED` 返回安全 `last_error.code` 和 `retryable`。
- API 不接受直接修改 Task Status、Attempt Count、Lease 或 Asynq ID。
- 重试必须调用所属业务资源的 Retry Command，不提供通用 `/tasks/{id}/retry`。
- Redis 丢失时 Task 仍可从 PostgreSQL 恢复；API 不把 Redis 队列状态当作终态。

## 21. Health、Readiness 与降级

### 21.1 Health Endpoints

| Method | Path | 身份 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/health/live` | 无 | 进程存活；不探测 Model Provider |
| `GET` | `/health/ready` | 无 | PostgreSQL、Redis 和必要 Object Volume 可用 |

Public Response 只返回 `status`、`version` 和组件安全状态，不返回 DSN、主机路径、凭据、数据库版本详情或堆栈。

### 21.2 Readiness

```json
{
  "status": "not_ready",
  "components": {
    "postgresql": "unavailable",
    "redis": "available",
    "object_storage": "available"
  }
}
```

Readiness 不可用返回 503。外部 Model Provider 不作为容器 Liveness/Readiness 条件，通过 Agent 业务错误与 Trace 暴露。

### 21.3 降级行为

| 故障 | API 行为 |
| --- | --- |
| PostgreSQL 不可用 | 503，拒绝新业务写入，不降级到 Redis |
| Redis 不可用 | 普通只读可用；异步任务和新 Agent Run 按安全策略 503/排队 |
| Object Volume 不可用 | 禁止上传/重试解析；已发布数据库知识按可用状态处理 |
| Model Provider 不可用 | Run 失败或切换允许的 Mock 模式，不伪造真实模型结果 |
| Knowledge 不可用 | 不生成无依据知识回答，进入错误/接管 |
| CreateTicket 结果未知 | 返回/记录 `TOOL_RESULT_UNKNOWN`，同 Key 核对，不假定成功 |
| Trace 关键写入失败 | 状态变更 Tool 前失败关闭，不执行不可追踪动作 |

## 22. 限流、配额与资源限制

### 22.1 Response Headers

受限端点返回 `RateLimit-Limit`、`RateLimit-Remaining`、`RateLimit-Reset`；429 同时返回 `Retry-After`。Header 只是提示，服务端 Redis/数据库配额为准。

### 22.2 Lite 默认限制

| 范围 | 默认策略 |
| --- | --- |
| 普通 Customer API | 每 Session 60 请求/分钟 |
| Agent Run Command | 每 Session 20 次/小时；每 Conversation 1 个活动 Run |
| Agent 执行并发 | Lite 默认 1、最大 2 |
| 同时在线 Session | 目标 20 |
| Knowledge 上传 | 每文件 10 MiB，每 Session 文件数受配额限制 |
| Customer Message | 8,000 Unicode 字符 |
| Feedback Comment | 500 字符 |
| Ticket/Knowledge Note | 依资源 Schema 限制 |
| SSE | 每 Run 1 个主连接；断线使用状态恢复 |

Public Demo 可进一步收紧，不得通过 Query 参数放宽。达到 Token 配额时返回 `QUOTA_EXCEEDED`，不在后台继续产生费用。

### 22.3 响应目标

- 普通 REST API（不含 Agent、RAG、LLM 和 Tool 链路）P95 不超过 500ms。
- Mock `GetOrder` P95 不超过 500ms。
- 10,000 Chunk 以内的 Knowledge Retrieval P95 不超过 1 秒。
- Agent Command 在 1 秒内发送 `run.accepted` 或首个 `business.status`，不要求完成回答。
- Provider 首字、总耗时和 Tool/检索阶段耗时分别记录到 Trace，不混入普通 API 指标。

## 23. 安全与隐私要求

### 23.1 输入校验

- JSON Command 拒绝未知字段、重复键、非法 Unicode 和超长嵌套。
- 所有 ID 先校验格式，再在 Tenant/主体范围查询；错误不泄漏其他对象存在性。
- 文件校验扩展名、MIME、Magic Bytes、大小、页数和文本可提取性；文件名不作为存储路径。
- Markdown/PDF 内容作为不可信数据，不允许改变系统规则、Route、Tool Permission 或确认要求。
- 导航路径和 Notification Target 使用服务端白名单，避免开放重定向。

### 23.2 输出与日志

- API Error 不返回 SQL、堆栈、环境变量、内部路径或 Provider 原始响应。
- Trace API 使用允许字段清单；Customer API 不暴露 Trace。
- 请求日志对 Cookie、CSRF、Authorization、Idempotency、邮箱、手机号、地址和订单号应用移除或掩码。
- SSE `message.delta` 内容经过与最终 Message 相同的输出安全策略；检测失败时停止继续流式输出。

### 23.3 Prompt Injection 与 Tool 防护

- Customer/Knowledge 内容只能作为数据，不得直接形成 Tool Context 身份字段。
- LLM Tool Proposal 必须经过固定 Registry、JSON Schema、权限、风险、对象归属和确认校验。
- 高风险请求只能生成 Human Handoff，不存在可调用的高风险 Tool Contract。
- 拒绝越权和绕过确认时写入结构化 Trace Reason Code，不把内部策略细节返回攻击者。

## 24. v0.3 延后接口

以下接口不进入 v0.2 实现和验收：

- Workspace CRUD、配置发布和 Workspace 级 Tool Permission API。
- Platform User、企业 Member 登录、邀请、密码、MFA、SSO、Session 管理和临时授权 API。
- 完整 RBAC Role/Permission API。
- Audit 查询和导出 API。
- 真实 Business Connector 配置、Secret、连通性测试和启停 API。
- 模型 Provider 配置、任意 Endpoint 和密钥管理 API。
- Analytics 聚合、成本看板和报表导出 API。
- Webhook、Workflow、Plugin、自动学习和多 Agent API。
- Customer 附件、语音、外部客服渠道和原生 App API。

v0.3 可以在保持 `/api/v1/customer`、`/api/v1/operations` 资源语义的前提下替换身份来源，不允许放松 Tenant、权限、确认和 Trace 安全边界。

## 25. API 验收与后续输入

### 25.1 Contract 验收

API 设计进入开发前至少验证：

1. Customer 无法通过 ID、Cursor、过滤参数或 Body 访问其他 Customer 数据。
2. Operations 角色矩阵由 Backend 强制执行，Knowledge Operator 无法读取完整会话和 Trace。
3. 同一个 Conversation 无法并发创建两个活动 Run。
4. 相同 `client_request_id` 和 Idempotency Key 不产生重复 Message、Run 或 Ticket。
5. ETag 冲突不静默覆盖 Handoff、Ticket、Knowledge 和 Todo 状态。
6. SSE 在 1 秒内返回 Accepted/业务状态，事件不包含 CoT，断线后可从 REST 恢复终态。
7. Tool Result 明确区分技术失败和业务拒绝。
8. Knowledge 上传、处理、失败、重试、待审核和原子发布状态可查询。
9. Trace 只返回允许字段，Customer、Knowledge Operator 和未授权 Member 均无法访问。
10. Provider、Knowledge、Tool、Redis 和 Trace 失败均有固定错误码和安全降级行为。

具体测试数量、Mock、自动化框架和 OpenAPI Contract Test 在测试设计阶段确定。

### 25.2 对 Task 拆分的输入

后续开发任务至少按以下独立交付单元拆分：

- API 基础协议、Error Middleware、Request ID、CSRF、Cursor、ETag 和 Idempotency。
- Demo Session 与沙箱角色。
- Customer Conversation、Message、Agent Run Command 和状态恢复。
- typed SSE Encoder、业务事件和断线处理。
- Feedback、Citation、Status Center、Notification。
- Operations Handoff 与 Ticket。
- Knowledge Upload、Version、Durable Task、Review 与 Publish。
- Knowledge Todo 与安全上下文。
- Agent Trace 和 Tool Call 安全查询。
- Tool/Connector Contract、Mock GetOrder 与 CreateTicket。
- Health、限流、配额、安全与 Contract Test。

## 26. API 设计结论

SupportFlow v0.2 以 `/api/v1/demo`、`/api/v1/customer` 和 `/api/v1/operations` 分离身份与业务边界，以 REST 查询持久化事实、以 Fetch POST + typed SSE 提供 Agent 流式体验，并用固定错误码、Keyset Pagination、ETag、Idempotency Key 和 `client_request_id` 保证一致性。所有 Tool 和 Business Connector 调用保留在 Backend 内部受控边界，Customer 只能看到业务状态、安全结果和引用，不能接触模型 CoT、权限上下文或可重放业务操作。
