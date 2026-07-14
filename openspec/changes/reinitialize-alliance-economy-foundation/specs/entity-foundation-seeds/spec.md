## ADDED Requirements

### Requirement: Change-Specific Alliance Economy Importer
系统 SHALL 为本 change 提供一个只接受 frozen approved artifact 的最小 importer，优先复用现有 entity-seed/repository，并拒绝演变为通用数据导入框架。

#### Scenario: 加载 Frozen Artifact
- **WHEN** importer 启动 preflight 或获授权的 rebuild
- **THEN** 必须验证固定版本、checksum、45 alliance、79 economy、133 `member_of`、四个联盟字段、端点和方向；不接受 Excel、旧 CSV、旧 223 disposition 或任意外部 manifest 作为执行输入

#### Scenario: 限制实现范围
- **WHEN** 开发者实现 importer
- **THEN** 只能增加本批所需的 loader/validator、最小 repository 适配和固定入口，不得增加通用 service、policy engine、任意实体 mapping framework、计划语言或复杂 dry-run/report 子系统

#### Scenario: 保持 Economy 现有结构
- **WHEN** importer 映射 approved economy
- **THEN** 必须写入现有 `country_code/currency_code/region` profile，不得要求 `identity_kind`、新区域/货币规则或全局 `entity_key` 唯一索引

### Requirement: Read-Only Dependency Preflight
系统 SHALL 在 R3 cleanup Review 前输出可审计的只读 dependency package，且不得在 R1 执行 migration、seed 或 database write。

#### Scenario: 报告目标与引用
- **WHEN** preflight 审计 local PostgreSQL
- **THEN** 必须按表、FK、relation type、方向和 endpoint type 报告目标 counts/hash，并覆盖 alliance/economy profiles、entity edges、external identifiers 以及 market/sector/industry-chain/company/person 等对 economy 的引用

#### Scenario: 报告跨域事实
- **WHEN** 发现不由 45/79/133 重建的 economy/alliance 跨域关系或引用
- **THEN** preflight 必须逐类列为“删除并丢弃”或“保留/重建”候选，未获人工决定即 fail-closed，不得由 importer 静默级联删除

#### Scenario: 防止错误环境
- **WHEN** 环境不能被明确证明为获批 local 探索数据库
- **THEN** cleanup/rebuild 入口必须拒绝执行，不得把 local 豁免推广到 UAT、prod 或 shared

### Requirement: Scoped Cleanup 与 Latest Rebuild
系统 SHALL 把 cleanup 与 rebuild 实现为两个独立、可审阅、fail-closed 的执行包；前者为 R3，后者为 R2。

#### Scenario: 精确清理并断言 Zero
- **WHEN** 4.1 R3 获得明确授权且 preflight 未漂移
- **THEN** importer 必须只按已批准表、谓词、relation types 和跨域处置清理 local scope，并在提交/进入下一包前证明批准的 alliance/economy/member scope 为零

#### Scenario: 精确重建并查询
- **WHEN** 4.1 zero Query 已验收且 4.2 R2 获得独立授权
- **THEN** importer 必须以单事务或明确 fail-closed 边界重建 45/79/133，并输出 exact counts、端点、方向、孤儿、重复、checksum 和失败统计

#### Scenario: 幂等复跑
- **WHEN** 对已经符合 frozen manifest 的 local 数据再次运行 4.2
- **THEN** 不得创建重复 entity/profile/edge 或改变集合，Query 必须报告 45/79/133 仍精确成立

#### Scenario: 漂移时停止
- **WHEN** 环境、manifest checksum、preflight count/hash、FK/关系类型、跨域决策或 Query assertion 与授权包不一致
- **THEN** importer 必须停止且不得把旧批准解释为扩大 scope 的权限

### Requirement: 联盟与 Economy Rebuild 自动化验证
系统 SHALL 对最小 migration、manifest validator、repository/importer、dependency classification、原子性、zero/post assertions 与幂等提供 targeted tests。

#### Scenario: 运行验证
- **WHEN** 开发者验证本 change 实现
- **THEN** 相关包测试、migration 静态或隔离 integration tests、受影响 backend suite、共享 architecture/contract tests 与 OpenSpec strict 必须通过；普通测试不得访问真实外部网络或写真实 PostgreSQL/Neo4j
