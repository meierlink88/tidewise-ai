## ADDED Requirements

### Requirement: 本地数据库和采集 smoke 架构边界
系统 SHALL 将真实数据库建表和真实采集 smoke 归入后端与基础设施边界，不得让前端、小程序或 prototype 参与数据库访问和采集执行。

#### Scenario: 运行真实建表
- **WHEN** 开发者需要创建或更新 PostgreSQL schema
- **THEN** 必须通过后端 migration 命令、API 启动检查或部署流程执行，而不是通过前端、小程序或手工散落 SQL 执行

#### Scenario: 运行真实采集
- **WHEN** 开发者需要验证公开来源采集和入库链路
- **THEN** 必须通过后端 ingestion job 和 repository 边界运行，并保持采集数据只作为原始事实材料

#### Scenario: 保持分析安全边界
- **WHEN** smoke 数据后续被用于事件抽取、Agent 分析或展示
- **THEN** 系统必须继续保持决策辅助定位，不得把采集原文直接表达为投资建议
