# Data Evidence Publication V1

## 状态与权威

- 状态：Accepted
- 所属上下文：Data
- 上位决策：ADR-0011
- Raw Evidence 写入口：`POST /api/data/v1/raw-evidence-publications`
- Evidence 写入口：`POST /api/data/v1/evidence-publications`
- 既有 Event Publication：不变，继续使用 `raw_documents` 与 `event_sources`

本合同让采集/清洗执行方分两阶段把完整原始材料和原子 Evidence 发布为 Data 正式事实。
Data 不调用 AgentOS/AgentRun、不运行 LLM、不判断语义相似性，也不读取执行方数据库或
Artifact；执行方不得直写 Data PostgreSQL。

## 领域模型

### Raw Evidence

`raw_evidences` 从本地参考 `tidewise_ai_server.raw_documents` 复制字段类型、空值、默认值、
注释和行级约束，只做领域改名并增加 Keywords：

| 字段 | PostgreSQL | 约束与含义 |
| --- | --- | --- |
| `raw_evidence_id` | `VARCHAR(32)` | PK、非空非空白；原 `raw_document_id` |
| `source_id` | `VARCHAR(32)` | 必填信源 ID |
| `source_name` | `VARCHAR(100)` | 必填信源名称 |
| `source_level` | `VARCHAR(20)` | `L1_OFFICIAL \| L2_WIRE \| L3_MEDIA \| L4_SOCIAL` |
| `source_url` | `TEXT` | 必填非空白原始链接 |
| `is_original` | `BOOLEAN` | 当前 source 是否原创 |
| `quoted_source_id` | `VARCHAR(32)` | 可空转载上游信源 ID |
| `quoted_source_name` | `VARCHAR(100)` | 转载时必填上游信源名称 |
| `title` | `VARCHAR(500)` | 可空；非空时不可仅空白 |
| `raw_text` | `TEXT` | 必填非空白完整正文 |
| `published_at` | `TIMESTAMPTZ` | 可空文章发布时间 |
| `collected_at` | `TIMESTAMPTZ` | 必填，数据库默认 `now()` |
| `content_hash` | `VARCHAR(64)` | `raw_text` 的 generated stored SHA-256 |
| `keywords` | `TEXT[]` | 非空，默认空数组，原样有序保存 |
| `created_at` | `TIMESTAMPTZ` | Data 数据库生成的新行创建时间；历史行可空 |

`is_original=true` 时两个 quoted 字段必须为空；`false` 时 `quoted_source_name` 必填。
Keywords 是阅读辅助元数据，发布方负责数量、长度、内容和去重规则。Data 只要求 JSON
字符串数组和 PostgreSQL `TEXT[]` 可存储性，不 trim、规范化、排序或增加 GIN 索引。

### Atomic Evidence

`evidences` 复制本地参考 Evidence 的 ID、拆分顺序、`SINGLE/DOUBLE` 与两层 5W1H 字段；
原 Group 三字段被正式去重身份替代，并增加 Data 派生的 `is_split`：

| 字段 | PostgreSQL | 约束与含义 |
| --- | --- | --- |
| `evidence_id` | `VARCHAR(32)` | PK、调用方稳定身份 |
| `raw_evidence_id` | `VARCHAR(32)` | FK 到 `raw_evidences`；原 `raw_document_id` |
| `split_order` | `INTEGER` | 默认 0；非负；同 Raw 唯一；从 0 连续 |
| `is_split` | `BOOLEAN` | Data 根据整包基数派生 |
| `layer_type` | `VARCHAR(10)` | `SINGLE \| DOUBLE` |
| `source_who` | `TEXT` | 第一层 5W1H：谁 |
| `source_what` | `TEXT` | 第一层 5W1H：发生了什么；必填非空白 |
| `source_when` | `TIMESTAMPTZ` | 第一层结构化事实发生时间 |
| `source_when_raw` | `TEXT` | 第一层原始时间表达 |
| `source_where` | `TEXT` | 第一层何地 |
| `source_why` | `TEXT` | 第一层为何 |
| `source_how` | `TEXT` | 第一层如何、程度或数量 |
| `source_who_core` | `TEXT` | 第二层核心 5W1H：谁 |
| `source_what_core` | `TEXT` | 第二层核心发生了什么 |
| `source_when_core` | `TIMESTAMPTZ` | 第二层核心结构化发生时间 |
| `source_when_raw_core` | `TEXT` | 第二层核心原始时间表达 |
| `source_where_core` | `TEXT` | 第二层核心何地 |
| `source_why_core` | `TEXT` | 第二层核心为何 |
| `source_how_core` | `TEXT` | 第二层核心如何、程度或数量 |
| `expression_fingerprint` | `VARCHAR(200)` | 必填非空白、确定性规范化后的可读表达 |
| `expression_key` | `VARCHAR(64)` | 必填非空白、发布方稳定机器去重键 |
| `fingerprint_version` | `VARCHAR(64)` | 必填非空白、规范化/哈希算法版本 |
| `created_at` | `TIMESTAMPTZ` | Data 数据库生成的新行创建时间；历史行可空 |

`SINGLE` 要求全部 core 字段为空；`DOUBLE` 要求 `source_what_core` 非空白。文章的
`published_at` 不能替代 Evidence 的 `source_when`。

`expression_key` 只有普通非唯一 B-tree 索引。多个来源对同一逻辑事实的 Evidence 可以
共享 key，且每一行都必须保留。所谓 Group 只是按该 key 查询得到的派生集合；V1 没有
Group 表、Group API、状态、合并、redirect、embedding 或 Data 语义召回。

两张领域表的 `created_at` 都是 Data 内部持久化元数据。发布方不得提交该字段，发布 API
也不返回它。新行由 PostgreSQL `transaction_timestamp()` 生成；自然身份复用时保留首次
创建时间。历史行不回填，保持 `NULL`，不得使用 migration 执行时间冒充历史创建时间。
领域事实发布后不可变，因此不增加 `updated_at`。

## 两阶段状态与不可变性

1. Raw Evidence Publication 成功后，Raw Evidence 可以暂时没有 Evidence；这只表示等待
   清洗，不是 `NO_EVIDENCE` 终态，也不需要冗余状态列。
2. Evidence Publication 必须一次提交该 Raw Evidence 的完整 `1..N` 集合。
3. 恰好一条时，Data 写 `is_split=false` 且要求 `split_order=0`。
4. 多条时，Data 为全部行写 `is_split=true`，并要求 `split_order` 恰为 `0..N-1`。
5. Evidence 首次成功发布后不可覆盖、追加、删除或重排。完整相同的后续调用复用；任一
   ID、数量、顺序、5W1H、层级或表达身份变化都返回冲突。

普通 CHECK 无法安全表达跨行 `1..N` 与连续顺序，因此 Biz 负责整包验证，真实 PostgreSQL
事务测试是最高可观察 seam。数据库负责 FK、`(raw_evidence_id, split_order)` 唯一、非负、
行级层约束和非空表达身份，不引入 deferrable constraint trigger。

## API 合同

两个入口均使用 Data Service 现有 Bearer service token、严格 JSON、未知/重复字段拒绝、
1 MiB request body、Data Service 内部强制执行的 3 秒预算和标准 Error Envelope。发布方不提交
timeout Header 或请求字段，V1 不使用 `Idempotency-Key`。

### Raw Evidence Publication

```json
{
  "raw_evidence": {
    "raw_evidence_id": "RAW_example_00000000000000000000",
    "source_id": "SRC_example_00000000000000000000",
    "source_name": "Example Wire",
    "source_level": "L2_WIRE",
    "source_url": "https://example.test/article/1",
    "is_original": true,
    "quoted_source_id": null,
    "quoted_source_name": null,
    "title": "Example title",
    "raw_text": "Complete original article.",
    "published_at": "2026-08-11T01:00:00Z",
    "collected_at": "2026-08-11T01:05:00Z",
    "keywords": ["AI芯片", "供应链"]
  }
}
```

`content_hash` 由 PostgreSQL 从 `raw_text` 生成，调用方不提交。相同
`raw_evidence_id` 的全部业务字段（包括 Keywords 顺序）必须逐字段一致才能复用；任一漂移
返回 `409`。相同 `content_hash` 不设唯一约束，不同信源可以保存相同正文。

### Evidence Publication

```json
{
  "raw_evidence_id": "RAW_example_00000000000000000000",
  "evidences": [
    {
      "evidence_id": "EVD_example_00000000000000000000",
      "split_order": 0,
      "layer_type": "SINGLE",
      "source_who": "Example Corp",
      "source_what": "Example Corp expanded production.",
      "source_when": "2026-08-10T00:00:00Z",
      "source_when_raw": "August 10",
      "source_where": null,
      "source_why": null,
      "source_how": null,
      "source_who_core": null,
      "source_what_core": null,
      "source_when_core": null,
      "source_when_raw_core": null,
      "source_where_core": null,
      "source_why_core": null,
      "source_how_core": null,
      "expression_fingerprint": "Example Corp expands production",
      "expression_key": "example-corp-expands-production-v1",
      "fingerprint_version": "evidence-expression.v1"
    }
  ]
}
```

`is_split` 由 Data 计算，不属于 request DTO。请求引用的 Raw Evidence 必须已经存在；不存在
返回安全的 `422`。全部 Evidence 写入和冲突校验使用一个 PostgreSQL transaction，
任一失败整包回滚。

### 成功响应

成功调用返回 `201 Created`。Raw 响应只返回正式 `raw_evidence_id`；Evidence 响应返回
`raw_evidence_id` 与按 `split_order` 排序的 `evidence_ids`。成功/失败由 HTTP 状态和标准
Envelope 表达，不增加布尔状态，不返回 `receipt_id`、`imported_at`、`content_hash`、
Keywords、`is_split`、`created|reused` 或计数，也不保存独立发布审计记录。

### 错误与重试

- `400 INVALID_REQUEST`：JSON、未知/重复字段、格式、枚举或行级合同错误。
- `401/403`：Bearer token 缺失、非法或不具备路由权限。
- `409 CONFLICT`：自然身份对应已存在但内容不一致，或并发调用收敛后发现漂移。
- `413 PAYLOAD_TOO_LARGE`：请求超过 1,048,576 bytes；不得截断原文。
- `422 UNPROCESSABLE_ENTITY`：Raw 不存在、Evidence 集合/连续顺序/跨引用业务约束失败。
- `500/503`：安全内部错误、依赖暂不可用或 3 秒执行预算耗尽；超时取消 SQL 并回滚事务，
  不返回 SQL、表名、正文或内部错误。

未知调用结果只能用相同自然身份和完全相同正文重试。并发写以自然身份锁/唯一约束收敛；
同内容重试得到同一正式业务对象和相同正式 ID，内容漂移得到冲突。

## Migration 与兼容

Data migration ledger 新增 forward-only `000042`：

- plain `CREATE TABLE` 创建 `raw_evidences`、`evidences` 和两个 Receipt 表；若 Data DB
  异常存在同名非 ledger 表则 fail closed，不使用 `IF NOT EXISTS` 接纳未知 schema；
- 使用 PostgreSQL 16 内置 `sha256(bytea)` 生成正文 hash，不增加扩展依赖；
- 新表不引用、复制、改名或修改 `raw_documents`、`event_sources` 或 Event Publication；
- 不读取 AgentRun `tidewise_ai_server`，不迁移本地旧 8/19 行；
- 更新 UAT migration risk manifest，保护所有既有 Data 表和行数。

后续 forward-only `000043` 先为两张领域表增加无默认值的 nullable `created_at`，确保历史
行不发生回填；随后仅为未来 INSERT 设置 `transaction_timestamp()` 默认值，并使用
`NOT VALID CHECK` 约束新写入非空。该约束不扫描或改写历史行，部署 scope 为 schema-only。

本版本停止读写 `raw_evidence_publication_receipts`、`evidence_publication_receipts`，但在
上一版应用镜像仍处于可回滚窗口时保留这两张兼容空表及其专用 trigger function。待本版本
完成部署且不再回滚到旧写入方后，另以独立 forward-only migration 物理删除；清理不得修改
正式 Raw Evidence/Evidence 行、身份、约束或内部 `created_at`。

应用回滚使用上一版已知良好镜像；数据库只允许 reviewed forward repair，不提供 destructive
down migration。

## 实现模块边界

实现按统一 Kratos 结构在 API、Biz、Data、Service 四层使用单数 `evidence` 领域目录；
publication 只作为公开方法和 operation，不作为 package 或手写源码文件名。Biz 与 Data 的
`transaction.go` 分别拥有事务 Port 与 PostgreSQL Adapter，Evidence SQL 不再留在共享
`internal/data/postgres` 包。

## 测试 seam 与验收

### Biz

- Raw 自然身份首次创建/一致重试/conflict，Keywords 空/非空、顺序和原样保存；Data 不执行发布方
  的数量、长度、trim 或去重规则。
- Evidence `1:1` 派生 `is_split=false`；`1:N` 全部 true；拒绝零项、缺号、重复号、混合或
  非零起点。
- `SINGLE/DOUBLE`、两层 5W1H、`source_what`、表达身份必填与发布时间/事实时间分离。
- 已发布整包完全一致时复用，任何字段/基数/顺序漂移冲突。

### HTTP/OpenAPI

- 两条路径、Bearer scope、严格 JSON、未知/重复字段、1 MiB/413、标准错误、201 与仅含正式
  ID 的成功响应。
- 两条发布路径均由 Data Service 内部施加 3 秒 deadline；预算耗尽返回安全 503，取消 SQL
  并回滚完整事务，不接受调用方 timeout Header 或请求字段。
- Keywords 为 string list 并原样保存；content hash 为数据库只读事实；二者不在发布成功响应
  重复返回；无 Group 资源或 API。

### PostgreSQL

- 两阶段事务、回滚、并发首次创建/一致重试/conflict，且发布不产生独立 Receipt 写入。
- 新建 Raw Evidence 与 Atomic Evidence 由数据库生成非空 `created_at`；完全相同的重试
  复用原行且时间不变；历史行不由 migration 回填。
- FK、唯一 split order、非唯一 expression key 索引、generated content hash、Keywords
  `TEXT[] DEFAULT '{}'`、schema comments 和 migration ledger/risk manifest。
- migration 前后既有 Data 表/行保持不变，异常同名表 fail closed。

### Provider/Consumer fixture

fixture 由 Data 合同拥有，模拟任意采集/清洗发布方；不得 import AgentRun/AgentOS 代码，
不得要求其数据库、运行时或生命周期存在。Fixture 覆盖 Raw 先发布、Evidence 后发布、
自然身份重试、表达 key 重复但 Evidence 全保留，以及任一步失败不产生部分事实。

## 明确非目标

- AgentOS/AgentRun consumer、LLM、采集、关键词生成、清洗、Event 提取或投资结论生成；
- Source Catalog 或采集控制面；
- Evidence Group 表/API、语义相似搜索、embedding、自动合并或重复 Evidence 删除；
- 旧 AgentRun 表/8 条 Raw Document/19 条 Evidence 数据迁移；
- 既有 Event Publication、`raw_documents`、`event_sources` 或历史 Event Evidence 改造；
- 已发布 Evidence 修订、版本化、删除或局部追加。
