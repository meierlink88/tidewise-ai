# 联盟组织与国家/经济体关系审阅清单

## 审阅口径

- 核验日期：2026-07-10。
- 本清单只使用当前 `economies.json` 已存在的 50 个实体，不新增或修改实体主数据。
- 所有候选关系方向统一为 `economy -> alliance_org`，关系类型统一为 `member_of`。
- `economy:global` 不表示可加入组织的成员，不生成任何 `member_of` 关系。
- 官网成员不存在对应经济体实体时，只记录覆盖缺口，不生成悬空关系。
- 本清单通过 review 前，不得写入正式关系 seed、PostgreSQL 或 Neo4j。

## 汇总

| 联盟组织 | 目标实体 | 候选关系数 | 备注 |
|---|---|---:|---|
| OPEC+ | `alliance_org:opec_plus` | 9 | 官网完整参与方中有 13 个尚无经济体实体 |
| OPEC | `alliance_org:opec` | 5 | 官网 12 个成员中有 7 个尚无经济体实体 |
| G7 | `alliance_org:g7` | 8 | 7 个国家加欧盟，当前实体完整覆盖 |
| G20 | `alliance_org:g20` | 20 | 覆盖 19 个国家和欧盟；非洲联盟尚无对应实体 |
| WTO | `alliance_org:wto` | 48 | 当前实体中仅伊朗不是 WTO 成员；全球实体排除 |
| IMF | `alliance_org:imf` | 46 | 欧盟、中国香港、中国台湾和全球实体不生成成员关系 |
| World Bank | `alliance_org:world_bank` | 46 | 以 IBRD 成员资格为口径 |
| OECD | `alliance_org:oecd` | 22 | 欧盟属于参与方但不是 38 个成员国之一，不生成成员关系 |
| EU | `alliance_org:eu` | 9 | 仅覆盖当前经济体主数据中已有的 9 个 EU 成员国 |
| BRICS | `alliance_org:brics` | 10 | 官网 11 个成员中埃塞俄比亚尚无经济体实体 |
| 合计 |  | 223 | 不含覆盖缺口 |

## OPEC+

- 目标：`alliance_org:opec_plus`（OPEC+）。
- 来源名称：石油输出国组织官网。
- 来源 URL：https://www.opec.org/pr-detail/585-10-december-2025.html
- 核验依据：OPEC 成员国与 10 个非 OPEC 产油国参与 Declaration of Cooperation。
- 候选关系：
  - `economy:ir`（伊朗）
  - `economy:kw`（科威特）
  - `economy:ng`（尼日利亚）
  - `economy:sa`（沙特阿拉伯）
  - `economy:ae`（阿联酋）
  - `economy:kz`（哈萨克斯坦）
  - `economy:my`（马来西亚）
  - `economy:mx`（墨西哥）
  - `economy:ru`（俄罗斯）
- 覆盖缺口：阿尔及利亚、刚果共和国、赤道几内亚、加蓬、伊拉克、利比亚、委内瑞拉、阿塞拜疆、巴林、文莱、阿曼、苏丹、南苏丹。

## OPEC

- 目标：`alliance_org:opec`（石油输出国组织）。
- 来源名称：石油输出国组织官网。
- 来源 URL：https://www.opec.org/member-countries.html
- 核验依据：官网当前列出 12 个成员国。
- 候选关系：
  - `economy:ir`（伊朗）
  - `economy:kw`（科威特）
  - `economy:ng`（尼日利亚）
  - `economy:sa`（沙特阿拉伯）
  - `economy:ae`（阿联酋）
- 覆盖缺口：阿尔及利亚、刚果共和国、赤道几内亚、加蓬、伊拉克、利比亚、委内瑞拉。

## G7

- 目标：`alliance_org:g7`（七国集团）。
- 来源名称：加拿大 G7 主席国官网。
- 来源 URL：https://g7.canada.ca/en/g7-information/about/
- 核验依据：7 个成员国以及欧盟共同构成 G7。
- 候选关系：
  - `economy:ca`（加拿大）
  - `economy:fr`（法国）
  - `economy:de`（德国）
  - `economy:it`（意大利）
  - `economy:jp`（日本）
  - `economy:gb`（英国）
  - `economy:us`（美国）
  - `economy:eu`（欧盟）
- 覆盖缺口：无。

## G20

- 目标：`alliance_org:g20`（二十国集团）。
- 来源名称：G20 官网。
- 来源 URL：https://g20.org/about-g20
- 核验依据：19 个国家、欧盟和非洲联盟共同构成 G20。
- 候选关系：
  - `economy:ar`（阿根廷）
  - `economy:au`（澳大利亚）
  - `economy:br`（巴西）
  - `economy:ca`（加拿大）
  - `economy:cn`（中国）
  - `economy:fr`（法国）
  - `economy:de`（德国）
  - `economy:in`（印度）
  - `economy:id`（印度尼西亚）
  - `economy:it`（意大利）
  - `economy:jp`（日本）
  - `economy:mx`（墨西哥）
  - `economy:ru`（俄罗斯）
  - `economy:sa`（沙特阿拉伯）
  - `economy:za`（南非）
  - `economy:kr`（韩国）
  - `economy:tr`（土耳其）
  - `economy:gb`（英国）
  - `economy:us`（美国）
  - `economy:eu`（欧盟）
- 覆盖缺口：非洲联盟没有对应经济体或联盟实体，本批不生成悬空关系。

## WTO

- 目标：`alliance_org:wto`（世界贸易组织）。
- 来源名称：世界贸易组织官网。
- 来源 URL：https://www.wto.org/english/thewto_e/countries_e/org6_map_e.htm
- 核验依据：官网 Members and observers 名单；中国香港和中国台湾分别以独立关税区身份加入 WTO，欧盟自身也是 WTO 成员。
- 候选关系：
  - `economy:cn`（中国）、`economy:us`（美国）、`economy:eu`（欧盟）
  - `economy:hk`（中国香港）、`economy:tw`（中国台湾）
  - `economy:jp`（日本）、`economy:kr`（韩国）、`economy:sg`（新加坡）
  - `economy:in`（印度）、`economy:id`（印度尼西亚）、`economy:vn`（越南）
  - `economy:th`（泰国）、`economy:my`（马来西亚）、`economy:ph`（菲律宾）
  - `economy:au`（澳大利亚）、`economy:nz`（新西兰）
  - `economy:gb`（英国）、`economy:de`（德国）、`economy:fr`（法国）、`economy:it`（意大利）
  - `economy:es`（西班牙）、`economy:nl`（荷兰）、`economy:ch`（瑞士）、`economy:se`（瑞典）
  - `economy:no`（挪威）、`economy:dk`（丹麦）、`economy:pl`（波兰）、`economy:tr`（土耳其）
  - `economy:ru`（俄罗斯）、`economy:ca`（加拿大）、`economy:mx`（墨西哥）
  - `economy:br`（巴西）、`economy:ar`（阿根廷）、`economy:cl`（智利）
  - `economy:sa`（沙特阿拉伯）、`economy:ae`（阿联酋）、`economy:qa`（卡塔尔）、`economy:kw`（科威特）
  - `economy:il`（以色列）、`economy:za`（南非）、`economy:eg`（埃及）、`economy:ng`（尼日利亚）
  - `economy:ma`（摩洛哥）、`economy:kz`（哈萨克斯坦）、`economy:pk`（巴基斯坦）
  - `economy:bd`（孟加拉国）、`economy:ua`（乌克兰）、`economy:cz`（捷克）
- 明确排除：`economy:ir`（伊朗）目前不是 WTO 成员；`economy:global` 不是成员主体。

## IMF

- 目标：`alliance_org:imf`（国际货币基金组织）。
- 来源名称：国际货币基金组织官网。
- 来源 URL：https://www.imf.org/external/np/sec/memdir/memdate.htm
- 核验依据：IMF 官方 191 个成员国名单。
- 候选关系：
  - `economy:cn`（中国）、`economy:us`（美国）
  - `economy:jp`（日本）、`economy:kr`（韩国）、`economy:sg`（新加坡）
  - `economy:in`（印度）、`economy:id`（印度尼西亚）、`economy:vn`（越南）
  - `economy:th`（泰国）、`economy:my`（马来西亚）、`economy:ph`（菲律宾）
  - `economy:au`（澳大利亚）、`economy:nz`（新西兰）
  - `economy:gb`（英国）、`economy:de`（德国）、`economy:fr`（法国）、`economy:it`（意大利）
  - `economy:es`（西班牙）、`economy:nl`（荷兰）、`economy:ch`（瑞士）、`economy:se`（瑞典）
  - `economy:no`（挪威）、`economy:dk`（丹麦）、`economy:pl`（波兰）、`economy:tr`（土耳其）
  - `economy:ru`（俄罗斯）、`economy:ca`（加拿大）、`economy:mx`（墨西哥）
  - `economy:br`（巴西）、`economy:ar`（阿根廷）、`economy:cl`（智利）
  - `economy:sa`（沙特阿拉伯）、`economy:ae`（阿联酋）、`economy:qa`（卡塔尔）、`economy:kw`（科威特）
  - `economy:ir`（伊朗）、`economy:il`（以色列）、`economy:za`（南非）
  - `economy:eg`（埃及）、`economy:ng`（尼日利亚）、`economy:ma`（摩洛哥）
  - `economy:kz`（哈萨克斯坦）、`economy:pk`（巴基斯坦）、`economy:bd`（孟加拉国）
  - `economy:ua`（乌克兰）、`economy:cz`（捷克）
- 明确排除：`economy:eu`、`economy:hk`、`economy:tw`、`economy:global` 不作为 IMF 成员实体写入。

## World Bank

- 目标：`alliance_org:world_bank`（世界银行）。
- 来源名称：世界银行官网。
- 来源 URL：https://www.worldbank.org/en/about/leadership/members
- 核验依据：以 International Bank for Reconstruction and Development（IBRD）成员资格为统一口径。
- 候选关系：与本清单 IMF 的 46 个国家实体相同。
- 明确排除：`economy:eu`、`economy:hk`、`economy:tw`、`economy:global` 不作为 IBRD 成员实体写入。

## OECD

- 目标：`alliance_org:oecd`（经济合作与发展组织）。
- 来源名称：经济合作与发展组织官网。
- 来源 URL：https://www.oecd.org/en/about/members-partners.html
- 核验依据：官网当前列出的 38 个 Member countries；欧盟属于参与方，不是成员国。
- 候选关系：
  - `economy:au`（澳大利亚）、`economy:nz`（新西兰）
  - `economy:jp`（日本）、`economy:kr`（韩国）
  - `economy:gb`（英国）、`economy:de`（德国）、`economy:fr`（法国）、`economy:it`（意大利）
  - `economy:es`（西班牙）、`economy:nl`（荷兰）、`economy:ch`（瑞士）、`economy:se`（瑞典）
  - `economy:no`（挪威）、`economy:dk`（丹麦）、`economy:pl`（波兰）、`economy:tr`（土耳其）
  - `economy:ca`（加拿大）、`economy:mx`（墨西哥）、`economy:cl`（智利）
  - `economy:il`（以色列）、`economy:cz`（捷克）、`economy:us`（美国）
- 明确排除：`economy:eu` 仅参与 OECD 工作，不写成 `member_of`。

## EU

- 目标：`alliance_org:eu`（欧洲联盟）。
- 来源名称：欧洲联盟官网。
- 来源 URL：https://european-union.europa.eu/easy-read_en
- 核验依据：官网 27 个 EU member countries 名单，页面更新于 2026-01-01。
- 候选关系：
  - `economy:cz`（捷克）
  - `economy:dk`（丹麦）
  - `economy:de`（德国）
  - `economy:fr`（法国）
  - `economy:it`（意大利）
  - `economy:es`（西班牙）
  - `economy:nl`（荷兰）
  - `economy:pl`（波兰）
  - `economy:se`（瑞典）
- 覆盖缺口：奥地利、比利时、保加利亚、克罗地亚、塞浦路斯、爱沙尼亚、芬兰、希腊、匈牙利、爱尔兰、拉脱维亚、立陶宛、卢森堡、马耳他、葡萄牙、罗马尼亚、斯洛伐克、斯洛文尼亚。

## BRICS

- 目标：`alliance_org:brics`（金砖国家合作机制）。
- 来源名称：BRICS 巴西主席国官网。
- 来源 URL：https://brics.br/en/about-the-brics
- 核验依据：官网当前列出 11 个成员国。
- 候选关系：
  - `economy:br`（巴西）
  - `economy:ru`（俄罗斯）
  - `economy:in`（印度）
  - `economy:cn`（中国）
  - `economy:za`（南非）
  - `economy:sa`（沙特阿拉伯）
  - `economy:eg`（埃及）
  - `economy:ae`（阿联酋）
  - `economy:id`（印度尼西亚）
  - `economy:ir`（伊朗）
- 覆盖缺口：埃塞俄比亚。

## Review 结论记录

- [x] OPEC+ 清单确认
- [x] OPEC 清单确认
- [x] G7 清单确认
- [x] G20 清单确认
- [x] WTO 清单确认
- [x] IMF 清单确认
- [x] World Bank 清单确认
- [x] OECD 清单确认
- [x] EU 清单确认
- [x] BRICS 清单确认
