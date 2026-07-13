## 1. Phase A：结构基础与受控 cleanup

- [x] 1.1 记录已完成人工字段 Review：`chain_node_profiles(entity_id, definition NOT NULL, boundary_note NULL)`，`theme_profiles(entity_id, definition NOT NULL, boundary_note NOT NULL)`；名称/aliases/状态复用 `entity_nodes`，不保留 position/category/unit/level/parent/market/source/observation，不建立 source mapping，不使用 `research_theme`。
- [x] 1.2 记录 2026-07-13 Material Proposal Change 已覆盖“保留旧 facts、复用旧 UUID/key、legacy→target 收敛”策略；未执行任何 PostgreSQL/Neo4j Write。
- [x] 1.3 记录结构专项 Review 整改范围：当前 checkpoint 仅包含目标 schema、cleanup migration 代码、完整只读 preflight 与生产入口切换；不包含候选数据、UUID/key 方案、seed 实现或关系阶段。
- [x] 1.4 测试先行重写 `000015` 静态契约，覆盖专用 session Write 授权门禁、精确目标集合、event links/entity edges、convergence/audit、physical constraint/topology/membership/source mapping/profile 的叶到根清理、未知 FK 阻断、最小 profile schema、禁止 TRUNCATE/CASCADE 与不可逆 down 边界。
- [x] 1.5 重写版本化 `000015_refactor_industry_chain_node_phase_a.sql`：编码受控 cleanup 与最小 chain_node/theme schema，只提交代码，不 apply migration、不写 PostgreSQL。
- [x] 1.6 测试先行实现 repeatable-read/read-only preflight：覆盖旧表与引用 counts、`event_entity_links`、全部 convergence/audit 子表、FK/trigger/function/view/rule catalog 引用、orphans、`entity_key` 条件门禁及非目标实体 counts/校验和。当前 checkpoint 不运行 cleanup Write。
- [x] 1.7 切换默认生产入口：普通 seed 不加载旧 sector/industry_chain/chain_node 数据文件或旧产业关系，service 拒绝旧实体、source mapping、membership/topology/constraint 与旧关系端点/类型，CLI 禁用旧 apply scopes/convergence flags；历史数据 fixture 仅存在于测试代码。
- [x] 1.8 运行 targeted tests、`go test ./...`、OpenSpec strict validation、diff/scope/secret 检查，创建并 push scoped structure implementation review checkpoint；随后停止等待主对话复验。
- [ ] 1.9 **Cleanup Review 门禁**：运行并提交只读 preflight、可恢复备份证据、版本化 cleanup diff、冻结目标 ID 集合、每表预计删除 counts、catalog/逻辑引用、锁/事务影响、非目标保护与 forward-fix；单独请求 cleanup Write 授权。
- [ ] 1.10 **Cleanup Write -> Query 门禁**：仅在 1.9 明确获批后 apply `000015`；立即 Query 证明旧专属表不存在、旧 sector/industry_chain/chain_node 与相关关系/链接/审计为 0、无孤儿、非目标 counts/校验和不变，等待验收。
- [ ] 1.11 **数据初始化设计门禁（延后）**：cleanup Query 验收后另行提交最终候选范围、definition/boundary、aliases、UUID/key 规则与去重策略供主对话 Review；本次结构 checkpoint 不引用任何工作簿作为可执行输入，不确定具体 theme 实例。
- [ ] 1.12 仅在 1.11 获批后按 TDD 实现 final seed/dry-run/report；不得复用旧 UUID/key，不得在设计批准前预置候选分类或身份生成逻辑。
- [ ] 1.13 **Final seed Review -> Write -> Query 门禁**：seed 实现完成后提交 scoped diff、dry-run、影响与备份边界并单独请求 seed Write；获批后写入并立即 Query，等待验收。
- [ ] 1.14 **Phase A 验收门禁**：cleanup 与 final seed 的 Query 均获验收后才可进入 Phase B。PG cleanup 后 Neo4j 将暂时陈旧，本 change 不清理、不写入、不 rebuild。

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
