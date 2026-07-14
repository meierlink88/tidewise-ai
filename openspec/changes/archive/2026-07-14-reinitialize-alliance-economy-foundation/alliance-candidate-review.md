# Package 1：Alliance Candidate Review（Excel 1.0 已批准）

## 0. 状态、来源与硬门禁

本文记录 Package 1.2 的最终人工决策。主对话于 2026-07-14 批准全部 45 条候选、9 keep + 36 create、两个 U+200C normalization，以及现有 10 条 alliance 的 9 keep + 1 forward inactivate disposition。

- 权威文件：`联盟组织列表1.0.xlsx`。
- SHA-256：`ac0d953c0cd93596fe6bf8a70541bbe658620e75d38a9b3178980071b2cdc102`。
- 唯一读取范围：首个 sheet `联盟组织`，`A1:K51`。
- 结构：45 条数据行；sheet rows 2、16、25、31、39 是 5 条分组标题，不是实体。
- 四个目标字段完整性：名称 45/45、缩写 45/45、核心主导方 45/45、核心影响范围说明 45/45 非空。
- 重复：名称 0，删除末尾 U+200C 后缩写 0。
- 旧 `表格_20260713.csv` 的 68 条 provisional 候选、recommendation、网页核验与排除表已 superseded，只留在 Git 历史，不能与本表合并或补充当前 manifest。
- 现有联盟基线：`backend/data/entity_foundation/alliance_orgs.json` SHA-256 `a797ed7b03a3f3acfc3e8fb885b3b19c16af8c4dc2f781efcb9d8ae2089ee37f`；本轮不连接 PostgreSQL，因此不抄录或声称真实 UUID。
- Identity 映射规则：源名称或规范化缩写与现有 name/canonical/aliases 做 NFKC + casefold 等价匹配，命中则复用 key；未命中则仅以规范化缩写生成 provisional `alliance_org:<slug>`。不使用旧 recommendation、英文名或网页知识补全 identity。
- Package 1.2 已通过；批准结果冻结在 `approved-alliance-manifest.md` v1，canonical checksum `4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a`。该批准只授权 Package 2 的 R0 候选工作，不授权 seed、源码/migration 或 PostgreSQL/Neo4j。

`proposed action` 只表示与当前 10 条文件基线比较得到的 `create/keep` 技术 diff，不是 `approve/reject/defer` recommendation，也不是写入授权。分组标题仅用于定位，不生成 category、标签或 profile 字段。

## 1. 45 条源数据映射与 Exact Diff

| Sheet row | 分组（仅定位） | 名称 → `name/canonical_name` | 缩写 → `abbreviation`/派生 alias | 核心主导方 → `leadership_summary` | 核心影响范围说明 → `influence_scope_summary` | proposed entity key | proposed action | final decision | normalization / exact diff |
|---:|---|---|---|---|---|---|---|---|---|
| 3 | 一、政治军事联盟 | 二十国集团 | `G20` | 轮值主席国制（美 / 中 / 德 / 日等大国轮流） | 全球宏观经济政策协调首要平台，金融危机 / 疫情等重大危机应对核心机制 | `alliance_org:g20` | keep | approve | 无；name/canonical 与源值一致；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases，并确保缩写 alias 存在 |
| 4 | 一、政治军事联盟 | 七国集团 | `G7` | 美国、德国、日本（轮值主导） | 西方发达国家最高协调机制，货币政策 / 地缘政治 / 科技制裁统一发声平台 | `alliance_org:g7` | keep | approve | 无；name/canonical 与源值一致；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases，并确保缩写 alias 存在 |
| 5 | 一、政治军事联盟 | 联合国安全理事会 | `UNSC` | 五常主导 | 唯一有权授权军事行动，决定战争与和平 | `alliance_org:unsc` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 6 | 一、政治军事联盟 | 北大西洋公约组织 | `NATO` | 美国主导 | 全球最强大军事同盟，主导欧洲安全秩序 | `alliance_org:nato` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 7 | 一、政治军事联盟 | 上海合作组织 | `SCO` | 中俄共同主导 | 欧亚大陆最大综合性区域组织，影响全球 30% 人口 | `alliance_org:sco` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 8 | 一、政治军事联盟 | 五眼联盟 | `Five Eyes` | 美英主导 | 全球最紧密情报体系，深度影响全球安全与科技博弈 | `alliance_org:five_eyes` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 9 | 一、政治军事联盟 | 欧洲联盟 | `EU` | 欧盟 | 全球第二大经济体，规则制定能力仅次于美国 | `alliance_org:eu` | keep | approve | 无；name/canonical 与源值一致；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases，并确保缩写 alias 存在 |
| 10 | 一、政治军事联盟 | 东南亚国家联盟 | `ASEAN` | 东盟 | 亚太秩序关键一极，RCEP 与东亚合作中心轮轴 | `alliance_org:asean` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 11 | 一、政治军事联盟 | 非洲联盟 | `AU` | 非盟 | 非洲最高政治机构，G20 正式成员 | `alliance_org:au` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 12 | 一、政治军事联盟 | 阿拉伯国家联盟 | `LAS` | 阿盟 | 阿拉伯世界政治核心，影响中东地缘与能源 | `alliance_org:las` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 13 | 一、政治军事联盟 | 海湾合作委员会 | `GCC` | 海合会 | 全球石油定价权核心，影响全球能源格局 | `alliance_org:gcc` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 14 | 一、政治军事联盟 | 伊斯兰合作组织 | `OIC` | 沙特等主导 | 伊斯兰世界最大政府间组织，覆盖 18 亿穆斯林 | `alliance_org:oic` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 15 | 一、政治军事联盟 | 美洲国家组织 | `OAS` | 美国主导 | 西半球最大政治组织，美国后院治理工具 | `alliance_org:oas` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 17 | 二、经济贸易与区域一体化 | 世界贸易组织 | `WTO` | 多边 | 全球贸易规则制定与争端解决，覆盖 98% 国际贸易 | `alliance_org:wto` | keep | approve | 无；name/canonical 与源值一致；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases，并确保缩写 alias 存在 |
| 18 | 二、经济贸易与区域一体化 | 区域全面经济伙伴关系协定 | `RCEP` | 东盟主导 / 中国核心参与 | 全球最大自贸区，重塑亚太供应链格局 | `alliance_org:rcep` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 19 | 二、经济贸易与区域一体化 | 全面与进步跨太平洋伙伴关系协定 | `CPTPP` | 日本主导 | 全球最高标准经贸规则，数字经济规则标杆 | `alliance_org:cptpp` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 20 | 二、经济贸易与区域一体化 | 美墨加协定 | `USMCA` | 美国主导 | 北美供应链核心框架，含对华 "毒丸条款" | `alliance_org:usmca` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 21 | 二、经济贸易与区域一体化 | 中国 - 东盟自由贸易区 | `CAFTA` | 中 - 东盟共同主导 | 发展中国家间最大自贸区，RCEP 基础框架 | `alliance_org:cafta` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 22 | 二、经济贸易与区域一体化 | 欧亚经济联盟 | `EAEU` | 俄罗斯主导 | 后苏联空间经济一体化核心 | `alliance_org:eaeu` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 23 | 二、经济贸易与区域一体化 | 南方共同市场 | `MERCOSUR` | 巴西阿根廷主导 | 拉美最大经济一体化组织 | `alliance_org:mercosur` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 24 | 二、经济贸易与区域一体化 | 非洲大陆自由贸易区 | `AfCFTA` | 非盟主导 | 全球成员最多自贸区，非洲工业化关键 | `alliance_org:afcfta` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 26 | 三、能源资源与大宗商品组织 | 石油输出国组织 | `OPEC` | 沙特主导 | 全球石油定价权核心，直接决定国际油价走势 | `alliance_org:opec` | keep | approve | 无；name/canonical 与源值一致；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases，并确保缩写 alias 存在 |
| 27 | 三、能源资源与大宗商品组织 | OPEC+（欧佩克 +） | `OPEC+` | 沙特 + 俄罗斯联合主导 | 掌控全球半数石油供应，减产决议影响全球经济 | `alliance_org:opec_plus` | keep | approve | 无；name/canonical `OPEC+` → `OPEC+（欧佩克 +）`；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases |
| 28 | 三、能源资源与大宗商品组织 | 国际能源署 | `IEA` | 美欧主导 | 发达国家能源安全核心，战略储备调控油价 | `alliance_org:iea` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 29 | 三、能源资源与大宗商品组织 | 天然气出口国论坛 | `GECF` | 俄罗斯 + 伊朗主导 | 全球天然气储量七成掌控，气价博弈核心 | `alliance_org:gecf` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 30 | 三、能源资源与大宗商品组织 | 矿产安全伙伴关系 | `MSP` | 美国主导 | 西方关键矿产供应链联盟，制衡中国资源优势 | `alliance_org:msp` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 32 | 四、金融货币与全球治理机构 | 国际货币基金组织 | `IMF` | 美欧主导 | 全球金融安全网核心，SDR 发行，危机救助决定国家命运 | `alliance_org:imf` | keep | approve | 无；name/canonical 与源值一致；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases，并确保缩写 alias 存在 |
| 33 | 四、金融货币与全球治理机构 | 世界银行集团 | `WBG` | 美欧主导 | 全球最大多边开发机构，影响发展中国家政策方向 | `alliance_org:world_bank` | keep | approve | 无；name/canonical `世界银行` → `世界银行集团`；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases，并新增缩写 alias `WBG` |
| 34 | 四、金融货币与全球治理机构 | 亚洲基础设施投资银行 | `AIIB` | 中国主导发起 | 新兴多边金融机构代表，重塑亚洲基建融资格局 | `alliance_org:aiib` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 35 | 四、金融货币与全球治理机构 | 新开发银行（金砖银行） | `NDB` | 金砖共同主导 | 金砖国家金融合作核心，推动去美元化 | `alliance_org:ndb` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 36 | 四、金融货币与全球治理机构 | 清迈倡议多边化 | `CMIM` | 东盟 + 中日韩 | 亚洲区域金融防火墙，2400 亿美元互换规模 | `alliance_org:cmim` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 37 | 四、金融货币与全球治理机构 | 世界卫生组织 | `WHO` | 联合国专门机构 | 全球公共卫生最高权威，疫情应对直接影响全球经济 | `alliance_org:who` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 38 | 四、金融货币与全球治理机构 | 国际原子能机构 | `IAEA` | 多边（美欧影响大） | 核不扩散核查权，决定伊朗、朝鲜等核问题走向 | `alliance_org:iaea` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 40 | 五、战略倡议与伙伴关系 | 一带一路倡议 | `BRI` | 中国首倡主导 | 中国全球经济外交旗舰，重塑全球基建与贸易格局 | `alliance_org:bri` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 41 | 五、战略倡议与伙伴关系 | 印太经济框架 | `IPEF` | 美国主导 | 美国印太经济战略核心，重塑供应链与数字规则 | `alliance_org:ipef` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 42 | 五、战略倡议与伙伴关系 | 四方安全对话 | `QUAD` | 美国主导 | 印太战略制衡核心，中美博弈关键小多边平台 | `alliance_org:quad` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 43 | 五、战略倡议与伙伴关系 | 金砖国家合作机制 | `BRICS` | 中俄巴印南共同主导 | 全球南方核心平台，推动多极化与去美元化 | `alliance_org:brics` | keep | approve | 无；name/canonical `金砖国家` → `金砖国家合作机制`；旧 profile 待替换为三业务字段；保留稳定 key/UUID 与既有合法 aliases |
| 44 | 五、战略倡议与伙伴关系 | 全球基础设施和投资伙伴关系 | `PGII` | 美国 / G7 主导 | G7 对标一带一路的基建方案，6000 亿美元规模 | `alliance_org:pgii` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 45 | 五、战略倡议与伙伴关系 | 美日韩三边合作机制 | `UJR` | 美国主导 | 东北亚安全架构升级，对华围堵前沿 | `alliance_org:ujr` | create | approve | 删除缩写末尾 U+200C：`UJR<U+200C>` → `UJR`；当前 10 条中不存在；UUID 未分配 |
| 46 | 五、战略倡议与伙伴关系 | 芯片四方联盟 | `Chip 4` | 美国主导 | 半导体供应链对华脱钩核心机制 | `alliance_org:chip_4` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 47 | 五、战略倡议与伙伴关系 | 中非合作论坛 | `FOCAC` | 中国主导 | 中非关系核心平台，深度影响非洲发展方向 | `alliance_org:focac` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 48 | 五、战略倡议与伙伴关系 | 中阿合作论坛 | `CASCF` | 中国主导 | 中阿能源与一带一路合作核心平台 | `alliance_org:cascf` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 49 | 五、战略倡议与伙伴关系 | 中拉论坛 | `CELAC` | 中国主导 | 中拉整体合作框架 | `alliance_org:celac` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |
| 50 | 五、战略倡议与伙伴关系 | 中国-中亚峰会 | `CCAS` | 中国主导 | 中亚战略升级，能源与安全走廊核心 | `alliance_org:ccas` | create | approve | 删除缩写末尾 U+200C：`CCAS<U+200C>` → `CCAS`；当前 10 条中不存在；UUID 未分配 |
| 51 | 五、战略倡议与伙伴关系 | 美欧贸易与技术委员会 | `TTC` | 美欧共同主导 | 西方科技联盟核心，出口管制与技术标准协调 | `alliance_org:ttc` | create | approve | 无；当前 10 条中不存在；UUID 未分配，alias 仅由缩写派生 |

### 候选 Diff 断言

```text
candidate rows = 45
proposed keep = 9
proposed create = 36
candidate final decisions completed = 45
name duplicates = 0
normalized abbreviation duplicates = 0
```

## 2. 现有 10 条 Active Alliance 的穷尽 Disposition

下表不引用网页或旧 recommendation。`final disposition` 是 Package 1.2 已批准结果；`inactivate` 仍须在未来 R2A 获得独立 Write 授权后 forward convergence。

| Existing key | 新 Excel 映射 | proposed disposition | final disposition | exact impact / Review note |
|---|---|---|---|---|
| `alliance_org:opec_plus` | sheet row 27，OPEC+（欧佩克 +） | keep | keep | 复用 key/UUID；name/canonical 按源值更新，保留既有合法 aliases |
| `alliance_org:opec` | sheet row 26，石油输出国组织 | keep | keep | 复用 key/UUID；name/canonical 与源值一致 |
| `alliance_org:g7` | sheet row 4，七国集团 | keep | keep | 复用 key/UUID；不再使用旧网页来源文本 |
| `alliance_org:g20` | sheet row 3，二十国集团 | keep | keep | 复用 key/UUID |
| `alliance_org:wto` | sheet row 17，世界贸易组织 | keep | keep | 复用 key/UUID |
| `alliance_org:imf` | sheet row 32，国际货币基金组织 | keep | keep | 复用 key/UUID |
| `alliance_org:world_bank` | sheet row 33，世界银行集团 | keep | keep | 复用 key/UUID；name/canonical 从“世界银行”更新为源值，派生 `WBG` alias |
| `alliance_org:oecd` | 新 45 条中无匹配 | inactivate | inactivate | 已批准未来 forward inactivate；保留 key/UUID/provenance，不物理删除；R2A 授权前保持现状 |
| `alliance_org:eu` | sheet row 9，欧洲联盟 | keep | keep | 复用 key/UUID |
| `alliance_org:brics` | sheet row 43，金砖国家合作机制 | keep | keep | 复用 key/UUID；name/canonical 从“金砖国家”更新为源值 |

```text
existing active covered = 10/10
proposed keep = 9
proposed merge = 0
proposed inactivate = 1
final dispositions completed = 10
```

## 3. QA、Review Notes 与 Fail-Closed 条件

- 确定性 QA sample：sheet rows 3、17、26、32、40、51，并强制包含两个 normalization rows 45、50、全部 9 个 keep 映射及缺席的 `alliance_org:oecd`。抽样不替代 45 条候选和 10 条现有 disposition 的逐项 Review。
- UJR/CCAS 只删除末尾 U+200C；工作簿其他名称、缩写、核心主导方和影响说明均原样呈现。疑似语义问题可由用户在单行 final Review 中备注，本草案不自行纠正。
- “核心主导方”只是 `leadership_summary` 文本，不能自动生成 `led_by`。
- 任何输入 SHA、sheet/range、45/5 counts、四字段完整性、重复断言、existing baseline hash 或 exact diff 发生漂移，Package 1 必须停止并重新生成 Review package。
- Package 1.2 canonical tuple checksum 必须与 `approved-alliance-manifest.md` v1 一致；漂移时 Package 2 输入失效并回到 Review。

## 4. 当前停止点

Package 1 已完成并只读冻结；Package 2 候选见 `package-2-candidate-review.md`，其快速 MVP 范围已获人工批准。当前停止等待产业链 change Deliver，不得进入 Package 3。
