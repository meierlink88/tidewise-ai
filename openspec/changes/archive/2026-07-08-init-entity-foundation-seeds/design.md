## Context

产品规划中的六层体系为“联盟 → 国家 → 大盘 → 产业 → 企业 → 人物”。当前 PostgreSQL 初始 schema 已具备 `entity_nodes`、`entity_edges` 和 12 类 profile 表，但缺少六层第一层的联盟组织实体，导致 `OPEC+`、`G7`、`WTO`、`IMF` 等跨经济体组织只能被勉强放进政策机构或经济体，语义不准确。

当前数据库表结构已经存在经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物等 profile 表。本 change 不推翻现有 schema，而是在增量 migration 中补充 `alliance_org_profiles`，并建立 repo 内版本化实体 seed 机制，使所有实体表都有一阶段基础数据和可审计初始化路径。

原 `seed-researched-source-catalogs` 的数据源接入、connector 修改和并发采集仍然重要，但优先级后移。实体基础库先落地后，后续数据源采集到的事件、行情、板块和证券数据才能稳定映射到统一实体图谱。

## Goals / Non-Goals

**Goals:**

- 新增 `alliance_org` 实体类型和 `alliance_org_profiles` 表。
- 建立所有实体 profile 表的一阶段基础 seed 数据，覆盖联盟组织、经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物。
- 将 review 已确认的具体初始化范围固化到 `seed-scope.md`，实现时必须以该文档作为 seed 内容和统计口径依据。
- 用 repo 内结构化 seed 文件表达实体节点、profile、实体关系和统计口径。
- 提供实体 seed loader、validator、service 和命令入口，支持幂等 upsert 和可审阅 report。
- 按 TDD 方式先写 migration 静态测试、loader/validator 测试、repository/service 测试，再实现生产代码。
- 保持初始化数据为“基础库”和“系统字典”定位，不写入事件评分、传导强度、预测结论或投资建议。

**Non-Goals:**

- 不在本 change 中实现调研数据源接入、真实 RSS/Eastmoney/SDK connector 修改或多来源并发采集。
- 不初始化全量 A 股、港股、美股公司和证券列表；全量证券和公司应由后续数据源 change 通过 Eastmoney、AKShare、Tushare 等来源动态导入或维护。
- 不引入独立图数据库或向量数据库。
- 不实现事件抽取、Agent 回写、RAG、报告生成或前端页面。
- 不修改 `prototype` 和 `doc` 目录。

## Decisions

### Decision: 联盟组织独立为 `alliance_org`

新增实体类型：

```text
entity_type = alliance_org
layer_code = alliance
profile = alliance_org_profiles
```

`alliance_org` 表达跨国家、跨经济体或多边规则组织，例如 `OPEC+`、`OPEC`、`G7`、`G20`、`WTO`、`IMF`、`World Bank`、`OECD`、`EU`。它和 `policy_body` 的边界是：`policy_body` 隶属于某个经济体或司法辖区，`alliance_org` 跨越多个经济体并影响全球规则、能源、贸易、金融或地缘格局。

备选方案是把联盟组织塞进 `policy_body_profiles`。该方案少一张表，但会混淆“国家政策机构”和“多边组织”，后续六层传导链路无法稳定区分联盟层和国家层。

### Decision: `alliance_org_profiles` 使用轻量字段

一阶段 profile 字段建议为：

```text
entity_id UUID PRIMARY KEY REFERENCES entity_nodes(id)
org_code VARCHAR(64) NOT NULL
org_type VARCHAR(64) NOT NULL
primary_domain VARCHAR(64) NOT NULL DEFAULT ''
scope_region VARCHAR(64) NOT NULL DEFAULT ''
official_url TEXT NOT NULL DEFAULT ''
```

成员国、影响对象和规则关系不放在 profile 字段里，而是通过 `entity_edges` 表达，例如 `member_of`、`influences`、`sets_rules_for`、`coordinates_policy_with`。

备选方案是在 profile 里保存成员列表数组。该方案读取简单，但成员关系会失去图谱边语义，也不利于后续查询“某经济体属于哪些组织”。

### Decision: 所有实体 profile 表都要有一阶段 seed 口径

一阶段 seed 覆盖所有实体类型，但不同类型使用不同粒度：

| 实体类型 | 一阶段 seed 口径 |
| --- | --- |
| `alliance_org` | 10 个核心联盟组织和国际组织 |
| `economy` | 50 个对全球经济或股票市场有重要影响的经济体，主名称使用“中国香港”“中国台湾” |
| `policy_body` | 30 个核心政策机构，国外机构使用中文主名称，英文简称进入 aliases |
| `market` | 32 个股票、交易所、外汇、商品期货、加密资产市场 |
| `index` | 45 个全球和中国市场关键指数 |
| `sector` | 同花顺概念、行业、指数三类板块各 20 个，共 60 个，按初始化热度快照记录 |
| `chain_node` | 具体产业链节点，不使用“上游资源”等大类节点作为唯一节点 |
| `company` | 按每个具体产业链节点维护不少于 10 个代表性上市公司快照，去重后写入唯一公司主体 |
| `security` | 每家公司至少关联 1 条主证券，必要时附带 A/H/ADR 等多证券 |
| `instrument` | 股票、商品期货、外汇、数字资产 4 类交易工具 |
| `metric` | 42 个行情、交易、估值、资金、宏观、商品和产业供需指标 |
| `commodity` | 45 个能源、金属、黑色系、电池材料、农产品和化工品 |
| `person` | 30 个核心政策人物、地缘人物、企业人物和 KOL |

备选方案是只初始化联盟组织、经济体和市场。该方案更小，但无法满足“初始化所有实体表”的目标，也会让 repository 和 seed 能力只覆盖局部表。

### Decision: 实体 seed 使用 repo 内结构化文件，不写进 migration

新增 `backend/data/entity_foundation` 或等价目录保存实体 seed。seed 文件按类型拆分或统一 manifest 均可，但必须表达：

- 稳定 ID 或稳定 key。
- `entity_nodes` 通用字段。
- profile 专属字段。
- 关系边和关系类型。
- 状态、来源说明和初始化阶段。

备选方案是把基础数据写进 SQL migration。该方案首次初始化简单，但后续基础库调整会把数据资产和 schema 变更绑死，也不利于单元测试和环境差异控制。

### Decision: company/security 按产业链节点维护代表性快照

公司和证券不逐一人工确认全量清单，而是按具体产业链节点维护一阶段代表性上市公司快照：每个已确认产业链节点后续通过 `participates_in` 关系关联不少于 10 个代表性上市公司。该快照用于基础字典和推理映射，不表达实时市值排名、投资优先级或动态强弱。每家公司必须能关联至少一个主证券代码。公司主体写入 `company_profiles`，证券代码、交易所、币种和上市状态写入 `security_profiles`。公司与证券通过 `issues` 关系连接，公司与产业链节点通过 `participates_in` 关系连接。同一家公司可关联多个产业链节点，但 company 实体只保留一份。

备选方案是本 change 导入全量股票池。该方案看起来完整，但会提前引入 provider 数据质量、代码映射、退市状态、更新频率和版权边界，超出实体基础库 change 的职责。

### Decision: seed report 作为验收依据

实体 seed 完成后必须输出结构化 report，至少包含：

- `total_entities`: 初始化实体总数。
- `by_entity_type`: 各实体类型数量。
- `by_layer_code`: 六层或扩展层级数量。
- `profile_counts`: 各 profile 表写入数量。
- `edge_counts`: 各关系类型数量。
- `created`、`updated`、`unchanged`、`failed` 统计。

## Risks / Trade-offs

- [Risk] 联盟组织和经济体存在重叠，例如 `EU` 既可理解为联盟组织，也可在市场语境中作为欧洲经济体代理。→ Mitigation：一阶段把 `EU` 建模为 `alliance_org`；如果后续需要欧盟经济体语义，再通过独立实体和关系表达。
- [Risk] 初始化“所有实体表”可能被误解为导入全量真实世界数据。→ Mitigation：明确一阶段是基础库和系统字典，不是全量证券主数据。
- [Risk] seed 数据稳定 ID 设计不当会导致重复实体。→ Mitigation：使用稳定 key、重复校验和幂等 upsert 测试。
- [Risk] profile 表字段不足以表达后续复杂属性。→ Mitigation：本 change 只补联盟组织必要字段，其他扩展字段通过后续增量 migration 处理。
- [Risk] 关系边过多会提前引入推理结论。→ Mitigation：只初始化客观基础关系，不写入利好利空、影响强度、预测结论或投资建议。

## Migration Plan

1. 新增 `alliance_org_profiles` 的增量 migration 和静态测试。
2. 扩展 domain model，加入 `AllianceOrgProfile` 和实体类型校验。
3. 设计实体 seed 文件格式，并先编写 loader/validator 测试。
4. 编写一阶段实体 seed 数据，覆盖所有 profile 表。
5. 编写 repository/service 测试，覆盖幂等 upsert、profile 写入、关系写入和 report 统计。
6. 实现实体 seed repository/service 和命令入口。
7. 运行 `go test ./...`、`openspec validate init-entity-foundation-seeds` 和 `openspec validate --all`。

回滚策略：schema migration 采用增量方式新增 `alliance_org_profiles`，不修改或删除既有业务表。若 seed 数据需要回退，应通过后续 seed 文件把对应实体置为 `inactive` 或修正关系，不依赖清空数据库。

## Open Questions

- company/security 一阶段基准集使用哪些具体公司和证券，需要在实现前确认；默认建议只使用少量跨市场代表样例，避免引入全量股票池。
- `EU` 是否只作为 `alliance_org` 初始化，还是同时作为虚拟经济体初始化，需要在后续经济体清单确认时决定。
