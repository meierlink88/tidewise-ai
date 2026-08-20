# Company 国家归属推断

## 结论

本次对 `companies-v1.json` 中的 13,264 家上市公司执行一次性国家归属推断，并把结果写入
`companies-v2.json` 的 `Company.registration_country_id`。业务已确认不要求法定注册地证据；该字段在本批数据中应理解为
"根据上市地、发行人信息、注册法域和公司官方资料推断的企业所属国家/地区"。

推断结果覆盖 13,264/13,264 家公司，无空值：

| 置信度 | 数量 | 主要来源 |
| --- | ---: | --- |
| 高 | 10,593 | A/B 股挂牌地、Nasdaq 发行人 Country、SEC 注册法域 |
| 中 | 708 | HKEX ISIN 前缀、SEC 经营地址、官方公司资料、离岸法域映射 |
| 低 | 1,963 | 港股发行人缺少可用国家线索时归到中国香港 |

## 数据语义与边界

- 仅使用当前 Country 目录中的 201 个国家/地区，不新增 Country。
- 百慕大、开曼群岛、英属维尔京群岛、根西岛和泽西岛不在当前 Country 目录中，
  故按英国海外领地/王室属地映射为 `GB`，且置信度不高于中等。
- 港股离岸控股公司如无可用经营地或发行人 Country，归到 `HK`。这是实用归属推断，
  不是对其法定注册地的声明。
- 本次不改动 Company 模型、Country 模型、API 或数据库 schema，也不建立 Industry 关系。
- 如未来需要可证明的法定注册地，应引入独立的事实来源、生效时间和证据模型，不应直接把
  本批推断值升格为法定事实。

## 推断顺序

1. 存在上交所、深交所或北交所挂牌：归到 `CN`，高置信。
2. Nasdaq 官方 Stock Screener 提供非空 Country：直接映射到 Country，高置信；离岸领地映射为
   当前目录可表达的母国，中置信。
3. 美股发行人 Country 为空：通过 SEC ticker 映射到 CIK，优先使用 SEC `state-of-incorporation`，
   其次使用 `state-location`。SEC 代码依据官方 EDGAR State and Country Codes 解析。
4. 官方结构化字段仍不足时，使用 Nasdaq 公司简介或公司官方 IR/公司页中的总部、法域、
   主要经营地等线索，中置信。
5. 港股优先使用 HKEX 证券列表的 ISIN 前缀；可直接映射到当前 Country 时为中置信。
6. 港股 ISIN 为开曼、百慕大等离岸法域，或缺少 ISIN 且无其他线索：归到 `HK`，低置信。

## 方法统计

| 方法 | 数量 |
| --- | ---: |
| 中国大陆交易所挂牌 | 5,555 |
| Nasdaq Country 直接映射 | 4,877 |
| Nasdaq Country 离岸领地映射 | 75 |
| SEC 注册法域直接映射 | 161 |
| SEC 注册法域离岸映射 | 57 |
| SEC 经营地址映射 | 10 |
| HKEX ISIN 直接映射 | 560 |
| 港股离岸法域归属回退 | 1,932 |
| 港股无 ISIN 归属回退 | 31 |
| Nasdaq 公司简介推断 | 3 |
| 公司官方页推断 | 2 |
| 官方页离岸领地映射 | 1 |

## 主要来源

- [SEC EDGAR Application Programming Interfaces](https://www.sec.gov/search-filings/edgar-application-programming-interfaces)
- [SEC EDGAR State and Country Codes](https://www.sec.gov/submit-filings/filer-support-resources/edgar-state-country-codes)
- [Nasdaq Stock Screener](https://www.nasdaq.com/market-activity/stocks/screener)
- [HKEX Securities Lists](https://www.hkex.com.hk/Services/Trading/Securities/Securities-Lists?sc_lang=en)
- [LPA official company overview](https://ir.lpamericas.com/about-us/overview/default.aspx)
- [Seadrill investor FAQ](https://ir.seadrill.com/resources/investor-faqs/default.aspx)
- [Solaris Resources company page](https://solarisresources.com/company/)
- [Bain Capital GSS Investment Corp. official announcement](https://www.baincapital.com/news/bain-capital-gss-investment-corp-announces-separate-trading-its-ordinary-shares-and-warrants)

## 验证要求

- 发布包中 Company 总数保持 13,264，`registration_country_id` 空值为 0。
- 每个 `registration_country_id` 都必须指向当前 Country 目录中的记录。
- 完整发布必须保持原子性和幂等性；国家引用不合法时整批回滚。
- Company–Industry 关系总数保持 0。
