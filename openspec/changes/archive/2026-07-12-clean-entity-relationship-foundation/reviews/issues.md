# 公司与证券发行关系审阅清单

> 状态：已暂缓。2026-07-12 经投研优先级复核，本清单仅作为后续候选输入，不视为 review 通过，不在当前 change 写入 seed、PostgreSQL 或 Neo4j。

## 审阅口径

- 核验日期：2026-07-12。
- 关系方向统一为 `company -> security`，关系类型统一为 `issues`。
- 关系只表达公司是该证券的发行主体，不表达证券表现、推荐、实际控制或产业链判断。
- 当前 77 个 security profile 与 77 个 company 实体形成一一对应候选，共 77 条关系。
- `security_profiles.issuer_company_entity_id` 是发行主体主数据字段；`issues` 是同一事实的图谱表达，二者必须完全一致，不允许独立维护出冲突结果。
- 同一公司未来存在多市场、多类别证券时，可以拥有多条 `issues`；当前清单仅覆盖已经确认进入实体主数据的主证券。
- 本清单通过 review 前，不得写入正式关系 seed、PostgreSQL 或 Neo4j。

## 一致性预检

| 检查项 | 结果 |
|---|---:|
| 公司实体 | 77 |
| 证券实体 | 77 |
| 唯一发行主体引用 | 77 |
| 缺失发行主体 | 0 |
| 当前重复发行主体引用 | 0 |
| 候选 `issues` | 77 |

## 官方来源

| 交易所代码 | 来源 | 官方查询入口 |
|---|---|---|
| `AMS` | Euronext | https://live.euronext.com/en/products/equities |
| `ASX` | Australian Securities Exchange | https://www.asx.com.au/markets/trade-our-cash-market/directory |
| `B3` | B3 | https://www.b3.com.br/en_us/market-data-and-indices/data-services/market-data/consultas/equities/ |
| `CSE` | Nasdaq Copenhagen | https://www.nasdaqomxnordic.com/shares/listed-companies/copenhagen |
| `EPA` | Euronext | https://live.euronext.com/en/products/equities |
| `HKEX` | 香港交易所 | https://www.hkex.com.hk/Market-Data/Securities-Prices/Equities |
| `KRX` | 韩国交易所 | https://data.krx.co.kr/ |
| `LSE` | 伦敦证券交易所 | https://www.londonstockexchange.com/stock |
| `NASDAQ` | Nasdaq | https://www.nasdaq.com/market-activity/stocks/screener |
| `NYSE` | 纽约证券交易所 | https://www.nyse.com/listings_directory/stock |
| `SIX` | SIX Swiss Exchange | https://www.six-group.com/en/market-data/shares.html |
| `SNSE` | Santiago Stock Exchange | https://www.bolsadesantiago.com/ |
| `SSE` | 上海证券交易所 | https://www.sse.com.cn/assortment/stock/list/share/ |
| `SZSE` | 深圳证券交易所 | https://www.szse.cn/market/product/stock/list/index.html |
| `TADAWUL` | Saudi Exchange | https://www.saudiexchange.sa/ |
| `TSE` | 日本交易所集团 | https://www.jpx.co.jp/english/listing/co-search/ |
| `TWSE` | 中国台湾证券交易所 | https://isin.twse.com.tw/isin/e_C_public.jsp?strMode=2 |
| `XETRA` | Deutsche Börse | https://live.deutsche-boerse.com/equities/search |

## 77 条候选关系

### AMS（1 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:asml` | `security:asml_primary`（阿斯麦普通股） | `ASML` / `ASML` | `issues` | Euronext，https://live.euronext.com/en/products/equities |

### ASX（1 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:bhp` | `security:bhp_primary`（必和必拓普通股） | `BHP` / `BHP` | `issues` | Australian Securities Exchange，https://www.asx.com.au/markets/trade-our-cash-market/directory |

### B3（1 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:vale` | `security:vale_primary`（淡水河谷普通股） | `VALE3` / `VALE3` | `issues` | B3，https://www.b3.com.br/en_us/market-data-and-indices/data-services/market-data/consultas/equities/ |

### CSE（1 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:maersk` | `security:maersk_primary`（马士基B股） | `MAERSK-B` / `MAERSK-B.CO` | `issues` | Nasdaq Copenhagen，https://www.nasdaqomxnordic.com/shares/listed-companies/copenhagen |

### EPA（2 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:totalenergies` | `security:totalenergies_primary`（道达尔能源普通股） | `TTE` / `TTE` | `issues` | Euronext，https://live.euronext.com/en/products/equities |
| `company:stmicroelectronics` | `security:stmicroelectronics_primary`（意法半导体普通股） | `STMPA` / `STMPA.PA` | `issues` | Euronext，https://live.euronext.com/en/products/equities |

### HKEX（3 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:alibaba` | `security:alibaba_primary`（阿里巴巴港股） | `9988` / `9988.HK` | `issues` | 香港交易所，https://www.hkex.com.hk/Market-Data/Securities-Prices/Equities |
| `company:tencent` | `security:tencent_primary`（腾讯控股港股） | `0700` / `0700.HK` | `issues` | 香港交易所，https://www.hkex.com.hk/Market-Data/Securities-Prices/Equities |
| `company:baidu` | `security:baidu_primary`（百度港股） | `9888` / `9888.HK` | `issues` | 香港交易所，https://www.hkex.com.hk/Market-Data/Securities-Prices/Equities |

### KRX（2 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:lg_energy_solution` | `security:lg_energy_solution_primary`（LG新能源普通股） | `373220` / `373220.KS` | `issues` | 韩国交易所，https://data.krx.co.kr/ |
| `company:samsung_electronics` | `security:samsung_electronics_primary`（三星电子普通股） | `005930` / `005930.KS` | `issues` | 韩国交易所，https://data.krx.co.kr/ |

### LSE（4 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:shell` | `security:shell_primary`（壳牌普通股） | `SHEL` / `SHEL` | `issues` | 伦敦证券交易所，https://www.londonstockexchange.com/stock |
| `company:bp` | `security:bp_primary`（英国石油普通股） | `BP` / `BP` | `issues` | 伦敦证券交易所，https://www.londonstockexchange.com/stock |
| `company:rio_tinto` | `security:rio_tinto_primary`（力拓普通股） | `RIO` / `RIO` | `issues` | 伦敦证券交易所，https://www.londonstockexchange.com/stock |
| `company:glencore` | `security:glencore_primary`（嘉能可普通股） | `GLEN` / `GLEN` | `issues` | 伦敦证券交易所，https://www.londonstockexchange.com/stock |

### NASDAQ（17 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:tesla` | `security:tesla_primary`（特斯拉普通股） | `TSLA` / `TSLA` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:apple` | `security:apple_primary`（苹果普通股） | `AAPL` / `AAPL` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:applied_materials` | `security:applied_materials_primary`（应用材料普通股） | `AMAT` / `AMAT` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:lam_research` | `security:lam_research_primary`（泛林集团普通股） | `LRCX` / `LRCX` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:kla` | `security:kla_primary`（科磊普通股） | `KLAC` / `KLAC` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:synopsys` | `security:synopsys_primary`（新思科技普通股） | `SNPS` / `SNPS` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:cadence` | `security:cadence_primary`（楷登电子普通股） | `CDNS` / `CDNS` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:nvidia` | `security:nvidia_primary`（英伟达普通股） | `NVDA` / `NVDA` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:amd` | `security:amd_primary`（超威半导体普通股） | `AMD` / `AMD` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:intel` | `security:intel_primary`（英特尔普通股） | `INTC` / `INTC` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:onsemi` | `security:onsemi_primary`（安森美普通股） | `ON` / `ON` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:nxp` | `security:nxp_primary`（恩智浦普通股） | `NXPI` / `NXPI` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:microsoft` | `security:microsoft_primary`（微软普通股） | `MSFT` / `MSFT` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:amazon` | `security:amazon_primary`（亚马逊普通股） | `AMZN` / `AMZN` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:alphabet` | `security:alphabet_primary`（谷歌母公司A类股） | `GOOGL` / `GOOGL` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:meta` | `security:meta_primary`（Meta普通股） | `META` / `META` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |
| `company:united_airlines` | `security:united_airlines_primary`（联合航空普通股） | `UAL` / `UAL` | `issues` | Nasdaq，https://www.nasdaq.com/market-activity/stocks/screener |

### NYSE（11 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:exxon_mobil` | `security:exxon_mobil_primary`（埃克森美孚普通股） | `XOM` / `XOM` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:chevron` | `security:chevron_primary`（雪佛龙普通股） | `CVX` / `CVX` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:conocophillips` | `security:conocophillips_primary`（康菲石油普通股） | `COP` / `COP` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:freeport_mcmoran` | `security:freeport_mcmoran_primary`（自由港麦克莫兰普通股） | `FCX` / `FCX` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:albemarle` | `security:albemarle_primary`（雅保普通股） | `ALB` / `ALB` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:general_motors` | `security:general_motors_primary`（通用汽车普通股） | `GM` / `GM` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:ford` | `security:ford_primary`（福特汽车普通股） | `F` / `F` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:oracle` | `security:oracle_primary`（甲骨文普通股） | `ORCL` / `ORCL` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:delta_air_lines` | `security:delta_air_lines_primary`（达美航空普通股） | `DAL` / `DAL` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:fedex` | `security:fedex_primary`（联邦快递普通股） | `FDX` / `FDX` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |
| `company:ups` | `security:ups_primary`（联合包裹普通股） | `UPS` / `UPS` | `issues` | 纽约证券交易所，https://www.nyse.com/listings_directory/stock |

### SIX（1 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:abb` | `security:abb_primary`（ABB普通股） | `ABBN` / `ABBN.SW` | `issues` | SIX Swiss Exchange，https://www.six-group.com/en/market-data/shares.html |

### SNSE（1 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:sqm` | `security:sqm_primary`（智利矿业化工B股） | `SQM-B` / `SQM-B` | `issues` | Santiago Stock Exchange，https://www.bolsadesantiago.com/ |

### SSE（12 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:petrochina` | `security:petrochina_primary`（中国石油A股） | `601857` / `601857.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:sinopec` | `security:sinopec_primary`（中国石化A股） | `600028` / `600028.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:cnooc` | `security:cnooc_primary`（中国海油A股） | `600938` / `600938.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:zijin_mining` | `security:zijin_mining_primary`（紫金矿业A股） | `601899` / `601899.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:cmoc` | `security:cmoc_primary`（洛阳钼业A股） | `603993` / `603993.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:smic` | `security:smic_primary`（中芯国际A股） | `688981` / `688981.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:china_yangtze_power` | `security:china_yangtze_power_primary`（长江电力A股） | `600900` / `600900.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:longi` | `security:longi_primary`（隆基绿能A股） | `601012` / `601012.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:jinkosolar` | `security:jinkosolar_primary`（晶科能源A股） | `688223` / `688223.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:trina_solar` | `security:trina_solar_primary`（天合光能A股） | `688599` / `688599.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:air_china` | `security:air_china_primary`（中国国航A股） | `601111` / `601111.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |
| `company:china_southern_airlines` | `security:china_southern_airlines_primary`（南方航空A股） | `600029` / `600029.SH` | `issues` | 上海证券交易所，https://www.sse.com.cn/assortment/stock/list/share/ |

### SZSE（9 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:tianqi_lithium` | `security:tianqi_lithium_primary`（天齐锂业A股） | `002466` / `002466.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |
| `company:ganfeng_lithium` | `security:ganfeng_lithium_primary`（赣锋锂业A股） | `002460` / `002460.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |
| `company:catl` | `security:catl_primary`（宁德时代A股） | `300750` / `300750.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |
| `company:byd` | `security:byd_primary`（比亚迪A股） | `002594` / `002594.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |
| `company:estun` | `security:estun_primary`（埃斯顿A股） | `002747` / `002747.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |
| `company:inovance` | `security:inovance_primary`（汇川技术A股） | `300124` / `300124.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |
| `company:sungrow` | `security:sungrow_primary`（阳光电源A股） | `300274` / `300274.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |
| `company:goldwind` | `security:goldwind_primary`（金风科技A股） | `002202` / `002202.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |
| `company:sf_holding` | `security:sf_holding_primary`（顺丰控股A股） | `002352` / `002352.SZ` | `issues` | 深圳证券交易所，https://www.szse.cn/market/product/stock/list/index.html |

### TADAWUL（1 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:saudi_aramco` | `security:saudi_aramco_primary`（沙特阿美普通股） | `2222` / `2222` | `issues` | Saudi Exchange，https://www.saudiexchange.sa/ |

### TSE（6 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:panasonic` | `security:panasonic_primary`（松下控股普通股） | `6752` / `6752.T` | `issues` | 日本交易所集团，https://www.jpx.co.jp/english/listing/co-search/ |
| `company:toyota` | `security:toyota_primary`（丰田汽车普通股） | `7203` / `7203.T` | `issues` | 日本交易所集团，https://www.jpx.co.jp/english/listing/co-search/ |
| `company:tokyo_electron` | `security:tokyo_electron_primary`（东京电子普通股） | `8035` / `8035.T` | `issues` | 日本交易所集团，https://www.jpx.co.jp/english/listing/co-search/ |
| `company:fanuc` | `security:fanuc_primary`（发那科普通股） | `6954` / `6954.T` | `issues` | 日本交易所集团，https://www.jpx.co.jp/english/listing/co-search/ |
| `company:yaskawa` | `security:yaskawa_primary`（安川电机普通股） | `6506` / `6506.T` | `issues` | 日本交易所集团，https://www.jpx.co.jp/english/listing/co-search/ |
| `company:keyence` | `security:keyence_primary`（基恩士普通股） | `6861` / `6861.T` | `issues` | 日本交易所集团，https://www.jpx.co.jp/english/listing/co-search/ |

### TWSE（1 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:tsmc` | `security:tsmc_primary`（台积电普通股） | `2330` / `2330.TW` | `issues` | 中国台湾证券交易所，https://isin.twse.com.tw/isin/e_C_public.jsp?strMode=2 |

### XETRA（3 条）

| 公司 | 证券 | 代码 | 关系 | 来源 |
|---|---|---|---|---|
| `company:volkswagen` | `security:volkswagen_primary`（大众汽车优先股） | `VOW3` / `VOW3.DE` | `issues` | Deutsche Börse，https://live.deutsche-boerse.com/equities/search |
| `company:infineon` | `security:infineon_primary`（英飞凌普通股） | `IFX` / `IFX.DE` | `issues` | Deutsche Börse，https://live.deutsche-boerse.com/equities/search |
| `company:siemens` | `security:siemens_primary`（西门子普通股） | `SIE` / `SIE.DE` | `issues` | Deutsche Börse，https://live.deutsche-boerse.com/equities/search |

## 建议结论

- 建议 77 条候选关系全部进入人工 review。
- review 重点是发行主体、证券代码、交易所和证券类别，不重复审核产业归属。
- 通过后先增加自动化测试，要求每条 `issues` 与 security profile 的 `issuer_company_entity_id` 完全一致，再写入 seed。
- PG 验收目标为新增 77 条 active `issues`；Neo4j 重建后的 `ISSUES` 目标同为 77。

## Review 结论记录

- [ ] 77 条发行主体关系确认
- [ ] 允许以交易所官方证券查询入口作为该组关系的权威来源
- [ ] 确认 `issues` 必须与 `issuer_company_entity_id` 保持一致
