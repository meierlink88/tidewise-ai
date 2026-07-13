# B：Alliance Candidate Review（Provisional Review Draft）

## 0. 状态、方法与硬门禁

本文是 tasks 2.1—2.2 的**只读临时候选草案**。联盟主数据尚未完全准备，所有 `recommendation`、identity、categories、摘要与 disposition 都只是供主对话逐项 Review 的建议；`final decision` 一律留空，不构成 approved alliance manifest、seed、Write 或清理授权。

- CSV 来源：`表格_20260713.csv`，只读审阅第 1—85 条；未复制为可执行 seed。
- 现有数据来源：`origin/main@f942d7615afd952840cdc478bbe7b4ecc990616d` 的 `backend/data/entity_foundation/alliance_orgs.json`，只读文件审计，不连接 PostgreSQL。
- 正式来源只核验组织/机制 identity、名称、职责与持续存在性，核验日为 2026-07-13；**未读取、汇总或冻结任何成员国全集**。
- 本文不生成 economy 范围、`member_of`/`led_by`/`part_of` 候选，不修改源码、migration、seed，也不连接或写 PostgreSQL/Neo4j。
- `recommendation` 仅允许 `approve/reject/merge/defer`；`final decision` 必须由主对话填写。任一 summary 缺失都会是 blocker；本草案已提供非空建议文本，但仍须逐项核验。
- task 2.3 保持未通过。只有 68 条 CSV 候选和 10 个现有 active alliance disposition 全部逐项确认后，才可形成 approved manifest；在此之前禁止进入 C。

字段缩写：`L` = `leadership_summary` 草案，`I` = `influence_scope_summary` 草案；`create/keep/merge/inactivate` 只是未来 disposition 建议，不是执行动作。所有 categories 均来自已批准 22-code allowlist，并已按 code 字典序排列。

UUID 边界：对既有 `keep` 或 `merge target` 只建议复用当前 key 对应的稳定 UUID，但本轮不连接 PostgreSQL，因此不抄录或声称真实 UUID 值；对 `create` 候选的 UUID 保持未分配，只有逐项批准并进入未来 dry-run 后才可按既有 deterministic identity 契约提出。任何 recommendation 都不得提前占用 UUID。

## 0.1 R0 Candidate Review package 元数据

- 输入指纹：CSV SHA-256 `584f990ddf3a0784d7586c0b0dc40aef7558620f8d8a0c27cb91a8b075002614`；现有联盟基线 `f942d76:backend/data/entity_foundation/alliance_orgs.json` SHA-256 `a797ed7b03a3f3acfc3e8fb885b3b19c16af8c4dc2f781efcb9d8ae2089ee37f`；本文件 adoption 前业务内容 SHA-256 `9536c4889a3f5fbb4676b8da7c5b1ba67d88fa7ffb1cad71825e7056b7cb83e8`。
- 生成边界：CSV 1—68 形成 provisional 候选，69—85 排除；现有 10 条全部进入 disposition Review；来源仅用于 identity、名称、职责和持续存在性，不读取成员全集。
- Counts：68 条候选，其中 recommend approve 62、defer 4、merge 1、reject 1；10 条现有 active alliance 全覆盖；17 条排除；67 条带正式来源链接，Chip 4 为 1 个正式来源 blocker。
- 确定性 QA sample：CSV 行 1、10、20、30、40、50、60、68，并追加全部非 approve、宽 identity 边界、来源 blocker、alias/abbreviation 冲突项。**抽样不替代逐项 Review**；2.3 仍须确认全部 68 条候选与 10 条现有 disposition。
- 冲突/异常全集：World Bank stable target merge；现有但 CSV 未列的 G7/G20/OECD；BRI、PGII、Chip 4、EU-US TTC defer；Silk Road Fund reject；ISO alias 冲突；无正式 abbreviation 项；Chip 4 来源 blocker；协议机制、倡议网络和联合国下属机构的宽 identity 边界。
- Expected action classification：`create/keep/merge/defer/reject/inactivate` 仅供审阅，不是 Write 指令。任一 final decision 留空，或 source、identity、alias、summary、category、existing disposition 冲突未解决，均 fail-closed 并阻断 2.3。

## 1. CSV 第 1—68 条组织/机制候选

| No. | recommendation | final decision | identity / 中文名 / English name | aliases；abbreviation | categories | 非空摘要草案 | 正式来源 | 与现有 10 条 exact diff / 建议 disposition |
|---:|---|---|---|---|---|---|---|---|
| 1 | approve | ____ | `alliance_org:un`；联合国；United Nations | 联合国组织、UN；`UN` | `development`, `governance`, `humanitarian`, `legal`, `political`, `security` | L：依《联合国宪章》由大会、安全理事会等主要机关及秘书处共同治理。I：覆盖和平安全、发展、人权、人道与国际法等全球议题。 | [UN About Us](https://www.un.org/en/about-us) | 不存在；建议 create。 |
| 2 | approve | ____ | `alliance_org:un_security_council`；联合国安全理事会；United Nations Security Council | 安理会、UN Security Council、UNSC；`UNSC` | `legal`, `military`, `political`, `security` | L：联合国主要机关，按宪章承担维护国际和平与安全的首要责任。I：覆盖制裁、维和、和平解决争端及宪章授权行动。 | [UN Security Council](https://main.un.org/securitycouncil/en/content/what-security-council) | 不存在；建议 create；其与 UN 的正式隶属只留待未来 `part_of` Review。 |
| 3 | approve | ____ | `alliance_org:nato`；北大西洋公约组织；North Atlantic Treaty Organization | 北约、North Atlantic Alliance、NATO；`NATO` | `military`, `political`, `security` | L：以北大西洋理事会共识决策和一体化军事结构协调集体防御。I：覆盖欧洲—北美集体防御、威慑、危机管理与合作安全。 | [NATO: What is NATO?](https://www.nato.int/content/what-is-nato/en.html) | 不存在；建议 create。 |
| 4 | approve | ____ | `alliance_org:csto`；集体安全条约组织；Collective Security Treaty Organization | 集安组织、ODKB、CSTO；`CSTO` | `military`, `security` | L：以集体安全理事会及常设机构协调条约框架下的安全合作。I：覆盖成员所在欧亚区域的集体安全与防务协作。 | [CSTO official site](https://en.odkb-csto.org/) | 不存在；建议 create；正式英文名/别名需复验站点语言版本。 |
| 5 | approve | ____ | `alliance_org:sco`；上海合作组织；Shanghai Cooperation Organisation | 上合组织、Shanghai Cooperation Organization、SCO；`SCO` | `economic`, `political`, `security`, `trade` | L：由元首理事会等机构决策，秘书处为常设执行机构。I：覆盖欧亚安全、政治互信、经贸及区域合作。 | [SCO official introduction](https://chn.sectsco.org/20151209/26996.html) | 不存在；建议 create。 |
| 6 | approve | ____ | `alliance_org:five_eyes`；五眼情报合作；Five Eyes | 五眼联盟、FVEY、Five Eyes Intelligence Partnership；`FVEY` | `intelligence`, `security` | L：由参与国情报与安全机构通过长期政府间安排协作，无统一超国家秘书处。I：覆盖信号情报、安全情报及相关防务科技合作。 | [UK Government: Five Eyes science and technology](https://www.gov.uk/government/case-studies/five-eyes-science-and-technology) | 不存在；建议 create；CSV abbreviation `Five Eyes` 收敛为展示别名，profile 建议 `FVEY`。 |
| 7 | approve | ____ | `alliance_org:aukus`；澳英美安全伙伴关系；AUKUS | 美英澳三边安全伙伴关系、Australia-United Kingdom-United States Partnership、AUKUS；`AUKUS` | `military`, `security`, `technology` | L：澳大利亚、英国和美国通过双支柱政府间伙伴关系推进能力合作。I：覆盖核动力潜艇、先进防务能力与互操作性。 | [Australian Defence: AUKUS advanced capabilities](https://www.defence.gov.au/defence-activities/programs-initiatives/aukus-advanced-capabilities) | 不存在；建议 create；规范中文名建议按英文参与方顺序。 |
| 8 | approve | ____ | `alliance_org:eu`；欧洲联盟；European Union | 欧盟、EU；`EU` | `economic`, `finance`, `governance`, `legal`, `political`, `trade` | L：由欧洲理事会、欧盟理事会、委员会、议会等条约机构共同治理。I：覆盖区域一体化、单一市场、货币及对外规则协调。 | [European Union: aims and values](https://european-union.europa.eu/principles-countries-history/principles-and-values/aims-and-values_en) | 已存在同 key；名称/aliases 可复用，旧 profile 需转换；建议 keep。 |
| 9 | approve | ____ | `alliance_org:asean`；东南亚国家联盟；Association of Southeast Asian Nations | 东盟、ASEAN；`ASEAN` | `economic`, `political`, `security`, `trade` | L：以峰会、协调理事会、共同体理事会和秘书处推动共识协作。I：覆盖东南亚政治安全、经济和社会文化共同体建设。 | [ASEAN: About ASEAN](https://asean.org/about-us/) | 不存在；建议 create。 |
| 10 | approve | ____ | `alliance_org:african_union`；非洲联盟；African Union | 非盟、AU；`AU` | `development`, `economic`, `governance`, `political`, `security` | L：由成员国大会、执行理事会、委员会等机构推进大陆议程。I：覆盖非洲一体化、和平安全、发展和大陆治理。 | [African Union overview](https://au.int/en/overview) | 不存在；建议 create。 |
| 11 | approve | ____ | `alliance_org:league_of_arab_states`；阿拉伯国家联盟；League of Arab States | 阿拉伯联盟、阿盟、Arab League、LAS；`LAS` | `political`, `security` | L：由理事会和秘书处协调阿拉伯国家间政策与共同事务。I：覆盖阿拉伯地区政治协调、争端与区域安全议题。 | [League of Arab States official site](https://www.leagueofarabstates.net/) | 不存在；建议 create；正式英文名页面可审计性需在最终 Review 复验。 |
| 12 | approve | ____ | `alliance_org:gcc`；海湾阿拉伯国家合作委员会；Cooperation Council for the Arab States of the Gulf | 海湾合作委员会、海合会、Gulf Cooperation Council、GCC；`GCC` | `economic`, `energy`, `political`, `security`, `trade` | L：最高理事会、部长理事会和秘书处依章程协调成员政策。I：覆盖海湾地区政治安全、经济一体化、贸易与能源政策协作。 | [GCC About Us](https://gcc-sg.org/en/AboutUs/Pages/default.aspx) | 不存在；建议 create。 |
| 13 | approve | ____ | `alliance_org:oic`；伊斯兰合作组织；Organisation of Islamic Cooperation | 伊合组织、Organization of Islamic Cooperation、OIC；`OIC` | `development`, `humanitarian`, `political`, `religion` | L：由伊斯兰峰会、外长理事会和总秘书处推进政府间合作。I：覆盖伊斯兰世界政治、发展、人道与文化宗教合作。 | [OIC History](https://www.oic-oci.org/page/?p_id=52&p_ref=26&lan=en) | 不存在；建议 create。 |
| 14 | approve | ____ | `alliance_org:cis`；独立国家联合体；Commonwealth of Independent States | 独联体、СНГ、CIS；`CIS` | `economic`, `political`, `security` | L：由国家元首理事会、政府首脑理事会及执行委员会协调合作。I：覆盖后苏联区域政治、经济和安全协作。 | [CIS Executive Committee](https://e-cis.info/) | 不存在；建议 create；英文官方名称需在最终 Review 保留多语来源。 |
| 15 | approve | ____ | `alliance_org:oas`；美洲国家组织；Organization of American States | 美洲组织、OEA、OAS；`OAS` | `governance`, `legal`, `political`, `security` | L：由大会、常设理事会和总秘书处依《美洲国家组织宪章》治理。I：覆盖美洲民主、人权、安全与发展合作。 | [OAS: Who We Are](https://www.oas.org/en/about/who_we_are.asp) | 不存在；建议 create。 |
| 16 | approve | ____ | `alliance_org:ecowas`；西非国家经济共同体；Economic Community of West African States | 西共体、CEDEAO、ECOWAS；`ECOWAS` | `economic`, `political`, `security`, `trade` | L：由国家元首和政府首脑机构、部长理事会及委员会推进区域决策。I：覆盖西非经济一体化、人员流动、和平与安全。 | [ECOWAS Basic Information](https://www.ecowas.int/about-ecowas/basic-information/) | 不存在；建议 create。 |
| 17 | approve | ____ | `alliance_org:wto`；世界贸易组织；World Trade Organization | 世贸组织、WTO；`WTO` | `dispute_resolution`, `governance`, `legal`, `trade` | L：由部长级会议、总理事会及秘书处按多边贸易协定运行。I：覆盖全球贸易规则、谈判、监督与争端解决。 | [WTO: What we do](https://www.wto.org/english/thewto_e/whatis_e/what_we_do_e.htm) | 已存在同 key；名称/aliases 可复用，旧 profile 需转换；建议 keep。 |
| 18 | approve | ____ | `alliance_org:rcep`；区域全面经济伙伴关系协定机制；Regional Comprehensive Economic Partnership | 区域全面经济伙伴关系、RCEP Agreement、RCEP；`RCEP` | `economic`, `trade` | L：由缔约方委员会及协定机制实施和审议区域经贸规则。I：覆盖亚太区域货物、服务、投资及相关贸易规则。 | [RCEP Secretariat](https://rcepsec.org/) | 不存在；建议 create；明确建模对象是协定治理机制而非条约文本。 |
| 19 | approve | ____ | `alliance_org:cptpp`；全面与进步跨太平洋伙伴关系协定机制；Comprehensive and Progressive Agreement for Trans-Pacific Partnership | 跨太平洋伙伴全面进步协定、CPTPP；`CPTPP` | `economic`, `trade` | L：由 CPTPP 委员会与缔约方依协定进行实施和加入审议。I：覆盖跨太平洋贸易、投资、数字及监管规则。 | [New Zealand MFAT: CPTPP](https://www.mfat.govt.nz/en/trade/free-trade-agreements/free-trade-agreements-in-force/cptpp) | 不存在；建议 create；建模对象为协定机制。 |
| 20 | approve | ____ | `alliance_org:usmca`；美国—墨西哥—加拿大协定机制；United States-Mexico-Canada Agreement | 美墨加协定、CUSMA、T-MEC、USMCA；`USMCA` | `economic`, `trade` | L：由三方自由贸易委员会及协定委员会机制监督实施。I：覆盖北美货物、服务、投资、劳工及数字贸易规则。 | [USTR: USMCA](https://ustr.gov/trade-agreements/free-trade-agreements/united-states-mexico-canada-agreement) | 不存在；建议 create；保留三种官方简称为 aliases。 |
| 21 | approve | ____ | `alliance_org:asean_china_fta`；中国—东盟自由贸易区机制；ASEAN-China Free Trade Area | 中国东盟自贸区、ACFTA、ASEAN-China FTA；`ACFTA` | `economic`, `trade` | L：中国与东盟通过联合委员会及相关协定机制推进自贸区实施。I：覆盖中国—东盟货物、服务、投资与经贸合作。 | [ASEAN-China economic relations](https://asean.org/our-communities/economic-community/integration-with-global-economy/asean-china-economic-relation/) | 不存在；建议 create。 |
| 22 | approve | ____ | `alliance_org:eaeu`；欧亚经济联盟；Eurasian Economic Union | 欧亚联盟、ЕАЭС、EAEU；`EAEU` | `economic`, `finance`, `trade` | L：由最高欧亚经济理事会、政府间理事会和欧亚经济委员会治理。I：覆盖欧亚区域共同市场、关税及经济政策协调。 | [Eurasian Economic Commission](https://eec.eaeunion.org/en/comission/about/) | 不存在；建议 create。 |
| 23 | approve | ____ | `alliance_org:mercosur`；南方共同市场；Southern Common Market | 南共市、Mercado Común del Sur、MERCOSUR；`MERCOSUR` | `economic`, `trade` | L：由共同市场理事会、共同市场小组等机构推进政府间决策。I：覆盖南美关税、贸易和区域经济一体化。 | [MERCOSUR in brief](https://www.mercosur.int/en/about-mercosur/mercosur-in-brief/) | 不存在；建议 create。 |
| 24 | approve | ____ | `alliance_org:afcfta`；非洲大陆自由贸易区机制；African Continental Free Trade Area | 非洲大陆自贸区、AfCFTA；`AfCFTA` | `development`, `economic`, `trade` | L：由缔约方机构和 AfCFTA 秘书处推进协定实施。I：覆盖非洲大陆货物、服务、投资与市场一体化。 | [AfCFTA Secretariat](https://au-afcfta.org/) | 不存在；建议 create；与 AU 保持独立 identity，隶属关系后审。 |
| 25 | approve | ____ | `alliance_org:efta`；欧洲自由贸易联盟；European Free Trade Association | 欧洲自贸联盟、EFTA；`EFTA` | `economic`, `trade` | L：由理事会及秘书处按《斯德哥尔摩公约》协调成员合作。I：覆盖成员间自由贸易及对外自由贸易协定网络。 | [EFTA: About EFTA](https://www.efta.int/about-efta) | 不存在；建议 create。 |
| 26 | approve | ____ | `alliance_org:sadc`；南部非洲发展共同体；Southern African Development Community | 南共体、SADC；`SADC` | `development`, `economic`, `security`, `trade` | L：由峰会、部长理事会及秘书处推进区域协议和计划。I：覆盖南部非洲发展、经济一体化、贸易与和平安全。 | [SADC: About SADC](https://www.sadc.int/about-sadc) | 不存在；建议 create。 |
| 27 | approve | ____ | `alliance_org:eac`；东非共同体；East African Community | 东共体、EAC；`EAC` | `development`, `economic`, `trade` | L：由国家元首峰会、部长理事会及秘书处推进条约目标。I：覆盖东非关税同盟、共同市场及区域一体化。 | [EAC Overview](https://www.eac.int/overview-of-eac) | 不存在；建议 create。 |
| 28 | approve | ____ | `alliance_org:saarc`；南亚区域合作联盟；South Asian Association for Regional Cooperation | 南盟、SAARC；`SAARC` | `development`, `economic`, `political` | L：由峰会、部长理事会、常设委员会和秘书处协调区域合作。I：覆盖南亚发展、社会经济及跨境合作议题。 | [SAARC: About SAARC](https://www.saarc-sec.org/index.php/about-saarc/about-saarc) | 不存在；建议 create；当前活动连续性在最终 Review 再核验。 |
| 29 | approve | ____ | `alliance_org:caricom`；加勒比共同体；Caribbean Community | 加共体、CARICOM；`CARICOM` | `development`, `economic`, `trade` | L：由政府首脑会议、共同体理事会及秘书处推进共同体事务。I：覆盖加勒比经济一体化、外交协调与功能合作。 | [CARICOM: Who we are](https://caricom.org/our-community/who-we-are/) | 不存在；建议 create。 |
| 30 | approve | ____ | `alliance_org:pacific_islands_forum`；太平洋岛国论坛；Pacific Islands Forum | 太平洋论坛、PIF；`PIF` | `development`, `economic`, `environment`, `political`, `security` | L：由论坛领导人会议和秘书处以共识推动区域优先事项。I：覆盖太平洋区域政治、发展、气候、海洋和安全合作。 | [Pacific Islands Forum: Who we are](https://forumsec.org/who-we-arepacific-islands-forum) | 不存在；建议 create。 |
| 31 | approve | ____ | `alliance_org:opec`；石油输出国组织；Organization of the Petroleum Exporting Countries | 欧佩克、OPEC；`OPEC` | `energy` | L：由大会、理事会和秘书处依组织章程协调石油政策。I：覆盖成员石油政策协调与国际石油市场信息合作。 | [OPEC: About us](https://www.opec.org/about-us.html) | 已存在同 key；名称/aliases 可复用，旧 profile 需转换；建议 keep。 |
| 32 | approve | ____ | `alliance_org:opec_plus`；OPEC+合作机制；Declaration of Cooperation between OPEC and non-OPEC Participating Countries | 欧佩克+、OPEC Plus、Declaration of Cooperation；`OPEC+` | `energy` | L：由 OPEC 与非 OPEC 参与国通过合作宣言、部长级会议及联合机制协调。I：覆盖参与产油经济体的石油产量政策与市场稳定协作。 | [OPEC: Declaration of Cooperation](https://www.opec.org/declaration-of-cooperation.html) | 已存在同 key；canonical name 与正式机制英文别名需 Review，旧 profile 需转换；建议 keep。 |
| 33 | approve | ____ | `alliance_org:iea`；国际能源署；International Energy Agency | IEA；`IEA` | `economic`, `energy`, `security` | L：由治理委员会和秘书处在经合组织框架内推进能源合作。I：覆盖能源安全、政策分析、市场数据和能源转型。 | [IEA: About](https://www.iea.org/about) | 不存在；建议 create；与 OECD 的制度关系留待未来 `part_of` Review。 |
| 34 | approve | ____ | `alliance_org:gecf`；天然气出口国论坛；Gas Exporting Countries Forum | 气体出口国论坛、GECF；`GECF` | `energy` | L：由部长级会议、执行理事会及秘书处协调天然气政策合作。I：覆盖天然气市场对话、数据研究与生产国合作。 | [GECF: About us](https://www.gecf.org/about-us) | 不存在；建议 create。 |
| 35 | approve | ____ | `alliance_org:anrpc`；天然橡胶生产国协会；Association of Natural Rubber Producing Countries | ANRPC；`ANRPC` | `agriculture`, `trade` | L：由成员政府代表机构和秘书处协调天然橡胶产业合作。I：覆盖天然橡胶生产、市场信息、可持续性和贸易议题。 | [ANRPC: About us](https://www.anrpc.org/about-us) | 不存在；建议 create；CSV“卡特尔”表述不进入摘要。 |
| 36 | approve | ____ | `alliance_org:cpopc`；棕榈油生产国理事会；Council of Palm Oil Producing Countries | 棕榈油生产国委员会、CPOPC；`CPOPC` | `agriculture`, `energy`, `trade` | L：由成员政府理事会和秘书处协调棕榈油产业政策。I：覆盖棕榈油生产、贸易、可持续标准及生物能源议题。 | [CPOPC: About us](https://cpopc.net/about-us/) | 不存在；建议 create。 |
| 37 | approve | ____ | `alliance_org:international_coffee_organization`；国际咖啡组织；International Coffee Organization | ICO；`ICO` | `agriculture`, `development`, `trade` | L：由国际咖啡理事会和秘书处依国际咖啡协定运行。I：覆盖全球咖啡政策对话、数据、可持续发展与贸易合作。 | [ICO: About us](https://ico.org/what-we-do/about-us/) | 不存在；建议 create。 |
| 38 | approve | ____ | `alliance_org:international_cocoa_organization`；国际可可组织；International Cocoa Organization | ICCO；`ICCO` | `agriculture`, `development`, `trade` | L：由国际可可理事会及秘书处依国际可可协定运行。I：覆盖可可经济、市场透明度与可持续产业合作。 | [ICCO: About us](https://www.icco.org/about-us/) | 不存在；建议 create。 |
| 39 | approve | ____ | `alliance_org:international_sugar_organization`；国际糖业组织；International Sugar Organization | 国际糖组织、ISO Sugar、ISO；`ISO` | `agriculture`, `trade` | L：由国际糖业理事会和秘书处依国际糖业协定运行。I：覆盖糖、乙醇及相关市场的统计、研究和政策对话。 | [International Sugar Organization](https://www.isosugar.org/aboutus) | 不存在；建议 create；`ISO` 简称与标准化组织冲突，必须进入 alias conflict Review。 |
| 40 | approve | ____ | `alliance_org:cairns_group`；凯恩斯集团；Cairns Group | Cairns Group of Fair Trading Nations；`Cairns Group` | `agriculture`, `trade` | L：由农业出口经济体通过部长级和技术协调参与 WTO 农业谈判。I：覆盖农业贸易自由化、补贴与市场准入政策协调。 | [Cairns Group official site](https://www.cairnsgroup.org/Pages/default.aspx) | 不存在；建议 create；正式 abbreviation 为空，`Cairns Group` 作为名称 alias，不强造缩写。 |
| 41 | approve | ____ | `alliance_org:minerals_security_partnership`；矿产安全伙伴关系；Minerals Security Partnership | 关键矿产安全伙伴关系、MSP；`MSP` | `economic`, `mineral`, `security`, `technology` | L：由参与方政府通过伙伴机制协调负责任关键矿产项目与供应链。I：覆盖关键矿产开采、加工、回收和供应链韧性合作。 | [Minerals Security Partnership](https://www.mineralssecuritypartnership.org/) | 不存在；建议 create；CSV 中文名建议收敛为官方通行“矿产安全伙伴关系”。 |
| 42 | approve | ____ | `alliance_org:imf`；国际货币基金组织；International Monetary Fund | 国际货币基金、IMF；`IMF` | `economic`, `finance`, `governance` | L：由理事会、执行董事会和管理层依协定条款治理。I：覆盖国际货币合作、宏观监督、融资与能力建设。 | [IMF: About](https://www.imf.org/en/About) | 已存在同 key；名称/aliases 可复用，旧 profile 需转换；建议 keep。 |
| 43 | merge | ____ | target `alliance_org:world_bank`；世界银行集团；World Bank Group | 世界银行、WBG、World Bank；`WBG` | `development`, `economic`, `finance` | L：由世界银行集团各机构的治理机构和管理层按各自协定共同运作。I：覆盖发展融资、减贫、基础设施与知识服务。 | [World Bank: Who we are](https://www.worldbank.org/en/who-we-are) | 已有 `alliance_org:world_bank`，CSV 指向同一现实集团；建议保留现有 stable target，规范名称/aliases/profile 逐项更新，不创建第二 active identity。 |
| 44 | approve | ____ | `alliance_org:aiib`；亚洲基础设施投资银行；Asian Infrastructure Investment Bank | 亚投行、AIIB；`AIIB` | `development`, `finance` | L：由理事会、董事会、行长和管理层依协定治理。I：覆盖亚洲及相关区域基础设施和可持续发展融资。 | [AIIB: Who we are](https://www.aiib.org/en/about-aiib/index.html) | 不存在；建议 create。 |
| 45 | approve | ____ | `alliance_org:new_development_bank`；新开发银行；New Development Bank | 金砖国家新开发银行、金砖银行、NDB；`NDB` | `development`, `economic`, `finance` | L：由理事会、董事会和行长依成立协定治理。I：覆盖新兴市场和发展中经济体的基础设施与可持续发展融资。 | [NDB: About us](https://www.ndb.int/about-ndb/) | 不存在；建议 create；与 BRICS 关系后续单独审阅。 |
| 46 | approve | ____ | `alliance_org:cmim`；清迈倡议多边化机制；Chiang Mai Initiative Multilateralisation | 清迈倡议多边机制、CMIM；`CMIM` | `economic`, `finance`, `security` | L：由 ASEAN+3 财长和央行行长进程及相关执行机制治理区域流动性安排。I：覆盖东亚区域流动性支持、危机预防与金融安全网。 | [ASEAN Plus Three finance cooperation](https://aseanplusthree.asean.org/areas-of-cooperation/finance-cooperation/) | 不存在；建议 create；建模对象为多边化安排机制。 |
| 47 | reject | ____ | 不建议分配 `alliance_org` key；丝路基金；Silk Road Fund | SRF；`SRF` | `development`, `finance` | L：由单一国家出资设立的投资机构按公司治理结构运营。I：投资范围涉及共建“一带一路”相关项目，但不构成跨 economy 联盟治理。 | [Silk Road Fund: About us](https://www.silkroadfund.com.cn/enweb/23773/23775/index.html) | 不存在；建议 reject，因为是单一国家投资基金而非跨 economy 联盟/多边机制；不自行改建其他实体类型。 |
| 48 | approve | ____ | `alliance_org:undp`；联合国开发计划署；United Nations Development Programme | 联合国开发署、UNDP；`UNDP` | `development`, `governance` | L：由联合国开发计划署执行局、署长及区域/国家体系治理。I：覆盖减贫、治理、韧性、环境与可持续发展支持。 | [UNDP: About us](https://www.undp.org/about-us) | 不存在；建议 create；与 UN 隶属关系后审。 |
| 49 | approve | ____ | `alliance_org:fao`；联合国粮食及农业组织；Food and Agriculture Organization of the United Nations | 联合国粮农组织、FAO；`FAO` | `agriculture`, `development`, `governance` | L：由大会、理事会、总干事和秘书处依组织章程治理。I：覆盖粮食安全、农业、渔业、林业和农村发展。 | [FAO: About FAO](https://www.fao.org/about/about-fao/en/) | 不存在；建议 create。 |
| 50 | approve | ____ | `alliance_org:who`；世界卫生组织；World Health Organization | 世卫组织、WHO；`WHO` | `governance`, `health` | L：由世界卫生大会、执行委员会和秘书处依组织法治理。I：覆盖全球公共卫生规范、监测、应急和技术合作。 | [WHO: About WHO](https://www.who.int/about) | 不存在；建议 create。 |
| 51 | approve | ____ | `alliance_org:unesco`；联合国教育、科学及文化组织；United Nations Educational, Scientific and Cultural Organization | 联合国教科文组织、UNESCO；`UNESCO` | `culture`, `education`, `governance`, `technology` | L：由大会、执行局和秘书处依组织法治理。I：覆盖教育、科学、文化、信息传播及遗产合作。 | [UNESCO: About us](https://www.unesco.org/en/brief) | 不存在；建议 create。 |
| 52 | approve | ____ | `alliance_org:unep`；联合国环境规划署；United Nations Environment Programme | 联合国环境署、UN Environment、UNEP；`UNEP` | `environment`, `governance` | L：由联合国环境大会、执行主任和秘书处推进环境议程。I：覆盖全球环境评估、规范协调和多边环境行动。 | [UNEP: About](https://www.unep.org/who-we-are/about-us) | 不存在；建议 create。 |
| 53 | approve | ____ | `alliance_org:iaea`；国际原子能机构；International Atomic Energy Agency | 国际原子能总署、IAEA；`IAEA` | `energy`, `governance`, `nuclear`, `security` | L：由大会、理事会和秘书处依《国际原子能机构规约》治理。I：覆盖核保障、核安全、核安保及和平利用核技术。 | [IAEA: Overview](https://www.iaea.org/about/overview) | 不存在；建议 create。 |
| 54 | approve | ____ | `alliance_org:iomed`；国际调解院；International Organization for Mediation | 国际调解组织、IOMed；`IOMed` | `dispute_resolution`, `legal` | L：由建立公约设立的政府间法律组织，通过调解制度和常设机构运作。I：覆盖国家间、国家与外国投资者及国际商事争议的调解。 | [IOMed official site](https://www.iomed.int/) | 不存在；建议 create；CSV `IMC` 与“筹建中”已过时，改为官方 `IOMed`，仍需核验公约生效与 active 状态。 |
| 55 | approve | ____ | `alliance_org:wfp`；世界粮食计划署；World Food Programme | 联合国世界粮食计划署、WFP；`WFP` | `agriculture`, `development`, `humanitarian` | L：由执行局、执行主任及全球运营体系治理。I：覆盖紧急粮食援助、营养、物流和韧性建设。 | [WFP: Who we are](https://www.wfp.org/who-we-are) | 不存在；建议 create。 |
| 56 | approve | ____ | `alliance_org:g77`；七十七国集团；Group of 77 | 七十七国集团和中国、G-77、G77 + China、G77；`G77` | `development`, `economic`, `political` | L：由成员在联合国体系内通过主席国轮值和分会协调共同立场。I：覆盖发展中经济体在发展、经济与联合国议程中的集体谈判。 | [Group of 77: About](https://www.g77.org/doc/) | 不存在；建议 create；“+中国”是合作表述，不直接写入 canonical name。 |
| 57 | defer | ____ | provisional `alliance_org:belt_and_road_initiative`；共建“一带一路”倡议；Belt and Road Initiative | 一带一路、丝绸之路经济带和21世纪海上丝绸之路、BRI；`BRI` | `development`, `economic`, `trade` | L：中国提出并通过双多边合作文件、论坛及项目网络推进，无单一成员制组织治理。I：覆盖基础设施、互联互通、贸易投资和发展合作。 | [Belt and Road Portal](https://eng.yidaiyilu.gov.cn/) | 不存在；建议 defer，需先决定“倡议网络”是否满足 alliance_org identity 边界，不能因项目参与自动形成成员关系。 |
| 58 | approve | ____ | `alliance_org:ipef`；印太经济繁荣框架；Indo-Pacific Economic Framework for Prosperity | 印太经济框架、IPEF；`IPEF` | `economic`, `technology`, `trade` | L：由伙伴经济体通过部长级进程及各支柱协议协调实施。I：覆盖贸易、供应链、清洁经济与公平经济合作。 | [USTR: IPEF](https://ustr.gov/trade-agreements/agreements-under-negotiation/indo-pacific-economic-framework-prosperity-ipef) | 不存在；建议 create；建模对象是框架合作机制，不等同自由贸易协定。 |
| 59 | approve | ____ | `alliance_org:quad`；四方安全对话；Quadrilateral Security Dialogue | 四边机制、Quad；`Quad` | `political`, `security` | L：澳大利亚、印度、日本和美国通过领导人、外长及工作组机制协调。I：覆盖印太海洋、安全、韧性、科技与公共产品合作。 | [Australian DFAT: Quad](https://www.dfat.gov.au/international-relations/regional-architecture/quad) | 不存在；建议 create；CSV 大写 `QUAD` 收敛为官方常用 `Quad` 展示。 |
| 60 | approve | ____ | `alliance_org:brics`；金砖国家合作机制；BRICS | 金砖国家、BRICS Cooperation；`BRICS` | `development`, `economic`, `finance`, `political` | L：通过主席国轮值、领导人峰会、部长级机制及相关机构开展合作。I：覆盖政治安全、经贸金融和人文交流等合作支柱。 | [BRICS official overview](https://brics.br/en/about-the-brics) | 已存在同 key；建议 canonical name 从“金砖国家”收敛为“金砖国家合作机制”并保留旧名 alias，旧 profile 转换；建议 keep。 |
| 61 | defer | ____ | provisional `alliance_org:pgii`；全球基础设施和投资伙伴关系；Partnership for Global Infrastructure and Investment | PGII；`PGII` | `development`, `economic` | L：由七国集团以共同承诺和各成员融资工具推进基础设施投资。I：覆盖发展中经济体基础设施、经济走廊和投资合作。 | [G7 PGII factsheet](https://www.consilium.europa.eu/media/1tylvgk0/g7-2023-factsheet-on-partnership-global-infrastructure-investment.pdf) | 不存在；建议 defer，需确认其是否具有足够独立、持续的治理 identity，而非 G7 项目组合。 |
| 62 | approve | ____ | `alliance_org:us_japan_rok_trilateral`；美日韩三边合作机制；United States-Japan-Republic of Korea Trilateral Partnership | 美日韩三边机制、Camp David trilateral partnership；空简称 | `military`, `political`, `security` | L：三方通过领导人、外长、防长及国家安全协调机制推进合作。I：覆盖东北亚安全、防务协调、经济安全和区域议题。 | [Japan MOFA: Camp David Principles](https://www.mofa.go.jp/a_o/na2/page1e_000744.html) | 不存在；建议 create；没有稳定正式 abbreviation，不把 `—` 或临时缩写写入 profile。 |
| 63 | defer | ____ | provisional `alliance_org:chip_4`；芯片四方协调构想；Chip 4 / Fab 4 | 晶片四方联盟、Fab 4、Chip 4；`Chip 4` | `economic`, `security`, `technology`, `trade` | L：公开材料显示为美、日、韩及中国台湾间的半导体供应链咨询构想，治理结构不清晰。I：拟覆盖半导体供应链、产业政策与技术协作。 | **正式来源 blocker**：未找到由四方共同维护、可确认正式名称和治理状态的权威页面 | 不存在；建议 defer；来源、正式名称、持续机制和简称均未达准入，不得 create。 |
| 64 | approve | ____ | `alliance_org:focac`；中非合作论坛；Forum on China-Africa Cooperation | 中非论坛、FOCAC；`FOCAC` | `development`, `economic`, `political`, `trade` | L：通过峰会、部长级会议、后续行动委员会及论坛机制推进合作。I：覆盖中国与非洲国家间政治、经贸、发展和人文合作。 | [FOCAC official site](http://www.focac.org/eng/) | 不存在；建议 create；不在本阶段读取参与方全集。 |
| 65 | approve | ____ | `alliance_org:china_arab_states_cooperation_forum`；中国—阿拉伯国家合作论坛；China-Arab States Cooperation Forum | 中阿合作论坛、CASCF；`CASCF` | `economic`, `energy`, `political`, `trade` | L：通过部长级会议、高官会和中方/阿方协调机制推进合作。I：覆盖中国与阿拉伯国家间政治、经贸、能源和人文合作。 | [China-Arab States Cooperation Forum](http://www.chinaarabcf.org/eng/) | 不存在；建议 create。 |
| 66 | approve | ____ | `alliance_org:china_celac_forum`；中国—拉共体论坛；China-CELAC Forum | 中拉论坛、中国—拉美和加勒比国家共同体论坛、CELAC Forum；`China-CELAC Forum` | `development`, `economic`, `political`, `trade` | L：通过部长级会议、协调人会议和领域分论坛推进整体合作。I：覆盖中国与拉美和加勒比地区政治、经贸、发展及人文合作。 | [China-CELAC Forum](http://www.chinacelacforum.org/eng/) | 不存在；建议 create；CSV abbreviation 不是真正缩写，保留为空简称并将英文通称放 aliases 的方案需主对话确认。 |
| 67 | approve | ____ | `alliance_org:china_central_asia_summit`；中国—中亚峰会机制；China-Central Asia Summit Mechanism | 中国中亚峰会、中国—中亚机制；空简称 | `economic`, `political`, `security`, `trade` | L：由中国与中亚国家通过元首峰会及常设秘书处安排推进合作。I：覆盖政治互信、经贸互联互通、安全与人文合作。 | [China MFA: China-Central Asia Summit](https://www.mfa.gov.cn/eng/xw/zyxw/202305/t20230519_11080116.html) | 不存在；建议 create；没有正式 abbreviation，不写占位符。 |
| 68 | defer | ____ | provisional `alliance_org:eu_us_ttc`；欧美贸易和技术委员会；EU-US Trade and Technology Council | 美欧贸易与技术委员会、TTC；`TTC` | `economic`, `technology`, `trade` | L：由欧盟与美国的部长级共同主席及工作组协调贸易和技术议题。I：覆盖跨大西洋贸易、技术标准、供应链与经济安全合作。 | [European Commission: EU-US TTC](https://commission.europa.eu/topics/international-partnerships/eu-us-trade-and-technology-council_en) | 不存在；建议 defer，正式 identity 可核验，但需补充 2026 年持续活动/active 状态证据后再决定是否进入 active manifest。 |

## 2. 未出现在 CSV 1—68、但现有 10 条中必须穷尽处置的联盟

下列 3 条连同上表中的 EU、WTO、OPEC、OPEC+、IMF、World Bank、BRICS，合计覆盖现有 10 个 `alliance_org`。本节同样只是 disposition 建议，final decision 留空。

| existing identity | recommendation | final decision | 建议 profile 草案 | 正式来源 | exact diff / disposition 建议 |
|---|---|---|---|---|---|
| `alliance_org:g7`；七国集团；Group of Seven；aliases `G7`；abbr `G7` | approve | ____ | categories `economic`, `finance`, `political`；L：通过主席国轮值、领导人峰会及部长级会议协调共同政策。I：覆盖全球经济、金融、外交、安全及发展议题。 | [G7 official site](https://www.g7italy.it/en/) | CSV 未列但现有 identity 合法；建议 keep，不得因未进 CSV 而 inactivate；正式来源需更新为执行时主席国或 G7 文档库。 |
| `alliance_org:g20`；二十国集团；Group of Twenty；aliases `G20`；abbr `G20` | approve | ____ | categories `development`, `economic`, `finance`, `governance`；L：通过轮值主席国、领导人峰会、财金轨道和协调人轨道运行。I：覆盖国际经济合作、金融稳定、发展和全球治理议题。 | [G20 official site](https://www.g20.org/) | CSV 未列但现有 identity 合法；建议 keep，不得因未进 CSV 而 inactivate。 |
| `alliance_org:oecd`；经济合作与发展组织；Organisation for Economic Co-operation and Development；aliases 经合组织、OECD；abbr `OECD` | approve | ____ | categories `development`, `economic`, `governance`, `trade`；L：由理事会、各委员会和秘书处推进成员间政策合作。I：覆盖经济、税收、贸易、发展、治理与政策标准。 | [OECD: About](https://www.oecd.org/en/about.html) | CSV 未列但现有 identity 合法；建议 keep，不得因未进 CSV 而 inactivate。 |

### 现有 10 条覆盖断言（草案）

```text
existing keys covered = {
  alliance_org:opec_plus, alliance_org:opec, alliance_org:g7,
  alliance_org:g20, alliance_org:wto, alliance_org:imf,
  alliance_org:world_bank, alliance_org:oecd, alliance_org:eu,
  alliance_org:brics
}
recommended existing dispositions = 10 keep targets, 0 inactivate, 0 merge source
```

其中 CSV 第 43 条 recommendation=`merge` 表示把候选语义收敛到既有 `alliance_org:world_bank` target，而不是把该既有 target 当作 merge source；因此当前草案没有建议停用任何现有 10 条。该断言仍须主对话逐项批准，不能直接成为 manifest。

## 3. CSV 第 69—85 条排除记录

这些条目都是资源/商品或观察性集中度主题，不是 `alliance_org`。只记录未来可能的建模方向；不创建实体、关系、seed 或后续 change。

| No. | CSV 条目 | 排除原因 | 未来候选方向（仅记录） |
|---:|---|---|---|
| 69 | 稀土资源 | 资源品类，不是组织 | `commodity` / `chain_node` / supply observation |
| 70 | 锂资源 | 资源品类，不是组织 | `commodity` / `chain_node` / supply observation |
| 71 | 钴资源 | 资源品类，不是组织 | `commodity` / `chain_node` / supply observation |
| 72 | 铁矿石 | 资源品类，不是组织 | `commodity` / `chain_node` / trade observation |
| 73 | 镍资源 | 资源品类，不是组织 | `commodity` / `chain_node` / supply observation |
| 74 | 铜矿资源 | 资源品类，不是组织 | `commodity` / `chain_node` / supply observation |
| 75 | 铂族金属 | 资源品类，不是组织 | `commodity` / `chain_node` / supply observation |
| 76 | 铬矿资源 | 资源品类，不是组织 | `commodity` / `chain_node` / supply observation |
| 77 | 工业硅 | 材料品类，不是组织 | `commodity` / `chain_node` / capacity observation |
| 78 | 天然石墨 | 资源品类，不是组织 | `commodity` / `chain_node` / supply observation |
| 79 | 大豆 | 农产品，不是组织 | `commodity` / agriculture `chain_node` / trade observation |
| 80 | 小麦 | 农产品，不是组织 | `commodity` / agriculture `chain_node` / trade observation |
| 81 | 玉米 | 农产品，不是组织 | `commodity` / agriculture `chain_node` / trade observation |
| 82 | 大米 | 农产品，不是组织 | `commodity` / agriculture `chain_node` / trade observation |
| 83 | 棉花 | 农产品/纺织原料，不是组织 | `commodity` / `chain_node` / trade observation |
| 84 | 牛肉 | 农产品，不是组织 | `commodity` / agriculture `chain_node` / trade observation |
| 85 | 猪肉 | 农产品，不是组织 | `commodity` / agriculture `chain_node` / price observation |

## 4. 主对话逐项 Review 清单

主对话需要对每一行明确填写 final decision，并在通过 task 2.3 前处理以下 blocker：

1. 逐项确认 68 条的 `approve/reject/merge/defer`；任何空白 decision 都阻止 approved manifest。
2. 逐项确认拟用 stable key、中文/英文规范名、aliases、abbreviation、categories 和两个摘要。
3. 确认第 43 条 World Bank Group 收敛到既有 `alliance_org:world_bank`，不创建第二 active identity。
4. 确认既有 G7、G20、OECD 虽不在 CSV 1—68 仍建议 keep；不得按 CSV 缺席自动 inactivate。
5. 重点决策 defer 项：57 BRI、61 PGII、63 Chip 4、68 TTC；其中 Chip 4 缺少共同正式来源，是明确 blocker。
6. 确认第 47 条 Silk Road Fund 的 reject 建议；本 change 不自行改建其他 entity type。
7. 复核带 alias 冲突或无简称项：International Sugar Organization 的 `ISO`、China-CELAC Forum、China-Central Asia Summit、US-Japan-ROK trilateral。
8. 确认 69—85 全部排除出 alliance_org；只保留未来候选方向记录。

## 5. 停止点

tasks 2.1—2.2 已准备供 Review，task 2.3 未通过。本文没有成员国全集、economy 差异、成员关系或可执行 seed；在主对话逐项确认前不得启动 C。
