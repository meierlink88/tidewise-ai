---
status: accepted
date: 2026-08-20
issue: 304
amended_by: 0037-independent-company-persistence
superseded_by: 0057-rebuild-geopolitical-storyline-facts.md
extends: 0019-database-independent-domain-object-identities.md, 0028-rebuild-event-domain-around-atomic-evidence.md, 0034-independent-narrative-blueprint-objects.md, 0035-independent-storyline-domain-catalogs.md
---

# 独立 Storyline 持久化与 Event 关联

## 背景

Data 需要保存可持续演进的 Storyline 主记录。原始字段稿把产业类锚点命名为 `concept_id`，
同时持久化首末 Event 时间；后续设计审查明确该锚点实际是 IndustryChain，而且首末时间可从
关联 Event 计算。StorylineDomain 与 StorylineDomainTactic 是独立静态目录，本需求没有成立
Storyline 到这些目录的关系。最新对账状态属于主记录快照，也没有对账历史审计需求。

## 决策

- Data 建立独立 `Storyline`，物理表为 `storylines`，使用
  `STL + canonical lowercase UUID`。Storyline 不使用通用 Entity、StorylineDomain、
  StorylineDomainTactic 或 OpenSPG shadow object。
- Storyline 类型为 `GEOPOLITICAL | MACRO | INDUSTRY | CORPORATE`。每行必须且只能拥有与
  类型匹配的一个 restrictive anchor：`rivalry_id` 引用 GeopoliticRivalry，
  `macro_economic_id` 引用 MacroEconomic，`industry_chain_id` 引用 IndustryChain，
  `company_entity_id` 引用 Company Profile 的 Entity 身份。产业类锚点不是 Concept。
  此处 Company 决策已由 ADR-0037 修订：`company_entity_id` 被移除，`CORPORATE` 类型当前无锚点。
- 生命周期为 `EMERGING | ACTIVE | DORMANT | ARCHIVED`，默认 `EMERGING`；置信度范围为
  0.00–0.99。最新对账状态为 `ALIGNED | LAGGING | ACCUMULATING | DIVERGING | NEW_FACTOR`，
  对账分数范围为 0.00–1.00，并与原因和检查时间一起保存在主记录中。不建立对账历史表。
- `StorylineEventLink` 是 Storyline 与 Event 的当前多对多关系，物理表为
  `storyline_event_links`，使用 `SLE + canonical lowercase UUID`；端点对唯一，两个外键均
  restrictive。本阶段只允许建立和读取关系，不提供 unlink 或 delete。
- `first_event_at` 与 `last_event_at` 不持久化。读取时间边界时只聚合已关联 Event 的非空
  `occurred_at`；`announced_at` 不作回退。没有关联或全部 `occurred_at` 为空时，两个结果均为空。
- 首轮只交付 migration、公开 Data Adapter、身份支持与权威文档。Adapter 提供 create、get、
  type/status 过滤的稳定 list、update、link Event、list Event links 和 occurred-at bounds；
  不增加 Biz、Service、HTTP/OpenAPI、Server wiring、初始化数据、OpenSPG 或 UI。
- 本期没有 Biz UseCase，Adapter Create 与 Link Event 不接收主键并调用 Data Application 的
  统一随机身份原语；未来 Biz/API 接入时必须把身份生成收敛到 owning Biz。

## 发布与回滚

Migration `000066` 只增加空 enum、表、约束与索引，不改写既有事实，旧应用不读取或写入这些
结构，可与新 schema 共存。发布前以候选 Data 镜像执行全账本 check-only，确认 `000066` 是
唯一 pending migration 并保留 PostgreSQL 恢复点；apply 后验证 ledger 为 66、空表、STL/SLE
identity、类型与唯一锚点、native enum、数值边界、restrictive Event 关系及时间默认值满足合同。
应用回退保留新增空结构；若必须撤销 schema，恢复 migration 66 前快照或提交另行审阅的
forward repair，不运行 down migration。Biz/API、初始化数据、StorylineDomain 关系、对账历史、
unlink/delete、OpenSPG 与 UI 均需后续独立设计和发布。
