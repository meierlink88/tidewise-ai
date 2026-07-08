## ADDED Requirements

### Requirement: 版本化实体基础库初始化
系统 SHALL 通过 repo 内版本化实体 seed 初始化和更新实体基础库，而不是依赖手工数据库操作、临时 SQL 或散落脚本。

#### Scenario: 初始化实体基础库
- **WHEN** local、uat 或 prod 环境需要初始化一阶段实体基础库
- **THEN** 运维或开发者必须能够运行统一 seed 命令，将 repo 内实体清单写入 PostgreSQL，并得到实体总数、类型分布、profile 写入数量和关系写入数量的统计报告

#### Scenario: 覆盖所有实体 profile
- **WHEN** 一阶段实体 seed 完成
- **THEN** PostgreSQL 中必须至少完成联盟组织、经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物这些实体 profile 表的初始化验证

#### Scenario: 幂等更新实体基础库
- **WHEN** 同一套实体 seed 在同一数据库中重复执行
- **THEN** 系统必须按稳定实体 key 幂等 upsert，不得创建重复实体、重复 profile 或重复关系

#### Scenario: 审计实体基础库变更
- **WHEN** 后续新增、禁用或修改基础实体、profile 属性或基础关系
- **THEN** 该变更必须体现在 repo 内实体 seed、测试和必要 migration 中，而不是只修改数据库现状

#### Scenario: 保持决策辅助边界
- **WHEN** 实体 seed 初始化基础实体和关系
- **THEN** seed 数据不得包含投资建议、预测结论、利好利空判断、传导强度或事件评分
