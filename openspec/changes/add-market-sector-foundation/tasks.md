## 1. Review 准备与候选清单

- [ ] 1.1 基于 `backend/data/entity_foundation/sectors.json` 整理同花顺概念板块、行业板块、指数板块 Top 20×3 候选 Review 清单，标注 external/source taxonomy、semantic sector 分类、market benchmark 关联判断、事件可映射性、传导差异、稳定性、市场覆盖、数据可获得性和重叠度
- [ ] 1.2 与用户确认三类各 Top 20 都作为 sector 候选进入 Review，总量约 60 个；筛选重点是去重、交叉关系和 benchmark 关联，而不是预先削减 index_sector 配额
- [ ] 1.3 按已确认权重计算候选评估：事件可解释性 25、传导独立性 20、行情敏感度 15、数据完整性 15、长期稳定性 15、市场代表性 10，原则上 70 分以上进入 MVP
- [ ] 1.4 与用户逐项确认同花顺指数板块候选是否保留为 `index_sector`、是否另有关联 benchmark 或正式 index 代码、是否与概念/行业候选合并
- [ ] 1.5 更新 Review 后的 seed 草案，确保 Top 排名只作为来源快照，不作为长期 stable key 或唯一入选依据，最终正式 sector 约 50-60 个
- [ ] 1.6 标注核心约 30、扩展约 20、观察约 10 的运行分层技术落点，明确其属于推理调度或 Review 边界，不属于实体身份字段

## 2. Profile 与 migration 测试先行

- [ ] 2.1 在 `backend/migrations` 增加 migration 静态测试，先验证 `sector_profiles` 增量字段、非破坏性 SQL、外键引用和回滚说明
- [ ] 2.2 在 `backend/internal/apps/entityfoundation/seed` 增加 sector profile loader 测试，覆盖领域分类、来源分类、来源系统、主要市场、主要经济体、中文主名、英文 alias 和 Review 状态，并验证动态评分不会被当成实体身份
- [ ] 2.3 运行新增 migration 与 loader 测试，确认在实现前失败且失败原因指向缺失字段或校验逻辑
- [ ] 2.4 追加 `sector_profiles` 非破坏性 migration，补充分类、来源、主要市场、主要经济体、方法 URL 和 Review 状态字段
- [ ] 2.5 更新 domain、repository 和 seed loader，使新版 sector profile 字段可校验、可写入、可幂等更新，并兼容现有快照字段
- [ ] 2.6 复跑对应 migration 与 loader 测试，确认通过

## 3. 关系策略与图谱投影

- [ ] 3.1 在 `backend/internal/apps/entityfoundation/seed` 增加 relationship policy 测试，覆盖 `covers_sector` 只允许 `market -> sector`、拒绝反向、拒绝推理文案和拒绝悬空端点
- [ ] 3.2 在 `backend/internal/apps/graphprojection` 增加 relation mapping 测试，覆盖 `covers_sector -> COVERS_SECTOR`
- [ ] 3.3 运行新增关系与投影测试，确认在实现前失败
- [ ] 3.4 更新 `relationship_policy.go`，增加 `covers_sector` 客观关系策略；暂不正式允许未审阅的 `sector -> benchmark` 或 `sector -> chain_node` 关系写入 seed
- [ ] 3.5 更新 `graphprojection/mapping.go`，增加 `COVERS_SECTOR` 映射，并保持未知或不安全关系 fallback 行为
- [ ] 3.6 复跑关系与投影测试，确认通过

## 4. Seed 数据与安全校验

- [ ] 4.1 增加 seed fixture 测试，验证首批 reviewed sector 清单数量约 50-60、三类来源候选映射、主要传导簇覆盖、中文主名/英文 alias、来源快照和 Review 状态
- [ ] 4.2 增加 forbidden reasoning 测试，验证 sector profile 和 sector relationship 中出现利好、利空、受益、承压、预测、投资建议等字段或文案时会被拒绝
- [ ] 4.3 运行新增 seed 测试，确认在数据更新前失败
- [ ] 4.4 按用户 Review 结果更新 `backend/data/entity_foundation/sectors.json`，完成候选去重、分类收敛、stable key 策略、来源字段和多来源映射补充
- [ ] 4.5 增加 `backend/data/entity_foundation/relationships/covers_sector.json`，仅写入已 Review 的 `market -> sector` 客观覆盖关系
- [ ] 4.6 复跑 seed 测试，确认通过

## 5. 验证与交付边界

- [ ] 5.1 运行 `go test ./...`，确认后端全部自动化测试通过
- [ ] 5.2 运行 `openspec validate add-market-sector-foundation`，确认 change artifacts 仍有效
- [ ] 5.3 检查 scoped diff，确认未修改 `prototype`、上级 `doc`、`add-ai-event-extraction-pipeline` 或 `add-sdk-source-worker-connectors` 相关内容
- [ ] 5.4 在用户明确批准 Apply 后，按阶段提交实现 commit；未经 Review 不执行 seed 写入、数据库迁移应用或 Neo4j 重建
