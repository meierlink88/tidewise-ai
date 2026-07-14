## ADDED Requirements

### Requirement: 产业节点关系独立策展
系统 SHALL 将 chain_node 静态关系与通用 `entity_edges` 分离，在独立 `chain_node_relations` 中按四类语义逐层 Review、Write 和 Query。

#### Scenario: Review 产业节点候选边
- **WHEN** 系统准备 `is_subcategory_of`、`is_component_of`、`input_to` 或 `depends_on` 候选边
- **THEN** 清单必须包含中文端点名称、entity key、方向、mechanism、condition、evidence/provenance、反例、不确定性和 Review disposition

#### Scenario: 分类和组成关系使用内部派生证据
- **WHEN** `is_subcategory_of` 或 `is_component_of` 只表达已批准节点 definition/boundary 可直接判定的稳定集合从属或物理/系统组成
- **THEN** evidence 可以使用 approved internal source artifact path、SHA-256、确定性 derivation rule 与两遍独立 AI Review 记录，不强制伪造外部来源 URL
- **AND** 最终可写 manifest 仍必须保存 `verified_at`，且不得将该证据扩张解释为投入、依赖、供给瓶颈或事件传导

#### Scenario: 因果和物理约束保持强外部证据
- **WHEN** 候选是 `input_to`、`depends_on` 或 physical constraint
- **THEN** 必须提供可定位的强外部证据、source URL、verified_at、成立条件和反例
- **AND** 只有名称、definition/boundary、词面邻近或内部派生规则时必须保持 blocked 或 rejected

#### Scenario: 委托双遍 AI Review
- **WHEN** 用户明确委托 AI 处理专业关系逐项判断
- **THEN** 第一遍生成与自检、第二遍独立 Reviewer 的 approve/reject/blocked 结论必须完整留痕
- **AND** 双遍 AI Review 不替代 schema/data R2 的人工授权

#### Scenario: 未批准候选不得写入
- **WHEN** 某条基于新 chain_node 的候选边尚未取得已批准 evidence contract 下的完整 Review disposition，或可写 manifest/R2 package 尚未冻结
- **THEN** 系统不得将其写入正式 seed、PostgreSQL 或 Neo4j

## MODIFIED Requirements

### Requirement: 关系批次写入和图谱重建
系统 SHALL 将已审阅的通用 `entity_edges` 关系先写入 PostgreSQL，再按其 change 契约从 PostgreSQL 重建 Neo4j，不允许直接手工维护 Neo4j 关系事实；产业节点关系例外地写入独立 `chain_node_relations`，且本 change 只执行 PostgreSQL Write 与 Query，不执行 Neo4j rebuild。

#### Scenario: 写入已审阅关系批次
- **WHEN** 某一通用关系族通过 review 并执行实体 seed
- **THEN** 系统必须幂等 upsert 该批 `entity_edges`，并输出 created、updated、unchanged、failed 和按关系类型统计

#### Scenario: 从 PG 重建 Neo4j
- **WHEN** 某一通用关系批次的 PG 验收通过并运行 `graph-projector rebuild-entities`
- **THEN** Neo4j 必须包含全部可投影实体节点和当前 PG 中 active 的已审阅关系，且不得包含 PG 中不存在的历史关系

#### Scenario: cleanup 后投影暂时陈旧
- **WHEN** 本 change 已清理 PostgreSQL 旧产业事实但后续投影 change 尚未执行
- **THEN** 系统必须将 Neo4j 标记为暂时陈旧
- **AND** 本 change 不得尝试清理、写入或重建 Neo4j

#### Scenario: 产业节点关系仅验收 PostgreSQL
- **WHEN** 本 change 的 chain_node relation 批次通过 Review 并获得 Write 授权
- **THEN** 系统必须幂等写入 `chain_node_relations` 并执行写后 Query
- **AND** 不得运行 Neo4j rebuild、清理或直接图写入

#### Scenario: 自动化验证关系清洗能力
- **WHEN** 开发者运行后端测试和 OpenSpec 校验
- **THEN** migration、loader、关系策略、repository、report 和 graph projection 边界必须具备自动化测试，且 `go test ./...` 与 OpenSpec 全局校验必须通过
