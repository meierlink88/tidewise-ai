---
status: accepted
date: 2026-08-17
issue: 263
supersedes_in_part: 0013-data-entity-domain-and-projection-retirement.md, 0022-independent-industry-and-concept.md
---

# 独立 ChainNode 与 IndustryChain 事实

## 背景

ChainNode 的名称、别名和审计时间曾由 `entity_nodes` 拥有，定义、边界备注与审核状态位于
profile；IndustryChain 的名称和别名也依赖通用 Entity，领域事实位于 definition 表。
Membership、Graph Edge、Relation、Constraint 与 Research 合同继续使用
`*_entity_id`，把两个业务对象错误表达为通用 Entity 的附属 profile。

## 决策

- Data 使用独立的单数 `chain_node` 与 `industry_chain` 表拥有正式事实，不引用
  `entity_nodes`，不保留 profile、definition shadow、双读或双写入口。
- 两类对象无损保留既有 `ENT + canonical lowercase UUID` ID。owning 主键列从
  `entity_id` 改为 `id`，不执行身份重排或前缀替换。
- 两类对象直接拥有非空 `name` 与 `aliases`。ChainNode 保留 definition、review status，
  删除 `boundary_note`，并从原 Entity 保留创建与更新时间；IndustryChain 保留 scope、
  target output、end use、geography、primary Country、as-of date、review facts、technology
  route、observable variables 与既有审计时间。
- Entity 的 type、key、layer、canonical name 与 status 不复制。迁移只接受 expected type、
  active status、`canonical_name = name`、非空名称与非空 aliases 的 profiled row；不能安全
  表达时在删除 shadow 前 fail closed。只有拥有 profile/definition 的 shadow Entity 被删除。
- Membership、Graph Edge、ChainNode Relation、Physical Constraint 与 Research 数据库列、
  Go model 和 wire contract 统一改用 `chain_node_id`、`industry_chain_id`、
  `from_chain_node_id` 与 `to_chain_node_id`。Research receipt map 列与 JSON 字段改用
  `reasoning_tree_ids_by_industry_chain_id`。旧字段名不保留兼容 alias。
- Data 的通用 Object type resolver、全局 ID 唯一性、支持的多态引用校验和 owner
  删除保护识别独立 ChainNode/IndustryChain；通用 Entity 写入口拒绝这两种新事实。
- Data API 为两个对象分别提供 create/list/get/update、独立 read/write scope 和稳定
  keyset pagination。ID 由 Biz 生成，不提供 delete。
- `doctype/chain-node.schema` 与 `doctype/industry-chain.schema` 是两个对象的 OpenSPG
  语义合同；PostgreSQL migration 仍是持久化权威。

## 发布与回滚

Migration `000057` 是无 mixed-version 写窗口的 forward-only 协调切换。发布前停止旧写入
并取得 PostgreSQL 恢复点，随后使用同一候选版本执行完整 migration ledger，校验对象 ID、
名称、别名、领域字段、时间戳、Membership、Graph Edge、Relation、Constraint、Research
事实及所有支持引用的数量和值一致，再发布 Data Service 和匹配消费者。

回滚必须同时恢复迁移前 PostgreSQL 快照与上一版应用；不得运行 down migration、恢复
profile 表或单独回退消费者。

## 影响

Data Context、OpenSPG、API、Biz、Data、Service、Event Semantic、Research、Miniapp 消费合同
和运维工具均使用独立对象及新 ID 字段。ADR-0022 中“Industry Chain 与 Chain Node ownership
不变”的结论由本决策取代；Industry 与 Concept 的独立对象决策保持有效。
