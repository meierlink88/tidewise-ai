## ADDED Requirements

### Requirement: 采集源扩展配置持久化
系统 SHALL 在 PostgreSQL 采集源目录中持久化非敏感扩展配置，使来源参数可以随 `source_catalogs` 一起查询、审计和迁移。

#### Scenario: 创建扩展配置字段
- **WHEN** 数据库迁移执行到本 change 的版本
- **THEN** PostgreSQL 必须为 `source_catalogs` 提供 `source_config` JSONB 字段，默认值为空 JSON 对象，并且不得影响既有来源记录读取

#### Scenario: 读取扩展配置
- **WHEN** repository 查询 active source 或 seed 后读取来源记录
- **THEN** 系统必须返回 `source_config` 中的非敏感结构化参数，供后续 connector、parser 或 job 使用

### Requirement: 版本化来源初始化
系统 SHALL 通过 repo 内版本化来源清单初始化和更新统一采集源目录，而不是依赖手工数据库操作或散落脚本。

#### Scenario: 初始化来源目录
- **WHEN** local、uat 或 prod 环境需要初始化第一批采集源
- **THEN** 运维或开发者必须能够运行统一 seed 命令，将 repo 内来源清单写入目标 PostgreSQL，并得到接入来源总数和分类统计

#### Scenario: 审计来源变更
- **WHEN** 后续新增、禁用或修改采集来源
- **THEN** 该变更必须体现在 repo 内来源清单、测试和必要 migration 中，而不是只修改数据库现状

#### Scenario: 保护敏感配置
- **WHEN** 来源初始化涉及需要凭证的 provider
- **THEN** seed 数据必须只写入授权类型和凭证引用，不得写入真实 API key、cookie、token 或私有数据库连接信息

#### Scenario: 管理多类型来源
- **WHEN** 来源清单包含内容、行情、板块或本地回灌来源
- **THEN** PostgreSQL 中的 `source_catalogs` 必须能够保存这些来源的用途、类型、provider、connector、parser、扩展配置、授权策略、限流策略和状态，便于统一查询和治理
