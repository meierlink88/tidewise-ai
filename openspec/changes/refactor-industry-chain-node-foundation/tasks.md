## 1. Phase A：受控 cleanup 与全新 chain_node 初始化

- [x] 1.1 记录已完成人工字段 Review：`chain_node_profiles(entity_id, definition NOT NULL, boundary_note NULL)`，`theme_profiles(entity_id, definition NOT NULL, boundary_note NOT NULL)`；名称/aliases/状态复用 `entity_nodes`，不保留 position/category/unit/level/parent/market/source/observation，不建立 source mapping，不使用 `research_theme`。
- [x] 1.2 记录原 Proposal Review 曾获批准，但 2026-07-13 Material Proposal Change 已覆盖“保留旧 facts、复用旧 UUID/key、legacy→target 收敛”的策略；当前 Apply 已暂停，未执行任何 PostgreSQL/Neo4j Write。
- [ ] 1.3 **Material Proposal Review 门禁**：提交更新后的 proposal/design/delta specs/tasks、preflight/候选证据与当前代码差异审计；主对话重新批准前不得恢复 Apply。
- [ ] 1.4 测试先行：新增 cleanup migration/受控命令测试，覆盖旧目标集合快照、FK 与逻辑引用扫描、精确删除谓词、从叶到根顺序、非目标保护、可恢复备份门禁、事务 rollback、forward-fix、dry-run/report 和重复执行 already-clean/unchanged。
- [ ] 1.5 实现 Phase A cleanup 与目标 schema 代码：清除旧 sector、industry_chain、chain_node、profiles、source mappings、membership、topology、physical constraints、相关 `entity_edges`、`event_entity_links` 和仅服务 sector 的 convergence/audit；删除专属表并建立最小 chain_node/theme schema。只提交代码与测试，不 apply migration、不写 PostgreSQL。
- [ ] 1.6 运行完整只读 preflight：输出每个旧表/类型/引用的精确 counts、FK/trigger/function、逻辑引用、orphans、非目标基线与 `entity_key` 状态；当前报告遗漏 event links 与完整 audit 依赖，补齐前不得进入 cleanup Review。
- [ ] 1.7 准备并验证可恢复备份方案：明确备份文件/快照、校验和、恢复命令、恢复演练结果、保留位置与责任边界。仅 `archive_mode` 或文件存在不构成备份验证。
- [ ] 1.8 更新候选证据：以 `产业链节点候选-第一轮语义过滤.xlsx` 为输入记录 1191 原始、955 初步保留、202 明确排除、34 待复核；明确该工作簿不是 final seed，主对话完成待复核、同义归并、definition/boundary、粒度及层级关系 Review 前不得生成可执行节点清单。
- [ ] 1.9 **Cleanup Review 门禁**：提交版本化 cleanup diff/dry-run、冻结目标 ID 集合或可重算谓词、每表预计删除 counts、引用顺序、锁/事务影响、非目标保护、备份与 forward-fix；单独请求 cleanup Write 授权。
- [ ] 1.10 **Cleanup Write -> Query 门禁**：仅在 1.9 明确获批后执行 cleanup Write；立即 Query 证明旧专属表不存在、旧 sector/industry_chain/chain_node 与相关关系/链接/审计为 0、无孤儿、非目标 counts/校验和不变、重复执行幂等，等待 cleanup Query 验收。
- [ ] 1.11 **最终节点清单 Review 门禁**：仅在 cleanup Query 验收后，提交主对话最终批准的 chain_node 清单；逐项包含唯一新 key/UUID、definition、boundary、aliases 与同义归并结果。不得提出具体 theme 实例，不得复用旧 ID/key。
- [ ] 1.12 测试先行并实现 final seed/dry-run/report：仅接受最终批准节点，拒绝第一轮工作簿直接入库；覆盖 deterministic 新 UUID、唯一 key、最小 profile、created/updated/unchanged/failed 与幂等。只提交代码，不写 PostgreSQL。
- [ ] 1.13 **Final seed Review -> Write -> Query 门禁**：提交最终 seed diff、dry-run、影响 counts 与备份边界，单独请求 seed Write 授权；获批后写入并立即 Query counts、definition/boundary、key/ID、重复、孤儿及幂等，等待 seed Query 验收。
- [ ] 1.14 **Phase A 验收门禁**：提交 cleanup 与 final seed 两套 Write/Query 证据并等待主对话明确验收；未验收不得进入 Phase B。PG cleanup 后 Neo4j 将暂时陈旧，本 change 不清理、不写入、不 rebuild。

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
