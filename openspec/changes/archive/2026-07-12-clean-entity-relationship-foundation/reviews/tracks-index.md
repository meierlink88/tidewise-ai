# 市场与指数关系审阅清单

## 语义复核修订

2026-07-12 在图谱验收前复核确认：`tracks_index` 只表达市场与正式编制指数的关系。原清单中的 5 个政府债券收益率、Brent 与 WTI 连续价格、黄金现货价格、CME CF Bitcoin 与 Ether 参考利率共 10 个概念属于 benchmark，不属于 index。

本修订是该清单的最终有效结论：

- 正式 `index` 和 `tracks_index` 均由 53 调整为 43。
- 上述 10 个概念从当前 seed 和 local 图谱中精确移除，不改变其权威来源审阅结果。
- 10 个概念延后到 `add-market-benchmark-foundation`，以 `benchmark` 和 `observes_benchmark` 重新建模。
- `market:global_equity` 与 `market:global_precious_metals` 保留；市场实体存在不要求当前必须拥有 `tracks_index`。
- 下文 53 条清单保留为原始审阅记录，其中 benchmark 相关条目已被本修订取代。

## 审阅口径

- 核验日期：2026-07-12。
- 关系方向统一为 `market -> index`，关系类型统一为 `tracks_index`。
- 关系只表达指数、价格基准或收益率基准所属的分析市场，不表达涨跌方向或投资判断。
- 现有45个指数中37个沿用当前归属，8个纠正归属或基准定义；另新增8个第一版跨资产基准，合计53条关系。
- 新增 `market:global_equity` 和 `market:global_precious_metals` 只作为跨地域分析市场，不建立 `economy:global -> market` 关系。
- 本清单通过 review 前，不得修改市场、指数、正式关系 seed、PostgreSQL 或 Neo4j。

## 汇总

| 分类 | 关系数 | 建议 |
|---|---:|---|
| 现有归属直接确认 | 37 | 建议确认 |
| 现有归属或基准纠正 | 8 | 建议按新定义确认 |
| 新增核心债券基准 | 5 | 建议确认 |
| 新增区域股票指数 | 3 | 建议确认 |
| 合计 | 53 | 全部确认后写入 |

## 37条现有归属直接确认

| 市场 | 指数 | 来源 |
|---|---|---|
| `market:sse`（上海证券交易所） | `index:sse_composite`（上证指数） | 上海证券交易所，https://www.sse.com.cn/market/sseindex/indexlist/ |
| `market:szse`（深圳证券交易所） | `index:szse_component`（深证成指） | 国证指数，https://www.cnindex.com.cn/ |
| `market:szse`（深圳证券交易所） | `index:chinext`（创业板指） | 国证指数，https://www.cnindex.com.cn/ |
| `market:sse`（上海证券交易所） | `index:star50`（科创50） | 上海证券交易所，https://www.sse.com.cn/market/sseindex/indexlist/ |
| `market:a_share`（A股市场） | `index:csi300`（沪深300） | 中证指数，https://www.csindex.com.cn/ |
| `market:a_share`（A股市场） | `index:csi500`（中证500） | 中证指数，https://www.csindex.com.cn/ |
| `market:a_share`（A股市场） | `index:csi1000`（中证1000） | 中证指数，https://www.csindex.com.cn/ |
| `market:bse`（北京证券交易所） | `index:bse50`（北证50） | 北京证券交易所，https://www.bse.cn/ |
| `market:hkex`（中国香港交易所） | `index:hsi`（恒生指数） | 恒生指数公司，https://www.hsi.com.hk/ |
| `market:hkex`（中国香港交易所） | `index:hstech`（恒生科技指数） | 恒生指数公司，https://www.hsi.com.hk/ |
| `market:hkex`（中国香港交易所） | `index:hscei`（恒生中国企业指数） | 恒生指数公司，https://www.hsi.com.hk/ |
| `market:us_stock`（美国股票市场） | `index:sp500`（标普500） | S&P Dow Jones Indices，https://www.spglobal.com/spdji/ |
| `market:nasdaq`（纳斯达克证券交易所） | `index:ixic`（纳斯达克综合指数） | Nasdaq，https://www.nasdaq.com/market-activity/index/comp |
| `market:nasdaq`（纳斯达克证券交易所） | `index:ndx`（纳斯达克100指数） | Nasdaq，https://indexes.nasdaq.com/ |
| `market:us_stock`（美国股票市场） | `index:rut`（罗素2000） | FTSE Russell，https://www.lseg.com/en/ftse-russell/indices/russell-us |
| `market:nasdaq`（纳斯达克证券交易所） | `index:sox`（费城半导体指数） | Nasdaq，https://indexes.nasdaq.com/ |
| `market:twse`（中国台湾证券交易所） | `index:twii`（中国台湾加权指数） | 中国台湾证券交易所，https://www.twse.com.tw/zh/indices/taiex/mi-5min-hist.html |
| `market:tse`（东京证券交易所） | `index:nikkei225`（日经225） | Nikkei Indexes，https://indexes.nikkei.co.jp/en/nkave |
| `market:tse`（东京证券交易所） | `index:topix`（东证指数） | JPX，https://www.jpx.co.jp/english/markets/indices/topix/ |
| `market:europe_stock`（欧洲股票市场） | `index:sx5e`（欧洲斯托克50） | STOXX，https://www.stoxx.com/index-details?symbol=SX5E |
| `market:deutsche_boerse`（德国证券交易所） | `index:dax`（德国DAX） | Deutsche Börse，https://www.deutsche-boerse.com/ |
| `market:europe_stock`（欧洲股票市场） | `index:cac40`（法国CAC40） | Euronext，https://live.euronext.com/en/product/indices/FR0003500008-XPAR |
| `market:lse`（伦敦证券交易所） | `index:ftse100`（英国富时100） | FTSE Russell，https://www.lseg.com/en/ftse-russell/indices/uk |
| `market:nse_india`（印度国家证券交易所） | `index:nifty50`（印度Nifty 50） | NSE Indices，https://www.niftyindices.com/ |
| `market:india_stock`（印度股票市场） | `index:sensex`（印度Sensex） | BSE India，https://www.bseindia.com/sensex/ |
| `market:krx`（韩国交易所） | `index:kospi`（韩国KOSPI） | Korea Exchange，https://global.krx.co.kr/ |
| `market:krx`（韩国交易所） | `index:kosdaq`（韩国KOSDAQ） | Korea Exchange，https://global.krx.co.kr/ |
| `market:global_fx`（全球外汇市场） | `index:dxy`（美元指数） | ICE，https://www.ice.com/products/194/US-Dollar-Index-Futures |
| `market:us_stock`（美国股票市场） | `index:vix`（VIX恐慌指数） | Cboe，https://www.cboe.com/tradable_products/vix/ |
| `market:global_commodity_futures`（全球商品期货市场） | `index:crb`（CRB商品指数） | LSEG，https://www.lseg.com/ |
| `market:cme`（芝加哥商品交易所） | `index:wti_continuous`（WTI原油连续指数） | CME Group，https://www.cmegroup.com/markets/energy/crude-oil/light-sweet-crude.html |
| `market:a_share`（A股市场） | `index:wind_all_a`（万得全A） | Wind指数体系，https://www.wind.com.cn/ |
| `market:a_share`（A股市场） | `index:csi_all`（中证全指） | 中证指数，https://www.csindex.com.cn/ |
| `market:a_share`（A股市场） | `index:cni2000`（国证2000） | 国证指数，https://www.cnindex.com.cn/ |
| `market:a_share`（A股市场） | `index:csi_dividend`（中证红利） | 中证指数，https://www.csindex.com.cn/ |
| `market:a_share`（A股市场） | `index:csi_a500`（中证A500） | 中证指数，https://www.csindex.com.cn/ |
| `market:global_fx`（全球外汇市场） | `index:rmb_index`（人民币汇率指数） | 中国外汇交易中心，https://www.chinamoney.com.cn/ |

## 8条现有归属或基准纠正

| 指数 | 当前错误归属 | 建议新归属 | 来源与理由 |
|---|---|---|---|
| `index:dji`（道琼斯工业平均指数） | `market:nyse` | `market:us_stock`（美国股票市场） | 成分股跨交易所，S&P Dow Jones Indices，https://www.spglobal.com/spdji/ |
| `index:msci_world`（MSCI全球指数） | `market:global_fx` | `market:global_equity`（全球股票市场，新建） | MSCI全球股票指数体系，https://www.msci.com/indexes |
| `index:msci_em`（MSCI新兴市场指数） | `market:global_fx` | `market:global_equity`（全球股票市场，新建） | MSCI新兴市场股票指数体系，https://www.msci.com/indexes |
| `index:brent_continuous`（布伦特原油连续指数） | `market:ice` | `market:ice_futures_europe` | ICE Futures Europe承载布伦特原油期货，https://www.ice.com/futures-europe |
| `index:xau_spot`（伦敦金现货指数） | `market:global_commodity_futures` | `market:global_precious_metals`（全球贵金属现货市场，新建） | LBMA贵金属价格体系，https://www.lbma.org.uk/prices-and-data |
| `index:ccdc_bond`（中债综合指数） | `market:a_share` | `market:cn_bond`（中国债券市场） | 中债指数，https://yield.chinabond.com.cn/ |
| `index:btc_price`（比特币价格指数） | provider=`crypto_market` | 保持 `market:global_crypto`，改为CME CF Bitcoin Reference Rate | CME Group加密资产基准，https://www.cmegroup.com/market-data/cme-group-benchmark-administration/cryptocurrency-benchmarks.html |
| `index:eth_price`（以太坊价格指数） | provider=`crypto_market` | 保持 `market:global_crypto`，改为CME CF Ether-Dollar Reference Rate | CME Group加密资产基准，https://www.cmegroup.com/market-data/cme-group-benchmark-administration/cryptocurrency-benchmarks.html |

## 2个新增全球分析市场

| Entity key | 名称 | `market_type` | 经济体profile | 币种 | 时区 |
|---|---|---|---|---|---|
| `market:global_equity` | 全球股票市场 | `stock_market` | `economy:global` | MULTI | UTC |
| `market:global_precious_metals` | 全球贵金属现货市场 | `commodity_spot_market` | `economy:global` | USD | UTC |

这两个实体允许在profile中引用 `economy:global` 作为分析范围，但不得生成 `has_market` 关系。

## 5个新增核心债券基准

| Market | 新增Index key与名称 | `index_code` | 来源 |
|---|---|---|---|
| `market:cn_bond` | `index:cn_10y_government_bond_yield`（中国10年期国债收益率） | `CN_10Y_GOV_YIELD` | 中债收益率曲线，https://yield.chinabond.com.cn/ |
| `market:us_treasury` | `index:us_10y_treasury_yield`（美国10年期国债收益率） | `US_10Y_TREASURY_YIELD` | 美国财政部Daily Treasury Rates，https://home.treasury.gov/resource-center/data-chart-center/interest-rates/TextView?type=daily_treasury_yield_curve |
| `market:euro_area_government_bond` | `index:euro_area_10y_government_bond_yield`（欧元区10年期政府债券收益率） | `YC.B.U2.EUR.4F.G_N_C.SV_C_YM.SR_10Y` | 欧洲央行Data Portal，https://data.ecb.europa.eu/data/datasets/YC/YC.B.U2.EUR.4F.G_N_C.SV_C_YM.SR_10Y |
| `market:jgb` | `index:jgb_10y_yield`（日本10年期国债收益率） | `JGB_10Y_YIELD` | 日本财务省JGB，https://www.mof.go.jp/english/policy/jgbs/index.html |
| `market:uk_gilt` | `index:uk_10y_gilt_yield`（英国10年期国债收益率） | `UK_10Y_GILT_YIELD` | UK DMO Gilt Market，https://www.dmo.gov.uk/data/gilt-market/ |

## 3个新增区域股票指数

| Market | 新增Index key与名称 | `index_code` | 来源 |
|---|---|---|---|
| `market:saudi_stock` | `index:tasi`（沙特全指） | `TASI` | Saudi Exchange，https://www.saudiexchange.sa/ |
| `market:indonesia_stock` | `index:jakarta_composite`（雅加达综合指数） | `JCI` | Indonesia Stock Exchange，https://www.idx.co.id/en |
| `market:vietnam_stock` | `index:vn_index`（越南VN-Index） | `VNINDEX` | Vietnam Exchange，https://vnx.vn/ |

## 建议结论

- 建议37条现有正确归属全部确认。
- 建议8条错误归属或泛化基准按新定义确认。
- 建议2个全球分析市场、5个债券基准和3个区域股票指数全部确认。
- 确认后正式写入53条 `tracks_index`；40条 `has_market` 保持不变。

## Review结论记录

- [x] 37条现有正确归属确认
- [x] 8条归属或基准纠正确认
- [x] 2个全球分析市场确认
- [x] 5个核心债券基准确认
- [x] 3个区域股票指数确认
- [x] 53条 `tracks_index` 关系确认

确认日期：2026-07-12。
