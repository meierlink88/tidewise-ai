# Approved Alliance Manifest R0 v1

## 状态与输入

- 人工批准记录：主对话于 2026-07-14 明确批准 Package 1.2。
- 输入：`联盟组织列表1.0.xlsx`，SHA-256 `ac0d953c0cd93596fe6bf8a70541bbe658620e75d38a9b3178980071b2cdc102`，`联盟组织!A1:K51`。
- canonical tuple checksum：`4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a`。canonical tuple 顺序为 `sheet_row, name, normalized_abbreviation, leadership_summary, influence_scope_summary, entity_key, action, decision`，UTF-8、TAB 分隔、LF 结尾，按 sheet row 排序。
- 45 条全部 `approve`：9 `keep`、36 `create`；`UJR`、`CCAS` 只删除源缩写末尾 U+200C。
- 未来 UUID：9 个 keep 复用执行时稳定 UUID；36 个 create 只冻结下表 stable key，UUID 留待未来确定性 manifest 实现生成。
- 本文件是 Package 2 的 R0 输入，不是 seed，不授权源码、migration 或任何数据库写入。

## 45 条批准 Active Alliance Target

| Sheet row | name/canonical_name | abbreviation | leadership_summary | influence_scope_summary | entity key | action | decision |
|---:|---|---|---|---|---|---|---|
| 3 | 二十国集团 | `G20` | 轮值主席国制（美 / 中 / 德 / 日等大国轮流） | 全球宏观经济政策协调首要平台，金融危机 / 疫情等重大危机应对核心机制 | `alliance_org:g20` | keep | approve |
| 4 | 七国集团 | `G7` | 美国、德国、日本（轮值主导） | 西方发达国家最高协调机制，货币政策 / 地缘政治 / 科技制裁统一发声平台 | `alliance_org:g7` | keep | approve |
| 5 | 联合国安全理事会 | `UNSC` | 五常主导 | 唯一有权授权军事行动，决定战争与和平 | `alliance_org:unsc` | create | approve |
| 6 | 北大西洋公约组织 | `NATO` | 美国主导 | 全球最强大军事同盟，主导欧洲安全秩序 | `alliance_org:nato` | create | approve |
| 7 | 上海合作组织 | `SCO` | 中俄共同主导 | 欧亚大陆最大综合性区域组织，影响全球 30% 人口 | `alliance_org:sco` | create | approve |
| 8 | 五眼联盟 | `Five Eyes` | 美英主导 | 全球最紧密情报体系，深度影响全球安全与科技博弈 | `alliance_org:five_eyes` | create | approve |
| 9 | 欧洲联盟 | `EU` | 欧盟 | 全球第二大经济体，规则制定能力仅次于美国 | `alliance_org:eu` | keep | approve |
| 10 | 东南亚国家联盟 | `ASEAN` | 东盟 | 亚太秩序关键一极，RCEP 与东亚合作中心轮轴 | `alliance_org:asean` | create | approve |
| 11 | 非洲联盟 | `AU` | 非盟 | 非洲最高政治机构，G20 正式成员 | `alliance_org:au` | create | approve |
| 12 | 阿拉伯国家联盟 | `LAS` | 阿盟 | 阿拉伯世界政治核心，影响中东地缘与能源 | `alliance_org:las` | create | approve |
| 13 | 海湾合作委员会 | `GCC` | 海合会 | 全球石油定价权核心，影响全球能源格局 | `alliance_org:gcc` | create | approve |
| 14 | 伊斯兰合作组织 | `OIC` | 沙特等主导 | 伊斯兰世界最大政府间组织，覆盖 18 亿穆斯林 | `alliance_org:oic` | create | approve |
| 15 | 美洲国家组织 | `OAS` | 美国主导 | 西半球最大政治组织，美国后院治理工具 | `alliance_org:oas` | create | approve |
| 17 | 世界贸易组织 | `WTO` | 多边 | 全球贸易规则制定与争端解决，覆盖 98% 国际贸易 | `alliance_org:wto` | keep | approve |
| 18 | 区域全面经济伙伴关系协定 | `RCEP` | 东盟主导 / 中国核心参与 | 全球最大自贸区，重塑亚太供应链格局 | `alliance_org:rcep` | create | approve |
| 19 | 全面与进步跨太平洋伙伴关系协定 | `CPTPP` | 日本主导 | 全球最高标准经贸规则，数字经济规则标杆 | `alliance_org:cptpp` | create | approve |
| 20 | 美墨加协定 | `USMCA` | 美国主导 | 北美供应链核心框架，含对华 "毒丸条款" | `alliance_org:usmca` | create | approve |
| 21 | 中国 - 东盟自由贸易区 | `CAFTA` | 中 - 东盟共同主导 | 发展中国家间最大自贸区，RCEP 基础框架 | `alliance_org:cafta` | create | approve |
| 22 | 欧亚经济联盟 | `EAEU` | 俄罗斯主导 | 后苏联空间经济一体化核心 | `alliance_org:eaeu` | create | approve |
| 23 | 南方共同市场 | `MERCOSUR` | 巴西阿根廷主导 | 拉美最大经济一体化组织 | `alliance_org:mercosur` | create | approve |
| 24 | 非洲大陆自由贸易区 | `AfCFTA` | 非盟主导 | 全球成员最多自贸区，非洲工业化关键 | `alliance_org:afcfta` | create | approve |
| 26 | 石油输出国组织 | `OPEC` | 沙特主导 | 全球石油定价权核心，直接决定国际油价走势 | `alliance_org:opec` | keep | approve |
| 27 | OPEC+（欧佩克 +） | `OPEC+` | 沙特 + 俄罗斯联合主导 | 掌控全球半数石油供应，减产决议影响全球经济 | `alliance_org:opec_plus` | keep | approve |
| 28 | 国际能源署 | `IEA` | 美欧主导 | 发达国家能源安全核心，战略储备调控油价 | `alliance_org:iea` | create | approve |
| 29 | 天然气出口国论坛 | `GECF` | 俄罗斯 + 伊朗主导 | 全球天然气储量七成掌控，气价博弈核心 | `alliance_org:gecf` | create | approve |
| 30 | 矿产安全伙伴关系 | `MSP` | 美国主导 | 西方关键矿产供应链联盟，制衡中国资源优势 | `alliance_org:msp` | create | approve |
| 32 | 国际货币基金组织 | `IMF` | 美欧主导 | 全球金融安全网核心，SDR 发行，危机救助决定国家命运 | `alliance_org:imf` | keep | approve |
| 33 | 世界银行集团 | `WBG` | 美欧主导 | 全球最大多边开发机构，影响发展中国家政策方向 | `alliance_org:world_bank` | keep | approve |
| 34 | 亚洲基础设施投资银行 | `AIIB` | 中国主导发起 | 新兴多边金融机构代表，重塑亚洲基建融资格局 | `alliance_org:aiib` | create | approve |
| 35 | 新开发银行（金砖银行） | `NDB` | 金砖共同主导 | 金砖国家金融合作核心，推动去美元化 | `alliance_org:ndb` | create | approve |
| 36 | 清迈倡议多边化 | `CMIM` | 东盟 + 中日韩 | 亚洲区域金融防火墙，2400 亿美元互换规模 | `alliance_org:cmim` | create | approve |
| 37 | 世界卫生组织 | `WHO` | 联合国专门机构 | 全球公共卫生最高权威，疫情应对直接影响全球经济 | `alliance_org:who` | create | approve |
| 38 | 国际原子能机构 | `IAEA` | 多边（美欧影响大） | 核不扩散核查权，决定伊朗、朝鲜等核问题走向 | `alliance_org:iaea` | create | approve |
| 40 | 一带一路倡议 | `BRI` | 中国首倡主导 | 中国全球经济外交旗舰，重塑全球基建与贸易格局 | `alliance_org:bri` | create | approve |
| 41 | 印太经济框架 | `IPEF` | 美国主导 | 美国印太经济战略核心，重塑供应链与数字规则 | `alliance_org:ipef` | create | approve |
| 42 | 四方安全对话 | `QUAD` | 美国主导 | 印太战略制衡核心，中美博弈关键小多边平台 | `alliance_org:quad` | create | approve |
| 43 | 金砖国家合作机制 | `BRICS` | 中俄巴印南共同主导 | 全球南方核心平台，推动多极化与去美元化 | `alliance_org:brics` | keep | approve |
| 44 | 全球基础设施和投资伙伴关系 | `PGII` | 美国 / G7 主导 | G7 对标一带一路的基建方案，6000 亿美元规模 | `alliance_org:pgii` | create | approve |
| 45 | 美日韩三边合作机制 | `UJR` | 美国主导 | 东北亚安全架构升级，对华围堵前沿 | `alliance_org:ujr` | create | approve |
| 46 | 芯片四方联盟 | `Chip 4` | 美国主导 | 半导体供应链对华脱钩核心机制 | `alliance_org:chip_4` | create | approve |
| 47 | 中非合作论坛 | `FOCAC` | 中国主导 | 中非关系核心平台，深度影响非洲发展方向 | `alliance_org:focac` | create | approve |
| 48 | 中阿合作论坛 | `CASCF` | 中国主导 | 中阿能源与一带一路合作核心平台 | `alliance_org:cascf` | create | approve |
| 49 | 中拉论坛 | `CELAC` | 中国主导 | 中拉整体合作框架 | `alliance_org:celac` | create | approve |
| 50 | 中国-中亚峰会 | `CCAS` | 中国主导 | 中亚战略升级，能源与安全走廊核心 | `alliance_org:ccas` | create | approve |
| 51 | 美欧贸易与技术委员会 | `TTC` | 美欧共同主导 | 西方科技联盟核心，出口管制与技术标准协调 | `alliance_org:ttc` | create | approve |

## 现有 10 条 Active Alliance Disposition

| Existing key | disposition | 收敛要求 |
|---|---|---|
| `alliance_org:opec_plus` | keep | 复用稳定 key/UUID，profile 按批准四字段收敛 |
| `alliance_org:opec` | keep | 复用稳定 key/UUID |
| `alliance_org:g7` | keep | 复用稳定 key/UUID |
| `alliance_org:g20` | keep | 复用稳定 key/UUID |
| `alliance_org:wto` | keep | 复用稳定 key/UUID |
| `alliance_org:imf` | keep | 复用稳定 key/UUID |
| `alliance_org:world_bank` | keep | 复用稳定 key/UUID；name/canonical 收敛为“世界银行集团” |
| `alliance_org:oecd` | forward inactivate | 不在批准 45 target set；保留 key/UUID/provenance，不物理删除 |
| `alliance_org:eu` | keep | 复用稳定 key/UUID |
| `alliance_org:brics` | keep | 复用稳定 key/UUID；name/canonical 收敛为“金砖国家合作机制” |

最终 active alliance key set 必须严格等于上表 45 keys；OECD 处置只能在未来 R2A 获得独立 Write 授权后 forward inactivate。
