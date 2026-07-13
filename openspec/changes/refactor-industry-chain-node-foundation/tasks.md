## 1. Phase A：统一节点模型与产业链节点初始化

- [x] 1.1 记录已完成人工字段 Review：`chain_node_profiles(entity_id, definition NOT NULL, boundary_note NULL)`，`theme_profiles(entity_id, definition NOT NULL, boundary_note NOT NULL)`；名称/aliases/状态复用 `entity_nodes`，不保留 position/category/unit/level/parent/market/source/observation，不建立 source mapping，不使用 `research_theme`。
- [ ] 1.2 在进入 Apply 前取得整个 Proposal 的人工 Review 批准；未批准时停止，不修改源码或数据库。
- [ ] 1.3 测试先行：新增 migration 静态测试与领域 table-driven tests，覆盖最小 profile DDL、definition/boundary 非空、`entity_type=theme`、禁用旧字段/旧类型、无 `chain_node_source_mappings`，以及 `entity_key` 全局唯一约束默认不实施。
- [ ] 1.4 测试先行：为只读 preflight、legacy→target 映射、stable ID/key、convergence 引用迁移、候选过滤、dry-run/report 与幂等行为新增 fixture、fake/sqlmock 或可重复集成测试，先确认测试按预期失败。
- [ ] 1.5 实现 Phase A 版本化 forward migration 与 Go 领域/repository/seed 边界：统一 `chain_node`、新增 `Theme` / `ThemeProfile`、收敛 profiles，保留旧 facts 作为输入；此任务只完成代码与测试，不 apply migration、不写 PostgreSQL。
- [ ] 1.6 扫描并切换生产读写路径，禁止新建 sector、industry_chain 容器、membership、source mapping 或具体 theme 实例；保留 alliance、economy/country、market、benchmark/index 行为不变，并用测试证明没有恢复平行模型。
- [ ] 1.7 运行只读全库 preflight，输出旧类型/profile/引用 counts、空值/重复 `entity_key`、合并状态、孤儿、备份可用性和 legacy→target 草案；若非零冲突则记录阻断，不添加全局唯一约束。
- [ ] 1.8 生成 chain_node 候选 Review 清单，逐项列出复用/合并/停用/拒绝、definition、boundary、stable ID/key 处理及参考证据；过滤涨停、融资融券、高股息等非产业标签。theme 只记录空数据边界，不自行提出实例。
- [ ] 1.9 **人工 Review 门禁**：提交 Phase A 实现 diff、schema/migration 设计、preflight 与候选清单；未经主对话明确批准，不得执行任何 schema/data Write。
- [ ] 1.10 **Schema Review -> Write -> Query 门禁**：展示最新 schema diff、preflight、影响表/row 范围、备份验证、事务与 forward-fix 回滚边界，单独请求 PostgreSQL schema Write 授权；获批后 apply Phase A migration，并立即 Query schema、约束、版本、引用与重复执行结果，等待 schema Query 验收。
- [ ] 1.11 **Data Review -> Write -> Query 门禁**：仅在 schema Query 验收后，再次展示节点候选与初始化影响并单独请求 PostgreSQL data Write 授权；获批后只写入批准的 chain_node，立即 Query counts、profile 完整性、重复、孤儿、stable references、created/updated/unchanged/failed 与幂等结果，等待 data Query 验收。
- [ ] 1.12 **Phase A 验收门禁**：提交全部 Write/Query 证据并等待主对话明确验收；未验收不得准备最终关系写入 manifest、实现 Phase B 生产写入或执行任何 Neo4j 操作。

## 2. Phase B：产业链节点关系建立

- [x] 2.1 记录已完成人工关系语义 Review：唯一表 `chain_node_relations`，不复用 `entity_edges`；MVP 仅含 `is_subcategory_of`、`is_component_of`、`input_to`、`depends_on`，删除 `contains`、`supplies_to`、`substitutes_for`、`transmits_to`，同一机制不得同时记为 `input_to` 与 `depends_on`，事件传导动态推导。
- [ ] 2.2 仅在 Phase A 明确验收后开始；测试先行新增 relation migration/领域 table-driven tests，覆盖字段、四类枚举、方向、chain_node 端点 FK、自环、tuple 唯一、mechanism/condition/evidence/provenance、状态、时间与 input/depends 机制互斥。
- [ ] 2.3 测试先行新增 repository/seed fixture、fake/sqlmock 或可重复集成测试，覆盖幂等 upsert、旧 edge 转换拒绝规则、`supplies_to` 非机械改名、`substitutes_for` 停用、old-edge→new-relation 映射与写入 report。
- [ ] 2.4 测试先行新增 physical constraint migration 测试，覆盖节点 subject 保留、旧 topology edge 唯一映射到 `chain_node_relations.id`、node/relation subject XOR、歧义映射阻断与无 `industry_chain_entity_id`。
- [ ] 2.5 实现 `ChainNodeRelation` 强类型契约、`chain_node_relations` 版本化 migration、repository/validator/seed/report 与 `chain_node_physical_constraints` 前向迁移代码；此任务只完成代码与测试，不 apply migration、不写 PostgreSQL。
- [ ] 2.6 输出四类关系契约与候选边清单，逐条列出方向、mechanism、condition、evidence/provenance、旧 edge 处置及 constraint subject 映射；不得加入 theme-node link、事件传导或替代关系。
- [ ] 2.7 **人工 Review 门禁**：提交 Phase B 实现 diff、候选边、old-edge 与 constraint 映射；未经主对话逐层明确批准，不得执行 relation/constraint schema 或 data Write。
- [ ] 2.8 **Relation schema Review -> Write -> Query 门禁**：展示最新 relation schema diff、preflight、影响、备份与回滚边界，单独请求 PostgreSQL schema Write 授权；获批后 apply migration，并立即 Query 表、约束、索引、FK、版本、旧结构状态与重复执行结果，等待 schema Query 验收。
- [ ] 2.9 **Relation/constraint data Review -> Write -> Query 门禁**：仅在 schema Query 验收后，再次展示候选边、old-edge/constraint 映射与 data 影响并单独请求 PostgreSQL data Write 授权；获批后只写入已批准关系和 subject 映射，立即 Query 方向、端点、tuple、自环、机制冲突、evidence、孤儿、未迁移阻断项与幂等结果，等待 data Query 验收。
- [ ] 2.10 **旧结构 cleanup Review -> Write -> Query 门禁**：仅在 relation/constraint data Query 验收后，如需停用或移除旧 `sector_profiles`、`sector_source_mappings`、`industry_chain_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges`，先提交引用扫描、备份、影响与回滚边界并取得独立 PostgreSQL cleanup Write 授权；只能用版本化 forward migration，Write 后立即 Query 验证旧结构状态、引用完整性、孤儿、counts 与重复执行幂等，等待 cleanup Query 验收，禁止手工清库或历史回滚。
- [ ] 2.11 运行相关 Go 包测试、migration 静态/集成测试、`go test ./...` 与 `openspec validate refactor-industry-chain-node-foundation`，检查 scoped diff 和 secret；明确证明未运行 Neo4j rebuild、未写 Neo4j、未加入观测/事件推理/股票推荐或联盟/国家/市场/benchmark/index 调整。
- [ ] 2.12 **Apply Review 门禁**：提交 scoped diff、测试与每层 Review/Write/Query 证据，停止等待主对话人工 Review；批准前不得 Sync、Archive 或 Deliver。
