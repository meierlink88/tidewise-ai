## ADDED Requirements

### Requirement: Neo4j 图谱投影架构边界
系统 SHALL 将 Neo4j 定义为从 PostgreSQL 权威事实源派生的图谱查询库，用于多跳关系查询、路径分析、图谱推理和后续可视化，而不是替代 PostgreSQL 成为主事实库。

#### Scenario: 规划 Neo4j 图谱能力
- **WHEN** 后续 change 需要使用 Neo4j 保存实体图或事件图
- **THEN** 该 change 必须说明 PostgreSQL 中的事实来源、投影规则、重建方式和 Neo4j 只作为图谱查询视图的边界

#### Scenario: 恢复 Neo4j 数据
- **WHEN** Neo4j 数据损坏、清空或图模型需要调整
- **THEN** 系统必须能够从 PostgreSQL 事实表重新投影恢复图谱数据

#### Scenario: 保持服务端部署边界
- **WHEN** 部署 Neo4j 或图谱投影 worker
- **THEN** 该能力必须作为服务端基础设施和后端运行时能力部署，不得出现在小程序或管理后台前端源码中
