## Why

现有实体基础库只有 33 个全局扁平 `chain_node`，缺少独立产业链身份、链内节点粒度、稳定拓扑和可审计的动态观测边界，无法可靠支撑“全球事件/benchmark 变化如何经产业链传导到中国市场板块”的事件驱动推理。现在需要先建立可复用、可核验且与推理结论分离的产业链事实基础，再由后续 change 接入事件抽取、推理和小程序展示。

## What Changes

- 新增独立 `industry_chain` 主数据类型与 profile，明确产业链的稳定身份、范围、版本和 Review 状态；保留并改进既有 `chain_node`，不建立平行节点体系。
- 新增链内节点成员与稳定拓扑模型，区分链内阶段/角色与节点全局属性，支持上下游、投入、产出、依赖、替代和瓶颈候选等有来源、可审阅的客观关系。
- 扩展实体关系 policy，使产业链及节点能够与 `economy`、`commodity`、`benchmark`、`sector`、`metric` 等实体建立方向明确的客观关系；禁止把推理方向、影响强度、利好利空或预测结论固化为主数据。
- 建立通用 observation governance envelope 与产业链 typed observation contracts，覆盖节点指标和拓扑流量/约束观测，禁止单一万能 EAV 表；首版只定义必要存储、幂等、质量和采集交接契约。
- 使 PostgreSQL 成为产业链、拓扑、跨实体关系与观测事实源；Neo4j 继续只投影 active 主数据和已审阅 active 稳定关系，不投影时序 observation 或未审阅候选。
- 定义事件提取、event-driven reasoning、benchmark/商品/指标、市场板块与小程序展示的消费边界，尤其禁止用海外市场 `COVERS_SECTOR` 直接指向中国板块。
- 将 Serenity 的公开方法论转化为系统设计目标：`market story → system change → required parts → value-chain layers → scarce constraints → evidence → risks/falsification`，同时把稳定事实、动态观测与时点推理结果严格分层。
- 以 AI 算力基础设施、半导体制造、机器人三条链作为首批试点 Review 候选，用于验证共享节点、跨链复用和不同制造链泛化；新能源汽车/储能、创新药/生物制造保留为第二批候选。本 change 的 Propose 阶段不把候选直接写入正式 seed。

## Capabilities

### New Capabilities

- `industry-chain-foundation`: 定义产业链主数据、链内节点成员、稳定拓扑、瓶颈分析输入契约、typed observations、MVP 候选 Review 与下游消费边界。

### Modified Capabilities

- `event-knowledge-schema`: 增加产业链 profile、拓扑和 observation 的 PostgreSQL 增量 schema 事实边界。
- `entity-foundation-seeds`: 将 `industry_chain` 纳入实体基础库，改进 `chain_node` profile，并为首批 seed 增加独立人工 Review 与写入门禁。
- `entity-relationship-curation`: 增加产业链相关客观关系的类型、方向、来源与分层写入规则。
- `neo4j-graph-projection-foundation`: 增加产业链定义、链内稳定拓扑和已审阅跨实体关系的 active-only 投影规则，并明确 observation 不投影。

## Impact

- 预期 Apply 影响 `backend/migrations/`、`backend/internal/domain/`、`backend/internal/apps/entityfoundation/seed/`、`backend/internal/repositories/`、`backend/internal/apps/graphprojection/`、`backend/data/entity_foundation/` 及相应 Go/SQL 测试。
- 首版采集只定义 source catalog/ingestion 到 typed observation writer 的结构化交接契约；具体外部 connector、Agent 推理实现、动态瓶颈结论和市场预测不在本 change 范围。
- 本 change 不修改 `frontend/miniapp/`、`frontend/admin/`、`prototype/` 或项目外 `doc/`；只定义未来 API/小程序展示消费边界，不实现 UI。
- 不引入新数据库、独立 migration 根、万能 EAV、前端密钥或模型编排依赖；不改变 PostgreSQL 事实源与 Neo4j 可重建投影的架构边界。
