# 第一版事件投研市场实体补充审阅清单

## 审阅口径

- 核验日期：2026-07-12。
- 本清单只补充第一版事件推导到市场层所必需的市场实体，不扩展到具体股票或公司。
- `stock_market`、`bond_market` 表达事件分析的抽象市场落点；`commodity_futures_exchange` 表达具体交易场所。
- 新增市场实体必须使用当前已存在的经济体 key，并在实体通过 review 后建立 `economy -> market` 的 `has_market` 关系。
- 现有 `market_type` 已能承担市场类别字段，不新增平行分类字段。
- 本清单通过 review 前，不得修改 `markets.json`、正式关系 seed、PostgreSQL 或 Neo4j。

## 汇总

| 分类 | 候选实体数 | 投研用途 |
|---|---:|---|
| 核心主权债券市场 | 5 | 利率、财政、通胀、货币政策和避险传导 |
| 关键商品及衍生品交易场所 | 5 | 能源、金属、农产品和供应链价格传导 |
| 高事件敏感区域股票市场 | 3 | 石油、关键矿产和制造业转移传导 |
| 合计 | 13 | 第一版跨资产市场最小覆盖 |

## 核心主权债券市场

| Entity key | 规范名称 | `market_type` | 所属经济体 | 币种 | 时区 | 权威来源 |
|---|---|---|---|---|---|---|
| `market:cn_bond` | 中国债券市场 | `bond_market` | `economy:cn`（中国） | CNY | Asia/Shanghai | 中国人民银行金融市场报告，https://www.pbc.gov.cn/en/3688247/3688978/3709134/5419278/2024073114515924526.pdf |
| `market:us_treasury` | 美国国债市场 | `bond_market` | `economy:us`（美国） | USD | America/New_York | TreasuryDirect，https://www.treasurydirect.gov/marketable-securities/ |
| `market:euro_area_government_bond` | 欧元区政府债券市场 | `bond_market` | `economy:eu`（欧盟） | EUR | Europe/Brussels | 欧洲中央银行金融市场与利率统计，https://www.ecb.europa.eu/stats/financial_markets_and_interest_rates/html/index.en.html |
| `market:jgb` | 日本国债市场 | `bond_market` | `economy:jp`（日本） | JPY | Asia/Tokyo | 日本财务省 Japanese Government Bonds，https://www.mof.go.jp/english/policy/jgbs/index.html |
| `market:uk_gilt` | 英国国债市场 | `bond_market` | `economy:gb`（英国） | GBP | Europe/London | UK Debt Management Office，https://www.dmo.gov.uk/responsibilities/gilt-market/ |

### 债券市场建模说明

- 中国第一版使用“中国债券市场”作为抽象落点，覆盖银行间市场中的国债、政策性金融债和信用债行情，不同时建立“中国银行间债券市场”和“中国国债市场”，避免事件推导重复计权。
- 欧元区不存在单一主权发行人，该实体表达欧元区政府债券的聚合分析市场，不替代德国、法国、意大利等成员国债券市场；成员国细分留待后续扩展。
- 美国、日本和英国使用各自国债市场作为利率与避险事件的主要落点。

## 关键商品及衍生品交易场所

| Entity key | 规范名称 | `market_type` | 所属经济体 | 币种 | 时区 | 权威来源 |
|---|---|---|---|---|---|---|
| `market:ine` | 上海国际能源交易中心 | `commodity_futures_exchange` | `economy:cn`（中国） | CNY | Asia/Shanghai | 上海国际能源交易中心官网，https://www.ine.cn/ |
| `market:dce` | 大连商品交易所 | `commodity_futures_exchange` | `economy:cn`（中国） | CNY | Asia/Shanghai | 大连商品交易所官网，https://www.dce.com.cn/ |
| `market:czce` | 郑州商品交易所 | `commodity_futures_exchange` | `economy:cn`（中国） | CNY | Asia/Shanghai | 郑州商品交易所官网，https://www.czce.com.cn/ |
| `market:lme` | 伦敦金属交易所 | `commodity_futures_exchange` | `economy:gb`（英国） | USD | Europe/London | London Metal Exchange，https://www.lme.com/en/ |
| `market:ice_futures_europe` | ICE Futures Europe | `commodity_futures_exchange` | `economy:gb`（英国） | MULTI | Europe/London | Intercontinental Exchange，https://www.ice.com/futures-europe |

### 商品交易场所建模说明

- 现有 `market:ice` 表达跨国交易所集团，继续暂缓；新增 `market:ice_futures_europe` 表达具体英国交易场所，可承接布伦特原油、天然气、碳排放和欧洲利率衍生品。
- LME 合约主要使用美元计价，因此币种使用 USD；ICE Futures Europe 覆盖多币种合约，因此币种使用 MULTI。
- 上期所和 CME 已存在，本清单不重复增加。

## 高事件敏感区域股票市场

| Entity key | 规范名称 | `market_type` | 所属经济体 | 币种 | 时区 | 权威来源 |
|---|---|---|---|---|---|---|
| `market:saudi_stock` | 沙特阿拉伯股票市场 | `stock_market` | `economy:sa`（沙特阿拉伯） | SAR | Asia/Riyadh | Saudi Exchange，https://www.saudiexchange.sa/ |
| `market:indonesia_stock` | 印度尼西亚股票市场 | `stock_market` | `economy:id`（印度尼西亚） | IDR | Asia/Jakarta | Indonesia Stock Exchange，https://www.idx.co.id/en |
| `market:vietnam_stock` | 越南股票市场 | `stock_market` | `economy:vn`（越南） | VND | Asia/Ho_Chi_Minh | Vietnam Exchange，https://vnx.vn/ |

### 区域市场建模说明

- 沙特阿拉伯用于承接 OPEC+、油价、中东财政和地缘事件。
- 印度尼西亚用于承接镍、煤炭、棕榈油和关键矿产政策事件。
- 越南用于承接制造业转移、贸易政策和亚洲供应链事件。
- 第一版只建立抽象股票市场，不同时增加 Saudi Exchange、IDX、HOSE、HNX 等交易场所实体，避免与抽象市场重复计权；具体交易场所在指数和行情接入需要时再补充。

## 建议结论

- 建议13个候选市场实体全部确认。
- 确认后，为每个实体建立一条所属经济体到市场的 `has_market` 关系。
- 现有27条已确认关系与本批13条新增关系合计形成40条第二批正式关系。
- `market:europe_stock`、`market:ice` 和3条全球聚合市场关系继续保持暂缓，不计入40条。

## Review 结论记录

- [x] 5个核心主权债券市场确认
- [x] 5个关键商品及衍生品交易场所确认
- [x] 3个高事件敏感区域股票市场确认
- [x] 13条新增 `has_market` 关系确认

确认日期：2026-07-12。
