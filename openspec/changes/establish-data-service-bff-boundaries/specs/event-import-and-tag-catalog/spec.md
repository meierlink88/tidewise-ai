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
