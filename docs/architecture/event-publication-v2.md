# Event Publication V2

## 状态与范围

- 状态：Accepted
- 所属上下文：Data
- 决策：ADR-0005
- 唯一写入口：`POST /api/data/v1/reviewed-event-imports`

本合同让 AgentRun 将已完成采集、提取和审核的正式 Event 连同轻量证据原子发布
到 Tidewise Data。它不实现采集、原文存储、Event 提取、异步任务或失败恢复。

## 服务鉴权

```http
Authorization: Bearer <service-token>
Content-Type: application/json
```

调用主体必须具备 `data.reviewed-events.import` scope。Data 从凭据解析稳定
`caller_subject`；请求体不能覆盖调用身份，数据库不保存 token。

## 请求合同

```json
{
  "package_id": "agentrun:event-publication:20260723:001",
  "provenance": {
    "extractor_execution_id": "extractor-exec-001",
    "extractor_agent_version": "event-extractor-v2.0.0",
    "collector_executions": [
      {
        "artifact_id": "artifact-001",
        "collector_execution_id": "collector-exec-101"
      }
    ]
  },
  "raw_documents": [
    {
      "artifact_id": "artifact-001",
      "content_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "source_ref": "source:reuters:world",
      "source_name": "Reuters",
      "source_type": "news",
      "source_url": "https://example.test/article/1",
      "title": "Example source title",
      "published_at": "2026-07-23T01:00:00Z",
      "collected_at": "2026-07-23T01:05:00Z",
      "language": "en",
      "mime_type": "text/markdown"
    }
  ],
  "events": [
    {
      "dedupe_key": "event:example:20260723:001",
      "title": "Example event",
      "factual_summary": "A verifiable state change occurred.",
      "occurred_at": "2026-07-23T00:30:00Z",
      "fact_payload": {
        "metric": "example"
      },
      "evidence": [
        {
          "artifact_id": "artifact-001",
          "evidence_relation": "supports",
          "evidence_excerpt": "A short excerpt supporting the event.",
          "supports_fields": [
            "title",
            "factual_summary"
          ],
          "source_level": "primary",
          "is_primary": true
        }
      ],
      "tags": [
        {
          "tag_id": "11111111-1111-5111-8111-111111111111",
          "tag_kind": "news_category",
          "tag_code": "technology_industry",
          "confidence": 0.94,
          "assignment_reason": "The event concerns technology industry supply.",
          "assign_source": "ai"
        }
      ],
      "review": {
        "review_id": "review-001",
        "evidence_grade": "A",
        "reasons": [
          "The source and event facts are internally consistent."
        ]
      }
    }
  ]
}
```

严格 JSON 解码拒绝未知字段。

## 批次约束

- `package_id` 必填但不唯一，只用于 AgentRun 与 Receipt 审计关联。
- `events` 数量为 `1..10`，同批次 Dedupe Key 不得重复。
- `raw_documents` 按 `artifact_id` 去重；同一 Artifact 必须恰好有一条
  `collector_execution_id`。
- 每个 Event 至少引用一个顶层 Artifact；每个顶层 Artifact 至少被一个 Event
  引用。
- 任一结构、引用或数据库写入失败，整个批次回滚且不生成 Receipt。

## 轻量 Raw Document

必填：

- `artifact_id`
- `content_sha256`，标准小写 64 位 SHA-256
- `source_ref`
- `source_name`
- `source_type`
- `title`
- `collected_at`

可选：

- `source_url`，非空时必须是绝对 HTTP(S) URL
- `published_at`
- `language`
- `mime_type`

V2 不接收 `content_text`、Artifact URI、采集通道、采集状态、内容层级或独立
来源外部 ID。`source_type` 只校验非空和长度；其业务枚举由 AgentRun 管理。
外部合同长度与现有 PostgreSQL 列一致：`artifact_id/source_ref` 最多 256 字符，
`source_name` 300 字符，`source_type` 64 字符，`source_url` 2048 字符，
`title` 1000 字符，`language` 16 字符，`mime_type` 128 字符。

相同 `artifact_id` 的全部轻量字段必须一致才能复用；任一字段不同返回冲突。
`content_sha256` 只检查 Artifact 身份漂移，不表示 Data 已验证来源真实性。

## Event 核心事实

每个 Event 的状态由 Data 固定为：

```text
event_status = confirmed
fact_status  = verified
```

不可变核心字段：

- `title`
- `factual_summary`
- 可空 `occurred_at`
- 按 JSONB 语义比较的 `fact_payload`

相同 `dedupe_key` 仅在核心字段完全一致时复用。事实修订必须使用新 Dedupe Key。
`first_seen_at` 取全部关联 Evidence 的最早 `collected_at`；`knowable_at` 取最早
`published_at`，缺失时回退 `collected_at`。新增更早证据时两者只向更早时间
收敛。

## Evidence Link

每个 Event 与同一 Artifact 只能存在一条 Link，数据库身份为
`(event_id, raw_document_id)`。

- `evidence_relation`：`supports | contradicts | context`
- `evidence_excerpt`：必填短摘录
- `supports_fields`：仅允许 `title | factual_summary | occurred_at | fact_payload`
- `source_level`：`primary | secondary`
- `is_primary`：每个 Event 必须且只能有一条 `true`

`supports`、`contradicts` 必须提供非空 `supports_fields`；`context` 可以为空。
Data 根据摘录计算 SHA-256 `evidence_hash`。再次提交已有 Link 时，以上所有语义
必须一致。`events.primary_source_id` 指向 `is_primary=true` 的 Link，后续不得
静默替换。

## Tag 与 Review

每个 Event：

- 必须包含 `1..2` 个 active `news_category`
- 可以包含 `0..3` 个 active `index_category`
- Tag ID、kind、code 必须与 Data 主数据一致
- `confidence` 必须位于 `[0,1]`
- `assignment_reason` 非空
- `assign_source` 为 `ai | rule`

V2 不接收 Tag review status；Data 统一写入 `approved`。同 Event、同 Tag 的既有
映射必须语义一致，否则冲突；新 Tag 可以追加。

Review 按 Event 提交，`review_id`、`evidence_grade` 和非空 `reasons` 均为必填。
V2 不接收 review decision、Event/Fact status 或 Prompt/Model/Profile 版本。

## 自然身份收敛

V2 没有 idempotency key、payload hash、replayed 响应或状态查询。

- Event：Dedupe Key
- Raw Document：AgentRun Artifact ID
- Evidence Link：Event ID + Raw Document ID
- Event Tag：Event ID + Tag ID

相同自然身份且内容一致时复用；内容冲突时整个批次返回 `409`。后续批次可以为
既有 Event 追加新 Evidence Link 或 Tag，但不能修改或删除已有关系。

## 成功响应

每次成功都创建新的 Receipt，因此返回 `201 Created`：

```json
{
  "request_id": "data-1753240000000000000",
  "result": {
    "receipt_id": "22222222-2222-5222-8222-222222222222",
    "package_id": "agentrun:event-publication:20260723:001",
    "imported_at": "2026-07-23T02:00:00Z",
    "events": [
      {
        "dedupe_key": "event:example:20260723:001",
        "event_id": "33333333-3333-5333-8333-333333333333",
        "disposition": "created"
      }
    ],
    "raw_documents": [
      {
        "artifact_id": "artifact-001",
        "raw_document_id": "44444444-4444-5444-8444-444444444444",
        "disposition": "reused"
      }
    ],
    "counts": {
      "events_created": 1,
      "events_reused": 0,
      "raw_documents_created": 0,
      "raw_documents_reused": 1,
      "event_sources_created": 1,
      "event_sources_reused": 0,
      "event_tags_created": 1,
      "event_tags_reused": 0
    }
  }
}
```

## 失败响应

- `401`：未认证
- `403`：缺少 `data.reviewed-events.import`
- `409`：自然身份内容冲突
- `422`：合同、枚举、Tag 或引用错误
- `500`：数据库或未知服务故障

可确定的校验错误一次性返回，按字段路径稳定排序：

```json
{
  "error": {
    "code": "EVENT_PUBLICATION_INVALID",
    "details": [
      {
        "path": "events[0].evidence[0].artifact_id",
        "code": "UNKNOWN_ARTIFACT",
        "message": "artifact_id is not declared in raw_documents"
      }
    ]
  }
}
```

任何失败均不产生部分业务写入或 Receipt。

## 数据库演进

forward migration：

1. 保留 `raw_documents` 表名和历史正文；为 V2 增加 Artifact/Source 外部身份，
   放宽仅适用于历史采集的字段，并增加 V2 条件约束与 Artifact 唯一索引。
2. 为 V2 Evidence Link 增加合同版本和 `(event_id, raw_document_id)` 条件唯一
   约束，避免要求历史 Link 重新整形。
3. 新建不可变 `event_publication_receipts`，保存调用主体、package、结果 ID、
   created/reused 计数、Review 与执行血缘。
4. 删除 `raw_documents.source_id` 及 Source FK/索引。
5. 删除 `ingestion_run_sources`、`ingestion_runs`、
   `ingestion_scheduler_configs`、`source_catalogs`。
6. 删除 `raw_document_import_receipts`、`event_import_receipts` 及专属触发器/
   函数。

迁移不得删除或改写历史 Event、Raw Document、Event Source、Event Tag 或 Research
数据。未经单独授权，不应用到 `tidewise_local`、UAT 或生产数据库。

## 退出接口

- `POST /internal/data/v1/raw-document-imports`：路由已删除，返回 `404`
- `GET /internal/data/v1/raw-document-imports/{idempotency_key}`：路由已删除，返回 `404`
- `POST /internal/data/v1/reviewed-event-imports`：路由已删除，返回 `404`
- Data 与 Admin Portal 的 Source Catalog/Source Metadata 查询接口及其实现删除

## 必测场景

1. 单 Artifact 支持多个 Event，只创建一条 Raw Document。
2. 单 Event 引用多个 Artifact。
3. 未被 Event 引用的 Artifact 整批拒绝且不落库。
4. 相同自然身份与内容复用，不重复创建事实或关联，但创建新 Receipt。
5. 相同 Artifact ID 或 Dedupe Key 内容冲突时整批回滚。
6. 任一 Evidence、Tag、Review 或 Receipt 写入失败时零部分写入。
7. 无正文、无 Artifact URI 的 V2 Evidence Record 可成功落库。
8. 历史 `content_text` 在 migration 后保持可读。
9. 主证据 FK、Event Source FK 与 Raw Document FK 保持完整。
10. Source/采集控制面与旧 Receipt 表已退出，Research Theme/Anchor 合同不变。

## 范围外

- AgentRun 代码、Source 迁移数据和完整 Artifact 存储
- Event 提取、Prompt、模型、Profile 与失败恢复
- UAT/生产部署
- Theme、Anchor、Miniapp、Neo4j 或前端改动
