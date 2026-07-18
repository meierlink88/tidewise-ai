# Entity Foundation 本地维护说明

## Benchmark 语义边界

- `benchmark` 是稳定、可引用的具体观测定义，包含 provider、官方 series code、币种、单位、频率、期限或标的代码。例如“美国 10 年期国债平价收益率”。
- `index` 只表示有明确编制方法和发布主体的正式指数。收益率、期货结算价、定盘价和参考汇率不得因为“可观察”而建成 index。
- `metric` 是跨实体复用的通用测量维度，例如 `government_bond_yield`、`oil_price`、`gold_price`、`exchange_rate`，不重复 benchmark 的市场、期限、provider、单位或频率事实。
- `benchmark_observations` 是带时间、数值和来源的时序事实，不是实体主数据。fixture 与 entity seed 不得写入或伪造 observation。

## PostgreSQL 与 Neo4j

PostgreSQL 是事实源，保存 `entity_nodes`、`benchmark_profiles`、`entity_edges` 和 `benchmark_observations`。Neo4j 是可重建投影，只保存单一 `Entity` 标签下的稳定实体定义及已审阅关系。benchmark observation 不进入图投影。

Graph projector 必须复用 `entity_nodes` 的 `name`、`canonical_name` 和 `aliases`，并写入唯一的 `projection_namespace=tidewise`。本地重建使用：

```bash
cd backend
APP_ENV=local DATABASE_PASSWORD="$DATABASE_PASSWORD" go run ./cmd/graph-projector rebuild-entities
```

## Review Gate

新增或修改 benchmark 前必须先审阅：实体 key、中文与英文检索名称、benchmark type、provider、官方 series code、币种、单位、频率、期限/标的、权威来源，以及 `observes_benchmark`、`measures`、`references` 的端点和方向。未获用户确认不得写入正式 fixture、PostgreSQL 或 Neo4j。

关系只表达客观定义，不表达行情方向、影响强度、预测或投资建议。缺少精确市场端点时必须记录临时端点和后续迁移计划，不得用语义相近但不等价的实体替代。

## 双语可检索规则

每个 benchmark 必须支持中文和英文检索：

- 中文 `name` 和 `canonical_name` 至少保留一个英文 alias。
- 未来确需使用纯英文 `name` 和 `canonical_name` 时，至少保留一个中文 alias。
- ICE、NYMEX、LBMA、CME 等已包含在主名或完整英文 alias 中的缩写不重复造同义 alias。

该规则由 entity-foundation loader 校验，并由首批 10 个 benchmark fixture 测试逐条验证。

## 后续 Ingestion 边界

真实行情采集应以独立、可审阅的开发任务实现 connector、历史回填、换月规则、质量状态、容量、分区与 retention。采集链路只能把经过来源追踪、去重和结构校验的数据写入 `benchmark_observations`，不得在采集器中隐式创建 benchmark、metric、index 或关系。

同一 benchmark、观测时间和来源使用唯一键幂等更新；不同权威来源可以共存。Neo4j 重建与 ingestion 解耦，新增 observation 不触发 observation 节点或关系写入。
