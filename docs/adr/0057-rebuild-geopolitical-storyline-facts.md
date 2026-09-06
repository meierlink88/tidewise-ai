---
status: accepted
date: 2026-09-06
issue: 413
amends: 0019-database-independent-domain-object-identities.md, 0034-independent-narrative-blueprint-objects.md
supersedes: 0035-independent-storyline-domain-catalogs.md, 0036-independent-storyline-persistence.md, 0038-initialize-storyline-domain-catalog.md
---

# 重构地缘政治故事线事实

## 背景

投研推理已将 GeopoliticRivalry 明确为地缘政治推理的具体故事线，而不是再由
通用 Storyline 包装的抽象蓝图。旧 StorylineDomain 与 StorylineDomainTactic 又彼此
无关，无法表达“一个领域拥有多个可识别手段”。这会使 Event 语义匹配需要跨越
多组不完整对象，也会保留已不再需要的 Storyline–Event 持久化概念。

## 决策

- `geopolitic_rivalries` 作为地缘政治故事线主表。每条保存唯一中文名称、非枚举
  故事线分类、核心命题、核心参与方和主要传导；不保存范围、中文名称以外的名称、
  生命周期或对抗类型。
- 新建 `geopolitic_domains`，以 `GPD + canonical lowercase UUID` 为身份，保存唯一大写
  ASCII `code`、唯一中文名称、描述和 tactics JSONB 数组。每个 tactic 仅包含中文
  `name` 与 `description`；数组非空，同领域名称唯一。Tactic 不再拥有独立主键或表。
- 每条 GeopoliticRivalry 必须且只能引用一个 GeopoliticDomain，外键为 restrictive。
- 全量退役 `storyline_domain_tactics`、`storyline_domains`、`storyline_event_links`、
  `storylines` 及其当前枚举、Data Adapter、发布命令和 `SLD/SDT/STL/SLE` 当前身份。
- Data-owned `geopolitical-storylines-v1.json` 是一个简单、完整、版本化的发布包：
  14 个领域、每领域 8 个手段、44 条地缘政治故事线。发布器以 code 确定性生成
  `GPD` 身份，以审阅后的唯一中文名称确定性生成 `GPR` 身份，单事务幂等发布并对
  任何目录外身份 fail closed。
- 本次只交付 Data persistence、Object Schema、公开 Data Adapter、初始化包与离线发布器；
  不增加 Biz、Service、HTTP/OpenAPI、Admin/Miniapp、AgentOS 或图谱投影。

## 发布与回滚

Migration `000082` 是高风险、零兼容、前向切换。执行前必须停止旧 Storyline 与
GeopoliticRivalry 写入，保留经审阅的 PostgreSQL 恢复点，并确认发布镜像与初始化包
来自同一版本。迁移删除四张旧 Storyline 表和旧 GeopoliticRivalry 结构，然后创建新表；
不从旧数据猜测新故事线。迁移后手动运行 `geopolitical-catalog-publish`，验证精确的
14 个领域、112 个手段、44 条故事线、零孤立领域引用，并抽查关键故事线。

旧应用不兼容新 schema。回滚必须同时恢复 migration 82 前的 PostgreSQL 快照和上一版应用，
不运行 down migration。图谱改造作为下一个独立设计与发布单元。
