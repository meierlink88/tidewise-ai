## ADDED Requirements

### Requirement: 双遍 AI 候选分析方法
系统 SHALL 使用两遍 AI 分析生成和复核首批节点/关系候选，但该方法只服务当前 change-specific manifest，不得建设产品能力、审核平台或 policy engine。

#### Scenario: 第一遍生成候选
- **WHEN** 用户已批准 AI 服务器算力基础设施边界
- **THEN** 第一遍必须记录候选 identity、端点、方向、来源、证据、反例、置信度和 disposition

#### Scenario: 第二遍独立复核
- **WHEN** 第一遍候选完成
- **THEN** 第二遍必须独立复核范围、identity、端点、关系类型、方向和证据，并输出 approve/reject/blocked/merge

#### Scenario: 分析结果不构成 Write 授权
- **WHEN** 双遍分析完成
- **THEN** 结果仍须人工 Review 冻结 manifest，且不得推定 PostgreSQL R2 或 Neo4j R3 授权

### Requirement: 最小 PG-first 分层写入
系统 SHALL 复用现有 schema、entity-seed/repository 与 graph-projector，以 change-specific manifest 完成最小 preflight、原子 PG Write、写后 Query、幂等复验和后续独立 Neo4j sync。

#### Scenario: 现有能力存在硬缺口
- **WHEN** Apply 只读 audit 证明现有 schema 或入口不能安全完成批准 manifest
- **THEN** 系统必须带证据回到 Review
- **AND** 不得自行新增 migration、repository/service、runner 或通用 framework

#### Scenario: 独立 PostgreSQL R2
- **WHEN** manifest、backup、identity/scope/count/hash/schema 和范围外保护已冻结且 PG Write 获得独立 R2 授权
- **THEN** 系统必须原子写入并立即 Query 端点、tuple、orphan、范围外保护与幂等

#### Scenario: 独立 Neo4j R3
- **WHEN** PG Query 已验收且 Neo4j sync 获得单独 R3 授权
- **THEN** 系统必须只从 PG accepted baseline 同步并 Query，不得反写 PostgreSQL 或扩张范围
