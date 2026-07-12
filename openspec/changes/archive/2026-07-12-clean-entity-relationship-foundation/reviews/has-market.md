# 经济体与市场关系审阅清单

## 审阅口径

- 核验日期：2026-07-10。
- 本清单只审阅 2026-07-10 时 `economies.json` 和 `markets.json` 已存在的实体；第一版投研覆盖所需的新增市场另行形成补充审阅清单。
- 候选关系方向统一为 `economy -> market`，关系类型统一为 `has_market`。
- 候选关系首先从市场 profile 的 `economy_entity_id` 提取，再用监管机构、交易所或权威市场组织官网核验。
- 每个市场实体只允许对应一个经济体实体；存在跨区域或聚合语义歧义的关系单独列为待决项。
- 本清单通过 review 前，不得写入正式关系 seed、PostgreSQL 或 Neo4j。

## 汇总

| 分类 | 候选关系数 | 建议 |
|---|---:|---|
| 中国内地、中国香港、中国台湾 | 9 | 建议确认 |
| 美国 | 5 | 其中 ICE 需要确认跨国运营口径 |
| 日本、印度、韩国、新加坡、澳大利亚 | 8 | 建议确认 |
| 欧盟、英国、德国 | 3 | 其中欧洲股票市场需要确认名称口径 |
| 加拿大、巴西 | 4 | 建议确认 |
| 全球聚合市场 | 3 | 需要确认是否保留聚合关系 |
| 合计 | 32 | 27 条可直接确认，5 条需要决策 |

## 建议直接确认的 27 条关系

| 经济体 | 市场 | 关系方向 | 来源名称 | 来源 URL |
|---|---|---|---|---|
| `economy:cn`（中国） | `market:a_share`（A 股市场） | 中国 -> A 股市场 | 中国证券监督管理委员会官网 | https://www.csrc.gov.cn/ |
| `economy:cn`（中国） | `market:sse`（上海证券交易所） | 中国 -> 上海证券交易所 | 上海证券交易所官网 | https://www.sse.com.cn/ |
| `economy:cn`（中国） | `market:szse`（深圳证券交易所） | 中国 -> 深圳证券交易所 | 深圳证券交易所官网 | https://www.szse.cn/ |
| `economy:cn`（中国） | `market:bse`（北京证券交易所） | 中国 -> 北京证券交易所 | 北京证券交易所官网 | https://www.bse.cn/ |
| `economy:cn`（中国） | `market:shfe`（上海期货交易所） | 中国 -> 上海期货交易所 | 上海期货交易所官网 | https://www.shfe.com.cn/ |
| `economy:hk`（中国香港） | `market:hk_stock`（中国香港股票市场） | 中国香港 -> 中国香港股票市场 | 中国香港交易所官网 | https://www.hkex.com.hk/ |
| `economy:hk`（中国香港） | `market:hkex`（中国香港交易所） | 中国香港 -> 中国香港交易所 | 中国香港交易所官网 | https://www.hkex.com.hk/ |
| `economy:tw`（中国台湾） | `market:tw_stock`（中国台湾股票市场） | 中国台湾 -> 中国台湾股票市场 | 中国台湾证券交易所官网 | https://www.twse.com.tw/zh/ |
| `economy:tw`（中国台湾） | `market:twse`（中国台湾证券交易所） | 中国台湾 -> 中国台湾证券交易所 | 中国台湾证券交易所官网 | https://www.twse.com.tw/zh/ |
| `economy:us`（美国） | `market:us_stock`（美国股票市场） | 美国 -> 美国股票市场 | 美国证券交易委员会官网 | https://www.sec.gov/about/divisions-offices/division-trading-markets |
| `economy:us`（美国） | `market:nyse`（纽约证券交易所） | 美国 -> 纽约证券交易所 | 纽约证券交易所官网 | https://www.nyse.com/ |
| `economy:us`（美国） | `market:nasdaq`（纳斯达克证券交易所） | 美国 -> 纳斯达克证券交易所 | 纳斯达克官网 | https://www.nasdaq.com/ |
| `economy:us`（美国） | `market:cme`（芝加哥商品交易所） | 美国 -> 芝加哥商品交易所 | CME Group 官网 | https://www.cmegroup.com/ |
| `economy:jp`（日本） | `market:jp_stock`（日本股票市场） | 日本 -> 日本股票市场 | 日本交易所集团官网 | https://www.jpx.co.jp/english/ |
| `economy:jp`（日本） | `market:tse`（东京证券交易所） | 日本 -> 东京证券交易所 | 日本交易所集团官网 | https://www.jpx.co.jp/english/ |
| `economy:gb`（英国） | `market:lse`（伦敦证券交易所） | 英国 -> 伦敦证券交易所 | 伦敦证券交易所官网 | https://www.londonstockexchange.com/ |
| `economy:de`（德国） | `market:deutsche_boerse`（德国证券交易所） | 德国 -> 德国证券交易所 | 德意志交易所集团官网 | https://www.deutsche-boerse.com/ |
| `economy:in`（印度） | `market:india_stock`（印度股票市场） | 印度 -> 印度股票市场 | 印度国家证券交易所官网 | https://www.nseindia.com/ |
| `economy:in`（印度） | `market:nse_india`（印度国家证券交易所） | 印度 -> 印度国家证券交易所 | 印度国家证券交易所官网 | https://www.nseindia.com/ |
| `economy:kr`（韩国） | `market:kr_stock`（韩国股票市场） | 韩国 -> 韩国股票市场 | 韩国交易所官网 | https://global.krx.co.kr/ |
| `economy:kr`（韩国） | `market:krx`（韩国交易所） | 韩国 -> 韩国交易所 | 韩国交易所官网 | https://global.krx.co.kr/ |
| `economy:sg`（新加坡） | `market:sgx`（新加坡交易所） | 新加坡 -> 新加坡交易所 | 新加坡交易所官网 | https://www.sgx.com/ |
| `economy:au`（澳大利亚） | `market:asx`（澳大利亚证券交易所） | 澳大利亚 -> 澳大利亚证券交易所 | 澳大利亚证券交易所官网 | https://www.asx.com.au/ |
| `economy:ca`（加拿大） | `market:canada_stock`（加拿大股票市场） | 加拿大 -> 加拿大股票市场 | 多伦多证券交易所官网 | https://www.tsx.com/ |
| `economy:ca`（加拿大） | `market:tsx`（多伦多证券交易所） | 加拿大 -> 多伦多证券交易所 | 多伦多证券交易所官网 | https://www.tsx.com/ |
| `economy:br`（巴西） | `market:brazil_stock`（巴西股票市场） | 巴西 -> 巴西股票市场 | 巴西 B3 交易所官网 | https://www.b3.com.br/en_us/ |
| `economy:br`（巴西） | `market:b3`（巴西证券交易所） | 巴西 -> 巴西证券交易所 | 巴西 B3 交易所官网 | https://www.b3.com.br/en_us/ |

## 需要决策的 5 条关系

### 欧洲股票市场

- 候选关系：`economy:eu -> market:europe_stock`。
- 当前名称：欧洲股票市场。
- 当前 profile：`economy_entity_id=economy:eu`、`currency_code=EUR`。
- 来源：欧洲证券和市场管理局官网，https://www.esma.europa.eu/ 。
- 问题：欧洲股票市场还包含英国、瑞士等非欧盟市场，“欧洲”与“欧盟”语义不完全一致。
- 建议：本批暂不写入；后续将实体名称调整为“欧盟股票市场”后，再建立 `economy:eu -> market:europe_stock`。

### 洲际交易所

- 候选关系：`economy:us -> market:ice`。
- 来源：洲际交易所官网，https://www.ice.com/ 。
- 问题：ICE 是总部位于美国的跨国交易所集团，经营美国、英国和欧洲等多个市场；把集团整体写成美国“拥有的市场”可能产生误导。
- 建议：本批暂不写入；后续按具体交易场所拆分实体，再分别建立经济体与市场关系。

### 三个全球聚合市场

| 候选关系 | 来源名称 | 来源 URL |
|---|---|---|
| `economy:global -> market:global_fx`（全球外汇市场） | 国际清算银行全球外汇统计 | https://www.bis.org/statistics/about_fx_stats.htm |
| `economy:global -> market:global_commodity_futures`（全球商品期货市场） | 世界交易所联合会统计 | https://www.world-exchanges.org/our-work/statistics |
| `economy:global -> market:global_crypto`（全球加密资产市场） | 金融稳定理事会加密资产专题 | https://www.fsb.org/work-of-the-fsb/financial-innovation-and-structural-change/crypto-assets-and-global-stablecoins/ |

- 问题：`economy:global` 是分析用聚合实体，不是真实经济体；三个市场也是跨地域聚合市场。
- Review 结论：不写入这 3 条关系。`economy:global` 是分析聚合实体，不代表真实属地，跨地域市场后续通过分析范围或适用关系表达。

## Review 结论记录

- [x] 27 条直接关系确认
- [x] 欧洲股票市场暂不写入，待名称和覆盖范围调整后重新审阅
- [x] 洲际交易所暂不写入，待拆分具体交易场所后重新审阅
- [x] 3 条全球聚合市场关系不写入 `has_market`

确认日期：2026-07-12。
