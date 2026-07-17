## ADDED Requirements

### Requirement: Agent 受控 HTTP 导入 API
Data Service SHALL 提供版本化、服务身份认证、幂等且可审计的 HTTP REST + JSON import API，接收既有 reviewed-outbox contract；Agent Server MUST NOT 直接连接 Tidewise PostgreSQL 或调用 Miniapp/Admin BFF。

#### Scenario: 认证 Agent 导入 package
- **WHEN** 已授权 Agent identity 提交 contract 完整且 review 状态有效的 package
- **THEN** Data Service 必须复用 canonical payload hash、幂等 receipt 和单 transaction 持久化语义并返回结构化结果

#### Scenario: 拒绝未认证调用方
- **WHEN** 缺少、无效或 scope 不匹配的 service identity 调用 import API
- **THEN** Data Service 必须在解析业务写入前拒绝请求且不产生 PostgreSQL 业务记录

#### Scenario: 网络超时后重试
- **WHEN** Agent 未收到首次导入响应并使用相同 idempotency key 与 payload hash 重试
- **THEN** Data Service 必须返回原 receipt/result；相同 key 不同 hash 必须返回 conflict 且不修改原结果

### Requirement: Event import CLI 兼容迁移
现有 `event-import` CLI SHALL 在迁移期保持 input、dry-run、machine JSON、exit code 与 reviewed-outbox 字段兼容；生产 Agent path 稳定后，非 dry-run CLI SHALL 作为 Data Service API 的受控 client 或明确限定的 Data Service maintenance adapter，不得长期保留绕过 service ownership 的独立数据库入口。

#### Scenario: 运行 dry-run
- **WHEN** 操作者对 reviewed-outbox 执行 CLI dry-run
- **THEN** CLI 必须继续在不连接 PostgreSQL 的情况下验证并输出兼容 plan/result 结构

#### Scenario: 迁移期执行 import
- **WHEN** compatibility window 内执行非 dry-run CLI import
- **THEN** CLI 必须遵循与 HTTP API 相同的 validation、idempotency、transaction 和 error contract，并记录所用 adapter mode

#### Scenario: 结束 compatibility window
- **WHEN** Agent HTTP import 已通过 local 与目标环境验收且旧入口无消费者
- **THEN** 后续 change 才能删除直接 maintenance adapter，并必须保留明确迁移说明和 contract tests

### Requirement: reviewed event 与 raw document 导入分离
Data Service SHALL将reviewed-outbox event import与通用raw-document batch import定义为两个独立受控contract和两个独立receipt schemas；raw import MUST NOT绕过event review状态、tag/source validation、canonical payload hash或event receipt transaction，reviewed event import也MUST NOT以`raw_document_import_receipts`替代`event_import_receipts`。

#### Scenario: Agent只提交原始材料
- **WHEN** `agent-run`尚未形成reviewed event package而只提交来源材料
- **THEN** 它必须调用raw-document import，Data Service只能在一个transaction写入/复用raw documents及独立immutable raw receipt，不得自动创建event、source/tag mapping、review状态或research结论

#### Scenario: Agent提交reviewed event
- **WHEN** Agent Server提交contract完整且review状态有效的event package
- **THEN** 它必须调用reviewed event import，并继续在单transaction中处理raw document、event、sources、tags与`event_import_receipts` receipt，不得写raw receipt placeholder

#### Scenario: 两类 receipt key 空间
- **WHEN** raw import和reviewed-event import碰巧使用相同文本idempotency key
- **THEN** 两者必须在独立table/repository/API contract中各自判断，不得共享generic uniqueness、result payload或review语义

#### Scenario: 删除旧采集runtime
- **WHEN** Tidewise scheduler/source-ingest/ingest-smoke/runtime被删除
- **THEN** event import API、CLI dry-run/compatibility contract、repository、fixture、幂等和transaction tests必须继续有效
