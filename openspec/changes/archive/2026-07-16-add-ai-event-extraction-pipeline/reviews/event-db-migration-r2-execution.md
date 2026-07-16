# Event DB Migration R2 Execution Evidence

执行时间：2026-07-16。环境仅为 local `tidewise_local`；未访问 UAT、prod、shared、Neo4j 或外部网络，未执行 seed、回填、实体关联或业务数据写入。

## Frozen Input

- Migration：`000019_add_event_fact_contract.sql`
- SHA-256：`c89c402836945fb55950b749b6cc4764ef1890cef223309eea161adbe9002d60`
- PostgreSQL：local container `tidewise-local-postgres`，image `postgres:16`，server `16.14`
- Before Goose：version `18`，applied rows `19`
- Before counts：`raw_documents/events/event_sources/event_tag_defs/event_tag_maps/event_entity_links = 407/0/0/0/0/0`
- Before target columns：`0`
- Before duplicate evidence triples：`0`

## Recovery Evidence

- Backup：`/private/tmp/tidewise_local_pre_event_schema_000019_20260716T091425Z.dump`
- Format：PostgreSQL custom format，完整 local database
- Size：`1,104,455` bytes
- SHA-256：`f35ee7a7588ce86acbc19cd7f256653d501daca413d9e301ce16da249b6a7768`
- `pg_restore --list`：成功，`200` TOC lines
- 本轮未执行 restore 或 forward-fix。

## Single Apply

`dbmigrate --apply --target-version 000019` 仅执行一次并成功。Goose 报告 migration `000019` applied，remaining 为 empty；未重试 migration。

## Post Assertions

- Goose：version `19`，applied rows `20`；fresh check 显示 pending/applied/remaining 均为空。
- Counts：`407/0/0/0/0/0`，与 fresh before 完全一致。
- 新增列：`events.fact_payload`；`event_sources.evidence_relation/supports_fields`；`event_tag_maps.confidence/assignment_reason`，共 `5` 个。
- 约束：evidence relation、supports fields、confidence 共 `3` 个，全部 validated。
- 幂等：`ux_event_sources_event_document_evidence` unique index 存在；duplicate evidence groups 为 `0`。
- `event_entity_links`：从 pre-migration backup 提取的 CREATE TABLE 与当前定义逐字一致；对应 table/constraint/index/FK 对象集合计数均为 `7`。

首次 post-check 对 `pg_dump` 原始输出直接取 hash，因 PostgreSQL 16 每次生成随机 `\\restrict` token 而产生非语义差异；未重跑 migration。随后只读诊断改为 backup/current CREATE TABLE 规范化比较及对象集合比较，断言通过。
