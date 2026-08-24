---
status: accepted
date: 2026-08-17
issue: 258
supersedes_in_part: 0013-data-entity-domain-and-projection-retirement.md, 0016-tidewise-ai-2-object-schema-and-independent-region.md
superseded_in_part_by: 0045-retire-entity-identifiers-and-redirects.md
---

# 独立 Industry 与 Concept 事实

## 背景

Industry 与 Concept 的名称、别名和活动状态曾由 `entity_nodes` 拥有，领域属性分别拆在
profile 表中。该双表聚合与 Tidewise AI 2.0 的独立 Object Schema 决策冲突，也让分类查询、
映射、外部标识、Redirect 与 Event Semantic 引用依赖同 ID 的通用 Entity shadow row。

## 决策

- Data 使用独立的单数 `industry` 与 `concept` 表拥有正式事实，不引用 `entity_nodes`，不再
  使用 profile 表。
- Migration `000056` 首次独立化时保留既有 `ENT + canonical lowercase UUID` ID；随后
  ADR-0024 与 migration `000058` 在保留 UUID 后缀的前提下，将正式身份切换为 Industry
  的 `IND` 与 Concept 的 `CON`。
- Industry 与 Concept 都直接拥有非空 `name` 和 `aliases`。旧 Entity 的 canonical name、
  layer、key、type discriminator 与 status 不进入独立对象；迁移只接受 canonical name 等于
  name 且 status 为 active 的 profiled row，不能安全表达时 fail closed。
- Industry 删除 `classification_version`、`classification_level` 和 `boundary_note`。根节点由
  `parent_industry_id IS NULL` 表示，深度由 `hierarchy_path_codes` 推导；同一分类体系内的直接
  父子路径仍由数据库约束和触发器保护。
- Concept 删除 `boundary_note`，保留 Concept Type、definition 与 review status。
- 既有 relation mapping、external identifier、redirect、Event Entity Link、Direct Impact 与
  Event Semantic binding 在 migration `000058` 中随 owner ID 同步改写前缀，保留 UUID
  后缀和业务内容，并通过 Data object-aware 引用校验识别独立对象。
- `entity_nodes`、`industry` 与 `concept` 共用全局唯一的 Data Object ID 命名空间。
  对象写入、引用写入与 owner 删除使用同一 ID advisory lock；全部多态引用统一为
  `RESTRICT`，不再为 Entity external identifier 单独保留 cascade 删除。
- 只有拥有 profile 的 Industry/Concept shadow row 被迁移后删除；没有 profile 的 legacy
  Entity row 不推测缺失属性、不删除，也不进入新的独立对象集合。
- API 中对象 ID 在 migration `000058` 后分别使用 `IND` 与 `CON` 前缀，UUID
  后缀保持不变。需要 Entity-shaped 展示的既有读取合同以独立对象 `name`
  同时提供 name/canonical display，并以自身 review contract 判断可用性；持久化层不恢复
  canonical name 或 status shadow 字段。

## 发布与回滚

Migration `000056` 是无 mixed-version 写窗口的 forward-only 协调切换。发布前停止 Data
写入者并确认 PostgreSQL 恢复点；使用候选 Data 镜像执行完整账本 migration，校验迁移前后
Industry/Concept ID、名称、别名、保留属性、时间戳及全部引用数量和值一致，再发布匹配的
Data Service。迁移遇到无法安全表达的对象、分类冲突或未支持的 typed Entity 外键时必须在
删除 shadow row 前失败。

回滚必须同时恢复迁移前 PostgreSQL 快照和上一版应用，不运行 down migration。旧应用依赖
profile 表和 shadow Entity，不能在 `000056` 之后单独回退。

## 影响

Data Context、OpenSPG Industry/Concept Object Schema、Biz model、Data Adapter、Event Semantic
读取、Research Graph、操作工具与测试均以独立对象为当前合同。ADR-0013 的其他 Entity
ownership 不变；Industry Chain 与 Chain Node 随后由 ADR-0023 独立化。ADR-0016 的
每种 Object 独立 Schema 决策继续有效。

ADR-0045 后续退役 Entity External Identifier 与 Entity Redirect；本 ADR 中对这两类历史
引用的保留和前缀改写只描述 `000056`–`000058` 当时的切换，不再是当前事实合同。
