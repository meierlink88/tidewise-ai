## Why

当前事件知识 schema 已经具备统一实体节点和多类 profile 表，但产品六层体系的第一层是“联盟组织”，当前 ER 只覆盖经济体、政策机构、市场、产业、企业和人物等实体，缺少 `OPEC+`、`G7`、`WTO` 这类跨经济体组织的独立表达。同时，实体表虽然已经创建，但还没有 repo 内版本化的一阶段基础实体初始化机制，后续事件抽取、图谱投影和传导分析缺少稳定基础字典。

本 change 作为当前第一优先级，先完成实体 schema 补齐和实体基础库初始化；原计划中的调研数据源接入、真实 connector 修改和多来源并发采集迁移到第二优先级 change `seed-researched-source-catalogs` 中执行。

## What Changes

- 新增 `alliance_org` 实体类型和 `alliance_org_profiles` 表，用于表达 `OPEC+`、`OPEC`、`G7`、`G20`、`WTO`、`IMF`、`World Bank`、`OECD`、`EU` 等联盟组织、国际组织或跨经济体协调机制。
- 调整事件知识实体模型，使当前基础实体类型覆盖：联盟组织、经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物。
- 新增 repo 内版本化 entity seed 数据，初始化所有实体 profile 表的一阶段基础数据。
- 新增实体 seed loader、validator、repository/service 和命令入口，使 local、uat、prod 可以用同一套 seed 资产幂等初始化实体基础库。
- 初始化一阶段实体关系，例如联盟组织成员关系、经济体与市场关系、市场与指数关系、公司与证券关系、人物与机构关系、指标与适用对象关系。
- 明确本 change 不再实现调研数据源接入、source catalog 大批量 seed、真实 RSS/Eastmoney/SDK connector 修改或并发采集；这些内容保留到第二优先级 change。

## Capabilities

### New Capabilities
- `entity-foundation-seeds`: 定义实体基础库的一阶段版本化初始化能力，覆盖所有实体 profile 表、基础关系、幂等 seed、统计报告和验证边界。

### Modified Capabilities
- `event-knowledge-schema`: 补充联盟组织实体类型和 profile 表，并更新事件知识 schema 对实体类型覆盖范围的要求。
- `persistence-and-contracts`: 增加实体基础库初始化必须通过 repo 内版本化 seed 和统一命令执行的持久化要求。

## Impact

- 影响 `backend/migrations/`：需要新增增量 SQL migration，创建 `alliance_org_profiles` 表和必要索引；不得重写已执行的初始 migration。
- 影响 `backend/internal/domain`：需要新增联盟组织 profile 类型，并补充实体类型枚举或校验边界。
- 影响 `backend/internal/repositories`：需要支持实体节点、profile、关系的幂等写入和基础统计。
- 影响 `backend/internal/entityseed` 或等价模块：需要新增 seed loader、validator、service、fixture 和测试。
- 影响 `backend/cmd`：需要新增或扩展实体 seed 命令，用于把 repo 内实体基础库写入目标环境 PostgreSQL。
- 影响 OpenSpec active changes：`init-entity-foundation-seeds` 为第一优先级；`seed-researched-source-catalogs` 保留为第二优先级，后续再处理数据源和 connector。
- 不修改 `prototype` 和 `doc` 目录，不把外部设计文档复制到工程内，不提交任何密钥、token、cookie 或生产连接信息。
