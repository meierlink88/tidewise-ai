---
status: accepted
date: 2026-08-15
supersedes_in_part: 0010-event-semantic-entity-first-cross-type-resolution.md, 0013-data-entity-domain-and-projection-retirement.md
extends: 0016-tidewise-ai-2-object-schema-and-independent-region.md, 0017-independent-country-and-economy-retirement.md
---

# 独立 Organization 与 Alliance 退役

## 背景

旧模型把联盟组织保存为 `entity_nodes` 中的 `alliance_org` UUID，并把组织属性拆入
`alliance_org_profiles`。该模型不能稳定表达可运营的分类、职能、中文语义标签和带时间的
Country 成员资格，也与 Region、Country 的独立 Object 方向冲突。

## 决策

- Data 建立独立 Organization Object，以 `ORG_ + code` 为唯一稳定身份；核心事实保存于
  `organizations`，不创建 shadow Entity 或 Profile。
- Category、Function、Domain Tag 分别使用三张可维护目录表。目录只保存 code、中文名称
  和时间戳，通过独立幂等 publication 发布；Domain Tag 必须唯一归属一个 Function。
- binding power、influence rating 与 membership type 是固定 PostgreSQL enum。成员关系使用
  闭日期区间，并在 PostgreSQL 阻止同一 Organization–Country 历史重叠。
- `headquarters_subdivision_id` 只是未来 Subdivision 的可空文本预留；本次不赋予外键、格式、
  查询或关系语义。
- Event Semantic 和 Research 正式识别 `organization` 与 `ORG_*`，从 Organization 权威表
  解析；不接受 `alliance_org` 类型或旧 Alliance UUID。
- migration `000048` 不猜测转换旧事实。它删除 Alliance Profile、旧 Entity 身份及依赖它们
  的完整 Event Semantic submission 和 Research publication aggregate，同时建立独立
  Organization、目录、标签、成员与正式 Event link 结构。

## 发布与回滚

这是没有混合版本窗口的协调式破坏性切换。发布前停止旧写入并取得 PostgreSQL 快照；随后
应用完整 migration ledger、运行 Organization catalog publication、部署新 Data Service，
最后恢复写入。失败时必须同时恢复切换前 PostgreSQL 快照和上一版应用。不得运行 down、
局部重建 Alliance 表或只回滚应用。
