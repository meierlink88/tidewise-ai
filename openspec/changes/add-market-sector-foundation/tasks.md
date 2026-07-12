## 1. Review 准备与候选清单

- [ ] 1.1 基于 `backend/data/entity_foundation/sectors.json` 整理同花顺概念板块、行业板块、指数板块 Top 20×3 候选 Review 清单，标注 external/source taxonomy、semantic sector 分类、market benchmark 关联判断、事件可映射性、传导差异、稳定性、市场覆盖、数据可获得性和重叠度
- [ ] 1.2 按已确认权重计算候选评估：事件可解释性 25、传导独立性 20、行情敏感度 15、数据完整性 15、长期稳定性 15、市场代表性 10；每项使用 0-5 基础量表并保存 evidence/source，缺失项按 0 分
- [ ] 1.3 生成完整约 60 个候选 Review 清单，包含 canonical sector key 草案、source mapping identity、评分分项、总分、传导簇覆盖、合并/保留建议、benchmark 关联建议、核心/扩展/观察建议和 override 字段
- [ ] 1.4 暂停并提交候选 Review 清单给用户逐项 Review；批准清单前不得执行 migration、正式 seed、PG 写入或 Neo4j 写入/重建
- [ ] 1.5 按用户 Review 结果确认具体候选、合并/保留、上下位/交叉关系、`tracked_by_benchmark` 关联和人工 override；三类各 Top 20 进入候选池和评分框架不再作为待确认事项
- [ ] 1.6 更新 Review 后的 seed 草案，确保 Top 排名只作为来源快照，不作为长期 stable key 或唯一入选依据，最终正式 sector 约 50-60 个
- [ ] 1.7 标注核心约 30、扩展约 20、观察约 10 的运行分层技术落点，明确其属于推理调度或 Review 边界，不属于实体身份字段

## 2. Profile 与 migration 测试先行

- [ ] 2.1 在 `backend/migrations` 增加 migration 静态测试，先验证 `sector_profiles` 增量字段、`sector_source_mappings` 结构、唯一约束、非破坏性 SQL、外键引用和回滚说明
- [ ] 2.2 在 `backend/internal/apps/entityfoundation/seed` 增加 sector profile 和 source mapping loader 测试，覆盖 semantic classification 不含 `index_sector`、source taxonomy 可为 `index_sector`、主要市场、主要经济体、中文主名、英文 alias 和 Review 状态，并验证动态评分不会被当成实体身份
- [ ] 2.3 运行新增 migration 与 loader 测试，确认在实现前失败且失败原因指向缺失字段或校验逻辑
- [ ] 2.4 追加非破坏性 migration，补充 `sector_profiles` semantic classification、主要市场、主要经济体、方法 URL、Review 状态字段，并新增 `sector_source_mappings` 作为多来源映射事实表
- [ ] 2.5 更新 domain、repository 和 seed loader，使新版 sector profile 与 source mapping 字段可校验、可写入、可幂等更新，并兼容现有快照字段
- [ ] 2.6 复跑对应 migration 与 loader 测试，确认通过

## 3. 关系策略与图谱投影

- [ ] 3.1 在 `backend/internal/apps/entityfoundation/seed` 增加 relationship policy 测试，覆盖 `covers_sector` 只允许 `market -> sector`、`tracked_by_benchmark` 只允许 `sector -> benchmark`、拒绝反向、拒绝复用 `observes_benchmark`、拒绝推理文案和拒绝悬空端点
- [ ] 3.2 在 `backend/internal/apps/graphprojection` 增加 relation mapping 测试，覆盖 `covers_sector -> COVERS_SECTOR` 和 `tracked_by_benchmark -> TRACKED_BY_BENCHMARK`
- [ ] 3.3 运行新增关系与投影测试，确认在实现前失败
- [ ] 3.4 更新 `relationship_policy.go`，增加 `covers_sector` 和 `tracked_by_benchmark` 客观关系策略；暂不正式允许未审阅的 `sector -> chain_node` 关系写入 seed
- [ ] 3.5 更新 `graphprojection/mapping.go`，增加 `COVERS_SECTOR` 和 `TRACKED_BY_BENCHMARK` 映射，并保持未知或不安全关系 fallback 行为
- [ ] 3.6 复跑关系与投影测试，确认通过

## 4. Seed 数据与安全校验

- [ ] 4.1 增加 seed fixture 测试，验证首批 reviewed sector 清单数量约 50-60、canonical key 来源无关、三类来源候选映射、主要传导簇覆盖、中文主名/英文 alias、来源快照和 Review 状态
- [ ] 4.2 增加 forbidden reasoning 测试，验证 sector profile 和 sector relationship 中出现利好、利空、受益、承压、预测、投资建议等字段或文案时会被拒绝
- [ ] 4.3 运行新增 seed 测试，确认在数据更新前失败
- [ ] 4.4 按用户 Review 结果更新 `backend/data/entity_foundation/sectors.json` 和 `backend/data/entity_foundation/sector_source_mappings.json`，完成候选去重、分类收敛、canonical key、来源映射和多来源映射补充
- [ ] 4.5 增加 `backend/data/entity_foundation/relationships/covers_sector.json`，仅写入已 Review 的 `market -> sector` 客观覆盖关系
- [ ] 4.6 增加 `backend/data/entity_foundation/relationships/tracked_by_benchmark.json`，仅写入已 Review 的 `sector -> benchmark` 客观跟踪关系，并保持现有 `observes_benchmark` 只用于 `market -> benchmark`
- [ ] 4.7 复跑 seed 测试，确认通过

## 5. 验证与交付边界

- [ ] 5.1 运行 `go test ./...`，确认后端全部自动化测试通过
- [ ] 5.2 运行 `openspec validate add-market-sector-foundation`，确认 change artifacts 仍有效
- [ ] 5.3 检查 scoped diff，确认未修改 `prototype`、上级 `doc`、`add-ai-event-extraction-pipeline` 或 `add-sdk-source-worker-connectors` 相关内容
- [ ] 5.4 在用户明确批准 Apply 后，按阶段提交实现 commit；未经 Review 不执行 seed 写入、数据库迁移应用或 Neo4j 重建
