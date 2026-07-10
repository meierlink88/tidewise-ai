## ADDED Requirements

### Requirement: 实体关系来源元数据
系统 SHALL 在 `entity_edges` 中保存最小来源与核验元数据，使实体关系事实能够与来源信息一同在 local、uat 和 prod 环境中增量演进。

#### Scenario: 对已有关系数据库执行增量 migration
- **WHEN** 已存在 `entity_edges` 业务数据的数据库执行版本化 migration
- **THEN** 系统必须新增 `source_name`、`source_url` 和 `verified_at` 字段
- **AND** 不得清空、删除或重建 `entity_edges` 或其他业务表

#### Scenario: 重复执行 migration
- **WHEN** 已经应用实体关系来源元数据 migration 的数据库再次执行 migration 检查
- **THEN** 系统必须识别已应用版本并成功完成检查
- **AND** 不得重复创建字段或修改既有关系记录

#### Scenario: 迁移文件自动化验证
- **WHEN** 开发者验证后端 migration
- **THEN** 自动化测试必须确认 migration 包含三个来源字段、可重复执行的追加语义和非破坏性 SQL 约束
