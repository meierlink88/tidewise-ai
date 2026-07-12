# Benchmark Foundation 审阅清单

## 审阅状态与写入边界

本文记录 tasks 4.1-4.3 的只读审计结果、首批 benchmark 候选定义和关系候选。本文不是正式 seed；task 4.4 获得用户明确确认前，`benchmarks.json` 与三类关系 seed 必须保持空数组，不得执行 PostgreSQL 业务数据迁移或 Neo4j 重建。

审阅日期为 2026-07-12。来源选择优先级为政府财政或央行统计机构、交易所或受监管 benchmark 管理机构。`official_series_code` 只有在权威来源公开稳定代码时填写；页面栏目名、内部别名和推测代码均不得写入该字段。

## 4.1 Metric 只读审计与迁移方案

### Repo 审计

- `backend/data/entity_foundation/metrics.json` 包含唯一 `metric:fear_index` 定义：名称“恐慌指数”，`metric_type=sentiment`、`unit=index`、`frequency=trading_day`。
- `backend/data/entity_foundation/indices.json` 独立包含 `index:vix`，provider 为 Cboe；`relationships/tracks_index.json` 独立包含 `market:us_stock -> tracks_index -> index:vix`。
- 除当前 change artifacts 和 observation repository 测试中的 `index:vix` 示例外，repo 未发现其他 `metric:fear_index`、`fear_index` 或 `implied_volatility` 引用。
- 决策：保留正式指数 `index:vix` 及其市场关系；将通用 metric 改为 `metric:implied_volatility`，名称和 canonical name 均为“隐含波动率”，profile 使用 `metric_type=market_volatility`、`unit=percent`、`frequency=trading_day`。
- `metric:latest_price` 已存在且 profile 为 `market_price / price / realtime`；它不适合作为 LBMA PM 定盘、BRR 或 ETHUSD_RR 的 `measures` 端点。
- `metric:exchange_rate` 已存在且 profile 为 `fx / rate / trading_day`，当前 repo 没有已审阅关系引用；它可以表达 BTCUSD、ETHUSD 的通用兑换比率，具体计价单位与 `daily` 频率继续由各 benchmark profile 保存。
- repo 不存在 `metric:gold_price`。本 change 候选新增该通用 metric，名称和 canonical name 均为“黄金价格”，profile 使用 `metric_type=market_price`、`unit=price`、`frequency=trading_day`。它与现有 `metric:oil_price` 同层表达资产类别价格维度，不复制 LBMA PM 的 provider、币种、每金衡盎司单位或定盘频率。

### Local PostgreSQL 只读审计

审计目标数据库为 `tidewise_local`，仅执行 `SELECT`。结果如下：

| 检查项 | 结果 |
|---|---:|
| `metric:fear_index` active entity | 1 |
| `metric:fear_index` 对应 `metric_profiles` | 1 |
| profile 内容 | `sentiment / index / trading_day` |
| `metric:implied_volatility` entity | 0 |
| `index:vix` active entity | 1 |
| `entity_edges.from_entity_id` 引用旧 metric | 0 |
| `entity_edges.to_entity_id` 引用旧 metric | 0 |
| `event_entity_links.entity_id` 引用旧 metric | 0 |
| 其余 profile 外键引用旧 metric | 0 |
| `benchmark_profiles` 或 `benchmark_observations` 引用旧 metric | 0 |
| `metric:latest_price` active entity/profile | 1，`market_price / price / realtime` |
| `metric:exchange_rate` active entity/profile | 1，`fx / rate / trading_day` |
| `metric:gold_price` entity/profile | 0 |
| 上述三个 metric 的现有 `entity_edges` 引用 | 0 |

审计通过 PostgreSQL catalog 枚举了全部 26 个指向 `entity_nodes` 的外键列，再逐列计数；除旧 metric 自身的 `metric_profiles.entity_id` 1 行外，其余均为 0。

### Task 5.2 精确事务方案

task 4.4 确认后，task 5.2 应在单一事务中按以下顺序执行，不使用模糊名称匹配：

1. 锁定 `entity_key='metric:fear_index'` 的旧实体，断言恰好 1 行、类型为 `metric`，并断言 `metric:implied_volatility` 尚不存在。
2. 再次运行全部外键引用计数。若除 `metric_profiles.entity_id` 外出现任何非零值，事务立即回滚并先补充逐表迁移步骤，不直接删除旧实体。
3. 以 `NormalizeUUID("entity", "metric:implied_volatility")` 对应的确定性 UUID 创建新 entity，字段为 `entity_type=metric`、`layer_code=metric`、名称与 canonical name“隐含波动率”、active；同步把 repo `metrics.json` 的旧定义替换为新定义。
4. 为新 UUID 创建 `metric_profiles`：`market_volatility / percent / trading_day`。
5. 当前审计状态下不存在 edge、event link 或其他 profile 引用，因此不执行无效引用更新；删除旧 UUID 的 `metric_profiles`，再删除旧 `entity_nodes`。
6. 提交前断言：新 entity/profile 各 1 行；旧 entity/profile 各 0 行；`index:vix` 仍为 active 且原 `tracks_index` 关系不变；所有外键无悬空；active entity 总数不变。
7. 任一断言失败则回滚。回滚后 repo seed 仍应保持旧定义，避免 repo/PG 状态分叉。

该方案选择“新 key 对应新确定性 UUID”，而不是原地修改旧 entity key，因为 entity seed 的 UUID 由 entity key 派生；原地保留旧 UUID 会导致后续 seed 为新 key 创建第二个实体。

### `metric:gold_price` Repo/PG 处理

1. task 5.1 的 fixture 先要求 repo `metrics.json` 新增且只新增一个 `metric:gold_price`，profile 固定为 `market_price / price / trading_day`，并验证它不与 `metric:latest_price`、`metric:oil_price` 或 `commodity:gold` 重复。
2. task 5.2 完成 `fear_index -> implied_volatility` 事务后，重新断言 local PG 中 `metric:gold_price` 不存在，防止未审阅数据提前写入。
3. task 5.3 由现有 `entity-seed` 幂等创建 `metric:gold_price` entity/profile，再写入 `LBMA Gold Price PM -> measures -> metric:gold_price`；不单独手写一条绕过 seed repository 的 SQL。
4. PG 验收要求 `metric:gold_price` entity/profile 各 1 行、profile 为 `market_price / price / trading_day`，且不存在 `LBMA Gold Price PM -> metric:latest_price` 关系。
5. benchmark profile 仍是精确事实源：LBMA 保存 `USD / usd_per_troy_ounce / business_day_pm`；通用 metric 不复制这些值，也不声称实时行情。

## 4.2 首批 10 个 Benchmark 审阅清单

| # | 名称 | entity key | benchmark type | official series code | provider | tenor | underlying symbol | currency | unit | frequency | measures metric | 权威来源 |
|---:|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 中国 10 年期国债收益率 | `benchmark:cn_10y_government_bond_yield` | `government_bond_yield` | `null` | `chinabond` | `10Y` | `null` | `CNY` | `percent` | `trading_day` | `metric:government_bond_yield` | 中债信息网“中国国债收益率曲线”，10Yr，CCDC，每个交易日更新；https://yield.chinabond.com.cn/cbweb-pbc-web/pbc/more?locale=en_US |
| 2 | 美国 10 年期国债平价收益率 | `benchmark:us_10y_treasury_par_yield` | `government_bond_yield` | `null` | `us_treasury` | `10Y` | `null` | `USD` | `percent` | `trading_day` | `metric:government_bond_yield` | 美国财政部 Daily Treasury Par Yield Curve Rates，10 Yr；https://home.treasury.gov/resource-center/data-chart-center/interest-rates/TextView?type=daily_treasury_yield_curve |
| 3 | 德国当前 10 年期联邦债券收益率 | `benchmark:de_10y_federal_bond_yield` | `government_bond_yield` | `BBSSY.D.REN.EUR.A630.000000WT1010.A` | `bundesbank` | `10Y` | `null` | `EUR` | `percent` | `trading_day` | `metric:government_bond_yield` | 德国联邦银行 Daily yield of the current 10 year federal bond；https://www.bundesbank.de/en/statistics/overview-of-the-statistical-series/-/3-yields-of-current-federal-securities-914558 |
| 4 | 日本 10 年期国债恒定期限收益率 | `benchmark:jp_10y_jgb_constant_maturity_yield` | `government_bond_yield` | `null` | `japan_mof` | `10Y` | `null` | `JPY` | `percent` | `trading_day` | `metric:government_bond_yield` | 日本财务省 Interest Rate，按 15:00 二级市场价格计算恒定期限利率，次营业日发布；https://www.mof.go.jp/english/policy/jgbs/reference/interest_rate/qa.htm |
| 5 | 英国 10 年期名义平价国债收益率 | `benchmark:uk_10y_gilt_nominal_par_yield` | `government_bond_yield` | `IUDMNPY` | `bank_of_england` | `10Y` | `null` | `GBP` | `percent` | `trading_day` | `metric:government_bond_yield` | 英格兰银行数据库 Yield from British Government Securities, 10 year Nominal Par Yield；https://www.bankofengland.co.uk/boeapps/database/FromShowColumns.asp?CategId=6&FromCategoryList=Yes&HighlightCatValueDisplay=Nominal+par+yield%2C+10+year&NewMeaningId=RNPY10&Travel=NIxAZxI3x |
| 6 | ICE Brent 原油近月期货结算价 | `benchmark:ice_brent_crude_front_month_settlement` | `futures_price` | `B` | `ice_futures_europe` | `null` | `B` | `USD` | `usd_per_barrel` | `trading_day` | `metric:oil_price` | ICE Brent Crude Futures，contract symbol B、settlement price 为美元/桶；https://www.ice.com/products/219/Brent-Crude-Futures |
| 7 | NYMEX WTI 原油近月期货结算价 | `benchmark:nymex_wti_crude_front_month_settlement` | `futures_price` | `CL` | `cme_group` | `null` | `CL` | `USD` | `usd_per_barrel` | `trading_day` | `metric:oil_price` | CME Group NYMEX WTI Light Sweet Crude Oil Futures，product code CL；https://www.cmegroup.com/markets/energy/crude-oil/light-sweet-crude.contractSpecs.html |
| 8 | LBMA 黄金价格 PM | `benchmark:lbma_gold_price_pm` | `spot_price` | `null` | `ice_benchmark_administration` | `null` | `XAUUSD` | `USD` | `usd_per_troy_ounce` | `business_day_pm` | `metric:gold_price`（本 change 新增） | LBMA Gold Price，伦敦交割黄金国际 benchmark，由 IBA 管理，15:00 London 以美元/金衡盎司设定；https://www.lbma.org.uk/prices-and-data/about-lbma-daily-auction-prices |
| 9 | CME CF Bitcoin Reference Rate | `benchmark:cme_cf_bitcoin_reference_rate` | `reference_rate` | `BRR` | `cme_cf` | `null` | `BTCUSD` | `USD` | `usd_per_btc` | `daily` | `metric:exchange_rate` | CME CF BRR，每日 16:00 London 的 1 BTC 美元参考利率；https://www.cmegroup.com/cryptooptionsfaq |
| 10 | CME CF Ether-Dollar Reference Rate | `benchmark:cme_cf_ether_dollar_reference_rate` | `reference_rate` | `ETHUSD_RR` | `cme_cf` | `null` | `ETHUSD` | `USD` | `usd_per_eth` | `daily` | `metric:exchange_rate` | CME CF ETHUSD_RR，每日 16:00 London 的 1 Ether 美元参考利率；https://www.cmegroup.com/cryptooptionsfaq |

### 定义决策依据

- 中国和日本来源页面未公开稳定 series code，保持 `null`；中国页面的“10Yr”和日本页面的“10-year”是期限栏目，不作为代码。
- 美国财政部页面公开“10 Yr”期限和 XML feed，但没有把页面列名或 XML 字段声明为稳定 series code，因此保持 `null`；名称明确为官方平价收益率，避免与单只 on-the-run 债券成交收益率混淆。
- 德国与英国来源公开稳定时间序列代码，按原样保存，不自行缩写。
- Brent 与 WTI 候选明确为近月期货结算价，不笼统命名为“现货油价”；后续 ingestion 必须定义换月规则，本 change 不实现连续合约拼接。
- LBMA Gold Price PM 是受监管的伦敦交割黄金 benchmark，可作为首批黄金现货 benchmark；它是每日 PM 定盘，不代表实时 `XAUUSD` 报价。
- BRR 与 ETHUSD_RR 是每日 reference rate，不是实时指数；BTC/ETH 通过 `underlying_symbol` 区分，继续引用通用 `instrument:digital_asset`。
- `metric:gold_price` 采用与 `metric:oil_price` 相同的资产类别价格维度模式；没有采用 `metric:latest_price`，因为后者的 `realtime` profile 与 LBMA PM 定盘冲突。
- BRR 与 ETHUSD_RR 复用 `metric:exchange_rate`，因为它们都是数字资产对美元的兑换比率；没有新增 `bitcoin_price`、`ether_price` 或 `crypto_reference_rate` metric，避免复制 benchmark 已表达的具体标的与方法。

## 4.3 三类关系审阅清单

所有关系仅表达客观定义，不包含方向、强度、受益承压、预测或投资建议。正式 seed 时每条关系必须复用对应 benchmark 行的权威 `source_name`、`source_url`，并填写实际确认日期 `verified_at`。

### `observes_benchmark`

| from market | to benchmark | 决策依据 |
|---|---|---|
| `market:cn_bond` | `benchmark:cn_10y_government_bond_yield` | 中国债券市场的 10 年期国债收益率观测 |
| `market:us_treasury` | `benchmark:us_10y_treasury_par_yield` | 美国国债市场的 10 年期官方平价收益率观测 |
| `market:euro_area_government_bond` | `benchmark:de_10y_federal_bond_yield` | 复用现有欧元区政府债券市场，德国联邦债为核心区域 benchmark；当前不新增平行德国债券市场实体 |
| `market:jgb` | `benchmark:jp_10y_jgb_constant_maturity_yield` | 日本国债市场的 10 年期恒定期限收益率观测 |
| `market:uk_gilt` | `benchmark:uk_10y_gilt_nominal_par_yield` | 英国 gilt 市场的 10 年期名义平价收益率观测 |
| `market:ice_futures_europe` | `benchmark:ice_brent_crude_front_month_settlement` | ICE Futures Europe 发布 Brent futures 合约与结算价 |
| `market:global_commodity_futures` | `benchmark:nymex_wti_crude_front_month_settlement` | 当前 change 不新增市场实体；现有 `market:cme` 明确是 Chicago Mercantile Exchange，不能替代 NYMEX。后续独立新增 `market:nymex` 后，将该关系迁移到精确端点 |
| `market:global_precious_metals` | `benchmark:lbma_gold_price_pm` | 伦敦交割黄金 benchmark 作为全球贵金属现货市场观测 |
| `market:global_crypto` | `benchmark:cme_cf_bitcoin_reference_rate` | 全球加密资产市场的 BTC 每日参考利率观测 |
| `market:global_crypto` | `benchmark:cme_cf_ether_dollar_reference_rate` | 全球加密资产市场的 ETH 每日参考利率观测 |

### `measures`

| from benchmark | to metric |
|---|---|
| `benchmark:cn_10y_government_bond_yield` | `metric:government_bond_yield` |
| `benchmark:us_10y_treasury_par_yield` | `metric:government_bond_yield` |
| `benchmark:de_10y_federal_bond_yield` | `metric:government_bond_yield` |
| `benchmark:jp_10y_jgb_constant_maturity_yield` | `metric:government_bond_yield` |
| `benchmark:uk_10y_gilt_nominal_par_yield` | `metric:government_bond_yield` |
| `benchmark:ice_brent_crude_front_month_settlement` | `metric:oil_price` |
| `benchmark:nymex_wti_crude_front_month_settlement` | `metric:oil_price` |
| `benchmark:lbma_gold_price_pm` | `metric:gold_price` |
| `benchmark:cme_cf_bitcoin_reference_rate` | `metric:exchange_rate` |
| `benchmark:cme_cf_ether_dollar_reference_rate` | `metric:exchange_rate` |

`metric:gold_price` 和 `metric:exchange_rate` 只表达通用测量维度。benchmark profile 继续作为币种、`usd_per_troy_ounce`、`usd_per_btc`、`usd_per_eth`、`business_day_pm` 与 `daily` 的精确事实源；三条关系均不得指向 `metric:latest_price`。

### `references`

| from benchmark | to commodity/instrument | 决策依据 |
|---|---|---|
| `benchmark:ice_brent_crude_front_month_settlement` | `commodity:brent_crude` | ICE Brent futures 对应 Brent 原油标的 |
| `benchmark:nymex_wti_crude_front_month_settlement` | `commodity:wti_crude` | NYMEX WTI futures 对应 WTI 原油标的 |
| `benchmark:lbma_gold_price_pm` | `commodity:gold` | LBMA Gold Price 对应伦敦交割黄金 |
| `benchmark:cme_cf_bitcoin_reference_rate` | `instrument:digital_asset` | BTC 不新增具体 instrument，使用 `BTCUSD` profile 区分 |
| `benchmark:cme_cf_ether_dollar_reference_rate` | `instrument:digital_asset` | ETH 不新增具体 instrument，使用 `ETHUSD` profile 区分 |

五个主权债收益率本批不创建 `references`：当前 repo 没有政府债券 instrument 类别，`measures -> metric:government_bond_yield` 与 `observes_benchmark` 已足以表达本 change 的定义边界；不得为凑关系数量创建未经 review 的新 instrument。
