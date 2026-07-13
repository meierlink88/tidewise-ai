## ADDED Requirements

### Requirement: 联盟关系图谱投影门禁
系统 SHALL 仅从 PostgreSQL 已验收的 active 联盟实体、economy 和关系重建 Neo4j，并保持单一 `Entity` label 与 `projection_namespace=tidewise`。

#### Scenario: PostgreSQL 未全部验收时阻止重建
- **WHEN** alliance、economy 或目标关系的 PostgreSQL Query 尚未全部获得人工验收
- **THEN** 系统不得执行 Neo4j rebuild、直接图写入或关系补写

#### Scenario: 单独审阅 Neo4j 重建
- **WHEN** PostgreSQL 全部目标层已验收并计划更新图投影
- **THEN** 必须先展示重建范围、预期节点与 `MEMBER_OF`/`LED_BY`/`PART_OF` 数量、清理命名空间和查询断言，并取得独立 Rebuild 授权

#### Scenario: 投影联盟关系
- **WHEN** 获批执行 graph rebuild
- **THEN** graph mapper 必须将 PostgreSQL active `member_of`、`led_by`、`part_of` 分别映射为 `MEMBER_OF`、`LED_BY`、`PART_OF`，保留原始 relation type、edge identity、来源、状态、更新时间和投影命名空间

#### Scenario: 保持单一实体标签
- **WHEN** 联盟或 economy 被投影到 Neo4j
- **THEN** 节点必须继续使用单一 `Entity` label，不得创建 alliance、economy、category 或事件标签，也不得把 CSV 候选或非 active 成员身份投影为关系

#### Scenario: 重建后查询闭环
- **WHEN** Neo4j rebuild 完成
- **THEN** Query 必须证明目标关系的方向、端点和按类型计数与 PostgreSQL active facts 一致，且不存在 PostgreSQL 中没有的历史联盟关系
