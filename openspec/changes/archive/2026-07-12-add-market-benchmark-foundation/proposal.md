## Why

当前系统能表达正式指数，却缺少政府债券收益率、商品价格、现货价格和参考利率等可观测市场基准的独立语义与时间序列存储。事件驱动投研需要先建立 benchmark 定义、市场关系和可审计观测值，才能可靠判断利率、能源、贵金属和数字资产等跨资产变化。

## What Changes

- 新增 `benchmark` 实体类型和 `benchmark_profiles`，明确区分正式指数、通用指标维度、商品标的、交易工具与具体可观测基准。
- 新增 PostgreSQL `benchmark_observations`，保存 benchmark 的时间戳、数值、单位、来源、质量状态和幂等标识；观测记录不作为 Neo4j 节点。
- 新增 `market -> observes_benchmark -> benchmark`、`benchmark -> measures -> metric` 和 `benchmark -> references -> commodity/instrument` 三类客观关系及其方向、来源和端点校验。
- 首批整理并经人工 review 迁移 10 个 benchmark：五个 10 年期政府债券收益率、Brent 与 WTI 价格、黄金现货价格、CME CF Bitcoin 与 Ether 参考利率。
- 修正 `metric:fear_index` 与 `index:vix` 的语义重叠：保留 VIX 为正式指数，将 metric 收敛为通用市场隐含波动率或风险情绪度量维度，不创建第二个 VIX 实体。
- 扩展实体 seed、PostgreSQL repository、report 和 Neo4j 投影，使 benchmark 定义及已审阅关系可从 PG 重建。
- 本 change 不实现外部实时行情 connector、定时采集、历史数据回填、行情 API 或投研推理结论；这些通过后续独立 change 建设。
- 变更仅影响 `tidewise-ai` 后端、OpenSpec 和本地验证说明，不修改 `prototype` 与 `doc`。

## Capabilities

### New Capabilities

- `market-benchmark-foundation`: 定义 benchmark 实体、profile、观测值、与市场/指标/标的的客观关系、首批 seed 和图投影边界。

### Modified Capabilities

- `entity-foundation-seeds`: 将 benchmark 纳入实体基础 seed、profile 校验、report 和人工 review 流程。
- `event-knowledge-schema`: 增加 benchmark profile、观测值表和幂等、来源、质量约束。
- `neo4j-graph-projection-foundation`: 将 benchmark 定义和已审阅关系投影到 Neo4j，同时禁止投影逐时点观测值。

## Impact

- 影响 `backend/internal/domain` 的实体与关系类型。
- 影响 `backend/migrations/`、`backend/internal/repositories` 和 `backend/internal/apps/entityfoundation/seed`。
- 新增 `backend/data/entity_foundation/benchmarks.json` 及 benchmark 关系族 seed 文件。
- 影响 `backend/internal/apps/graphprojection` 的关系映射和实体 profile 投影。
- local PostgreSQL 将新增 benchmark 定义、profile、观测值 schema 和首批已审阅关系；Neo4j 只新增可重建的定义节点与关系。
- 不新增外部依赖，不修改前端或公开 API。
