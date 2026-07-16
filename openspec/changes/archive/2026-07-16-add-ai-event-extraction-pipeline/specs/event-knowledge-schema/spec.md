## ADDED Requirements

### Requirement: Event 原子事实载荷
系统 SHALL 在既有 `events` 事实表上增加 `fact_payload JSONB NOT NULL DEFAULT '{}'`，用于保存可独立验证的原子 Event 事实；该载荷不得表达预测、评分或直接投资建议。

#### Scenario: 保存 JSON object 事实载荷
- **WHEN** 确定性执行器写入通过 evidence 校验的 Event package
- **THEN** `fact_payload` 必须接受 JSON object（包括空 object），并与既有 title、summary、时间、状态和 `dedupe_key` 字段共同保存

#### Scenario: 拒绝非 object 或越界载荷
- **WHEN** candidate payload 不是 JSON object，或包含买入卖出、涨跌预测、利好利空、传导强度、事件评分或直接投资建议
- **THEN** domain/repository contract 必须拒绝该 payload，不得写入事件事实

### Requirement: Event 证据镜像与最小归因
系统 SHALL 通过既有 `event_sources` 将 Event 与 `raw_documents` 关联，并支持已批准的最小证据归因字段；`raw_documents` 仍是证据镜像，不能被 Event 事实替代。

#### Scenario: 保存可追溯 evidence
- **WHEN** Event package 由一个或多个 raw document 支持
- **THEN** 系统必须保留现有 `source_level`、`evidence_excerpt`、`evidence_hash`，并可在批准后写入 `evidence_relation VARCHAR(32) NULL` 与 `supports_fields TEXT[] NOT NULL DEFAULT '{}'`

#### Scenario: 解释 evidence 字段
- **WHEN** 新执行器写入 `evidence_relation`
- **THEN** `supports`、`contradicts`、`context` 必须使用受控值，legacy NULL 必须解释为 unknown；新 `supports`/`contradicts` 写入必须有非空 `supports_fields`

#### Scenario: 评估 evidence 幂等
- **WHEN** 同一 Event/raw document/evidence hash 被重复提交
- **THEN** repository 必须先检查重复并跳过无意义写入；只有在 preflight 证明既有数据与并发语义安全时，才可增加 `(event_id, raw_document_id, evidence_hash)` 唯一约束，否则必须 defer

### Requirement: 受控 Tag 分类和归因
系统 SHALL 以 Tidewise DB 的 `event_tag_defs` 作为 Tag 主数据唯一权威，保留 `event_tag_maps` 的既有唯一约束，并支持已批准的 Tag 置信度和分配理由；YAML 只能作为策略输入。

#### Scenario: 映射受控 Tag
- **WHEN** Event candidate 携带 Tag candidate
- **THEN** 系统必须按 DB 中已注册的 `event_tag_defs` 映射；YAML 不得创建或替代 Tag 主数据

#### Scenario: 保存 Tag 归因
- **WHEN** AI 或规则分配通过确定性校验
- **THEN** 可写入 `confidence NUMERIC(5,4) NULL CHECK (confidence >= 0 AND confidence <= 1)` 与 `assignment_reason TEXT NOT NULL DEFAULT ''`；新 AI/规则分配的 `assignment_reason` 必须非空，既有记录保持可读兼容

### Requirement: 事件字段扩展审慎门槛
系统 SHALL 对除 `fact_payload` 外的 `events` 新字段逐项评估，不得将未 Review 字段预批准。

#### Scenario: 延后非必要字段
- **WHEN** Review 讨论 `event_type_code`、`action_code`、`lifecycle_status`、`reference_period_start/end`、`last_seen_at` 或 `event_time_precision`
- **THEN** `event_type_code`/`action_code` 优先 defer 至受控 Tag 或 payload；`lifecycle_status` 因与 `event_status` 重叠 defer；`reference_period_start/end` 先放 payload；`last_seen_at` 可由 event_sources 聚合而 defer；`event_time_precision` 仅作为待用户 Review 的独立事实语义候选

### Requirement: 非破坏性增量 migration
系统 SHALL 通过编号 `000019` 的增量 migration 增加经 Review 批准的字段/约束，并保留既有数据；migration 及其 contract tests 必须在未来 Apply 的 R2 gate 下执行。

#### Scenario: 保留 fresh before 状态
- **WHEN** local PostgreSQL preflight 在 migration 前读取当前 schema/version、受影响表行数、约束/索引和 backup identity
- **THEN** migration 后必须以各表 fresh before counts 作为相等断言；2026-07-16 的 407/0 audit snapshot 只能作为历史审计信息，不得作为固定 live assertion

#### Scenario: 失败安全
- **WHEN** schema/hash/count 漂移、backup 失败、migration 失败、超时或 after assertion 失败
- **THEN** 系统必须立即停止剩余 scope，并采用 backup recovery 或 fail-closed/forward-fix 兼容策略，不得清空、回填或删除业务数据
