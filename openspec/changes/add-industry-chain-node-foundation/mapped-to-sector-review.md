# Layer 6 `mapped_to_sector` 候选 Review

## 1. 状态与口径

- 审查日期：2026-07-13。
- 本文件只审查 `backend/data/entity_foundation/review/industry_chain_candidates_v1.json` 中的 12 条 review-only candidate，不修改 fixture、正式 seed 或数据库状态。
- `mapped_to_sector` 只表示“该产业链或节点可稳定映射到中国市场分析板块”，不表示实体身份、法定覆盖、因果影响、行情相关性或事件传导方向。
- 下表的 candidate ID 是本 Review 的稳定审查标识；原 fixture 尚未提供 ID、`source_name`、`source_url` 或 `verified_at`。因此没有任何候选达到“可原样晋级”的直接证据闭合标准。
- sector 自身的 `openspec_review` / 同花顺 legacy source mapping 只证明目标 sector 已存在并经过主数据 Review，不能单独证明 chain/node 到 sector 的映射。

## 2. 实时只读基线

显式 `BEGIN READ ONLY` 查询确认：migration version=14；2 条 chain active+approved；26 个去重 pilot nodes active；27 个 active memberships；24 个 active topology；4 个 active+approved physical constraints；`entity_edges=383`、`sector_source_mappings=89`。12 条候选的 from/to 端点全部 active，现有 `mapped_to_sector` 为 0，同候选端点之间的其他 `entity_edges` 冲突为 0。

目标 sector 均为 `exchange_scope=CN`、`review_status=approved`。现有关系基线只有 `covers_sector=52`、`has_market=40`、`measures=10`、`member_of=223`、`observes_benchmark=10`、`references=5`、`tracks_index=43`；本 Review 不创建海外 market `covers_sector` 中国 sector，也不复用 `covers_sector` 表达分析映射。

## 3. 逐条审查

共同 PG 结论：下列 from/to 均为 active，现有 edge 冲突均为 0。共同 candidate provenance：`source_name/source_url/verified_at` 均未在 fixture 提供；“目标 sector provenance”列只用于定位已有 sector 定义，晋级时必须另补直接映射证据并写入正式 edge provenance。

本次可复核的来源登记如下；它们不是 candidate edge 已有 provenance：

| 来源标识 | source_name | source_url | verified_at / 状态 | 可支持范围 |
|---|---|---|---|---|
| `S15` | Tidewise 市场板块 MVP 候选 Review | `https://github.com/meierlink88/tidewise-ai/blob/03273effecb946ba21c953f6d12165d65b3dee88/openspec/changes/add-market-sector-foundation/candidate-review.md` | PG 未保存 snapshot/verified date；2026-07-13 本次只读复核 | 只证明 canonical sector 与 Review 状态，不直接证明 chain/node mapping |
| `THS legacy` | 同花顺 legacy sector convergence mapping | `https://github.com/meierlink88/tidewise-ai/blob/8abf03c6f9ef91b739a2c4cbd12da28ffc3f75ae/openspec/changes/add-market-sector-foundation/design.md` | PG 未保存 snapshot/verified date；2026-07-13 本次只读复核 | 只证明 legacy 名称被合并，不能替代具体分类页或成分证据 |
| `S5` | 上交所：科创板芯片设计、半导体材料设备主题指数公告 | `https://www.sse.com.cn/market/sseindex/diclosure/c/c_20240715_10759821.shtml` | 2026-07-13 本地已审阅来源登记复核；晋级前仍需访问原页核验 | 可支持两类指数板块定义；具体 node mapping 最好再以编制方案/成分主营交叉核验 |

| Candidate ID | From（类型） | To sector | 分类 | 映射语义审查 | 目标 sector provenance | 审核结论 |
|---|---|---|---|---|---|---|
| `sector-map:ai-chain:computing-infrastructure` | `industry_chain:ai_compute_infrastructure` / AI算力基础设施（industry_chain） | `sector:theme_computing_infrastructure` / 算力基础设施 | `theme_sector` / theme | 名称与边界高度重合，可表达整条试点链对该主题板块的稳定分析映射；不得推导板块涨跌。 | `openspec_review`；市场板块 Review C02；candidate source/date 缺失 | **语义认可但 provenance 需校正**：MVP 优先，补同花顺“算力”稳定分类/成分定义或等价官方主题方法后再晋级。 |
| `sector-map:ai-chain:data-centers-cloud` | `industry_chain:ai_compute_infrastructure` / AI算力基础设施（industry_chain） | `sector:theme_data_centers_cloud` / 数据中心与云 | `theme_sector` / theme | 整条 AI 链还包含芯片、封装、互连、电网等，不能因为数据中心是承载环节就把全链归入更窄的“数据中心与云”。 | `openspec_review`；市场板块 Review C03；candidate source/date 缺失 | **过宽应删除或改写**：删除 chain→sector，保留更精确的 data_center node 候选。 |
| `sector-map:data-center:data-centers-cloud` | `chain_node:data_center` / 数据中心（chain_node） | `sector:theme_data_centers_cloud` / 数据中心与云 | `theme_sector` / theme | 节点与板块核心定义直接重合，是稳定分类映射，不表达云需求或事件影响。 | `openspec_review`；市场板块 Review C03；candidate source/date 缺失 | **语义认可但 provenance 需校正**：MVP 优先，晋级前补稳定板块定义/成分分类直接来源。 |
| `sector-map:power-grid:power-utilities` | `chain_node:power_grid` / 电网（chain_node） | `sector:industry_power_utilities` / 电力与公用事业 | `industry_sector` / industry | 电网基础设施与电力公用事业有关，但目标同时包含发电、公用事业等更宽经济活动；“电网节点属于整个板块”仍需成分或行业分类证据。 | `openspec_review`；市场板块 Review I06；candidate source/date 缺失 | **需补证**：优先核对同花顺/东方财富行业定义及电网运营主体分类；不能仅以 AI 用电传导为证。 |
| `sector-map:rack-power:power-equipment` | `chain_node:rack_power_system` / 机架级供电系统（chain_node） | `sector:industry_power_equipment_batteries` / 电力设备与电池 | `industry_sector` / industry | 机架 UPS、配电等可落入电力设备子集，但 canonical sector 还合并电池，范围较宽；需要稳定产品/成分分类证明主要暴露。 | `openspec_review` + 已合并同花顺“电力设备/电池”source mapping；candidate source/date 缺失 | **需补证**：核对板块定义及代表成分主营，避免用数据中心资本开支故事替代分类证据。 |
| `sector-map:data-center-networking:software-communications` | `chain_node:data_center_networking` / 数据中心网络（chain_node） | `sector:industry_software_communications` / 软件与通信 | `industry_sector` / industry | 目标由软件与通信设备合并而成，明显宽于数据中心网络节点；映射会混入应用软件等无关暴露。 | `openspec_review` + 已合并同花顺“计算机应用/通信设备”source mapping；candidate source/date 缺失 | **过宽应删除或改写**：等待更精确的通信设备/数据中心网络 sector，不以当前宽板块落库。 |
| `sector-map:semi-chain:semiconductors-electronics` | `industry_chain:semiconductor_manufacturing` / 半导体制造（industry_chain） | `sector:industry_semiconductors_electronics` / 半导体与电子 | `industry_sector` / industry | 制造链是该行业骨架的核心子集，适合作为稳定分析映射；不得把电子全行业的行情反向当成制造链事实。 | `openspec_review` + 已合并同花顺“半导体及元件”source mapping；candidate source/date 缺失 | **语义认可但 provenance 需校正**：MVP 优先，补稳定行业定义/成分分类直接来源。 |
| `sector-map:semi-chain:resilience` | `industry_chain:semiconductor_manufacturing` / 半导体制造（industry_chain） | `sector:theme_semiconductor_resilience` / 半导体自主可控 | `theme_sector` / theme | “自主可控”是中国政策/事件暴露主题，不是全球半导体制造链的稳定分类身份；直接映射会把推理结论固化为主数据。 | `openspec_review`；市场板块 Review C09；candidate source/date 缺失 | **过宽应删除或改写**：未来由事件 reasoning 依据国产替代证据生成时点化传导，不进入静态 mapping。 |
| `sector-map:lithography:star-materials-equipment` | `chain_node:lithography_machine` / 光刻机（chain_node） | `sector:industry_star_semiconductor_materials_equipment` / 科创半导体材料设备 | `industry_sector` / industry（来源 taxonomy 为 index_sector） | 光刻设备属于半导体设备子集，语义稳定；需用指数编制方案/成分业务分类直接证明，不把指数表现当证据。 | 上交所科创半导体材料设备主题指数公告 S5；candidate source/date 缺失 | **语义认可但 provenance 需校正**：MVP 优先，正式 edge 应使用 S5 或对应编制方案的直接 URL 与核验日期。 |
| `sector-map:deposition:star-materials-equipment` | `chain_node:deposition_equipment` / 薄膜沉积设备（chain_node） | `sector:industry_star_semiconductor_materials_equipment` / 科创半导体材料设备 | `industry_sector` / industry（来源 taxonomy 为 index_sector） | 沉积设备属于半导体设备子集，语义稳定；仍需方法文件或成分主营分类直证。 | 上交所科创半导体材料设备主题指数公告 S5；candidate source/date 缺失 | **语义认可但 provenance 需校正**：MVP 优先，正式 edge 使用 S5/编制方案并补 verified_at。 |
| `sector-map:etch:star-materials-equipment` | `chain_node:etch_equipment` / 刻蚀设备（chain_node） | `sector:industry_star_semiconductor_materials_equipment` / 科创半导体材料设备 | `industry_sector` / industry（来源 taxonomy 为 index_sector） | 刻蚀设备属于半导体设备子集，语义稳定；仍需方法文件或成分主营分类直证。 | 上交所科创半导体材料设备主题指数公告 S5；candidate source/date 缺失 | **语义认可但 provenance 需校正**：MVP 优先，正式 edge 使用 S5/编制方案并补 verified_at。 |
| `sector-map:eda:star-chip-design` | `chain_node:eda` / EDA软件（chain_node） | `sector:industry_star_chip_design` / 科创芯片设计 | `industry_sector` / industry（来源 taxonomy 为 index_sector） | EDA 是设计工具/服务，不等同芯片设计公司或芯片设计业务；该边表达的是支持关系而非分类归属。 | 上交所科创芯片设计主题指数公告 S5；candidate source/date 缺失 | **过宽应删除或改写**：不落 `mapped_to_sector`；未来如有 EDA/工业软件细分 sector 再审查。 |

## 4. 汇总与 MVP 建议

| 分级 | 数量 | Candidate |
|---|---:|---|
| 直接证据闭合、可原样晋级 | 0 | 无；fixture 缺少全部 edge provenance |
| 语义认可但 provenance 必须校正 | 6 | AI chain→算力基础设施、data_center→数据中心与云、半导体链→半导体与电子、光刻/沉积/刻蚀→科创半导体材料设备 |
| 需补稳定分类证据 | 2 | power_grid→电力与公用事业、rack_power_system→电力设备与电池 |
| 过宽，应删除或改写 | 4 | AI chain→数据中心与云、数据中心网络→软件与通信、半导体链→自主可控、EDA→科创芯片设计 |

MVP 第一批建议只考虑上述 6 条“语义认可”项，并在逐条补齐直接 `source_name/source_url/verified_at`、再次人工批准后进入无状态 seed 准备。power_grid 与 rack power 两条先补成分/行业分类证据；4 条删除/改写项不进入正式 seed。

本结论只能回答“静态 chain/node 与哪个中国 sector 存在稳定分析映射”，不能回答事件影响方向、板块涨跌、因果强度、行情热度或海外 market 对中国板块的法定覆盖。
