## ADDED Requirements

### Requirement: 有限首批 chain node 增量候选与写入
系统 SHALL 只对人工 Spec gate 批准的有限首批范围提出 chain_node 增量候选，并复用既有 identity/profile/seed 边界完成冲突检查、dry-run、Review 和幂等写入；不得把首批扩张为 842 节点全量重建或治理。

#### Scenario: 与现有 842 节点基线对齐
- **WHEN** 首批研究提出节点候选
- **THEN** 系统必须先按 entity key、canonical name、aliases、definition 和 boundary 与现有基线判定 reuse、merge、update、create 或 conflict
- **AND** 不得因名称相近创建重复节点或恢复 sector/industry_chain 容器

#### Scenario: 输出可审阅 dry-run
- **WHEN** 首批 final node manifest 进入 PostgreSQL Write Review
- **THEN** report 必须列出范围指纹、manifest hash、created/updated/unchanged/conflict、stable identity、profile diff、重复 alias、宽边界、orphan 和重复执行预期
- **AND** 任一 conflict 必须阻断 Write

#### Scenario: 仅写入批准首批
- **WHEN** `first-batch-postgres-write` 获得明确 R2 授权
- **THEN** seed runner 必须只写入 final manifest 中人工批准的节点和 profile，并保持范围外 842 节点 identity、profile 与 checksum 不变

#### Scenario: 自动化验证增量 seed
- **WHEN** 开发者运行 entity foundation 测试
- **THEN** fixture、loader、validator、repository fake/sqlmock、事务原子性、dry-run/report 和幂等测试必须覆盖 reuse/merge/update/create/conflict 与范围外保护
