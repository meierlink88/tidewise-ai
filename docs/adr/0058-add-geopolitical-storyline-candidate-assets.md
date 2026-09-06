---
status: accepted
date: 2026-09-06
issue: 415
amends: 0057-rebuild-geopolitical-storyline-facts.md
---

# 地缘政治故事线保存候选资产

## 背景

GeopoliticRivalry 已保存核心命题与主要传导，但没有保留经过审阅的下游投研对象。
如果每次都从领域或传导文本重新生成，不同执行的研究范围会漂移，也无法对初始化基础数据做
确定性验收。

## 决策

- `geopolitic_rivalries` 增加非空 `candidate_assets JSONB`，只接受非空、无首尾空白、单项不超过
  100 字符且不重复的字符串数组；数组顺序是审阅后的研究优先顺序。
- 候选资产可以是可交易资产、市场指标、板块或产业链节点；它仅定义故事线匹配后的下游投研
  候选集，不表达看好、不看好、方向、置信度或最终结论。
- Event 到 GeopoliticRivalry 的匹配只使用故事线名称、分类、领域、核心命题、核心参与方与手段等
  地缘政治语义；不得用 `candidate_assets` 反向扩大 Event 匹配范围。
- Data-owned `geopolitical-storylines-v2.json` 作为当前发布包，为每条故事线完整保存候选资产；
  v1 只作为已发布历史制品保留。发布器仅接受 schema version 2。
- 本次范围只包含 Data persistence、Object Schema、Data Adapter、初始化包和离线发布器；不改
  HTTP/OpenAPI、Admin、Miniapp、AgentOS、Event 匹配或图谱投影。

## 发布与回滚

Migration `000083` 是空故事线表前置的 forward-only 切换。由于 v1 故事线是可重建的目录事实，
操作员必须先停止 GeopoliticRivalry 写入、保留恢复点，再显式清空这些可重建行。如果表中仍有任何行，
迁移以 SQLSTATE `55000` fail closed，不自动删除、补齐或猜测候选资产。迁移完成后，使用同一发布镜像的
v2 包重建全量故事线并验收 JSON 数组。完整回滚必须恢复迁移前 PostgreSQL 快照并同时回退应用，
不运行 down migration。
