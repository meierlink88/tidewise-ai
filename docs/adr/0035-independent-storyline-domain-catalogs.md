---
status: accepted
date: 2026-08-19
issue: 302
extends: 0019-database-independent-domain-object-identities.md
---

# 独立 StorylineDomain 与 StorylineDomainTactic 目录

## 背景

Data 需要保存第二组静态叙事定义：叙事线领域及领域手段。原始字段稿把手段描述为挂载在
特定领域下的标签，但没有提供 `domain_id` 或关系字段；当前需求进一步明确，本阶段不考虑
StorylineDomain 与 StorylineDomainTactic 的关联关系。因此不能通过隐式命名、分类或未来
Storyline Thread Template 假设建立关系。

## 决策

- Data 在 Entity 父领域下建立独立 `StorylineDomain` 与 `StorylineDomainTactic` 子领域，
  物理表分别为 `storyline_domains` 与 `storyline_domain_tactics`。Data Adapter 位于
  `internal/data/entity/<subdomain>`，与既有 Entity 子领域拓扑一致。
- StorylineDomain 使用 `SLD + canonical lowercase UUID`，保存中英文名称、描述、内容边界、
  闭集分类和启用状态。分类只允许
  `GEOPOLITICAL | MACRO | INDUSTRY | CORPORATE`，默认启用；名称允许重复。
- StorylineDomainTactic 使用 `SDT + canonical lowercase UUID`。`key` 是全表唯一且不可变的
  机器自然键，只允许大写 ASCII 字母、数字与下划线并以大写字母开头；名称允许重复。
- 两个对象当前彼此独立，不增加 `domain_id`、外键或关系表，也不引用 Storyline、
  Storyline Thread Template、GeopoliticRivalry、MacroEconomic 或其他对象。未来关系需要
  独立设计关系 owner、方向、基数、迁移和 API 合同。
- 首轮只交付 persistence、公开 Data Adapter、身份支持与权威文档。Domain Adapter 提供
  create、get、category/is_active 组合过滤的稳定 list 和 update；Tactic Adapter 提供
  create、get、按 key 稳定 list 和不修改 key 的 update。两者均不提供 delete。
- 本期没有 Biz UseCase，Adapter Create 不接收主键并调用 Data Application 的统一随机身份
  原语；未来 Biz/API 接入时必须把身份生成收敛到 owning Biz。

## 发布与回滚

Migration `000065` 只增加空 enum、表、约束与索引，不改写既有事实，旧应用不读取或写入这些
结构，可与新 schema 共存。发布前以候选 Data 镜像执行全账本 check-only，确认 `000065` 是
唯一 pending migration 并保留 PostgreSQL 恢复点；apply 后验证 ledger 为 65、空表、
SLD/SDT identity、native enum、Tactic key、必填字段、默认启用状态与时间满足合同。应用回退
保留新增空结构；若必须撤销 schema，恢复 migration 65 前快照或提交另行审阅的 forward repair，
不运行 down migration。关系、初始化数据、Biz/API wiring、OpenSPG 与 UI 均需后续独立发布。
