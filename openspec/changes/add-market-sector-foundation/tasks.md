## 1. Review 准备与候选清单

- [x] 1.1（首次清单验收否决后已返工）不机械沿用旧 `sectors.json`，从事件推理角度重新推荐 concept、industry、index_sector 各 20 个原始候选，并把旧 60 条仅作为迁移对照
- [x] 1.2（返工完成）按已确认权重和六维独立 0/3/5 锚点评估新 60 个候选，逐项保存 evidence/source；未核验行情敏感度按 0 分，不为数量目标虚增评分
- [x] 1.3（返工完成）重构 `candidate-review.md`，同时呈现原始候选 60、建议正式 52、旧 PG 逐项迁移、source/benchmark 建议、重复与覆盖统计；由 Git 保留 Review 历史
- [x] 1.4 用户已批准候选方向与语义候选池；批准不等于行情证据，migration、正式 seed、PG 写入或 Neo4j 写入/重建仍未获授权
- [x] 1.5 按用户 Review 收敛八组 canonical 合并、官方指数 source mapping/`tracked_by_benchmark` 候选和低分组 `MVP覆盖override/待后续行情验证`；原始六维评分保持不变
- [x] 1.6 在 Review artifact 中完成旧 PG 替换对照与建议正式 52 个 canonical sector 的 seed 草案边界；Top/顺序不进入 stable key 或实体身份，尚未修改正式 seed
- [x] 1.7 标注核心30、扩展15、观察7的推理调度建议，明确分层不属于实体身份字段

> **阶段记录：** tasks 1.x Review 收敛完成，主对话已批准进入 task 2；正式 seed、migration apply、PostgreSQL 写入和 Neo4j 仍未获授权。

## 2. Profile 与 migration 测试先行

- [x] 2.1 在 `backend/migrations` 增加 migration 静态测试，先验证 `sector_profiles` 增量字段、`sector_source_mappings` 结构、有代码与无代码稳定唯一约束不含 `snapshot_date`、`source_market_scope` 非空且默认空串、非破坏性 SQL、外键引用和回滚说明
- [x] 2.2 在 `backend/internal/apps/entityfoundation/seed` 增加 sector profile 和 source mapping loader 测试，覆盖 semantic classification 不含 `index_sector`、source taxonomy 可为 `index_sector`、主要市场、主要经济体、中文主名、英文 alias 和 Review 状态，并验证有代码/无代码 identity、名称规范化与最新快照幂等覆盖
- [x] 2.3 已取得 RED：migration 测试因缺少 `000010` 失败；domain/seed 测试因缺少 semantic classification、source mapping 类型、loader 与 upsert 能力失败；引用测试确认错误 market 类型在实现前被接受
- [x] 2.4 追加非破坏性 `000010_add_market_sector_foundation.sql`，先按旧 `sector_type` 确定性回填 semantic classification，再设置 default、NOT NULL 与 check；补充主要市场、主要经济体、方法 URL、Review 状态字段，并新增 `sector_source_mappings` 作为多来源映射事实表
- [x] 2.5 更新 domain、repository、service 和 seed loader，使新版 sector profile 与 source mapping 字段可校验、可写入、可幂等更新；source identity 和非快照审阅字段始终按输入更新，只有不旧于现有记录的输入才覆盖 `rank_snapshot`、`snapshot_date` 和 `source_url`，未新增历史 snapshot 表
- [x] 2.6 复跑 migration 回填与约束静态测试、Memory/PG 快照门禁一致性回归、相关 package tests 和 `go test ./...`，确认通过；未执行 migration apply 或任何 PostgreSQL 写入

> **暂停门：** task 2 已完成并等待主对话验收。进入 task 3 关系策略与图谱投影前必须再次获得批准。

## 3. 关系策略与图谱投影

- [x] 3.1 在 `backend/internal/apps/entityfoundation/seed` 增加 relationship policy 测试，覆盖 `covers_sector` 只允许 `market -> sector`、`tracked_by_benchmark` 只允许 `sector -> benchmark`、拒绝反向、拒绝复用 `observes_benchmark`、拒绝推理文案和拒绝悬空端点
- [x] 3.2 在 `backend/internal/apps/graphprojection` 增加 relation mapping 测试，覆盖 `covers_sector -> COVERS_SECTOR` 和 `tracked_by_benchmark -> TRACKED_BY_BENCHMARK`
- [x] 3.3 运行新增关系与投影测试，确认 seed policy 因不支持 `covers_sector` 失败，projection mapping 因两条新关系均回退为 `RELATED_TO` 失败
- [x] 3.4 更新 `relationship_policy.go`，增加 `covers_sector` 和 `tracked_by_benchmark` 客观关系策略；暂不正式允许未审阅的 `sector -> chain_node` 关系写入 seed
- [x] 3.5 更新 `graphprojection/mapping.go`，增加 `COVERS_SECTOR` 和 `TRACKED_BY_BENCHMARK` 映射，并保持未知或不安全关系 fallback 行为
- [x] 3.6 复跑关系与投影聚焦测试及两个完整 package tests，确认通过

> **暂停门：** task 3 已完成并等待主对话验收。进入 task 4 正式 seed 数据与关系文件前必须再次获得批准。

## 4. Seed 数据与安全校验

- [x] 4.1 增加 seed fixture 测试，验证 52 个 reviewed sector、canonical key 来源无关、三类各 20 个来源候选映射、主要传导簇代表覆盖、中文主名/英文 alias 和 Review 状态
- [x] 4.2 增加 forbidden reasoning 测试，验证 sector profile 和 sector relationship 中出现受益、投资建议等字段或文案时会被拒绝
- [x] 4.3 运行新增 seed 测试，确认旧 sector 数量、缺失 source mapping/关系文件和默认路径在数据更新前失败
- [x] 4.4 按用户 Review 结果更新 `backend/data/entity_foundation/sectors.json` 和 `backend/data/entity_foundation/sector_source_mappings.json`，完成 60 候选到 52 canonical 的去重、分类收敛和八组多来源映射；未核验来源代码均使用无代码 Review identity
- [x] 4.5 增加 `backend/data/entity_foundation/relationships/covers_sector.json`，仅写入 52 条已 Review 的 `market:a_share -> sector` 客观覆盖关系
- [x] 4.6 增加空的 `backend/data/entity_foundation/relationships/tracked_by_benchmark.json` manifest；当前 benchmark seed 不包含已审阅产业指数实体，因此不创建悬空或臆造关系，并保持现有 `observes_benchmark` 只用于 `market -> benchmark`
- [x] 4.7 复跑聚焦 seed 测试和完整 entityfoundation seed package tests，确认通过

> **暂停门：** task 4 已完成并等待主对话验收。进入 task 5 全 change 验证、sync 或 archive 前必须再次获得批准。

## 5. 验证与交付边界

- [x] 5.1 运行 `go test ./... -count=1`，确认后端全部自动化测试通过
- [x] 5.2 运行 `openspec validate add-market-sector-foundation` 并检查 sync/archive 前 artifact 阶段措辞，确认 change artifacts 有效且与已完成实现一致
- [x] 5.3 从 change merge-base 到 HEAD 审计完整 scoped diff、secrets、临时 provenance、stable key、semantic classification、禁止文案、关系端点与数据闭合，确认未修改 `prototype`、上级 `doc`、`add-ai-event-extraction-pipeline` 或 `add-sdk-source-worker-connectors` 相关内容
- [x] 5.4 用户已逐阶段批准本 change 的 Apply 实现与 checkpoint commits；该批准不包含真实 migration apply、seed 写入、PostgreSQL/Neo4j 写入或图谱重建，以上操作仍需独立审批

## 6. Canonical convergence 重新设计与实现

- [x] 6.1 记录 local PostgreSQL 已应用 migration `000010`、仍有 60 个 active legacy sector、mapping/covers 均为 0，正式 entity seed 因预检会产生 112 active sector 而未执行
- [x] 6.2 在 proposal/design/delta specs 中固定 60 项处置矩阵、12 replace/28 merge/20 benchmark-only retire、旧 UUID 保留、引用迁移、显式 CLI、单事务和最终 52 active 边界；`candidate-review.md` 保持不变
- [ ] 6.3 TDD RED：增加 convergence manifest loader/domain 测试，覆盖 60 项完整性、action/target 约束、canonical/legacy key 唯一性、alias/source/reference policy 和禁止推理字段
- [ ] 6.4 TDD RED：增加 Memory/PG service/repository 测试，覆盖普通 seed fail-closed、显式 convergence、单事务 rollback、40 条引用重定向与 legacy mapping、20 条 retirement、未知 FK 阻断和 edge 冲突收敛
- [ ] 6.5 TDD RED：增加 CLI 与 migration 静态测试，覆盖 `-apply-sector-convergence` 显式门禁、`entity_convergences` schema/FK/check/unique 约束和非破坏性 rollback 说明
- [ ] 6.6 实现 `sector_convergences.json`、domain/loader、`entity_convergences` migration、`SectorReferenceRegistry`、Memory clone-on-write、PostgreSQL `sql.Tx`、service report 和 CLI 显式模式；不得修改 `candidate-review.md` 的批准快照
- [ ] 6.7 运行聚焦与 `go test ./... -count=1`，验证普通 seed 在 legacy active 时零写入、显式 convergence 结果为 52 active/60 inactive/60 audit/100 mappings/52 covers/0 tracked，重复执行幂等
- [ ] 6.8 运行 `openspec validate add-market-sector-foundation` 和完整 scoped diff/secret 检查，提交并推送 convergence 实现 checkpoint；未经主对话批准不得写 local PG
- [ ] 6.9 经独立审批后在 local 先 apply 新 migration，再执行显式 convergence；记录前后实体、profile、mapping、edge、audit、引用和状态计数，失败时保留现场且不得手工修库
- [ ] 6.10 local 重复执行显式 convergence 和普通 seed，确认幂等；在 PostgreSQL 验收 52 active canonical、60 inactive legacy、无悬空引用后暂停，等待 Neo4j graph projection 独立审批

> **重新打开门：** 当前 change 因 canonical convergence 缺口不再是完成状态。下一阶段只能在主对话批准本设计后进入 6.3-6.8；本轮不得修改生产代码、seed 或数据库。
