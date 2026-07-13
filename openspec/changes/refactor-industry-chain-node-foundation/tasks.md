## 1. Phase A：结构基础与受控 cleanup

- [x] 1.1 记录已完成人工字段 Review：`chain_node_profiles(entity_id, definition NOT NULL, boundary_note NULL)`，`theme_profiles(entity_id, definition NOT NULL, boundary_note NOT NULL)`；名称/aliases/状态复用 `entity_nodes`，不保留 position/category/unit/level/parent/market/source/observation，不建立专属 chain_node source mapping，不使用 `research_theme`。
- [x] 1.2 记录 2026-07-13 Material Proposal Change 已覆盖“保留旧 facts、复用旧 UUID/key、legacy→target 收敛”策略；未执行任何 PostgreSQL/Neo4j Write。
- [x] 1.3 记录结构专项 Review 整改范围：当前 checkpoint 仅包含目标 schema、cleanup migration 代码、完整只读 preflight 与生产入口切换；不包含候选数据、UUID/key 方案、seed 实现或关系阶段。
- [x] 1.4 测试先行重写 `000015` 静态契约，覆盖专用 session Write 授权门禁、精确目标集合、event links/entity edges、convergence/audit、physical constraint/topology/membership/source mapping/profile 的叶到根清理、未知 FK 阻断、最小 profile schema、禁止 TRUNCATE/CASCADE 与不可逆 down 边界。
- [x] 1.5 重写版本化 `000015_refactor_industry_chain_node_phase_a.sql`：编码受控 cleanup 与最小 chain_node/theme schema，只提交代码，不 apply migration、不写 PostgreSQL。
- [x] 1.6 测试先行实现 repeatable-read/read-only preflight：覆盖旧表与引用 counts、`event_entity_links`、全部 convergence/audit 子表、FK/trigger/function/view/rule catalog 引用、orphans、`entity_key` 条件门禁及非目标实体 counts/校验和。当前 checkpoint 不运行 cleanup Write。
- [x] 1.7 切换默认生产入口：普通 seed 不加载旧 sector/industry_chain/chain_node 数据文件或旧产业关系，service 拒绝旧实体、source mapping、membership/topology/constraint 与旧关系端点/类型，CLI 禁用旧 apply scopes/convergence flags；历史数据 fixture 仅存在于测试代码。
- [x] 1.8 运行 targeted tests、`go test ./...`、OpenSpec strict validation、diff/scope/secret 检查，创建并 push scoped structure implementation review checkpoint；随后停止等待主对话复验。
- [x] 1.9 记录主对话已复验 structure implementation checkpoint `0f20171`：targeted tests、`go test ./...`、OpenSpec strict validation 与 diff check 均通过；该验收只允许继续 first-batch data contract Review，不授权 migration、cleanup、seed 或任何 PostgreSQL/Neo4j Write。
- [x] 1.10 只读核验已批准工作簿 Sheet「标准化保留」并提交数据契约审阅材料：842 个互异 canonical、950 个互异原始名称、108 个同义合并、79 个宽边界审阅节点；1,156 条外部代码为 eastmoney 811、ths 345，241 个节点两侧均有代码且无跨节点代码冲突。当前不生成可执行 seed。
- [x] 1.11 **First-batch data contract Review 门禁**：主对话审阅 842 节点范围、aliases 规则、definition/boundary 生成与逐项 Review 策略、全新 UUID/entity_key、去重/幂等、dry-run/report，以及通用 `entity_external_identifiers` schema。6 条组合来源分类记录涉及 13 个代码，逐代码 taxonomy 未消歧前不得批准可执行 seed；不得确定具体 theme 实例。
- [x] 1.12 **Schema/TDD implementation Review 门禁**：仅在 1.11 明确获批后，测试先行实现 `entity_external_identifiers` migration/domain/repository、节点 identity、final seed dry-run/report 与逐行映射解析；提交 scoped diff、测试、schema diff 和 dry-run 格式供 Review，不 apply migration、不写数据库。
- [ ] 1.13 **Cleanup Review 门禁**：1.12 通过后运行并提交只读 preflight、可恢复备份证据、版本化 cleanup diff、冻结目标 ID 集合、每表预计删除 counts、catalog/逻辑引用、锁/事务影响、非目标保护与 forward-fix；单独请求 cleanup Write 授权。cleanup 必须先于新节点写入，避免新旧 chain_node 混入删除集合。
- [ ] 1.14 **Cleanup Write -> Query 门禁**：仅在 1.13 明确获批后 apply `000015`；立即 Query 证明旧专属表不存在、旧 sector/industry_chain/chain_node 与相关关系/链接/审计为 0、无孤儿、非目标 counts/校验和不变、重复执行幂等，等待验收。
- [ ] 1.15 **External identifier schema Review -> Write -> Query 门禁**：cleanup Query 验收后，展示 `entity_external_identifiers` schema diff、preflight、影响、备份和 forward-fix 并单独请求 schema Write；获批后执行并立即 Query 表/列/FK/唯一约束/索引/版本/幂等，等待验收。
- [ ] 1.16 **Final seed dry-run Review 门禁**：schema Query 验收后提交 842 个 node/profile 的全新 UUID/entity_key、canonical/name、aliases、逐条 definition/boundary、预计动作、冲突与幂等 report；单独请求 node/profile seed Write 授权，不包含 theme 实例或关系。
- [ ] 1.17 **Node/profile seed Write -> Query 门禁**：仅在 1.16 明确获批后写入 842 个 chain_node 与 profile；立即 Query counts、profile 完整性、new key/ID、aliases、重复、孤儿与重复执行幂等，等待验收。
- [ ] 1.18 **Mapping data Review -> Write -> Query 门禁**：node/profile Query 验收后提交 1,156 条逐行 mapping report，包含 entity、source_system、source_taxonomy_type、external_code、external_name 与冲突检查并单独请求 mapping Write；获批后写入并立即 Query eastmoney=811、ths=345、总数=1,156、241 个双来源节点、唯一性、绑定、孤儿与幂等，等待验收。
- [ ] 1.19 **Phase A 验收门禁**：cleanup、external identifier schema、node/profile seed 与 mapping data 的 Query 均获验收后才可进入 Phase B。PG cleanup 后 Neo4j 将暂时陈旧，本 change 不清理、不写入、不 rebuild。

## 2. Phase B：基于全新节点建立关系

- [x] 2.1 记录已完成人工关系语义 Review：唯一表 `chain_node_relations`，不复用 `entity_edges`；MVP 仅含 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`，删除 `contains`、`supplies_to`、`substitutes_for`、`transmits_to`，同一机制不得同时记为 `input_to` 与 `depends_on`，事件传导动态推导。
- [ ] 2.2 仅在 Phase A 明确验收后开始；测试先行新增 relation migration/领域 table-driven tests，覆盖字段、四类枚举、方向、新 chain_node 端点 FK、自环、tuple 唯一、mechanism/condition/evidence/provenance、状态、时间与 input/depends 机制互斥。
- [ ] 2.3 测试先行新增 repository/seed fixture、fake/sqlmock 或可重复集成测试，覆盖幂等 upsert、只接受 Phase A 新节点、禁止旧 topology/constraint ID 输入与写入 report。
- [ ] 2.4 实现 `ChainNodeRelation` 强类型契约、`chain_node_relations` 版本化 migration、repository/validator/seed/report；如提出新 physical constraint，必须使用新 ID/subject 与独立候选 Review。只完成代码与测试，不 apply migration、不写 PostgreSQL。
- [ ] 2.5 输出四类关系契约与候选边清单，逐条列出方向、mechanism、condition、evidence/provenance；不得加入旧 edge 映射、theme-node link、事件传导或替代关系。
- [ ] 2.6 **Relation schema Review -> Write -> Query 门禁**：展示 schema diff、preflight、影响、备份与回滚边界并单独请求 schema Write；获批后执行并立即 Query 表、约束、索引、FK、版本与幂等，等待验收。
- [ ] 2.7 **Relation data Review -> Write -> Query 门禁**：仅在 schema Query 验收后提交候选边/任何新 constraint 与 data 影响，单独请求 data Write；获批后写入并立即 Query 方向、端点、tuple、自环、机制冲突、evidence、孤儿与幂等，等待验收。
- [ ] 2.8 运行相关 Go 包测试、migration 静态/集成测试、`go test ./...` 与 OpenSpec strict validation，检查 scoped diff 和 secret；明确证明未运行 Neo4j rebuild、未写 Neo4j、未加入观测/事件推理/股票推荐或联盟/国家/市场/benchmark/index 调整。
- [ ] 2.9 **Apply Review 门禁**：提交 scoped diff、测试与每层 Review/Write/Query 证据，停止等待主对话人工 Review；批准前不得 Sync、Archive 或 Deliver。
