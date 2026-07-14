## ADDED Requirements

### Requirement: 首批产业关系双遍研究审阅
系统 SHALL 对有限首批范围内的节点、关系和 physical constraint 候选执行可追踪的两遍 AI Review，并把候选 Review 与任何 PostgreSQL/Neo4j Write 授权严格分离。

#### Scenario: 第一遍研究与自检
- **WHEN** AI researcher 生成首批候选
- **THEN** 产物必须记录输入指纹、生成规则、总体 counts、逐项证据、反例、置信度、冲突、宽边界、低置信度和初始 disposition

#### Scenario: 第二遍独立审核
- **WHEN** 第一遍候选完成
- **THEN** 独立 reviewer 必须逐项给出 approve、reject、blocked 或 merge 结论并记录理由
- **AND** `input_to`、`depends_on` 与 physical constraint 缺少强外部证据时不得 approve

#### Scenario: 双遍 Review 不构成写入授权
- **WHEN** 两遍 AI Review 均完成
- **THEN** 产物仍只能作为人工候选 Review 输入，未冻结 final manifest 且未取得命名 R2/R3 授权前不得写入 PostgreSQL 或 Neo4j

### Requirement: 首批关系先 PG 后 Neo4j 分层验收
系统 SHALL 先把人工批准的首批事实通过独立 R2 层写入 PostgreSQL 并 Query 验收，再通过单独 R3 层从 PostgreSQL 同步 local Neo4j；任一层的批准不得推定另一层、环境或范围。

#### Scenario: 写入 PostgreSQL 首批事实
- **WHEN** final manifest、identity/scope/count/hash/schema、backup、before/after assertions 和停止条件已冻结且 `first-batch-postgres-write` 获得明确 R2 授权
- **THEN** 系统必须在 approved scope 内原子写入并立即 Query created/updated/unchanged/conflict、证据完整性、重复、孤儿和幂等结果

#### Scenario: PostgreSQL 写入失败
- **WHEN** manifest 漂移、identity conflict、部分写入、证据断言或 Query 失败
- **THEN** 系统必须 fail-closed 并停止，不得继续同步 Neo4j；未执行授权自动失效

#### Scenario: PostgreSQL 验收后同步 Neo4j
- **WHEN** PostgreSQL Query 已人工验收且 `first-batch-neo4j-sync` 获得单独 R3 授权
- **THEN** 系统必须只读 accepted PostgreSQL facts 进行同步并执行 PG/Neo4j 一致性 Query
- **AND** 不得从 Neo4j 创建、修正或反写 PostgreSQL 事实
