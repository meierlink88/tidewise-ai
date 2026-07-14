# Package 2：Economy + Relationship Candidate Review（R0 v1，待人工 Review）

## 0. 状态与边界

本 package 以 `approved-alliance-manifest.md` v1（canonical checksum `4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a`）为唯一联盟输入。它是只读候选 Review，不是 approved economy/member manifest，不生成 seed，不修改源码/migration，不连接 PostgreSQL/Neo4j。

- 审计基线：`economies.json` 50 条，SHA-256 `c4fed107d5309d5f32606583bd2b4db84fa1b537b1b7b8c59be84150acb94696`；`member_of.json` 223 条 active，SHA-256 `14b935e12df0ac8ca49b16cdf194c4c4d70862b6dcca20437a474ff55f5da9c5`。
- 45 个联盟 model：10 resolved formal sets、13 blocked formal sets、1 rotating/term-bound blocked、21 participant/signatory/framework/no-formal-membership。
- 只有 10 个 resolved formal sets 生成候选：133 条 formal-active tuples；31 条复用现有 edge，102 条 create。
- economy 并集：79；现有 reuse 35，create 44；现有 50 中有 15 条不在本次并集，全部进入保护快照，不得停用。
- 223 条现有 active edge 穷尽 disposition：keep 31、proposed inactivate 32、blocked/source-conflict 160；blocked 行在 Review 解决前禁止 R2B Write。
- `led_by`/`part_of` 附录为空并明确排除：Excel leadership 文本不能自动造边，本轮没有满足“无需主观推断 + 完整正式证据”的候选。

## 1. 45 个 Alliance Membership Model 与 Source Register

`resolved` 表示官方来源可在本核验时点形成完整 formal-active economy set；`blocked` 表示精确缺口仍需主对话决策或更新契约。`not_applicable` 不生成 member_of。

| Alliance | Membership model | 状态 | 准入/缺口 | 官方来源 | verified_at |
|---|---|---|---|---|---|
| `alliance_org:g20` | `formal_member_set` | blocked | 官方集合含 19 个国家、EU 与 AU；现有 member_of 仅允许 economy -> alliance_org，无法在不伪装聚合组织端点的情况下穷尽。 | [https://g20.org/about-g20](https://g20.org/about-g20) | `2026-07-14` |
| `alliance_org:g7` | `formal_member_set` | resolved | 7 个国家成员；EU 是 fully involved participant，不作为 G7 member_of。 | [https://www.elysee.fr/en/G7evian/2025/12/13/what-is-the-g7-1](https://www.elysee.fr/en/G7evian/2025/12/13/what-is-the-g7-1) | `2026-07-14` |
| `alliance_org:unsc` | `rotating_or_term_bound` | blocked | 15 席含 10 个有任期的非常任理事国；当前 edge 契约无 valid_from/valid_to，不能准确表达。 | [https://main.un.org/securitycouncil/en/content/current-members](https://main.un.org/securitycouncil/en/content/current-members) | `2026-07-14` |
| `alliance_org:nato` | `formal_member_set` | resolved | 官方页面列明 32 个 member countries。 | [https://www.nato.int/en/about-us/organization/nato-member-countries](https://www.nato.int/en/about-us/organization/nato-member-countries) | `2026-07-14` |
| `alliance_org:sco` | `formal_member_set` | resolved | 官方 About 页面区分 10 个 member states、observers 与 dialogue partners。 | [https://eng.sectsco.org/20170109/192193.html](https://eng.sectsco.org/20170109/192193.html) | `2026-07-14` |
| `alliance_org:five_eyes` | `participant/signatory/framework/no_formal_membership` | not_applicable | 情报合作框架，不将五个参与方机械解释为组织正式成员。 | [https://www.dni.gov/index.php/ncsc-how-we-work/217-about/organization/icig-pages/2660-icig-fiorc](https://www.dni.gov/index.php/ncsc-how-we-work/217-about/organization/icig-pages/2660-icig-fiorc) | `2026-07-14` |
| `alliance_org:eu` | `formal_member_set` | resolved | 官方页面列明 27 个 Member States。 | [https://european-union.europa.eu/easy-read_en](https://european-union.europa.eu/easy-read_en) | `2026-07-14` |
| `alliance_org:asean` | `formal_member_set` | resolved | Timor-Leste 已于 2025-10-26 成为第 11 个 Member State；2026 材料确认 full and active participation。 | [https://asean.org/wp-content/uploads/2026/01/FINAL-Press-Statement-by-the-Chair-of-the-ASEAN-Foreign-Ministers-Retreat.pdf](https://asean.org/wp-content/uploads/2026/01/FINAL-Press-Statement-by-the-Chair-of-the-ASEAN-Foreign-Ministers-Retreat.pdf) | `2026-07-14` |
| `alliance_org:au` | `formal_member_set` | blocked | 55 个成员中含 Sahrawi Arab Democratic Republic；需先单独批准其 ISO/territory identity，禁止只生成其余 54 条。 | [https://oau60.au.int/en/member-states](https://oau60.au.int/en/member-states) | `2026-07-14` |
| `alliance_org:las` | `formal_member_set` | blocked | 需取得可固定版本且明确 suspended/returned 状态的官方 active roster。 | [https://www.leagueofarabstates.net/](https://www.leagueofarabstates.net/) | `2026-07-14` |
| `alliance_org:gcc` | `formal_member_set` | resolved | GCC Charter/About 列明 6 个 member states。 | [https://www.gcc-sg.org/en/AboutUs/Pages/default.aspx](https://www.gcc-sg.org/en/AboutUs/Pages/default.aspx) | `2026-07-14` |
| `alliance_org:oic` | `formal_member_set` | blocked | 官方 57 国目录需要冻结可审计快照并逐项排除 suspended 状态后才能形成 manifest。 | [https://www.oic-oci.org/states/?lan=en](https://www.oic-oci.org/states/?lan=en) | `2026-07-14` |
| `alliance_org:oas` | `formal_member_set` | blocked | Nicaragua 退出及 Cuba 参与状态要求按核验日形成 active roster，当前未得到无冲突官方全集。 | [https://www.oas.org/en/member_states/default.asp](https://www.oas.org/en/member_states/default.asp) | `2026-07-14` |
| `alliance_org:wto` | `formal_member_set` | blocked | WTO members 包含独立关税区；需先审阅 territory economy identity 与当前 membership effective date。 | [https://www.wto.org/english/thewto_e/whatis_e/tif_e/org6_e.htm](https://www.wto.org/english/thewto_e/whatis_e/tif_e/org6_e.htm) | `2026-07-14` |
| `alliance_org:rcep` | `participant/signatory/framework/no_formal_membership` | not_applicable | 自由贸易协定参加方/缔约方，不自动等同联盟组织正式成员。 | [https://rcepsec.org/](https://rcepsec.org/) | `2026-07-14` |
| `alliance_org:cptpp` | `participant/signatory/framework/no_formal_membership` | not_applicable | 条约 parties/accession，不自动写 member_of。 | [https://www.mfat.govt.nz/en/trade/free-trade-agreements/free-trade-agreements-in-force/cptpp](https://www.mfat.govt.nz/en/trade/free-trade-agreements/free-trade-agreements-in-force/cptpp) | `2026-07-14` |
| `alliance_org:usmca` | `participant/signatory/framework/no_formal_membership` | not_applicable | 三方贸易协定，不建组织正式成员关系。 | [https://ustr.gov/trade-agreements/free-trade-agreements/united-states-mexico-canada-agreement](https://ustr.gov/trade-agreements/free-trade-agreements/united-states-mexico-canada-agreement) | `2026-07-14` |
| `alliance_org:cafta` | `participant/signatory/framework/no_formal_membership` | not_applicable | 中国—东盟自贸协定框架，不复制 ASEAN membership。 | [https://fta.mofcom.gov.cn/topic/chinaasean.shtml](https://fta.mofcom.gov.cn/topic/chinaasean.shtml) | `2026-07-14` |
| `alliance_org:eaeu` | `formal_member_set` | resolved | 官方页面列明 5 个 Member-States。 | [https://www.eaeunion.org/?about-info=&lang=en](https://www.eaeunion.org/?about-info=&lang=en) | `2026-07-14` |
| `alliance_org:mercosur` | `formal_member_set` | blocked | Bolivia 加入与 Venezuela suspended disposition 必须先冻结同一时点的 active full-member roster。 | [https://www.mercosur.int/en/about-mercosur/mercosur-countries/](https://www.mercosur.int/en/about-mercosur/mercosur-countries/) | `2026-07-14` |
| `alliance_org:afcfta` | `participant/signatory/framework/no_formal_membership` | not_applicable | 协定签署/批准状态不直接等同 alliance_org 正式成员。 | [https://au-afcfta.org/](https://au-afcfta.org/) | `2026-07-14` |
| `alliance_org:opec` | `formal_member_set` | resolved | 官方 Member Countries 页面列明 12 个正式成员。 | [https://www.opec.org/member-countries.html](https://www.opec.org/member-countries.html) | `2026-07-14` |
| `alliance_org:opec_plus` | `participant/signatory/framework/no_formal_membership` | not_applicable | Declaration of Cooperation 参与机制，不把参与方伪装为 OPEC+ 正式成员。 | [https://www.opec.org/declaration-of-cooperation.html](https://www.opec.org/declaration-of-cooperation.html) | `2026-07-14` |
| `alliance_org:iea` | `formal_member_set` | blocked | 官方 countries 页面混列 members、association 与 accession；需固定 member-only 导出。 | [https://www.iea.org/about/membership](https://www.iea.org/about/membership) | `2026-07-14` |
| `alliance_org:gecf` | `formal_member_set` | resolved | 官方 Overview 明列 12 个 Member Countries，并与 observers 分开。 | [https://www.gecf.org/About-Us/Overview](https://www.gecf.org/About-Us/Overview) | `2026-07-14` |
| `alliance_org:msp` | `participant/signatory/framework/no_formal_membership` | not_applicable | 伙伴关系 participant/partner 不是组织正式 membership。 | [https://www.state.gov/minerals-security-partnership](https://www.state.gov/minerals-security-partnership) | `2026-07-14` |
| `alliance_org:imf` | `formal_member_set` | blocked | 191 个 members 含 ISO identity 例外，需冻结官方按日期名单并先审阅例外 identity。 | [https://www.imf.org/external/np/sec/memdir/memdate.htm](https://www.imf.org/external/np/sec/memdir/memdate.htm) | `2026-07-14` |
| `alliance_org:world_bank` | `participant/signatory/framework/no_formal_membership` | blocked | WBG 是五个机构聚合，各机构 member set 不同；不能用单一 WBG member_of 集合。 | [https://www.worldbank.org/en/about/leadership/members](https://www.worldbank.org/en/about/leadership/members) | `2026-07-14` |
| `alliance_org:aiib` | `formal_member_set` | blocked | 需区分 full member 与 prospective member，并审阅非主权/地区成员 identity。 | [https://www.aiib.org/en/about-aiib/governance/members-of-bank/index.html](https://www.aiib.org/en/about-aiib/governance/members-of-bank/index.html) | `2026-07-14` |
| `alliance_org:ndb` | `formal_member_set` | blocked | 需从官方页面固定 effective member 与 prospective member 边界。 | [https://www.ndb.int/about-ndb/members/](https://www.ndb.int/about-ndb/members/) | `2026-07-14` |
| `alliance_org:cmim` | `participant/signatory/framework/no_formal_membership` | not_applicable | 货币互换安排 participants，不自动建 alliance membership。 | [https://amro-asia.org/about-amro/cmim/](https://amro-asia.org/about-amro/cmim/) | `2026-07-14` |
| `alliance_org:who` | `formal_member_set` | blocked | 成员状态受 2026 退出生效事件影响；需取得核验日官方 current roster 与冲突报告。 | [https://www.who.int/countries](https://www.who.int/countries) | `2026-07-14` |
| `alliance_org:iaea` | `formal_member_set` | blocked | 官方成员含 Holy See 等 ISO/identity 边界项；需先逐项批准 identity 例外。 | [https://www.iaea.org/about/governance/list-of-member-states](https://www.iaea.org/about/governance/list-of-member-states) | `2026-07-14` |
| `alliance_org:bri` | `participant/signatory/framework/no_formal_membership` | not_applicable | 倡议合作参与不等于具有正式成员制度的组织。 | [https://eng.yidaiyilu.gov.cn/](https://eng.yidaiyilu.gov.cn/) | `2026-07-14` |
| `alliance_org:ipef` | `participant/signatory/framework/no_formal_membership` | not_applicable | 谈判/框架 partners 不写 member_of。 | [https://ustr.gov/trade-agreements/agreements-under-negotiation/indo-pacific-economic-framework-prosperity-ipef](https://ustr.gov/trade-agreements/agreements-under-negotiation/indo-pacific-economic-framework-prosperity-ipef) | `2026-07-14` |
| `alliance_org:quad` | `participant/signatory/framework/no_formal_membership` | not_applicable | 领导人协调机制，不推导正式组织 membership。 | [https://www.dfat.gov.au/international-relations/regional-architecture/quad](https://www.dfat.gov.au/international-relations/regional-architecture/quad) | `2026-07-14` |
| `alliance_org:brics` | `formal_member_set` | resolved | 官方 Brazil 页面明确 11 个 full members；partners 排除。 | [https://brics.br/en/about-the-brics](https://brics.br/en/about-the-brics) | `2026-07-14` |
| `alliance_org:pgii` | `participant/signatory/framework/no_formal_membership` | not_applicable | G7 投资伙伴计划，不另建成员关系。 | [https://www.consilium.europa.eu/media/1tylvgk0/g7-2023-factsheet-on-partnership-global-infrastructure-investment.pdf](https://www.consilium.europa.eu/media/1tylvgk0/g7-2023-factsheet-on-partnership-global-infrastructure-investment.pdf) | `2026-07-14` |
| `alliance_org:ujr` | `participant/signatory/framework/no_formal_membership` | not_applicable | 三边合作机制，不推导正式成员制度。 | [https://www.whitehouse.gov/briefing-room/statements-releases/2023/08/18/camp-david-principles/](https://www.whitehouse.gov/briefing-room/statements-releases/2023/08/18/camp-david-principles/) | `2026-07-14` |
| `alliance_org:chip_4` | `participant/signatory/framework/no_formal_membership` | not_applicable | 缺少可审计的正式组织成员契约；不因俗称覆盖四方而建边。 | 无可审计正式来源；缺口 | `2026-07-14` |
| `alliance_org:focac` | `participant/signatory/framework/no_formal_membership` | not_applicable | 论坛参与范围不等同正式组织 membership。 | [https://www.focac.org/eng/](https://www.focac.org/eng/) | `2026-07-14` |
| `alliance_org:cascf` | `participant/signatory/framework/no_formal_membership` | not_applicable | 论坛/合作机制不自动生成 member_of。 | [https://www.mfa.gov.cn/eng/wjb_663304/zzjg_663340/xybfs_663590/xwlb_663592/](https://www.mfa.gov.cn/eng/wjb_663304/zzjg_663340/xybfs_663590/xwlb_663592/) | `2026-07-14` |
| `alliance_org:celac` | `participant/signatory/framework/no_formal_membership` | not_applicable | 中国—拉共体论坛参与范围不等同本联盟实体正式成员。 | [https://www.chinacelacforum.org/eng/](https://www.chinacelacforum.org/eng/) | `2026-07-14` |
| `alliance_org:ccas` | `participant/signatory/framework/no_formal_membership` | not_applicable | 峰会/机制参与方不自动写正式组织 membership。 | [https://www.mfa.gov.cn/eng/xw/wjbxw/202504/t20250426_11605128.html](https://www.mfa.gov.cn/eng/xw/wjbxw/202504/t20250426_11605128.html) | `2026-07-14` |
| `alliance_org:ttc` | `participant/signatory/framework/no_formal_membership` | not_applicable | 欧盟页面已 Archived / no longer updated，且最后部长级会议为 2024；不进入 active member manifest。 | [https://digital-strategy.ec.europa.eu/en/policies/trade-and-technology-council](https://digital-strategy.ec.europa.eu/en/policies/trade-and-technology-council) | `2026-07-14` |

## 2. Economy Target Manifest Candidate

- canonical checksum：`95613a931adf3d7231cbb1d311e5051f3695d9da40c60bbeeccb39d006118cb3`。
- 每行 identity code 共用 [ISO 3166 Online Browsing Platform](https://www.iso.org/obp/ui/#search)；currency code 共用 [ISO 4217 维护机构 SIX](https://www.six-group.com/en/products-services/financial-information/data-standards.html)。成员准入来源见第 1 节和对应第 3 节 bulk official source。中文名与 aliases 是候选显示值，不替代本 Package 人工 Review。
- identity kind 全部为 `sovereign_state`；`economy:eu`、`economy:global` 不在国家并集中，也不替代任何国家。
- 兼容 region 的待决歧义：`AM`、`CY` 暂列 `europe_asia`，`TT` 暂列 `north_america`；Review 可逐项修订，不自动另建 region taxonomy。

| entity key | 规范中文名 | 英文名 / aliases | identity_kind | ISO | currency | region | exact diff | Review note |
|---|---|---|---|---|---|---|---|---|
| `economy:ae` | 阿联酋 | United Arab Emirates；`AE` | sovereign_state | `AE` | `AED` | `middle_east` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:al` | 阿尔巴尼亚 | Albania；`AL` | sovereign_state | `AL` | `ALL` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:am` | 亚美尼亚 | Armenia；`AM` | sovereign_state | `AM` | `AMD` | `europe_asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:at` | 奥地利 | Austria；`AT` | sovereign_state | `AT` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:be` | 比利时 | Belgium；`BE` | sovereign_state | `BE` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:bg` | 保加利亚 | Bulgaria；`BG` | sovereign_state | `BG` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:bh` | 巴林 | Bahrain；`BH` | sovereign_state | `BH` | `BHD` | `middle_east` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:bn` | 文莱 | Brunei；`BN` | sovereign_state | `BN` | `BND` | `asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:bo` | 玻利维亚 | Bolivia；`BO` | sovereign_state | `BO` | `BOB` | `south_america` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:br` | 巴西 | Brazil；`BR` | sovereign_state | `BR` | `BRL` | `south_america` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:by` | 白俄罗斯 | Belarus；`BY` | sovereign_state | `BY` | `BYN` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:ca` | 加拿大 | Canada；`CA` | sovereign_state | `CA` | `CAD` | `north_america` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:cg` | 刚果共和国 | Congo - Brazzaville；`CG` | sovereign_state | `CG` | `XAF` | `africa` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:cn` | 中国 | China；`CN` | sovereign_state | `CN` | `CNY` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:cy` | 塞浦路斯 | Cyprus；`CY` | sovereign_state | `CY` | `EUR` | `europe_asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:cz` | 捷克 | Czechia；`CZ` | sovereign_state | `CZ` | `CZK` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:de` | 德国 | Germany；`DE` | sovereign_state | `DE` | `EUR` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:dk` | 丹麦 | Denmark；`DK` | sovereign_state | `DK` | `DKK` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:dz` | 阿尔及利亚 | Algeria；`DZ` | sovereign_state | `DZ` | `DZD` | `africa` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:ee` | 爱沙尼亚 | Estonia；`EE` | sovereign_state | `EE` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:eg` | 埃及 | Egypt；`EG` | sovereign_state | `EG` | `EGP` | `africa` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:es` | 西班牙 | Spain；`ES` | sovereign_state | `ES` | `EUR` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:et` | 埃塞俄比亚 | Ethiopia；`ET` | sovereign_state | `ET` | `ETB` | `africa` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:fi` | 芬兰 | Finland；`FI` | sovereign_state | `FI` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:fr` | 法国 | France；`FR` | sovereign_state | `FR` | `EUR` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:ga` | 加蓬 | Gabon；`GA` | sovereign_state | `GA` | `XAF` | `africa` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:gb` | 英国 | United Kingdom；`GB` | sovereign_state | `GB` | `GBP` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:gq` | 赤道几内亚 | Equatorial Guinea；`GQ` | sovereign_state | `GQ` | `XAF` | `africa` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:gr` | 希腊 | Greece；`GR` | sovereign_state | `GR` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:hr` | 克罗地亚 | Croatia；`HR` | sovereign_state | `HR` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:hu` | 匈牙利 | Hungary；`HU` | sovereign_state | `HU` | `HUF` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:id` | 印度尼西亚 | Indonesia；`ID` | sovereign_state | `ID` | `IDR` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:ie` | 爱尔兰 | Ireland；`IE` | sovereign_state | `IE` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:in` | 印度 | India；`IN` | sovereign_state | `IN` | `INR` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:iq` | 伊拉克 | Iraq；`IQ` | sovereign_state | `IQ` | `IQD` | `middle_east` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:ir` | 伊朗 | Iran；`IR` | sovereign_state | `IR` | `IRR` | `middle_east` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:is` | 冰岛 | Iceland；`IS` | sovereign_state | `IS` | `ISK` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:it` | 意大利 | Italy；`IT` | sovereign_state | `IT` | `EUR` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:jp` | 日本 | Japan；`JP` | sovereign_state | `JP` | `JPY` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:kg` | 吉尔吉斯斯坦 | Kyrgyzstan；`KG` | sovereign_state | `KG` | `KGS` | `central_asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:kh` | 柬埔寨 | Cambodia；`KH` | sovereign_state | `KH` | `KHR` | `asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:kw` | 科威特 | Kuwait；`KW` | sovereign_state | `KW` | `KWD` | `middle_east` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:kz` | 哈萨克斯坦 | Kazakhstan；`KZ` | sovereign_state | `KZ` | `KZT` | `central_asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:la` | 老挝 | Laos；`LA` | sovereign_state | `LA` | `LAK` | `asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:lt` | 立陶宛 | Lithuania；`LT` | sovereign_state | `LT` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:lu` | 卢森堡 | Luxembourg；`LU` | sovereign_state | `LU` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:lv` | 拉脱维亚 | Latvia；`LV` | sovereign_state | `LV` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:ly` | 利比亚 | Libya；`LY` | sovereign_state | `LY` | `LYD` | `africa` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:me` | 黑山 | Montenegro；`ME` | sovereign_state | `ME` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:mk` | 北马其顿 | North Macedonia；`MK` | sovereign_state | `MK` | `MKD` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:mm` | 缅甸 | Myanmar (Burma)；`MM` | sovereign_state | `MM` | `MMK` | `asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:mt` | 马耳他 | Malta；`MT` | sovereign_state | `MT` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:my` | 马来西亚 | Malaysia；`MY` | sovereign_state | `MY` | `MYR` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:ng` | 尼日利亚 | Nigeria；`NG` | sovereign_state | `NG` | `NGN` | `africa` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:nl` | 荷兰 | Netherlands；`NL` | sovereign_state | `NL` | `EUR` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:no` | 挪威 | Norway；`NO` | sovereign_state | `NO` | `NOK` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:om` | 阿曼 | Oman；`OM` | sovereign_state | `OM` | `OMR` | `middle_east` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:ph` | 菲律宾 | Philippines；`PH` | sovereign_state | `PH` | `PHP` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:pk` | 巴基斯坦 | Pakistan；`PK` | sovereign_state | `PK` | `PKR` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:pl` | 波兰 | Poland；`PL` | sovereign_state | `PL` | `PLN` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:pt` | 葡萄牙 | Portugal；`PT` | sovereign_state | `PT` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:qa` | 卡塔尔 | Qatar；`QA` | sovereign_state | `QA` | `QAR` | `middle_east` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:ro` | 罗马尼亚 | Romania；`RO` | sovereign_state | `RO` | `RON` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:ru` | 俄罗斯 | Russia；`RU` | sovereign_state | `RU` | `RUB` | `europe_asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:sa` | 沙特阿拉伯 | Saudi Arabia；`SA` | sovereign_state | `SA` | `SAR` | `middle_east` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:se` | 瑞典 | Sweden；`SE` | sovereign_state | `SE` | `SEK` | `europe` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:sg` | 新加坡 | Singapore；`SG` | sovereign_state | `SG` | `SGD` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:si` | 斯洛文尼亚 | Slovenia；`SI` | sovereign_state | `SI` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:sk` | 斯洛伐克 | Slovakia；`SK` | sovereign_state | `SK` | `EUR` | `europe` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:th` | 泰国 | Thailand；`TH` | sovereign_state | `TH` | `THB` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:tj` | 塔吉克斯坦 | Tajikistan；`TJ` | sovereign_state | `TJ` | `TJS` | `central_asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:tl` | 东帝汶 | Timor-Leste；`TL` | sovereign_state | `TL` | `USD` | `asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:tr` | 土耳其 | Türkiye；`TR` | sovereign_state | `TR` | `TRY` | `europe_asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:tt` | 特立尼达和多巴哥 | Trinidad & Tobago；`TT` | sovereign_state | `TT` | `TTD` | `north_america` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:us` | 美国 | United States；`US` | sovereign_state | `US` | `USD` | `north_america` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:uz` | 乌兹别克斯坦 | Uzbekistan；`UZ` | sovereign_state | `UZ` | `UZS` | `central_asia` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:ve` | 委内瑞拉 | Venezuela；`VE` | sovereign_state | `VE` | `VES` | `south_america` | create | 拟新增；中文名/英文 alias/ISO/currency/region 均待本 Package 人工确认 |
| `economy:vn` | 越南 | Vietnam；`VN` | sovereign_state | `VN` | `VND` | `asia` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |
| `economy:za` | 南非 | South Africa；`ZA` | sovereign_state | `ZA` | `ZAR` | `africa` | reuse | 复用现有稳定 identity/profile；aliases 在未来 R2A exact diff 再核对 |

### Economy Exception / Protection

- exception candidate：0。没有任何现有 economy 因“不属于本次已解析联盟并集”而 merge/inactivate。
- protected existing keys（15）：`economy:ar`, `economy:au`, `economy:bd`, `economy:ch`, `economy:cl`, `economy:eu`, `economy:global`, `economy:hk`, `economy:il`, `economy:kr`, `economy:ma`, `economy:mx`, `economy:nz`, `economy:tw`, `economy:ua`。
- 未来 R2A Query 必须证明上述 protected key/UUID/status 原样保持；EU/GLOBAL 聚合继续存在但不进入本批 member_of。

## 3. Formal-Active Member Of Candidate Manifest

- canonical checksum：`c3d652571fa93307088633cbfe06dce18b1257d8ac2b3ee2ae88c5a27e69fcf7`。
- 方向固定为 `economy -> alliance_org`；每行身份均为 formal_active，来源按联盟批量复用，不逐边重复检索。
- candidate tuples = 133；duplicate tuples = 0；orphan endpoints after approved economy creates = 0；wrong direction/type = 0；non-formal identities = 0。

| edge key | direction | membership status | bulk official source | verified_at | exact diff |
|---|---|---|---|---|---|
| `relationship:ca_member_of_g7` | `economy:ca` → `alliance_org:g7` | formal_active | G7 法国 2026 主席国官网 | `2026-07-14` | keep |
| `relationship:de_member_of_g7` | `economy:de` → `alliance_org:g7` | formal_active | G7 法国 2026 主席国官网 | `2026-07-14` | keep |
| `relationship:fr_member_of_g7` | `economy:fr` → `alliance_org:g7` | formal_active | G7 法国 2026 主席国官网 | `2026-07-14` | keep |
| `relationship:gb_member_of_g7` | `economy:gb` → `alliance_org:g7` | formal_active | G7 法国 2026 主席国官网 | `2026-07-14` | keep |
| `relationship:it_member_of_g7` | `economy:it` → `alliance_org:g7` | formal_active | G7 法国 2026 主席国官网 | `2026-07-14` | keep |
| `relationship:jp_member_of_g7` | `economy:jp` → `alliance_org:g7` | formal_active | G7 法国 2026 主席国官网 | `2026-07-14` | keep |
| `relationship:us_member_of_g7` | `economy:us` → `alliance_org:g7` | formal_active | G7 法国 2026 主席国官网 | `2026-07-14` | keep |
| `relationship:al_member_of_nato` | `economy:al` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:be_member_of_nato` | `economy:be` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:bg_member_of_nato` | `economy:bg` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:ca_member_of_nato` | `economy:ca` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:cz_member_of_nato` | `economy:cz` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:de_member_of_nato` | `economy:de` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:dk_member_of_nato` | `economy:dk` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:ee_member_of_nato` | `economy:ee` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:es_member_of_nato` | `economy:es` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:fi_member_of_nato` | `economy:fi` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:fr_member_of_nato` | `economy:fr` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:gb_member_of_nato` | `economy:gb` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:gr_member_of_nato` | `economy:gr` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:hr_member_of_nato` | `economy:hr` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:hu_member_of_nato` | `economy:hu` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:is_member_of_nato` | `economy:is` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:it_member_of_nato` | `economy:it` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:lt_member_of_nato` | `economy:lt` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:lu_member_of_nato` | `economy:lu` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:lv_member_of_nato` | `economy:lv` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:me_member_of_nato` | `economy:me` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:mk_member_of_nato` | `economy:mk` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:nl_member_of_nato` | `economy:nl` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:no_member_of_nato` | `economy:no` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:pl_member_of_nato` | `economy:pl` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:pt_member_of_nato` | `economy:pt` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:ro_member_of_nato` | `economy:ro` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:se_member_of_nato` | `economy:se` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:si_member_of_nato` | `economy:si` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:sk_member_of_nato` | `economy:sk` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:tr_member_of_nato` | `economy:tr` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:us_member_of_nato` | `economy:us` → `alliance_org:nato` | formal_active | NATO Member Countries | `2026-07-14` | create |
| `relationship:by_member_of_sco` | `economy:by` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:cn_member_of_sco` | `economy:cn` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:in_member_of_sco` | `economy:in` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:ir_member_of_sco` | `economy:ir` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:kg_member_of_sco` | `economy:kg` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:kz_member_of_sco` | `economy:kz` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:pk_member_of_sco` | `economy:pk` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:ru_member_of_sco` | `economy:ru` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:tj_member_of_sco` | `economy:tj` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:uz_member_of_sco` | `economy:uz` → `alliance_org:sco` | formal_active | SCO About | `2026-07-14` | create |
| `relationship:at_member_of_eu` | `economy:at` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:be_member_of_eu` | `economy:be` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:bg_member_of_eu` | `economy:bg` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:cy_member_of_eu` | `economy:cy` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:cz_member_of_eu` | `economy:cz` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:de_member_of_eu` | `economy:de` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:dk_member_of_eu` | `economy:dk` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:ee_member_of_eu` | `economy:ee` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:es_member_of_eu` | `economy:es` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:fi_member_of_eu` | `economy:fi` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:fr_member_of_eu` | `economy:fr` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:gr_member_of_eu` | `economy:gr` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:hr_member_of_eu` | `economy:hr` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:hu_member_of_eu` | `economy:hu` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:ie_member_of_eu` | `economy:ie` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:it_member_of_eu` | `economy:it` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:lt_member_of_eu` | `economy:lt` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:lu_member_of_eu` | `economy:lu` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:lv_member_of_eu` | `economy:lv` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:mt_member_of_eu` | `economy:mt` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:nl_member_of_eu` | `economy:nl` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:pl_member_of_eu` | `economy:pl` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:pt_member_of_eu` | `economy:pt` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:ro_member_of_eu` | `economy:ro` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:se_member_of_eu` | `economy:se` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | keep |
| `relationship:si_member_of_eu` | `economy:si` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:sk_member_of_eu` | `economy:sk` → `alliance_org:eu` | formal_active | European Union countries | `2026-07-14` | create |
| `relationship:bn_member_of_asean` | `economy:bn` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:id_member_of_asean` | `economy:id` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:kh_member_of_asean` | `economy:kh` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:la_member_of_asean` | `economy:la` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:mm_member_of_asean` | `economy:mm` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:my_member_of_asean` | `economy:my` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:ph_member_of_asean` | `economy:ph` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:sg_member_of_asean` | `economy:sg` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:th_member_of_asean` | `economy:th` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:tl_member_of_asean` | `economy:tl` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:vn_member_of_asean` | `economy:vn` → `alliance_org:asean` | formal_active | ASEAN 2026 Member State evidence | `2026-07-14` | create |
| `relationship:ae_member_of_gcc` | `economy:ae` → `alliance_org:gcc` | formal_active | GCC Charter/About | `2026-07-14` | create |
| `relationship:bh_member_of_gcc` | `economy:bh` → `alliance_org:gcc` | formal_active | GCC Charter/About | `2026-07-14` | create |
| `relationship:kw_member_of_gcc` | `economy:kw` → `alliance_org:gcc` | formal_active | GCC Charter/About | `2026-07-14` | create |
| `relationship:om_member_of_gcc` | `economy:om` → `alliance_org:gcc` | formal_active | GCC Charter/About | `2026-07-14` | create |
| `relationship:qa_member_of_gcc` | `economy:qa` → `alliance_org:gcc` | formal_active | GCC Charter/About | `2026-07-14` | create |
| `relationship:sa_member_of_gcc` | `economy:sa` → `alliance_org:gcc` | formal_active | GCC Charter/About | `2026-07-14` | create |
| `relationship:am_member_of_eaeu` | `economy:am` → `alliance_org:eaeu` | formal_active | Eurasian Economic Union | `2026-07-14` | create |
| `relationship:by_member_of_eaeu` | `economy:by` → `alliance_org:eaeu` | formal_active | Eurasian Economic Union | `2026-07-14` | create |
| `relationship:kg_member_of_eaeu` | `economy:kg` → `alliance_org:eaeu` | formal_active | Eurasian Economic Union | `2026-07-14` | create |
| `relationship:kz_member_of_eaeu` | `economy:kz` → `alliance_org:eaeu` | formal_active | Eurasian Economic Union | `2026-07-14` | create |
| `relationship:ru_member_of_eaeu` | `economy:ru` → `alliance_org:eaeu` | formal_active | Eurasian Economic Union | `2026-07-14` | create |
| `relationship:ae_member_of_opec` | `economy:ae` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | keep |
| `relationship:cg_member_of_opec` | `economy:cg` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | create |
| `relationship:dz_member_of_opec` | `economy:dz` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | create |
| `relationship:ga_member_of_opec` | `economy:ga` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | create |
| `relationship:gq_member_of_opec` | `economy:gq` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | create |
| `relationship:iq_member_of_opec` | `economy:iq` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | create |
| `relationship:ir_member_of_opec` | `economy:ir` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | keep |
| `relationship:kw_member_of_opec` | `economy:kw` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | keep |
| `relationship:ly_member_of_opec` | `economy:ly` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | create |
| `relationship:ng_member_of_opec` | `economy:ng` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | keep |
| `relationship:sa_member_of_opec` | `economy:sa` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | keep |
| `relationship:ve_member_of_opec` | `economy:ve` → `alliance_org:opec` | formal_active | OPEC Member Countries | `2026-07-14` | create |
| `relationship:ae_member_of_gecf` | `economy:ae` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:bo_member_of_gecf` | `economy:bo` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:dz_member_of_gecf` | `economy:dz` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:eg_member_of_gecf` | `economy:eg` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:gq_member_of_gecf` | `economy:gq` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:ir_member_of_gecf` | `economy:ir` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:ly_member_of_gecf` | `economy:ly` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:ng_member_of_gecf` | `economy:ng` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:qa_member_of_gecf` | `economy:qa` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:ru_member_of_gecf` | `economy:ru` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:tt_member_of_gecf` | `economy:tt` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:ve_member_of_gecf` | `economy:ve` → `alliance_org:gecf` | formal_active | GECF Overview | `2026-07-14` | create |
| `relationship:ae_member_of_brics` | `economy:ae` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:br_member_of_brics` | `economy:br` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:cn_member_of_brics` | `economy:cn` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:eg_member_of_brics` | `economy:eg` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:et_member_of_brics` | `economy:et` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | create |
| `relationship:id_member_of_brics` | `economy:id` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:in_member_of_brics` | `economy:in` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:ir_member_of_brics` | `economy:ir` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:ru_member_of_brics` | `economy:ru` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:sa_member_of_brics` | `economy:sa` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |
| `relationship:za_member_of_brics` | `economy:za` → `alliance_org:brics` | formal_active | BRICS Brazil About | `2026-07-14` | keep |

## 4. 现有 223 条 Active Member Of Disposition

- canonical checksum：`6be2a8659257f321613feaf1ff5bfec81f4f2ce899af4263ba587698796f73c9`。
- `inactivate` 是 Package 2 候选，不是 Write 授权；必须在本 Package Review 批准后进入未来 R2B exact diff。
- `blocked` 不是 keep：它表示尚不能对当前 active edge 作权威收敛。任一 blocked 未解决时，最终 active member_of 集合相等断言不能成立，R2B 必须停止。

| existing edge | direction | candidate disposition | reason |
|---|---|---|---|
| `relationship:ir_member_of_opec_plus` | `economy:ir` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:kw_member_of_opec_plus` | `economy:kw` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:ng_member_of_opec_plus` | `economy:ng` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:sa_member_of_opec_plus` | `economy:sa` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:ae_member_of_opec_plus` | `economy:ae` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:kz_member_of_opec_plus` | `economy:kz` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:my_member_of_opec_plus` | `economy:my` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:mx_member_of_opec_plus` | `economy:mx` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:ru_member_of_opec_plus` | `economy:ru` → `alliance_org:opec_plus` | inactivate | source_conflict：OPEC+ 是合作参与机制，不是正式成员组织 |
| `relationship:ir_member_of_opec` | `economy:ir` → `alliance_org:opec` | keep | formal_active |
| `relationship:kw_member_of_opec` | `economy:kw` → `alliance_org:opec` | keep | formal_active |
| `relationship:ng_member_of_opec` | `economy:ng` → `alliance_org:opec` | keep | formal_active |
| `relationship:sa_member_of_opec` | `economy:sa` → `alliance_org:opec` | keep | formal_active |
| `relationship:ae_member_of_opec` | `economy:ae` → `alliance_org:opec` | keep | formal_active |
| `relationship:ca_member_of_g7` | `economy:ca` → `alliance_org:g7` | keep | formal_active |
| `relationship:fr_member_of_g7` | `economy:fr` → `alliance_org:g7` | keep | formal_active |
| `relationship:de_member_of_g7` | `economy:de` → `alliance_org:g7` | keep | formal_active |
| `relationship:it_member_of_g7` | `economy:it` → `alliance_org:g7` | keep | formal_active |
| `relationship:jp_member_of_g7` | `economy:jp` → `alliance_org:g7` | keep | formal_active |
| `relationship:gb_member_of_g7` | `economy:gb` → `alliance_org:g7` | keep | formal_active |
| `relationship:us_member_of_g7` | `economy:us` → `alliance_org:g7` | keep | formal_active |
| `relationship:eu_member_of_g7` | `economy:eu` → `alliance_org:g7` | inactivate | source_conflict：EU fully involved，但官方来源未列为七个 member state 之一 |
| `relationship:ar_member_of_g20` | `economy:ar` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:au_member_of_g20` | `economy:au` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:br_member_of_g20` | `economy:br` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ca_member_of_g20` | `economy:ca` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cn_member_of_g20` | `economy:cn` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:fr_member_of_g20` | `economy:fr` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:de_member_of_g20` | `economy:de` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:in_member_of_g20` | `economy:in` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:id_member_of_g20` | `economy:id` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:it_member_of_g20` | `economy:it` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:jp_member_of_g20` | `economy:jp` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:mx_member_of_g20` | `economy:mx` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ru_member_of_g20` | `economy:ru` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:sa_member_of_g20` | `economy:sa` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:za_member_of_g20` | `economy:za` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kr_member_of_g20` | `economy:kr` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:tr_member_of_g20` | `economy:tr` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:gb_member_of_g20` | `economy:gb` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:us_member_of_g20` | `economy:us` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:eu_member_of_g20` | `economy:eu` → `alliance_org:g20` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cn_member_of_wto` | `economy:cn` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:us_member_of_wto` | `economy:us` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:eu_member_of_wto` | `economy:eu` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:hk_member_of_wto` | `economy:hk` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:tw_member_of_wto` | `economy:tw` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:jp_member_of_wto` | `economy:jp` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kr_member_of_wto` | `economy:kr` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:sg_member_of_wto` | `economy:sg` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:in_member_of_wto` | `economy:in` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:id_member_of_wto` | `economy:id` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:vn_member_of_wto` | `economy:vn` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:th_member_of_wto` | `economy:th` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:my_member_of_wto` | `economy:my` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ph_member_of_wto` | `economy:ph` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:au_member_of_wto` | `economy:au` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:nz_member_of_wto` | `economy:nz` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:gb_member_of_wto` | `economy:gb` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:de_member_of_wto` | `economy:de` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:fr_member_of_wto` | `economy:fr` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:it_member_of_wto` | `economy:it` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:es_member_of_wto` | `economy:es` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:nl_member_of_wto` | `economy:nl` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ch_member_of_wto` | `economy:ch` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:se_member_of_wto` | `economy:se` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:no_member_of_wto` | `economy:no` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:dk_member_of_wto` | `economy:dk` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:pl_member_of_wto` | `economy:pl` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:tr_member_of_wto` | `economy:tr` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ru_member_of_wto` | `economy:ru` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ca_member_of_wto` | `economy:ca` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:mx_member_of_wto` | `economy:mx` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:br_member_of_wto` | `economy:br` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ar_member_of_wto` | `economy:ar` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cl_member_of_wto` | `economy:cl` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:sa_member_of_wto` | `economy:sa` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ae_member_of_wto` | `economy:ae` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:qa_member_of_wto` | `economy:qa` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kw_member_of_wto` | `economy:kw` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:il_member_of_wto` | `economy:il` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:za_member_of_wto` | `economy:za` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:eg_member_of_wto` | `economy:eg` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ng_member_of_wto` | `economy:ng` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ma_member_of_wto` | `economy:ma` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kz_member_of_wto` | `economy:kz` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:pk_member_of_wto` | `economy:pk` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:bd_member_of_wto` | `economy:bd` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ua_member_of_wto` | `economy:ua` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cz_member_of_wto` | `economy:cz` → `alliance_org:wto` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cn_member_of_imf` | `economy:cn` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:us_member_of_imf` | `economy:us` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:jp_member_of_imf` | `economy:jp` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kr_member_of_imf` | `economy:kr` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:sg_member_of_imf` | `economy:sg` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:in_member_of_imf` | `economy:in` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:id_member_of_imf` | `economy:id` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:vn_member_of_imf` | `economy:vn` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:th_member_of_imf` | `economy:th` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:my_member_of_imf` | `economy:my` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ph_member_of_imf` | `economy:ph` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:au_member_of_imf` | `economy:au` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:nz_member_of_imf` | `economy:nz` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:gb_member_of_imf` | `economy:gb` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:de_member_of_imf` | `economy:de` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:fr_member_of_imf` | `economy:fr` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:it_member_of_imf` | `economy:it` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:es_member_of_imf` | `economy:es` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:nl_member_of_imf` | `economy:nl` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ch_member_of_imf` | `economy:ch` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:se_member_of_imf` | `economy:se` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:no_member_of_imf` | `economy:no` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:dk_member_of_imf` | `economy:dk` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:pl_member_of_imf` | `economy:pl` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:tr_member_of_imf` | `economy:tr` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ru_member_of_imf` | `economy:ru` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ca_member_of_imf` | `economy:ca` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:mx_member_of_imf` | `economy:mx` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:br_member_of_imf` | `economy:br` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ar_member_of_imf` | `economy:ar` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cl_member_of_imf` | `economy:cl` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:sa_member_of_imf` | `economy:sa` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ae_member_of_imf` | `economy:ae` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:qa_member_of_imf` | `economy:qa` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kw_member_of_imf` | `economy:kw` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ir_member_of_imf` | `economy:ir` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:il_member_of_imf` | `economy:il` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:za_member_of_imf` | `economy:za` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:eg_member_of_imf` | `economy:eg` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ng_member_of_imf` | `economy:ng` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ma_member_of_imf` | `economy:ma` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kz_member_of_imf` | `economy:kz` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:pk_member_of_imf` | `economy:pk` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:bd_member_of_imf` | `economy:bd` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ua_member_of_imf` | `economy:ua` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cz_member_of_imf` | `economy:cz` → `alliance_org:imf` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cn_member_of_world_bank` | `economy:cn` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:us_member_of_world_bank` | `economy:us` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:jp_member_of_world_bank` | `economy:jp` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kr_member_of_world_bank` | `economy:kr` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:sg_member_of_world_bank` | `economy:sg` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:in_member_of_world_bank` | `economy:in` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:id_member_of_world_bank` | `economy:id` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:vn_member_of_world_bank` | `economy:vn` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:th_member_of_world_bank` | `economy:th` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:my_member_of_world_bank` | `economy:my` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ph_member_of_world_bank` | `economy:ph` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:au_member_of_world_bank` | `economy:au` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:nz_member_of_world_bank` | `economy:nz` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:gb_member_of_world_bank` | `economy:gb` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:de_member_of_world_bank` | `economy:de` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:fr_member_of_world_bank` | `economy:fr` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:it_member_of_world_bank` | `economy:it` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:es_member_of_world_bank` | `economy:es` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:nl_member_of_world_bank` | `economy:nl` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ch_member_of_world_bank` | `economy:ch` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:se_member_of_world_bank` | `economy:se` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:no_member_of_world_bank` | `economy:no` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:dk_member_of_world_bank` | `economy:dk` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:pl_member_of_world_bank` | `economy:pl` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:tr_member_of_world_bank` | `economy:tr` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ru_member_of_world_bank` | `economy:ru` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ca_member_of_world_bank` | `economy:ca` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:mx_member_of_world_bank` | `economy:mx` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:br_member_of_world_bank` | `economy:br` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ar_member_of_world_bank` | `economy:ar` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cl_member_of_world_bank` | `economy:cl` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:sa_member_of_world_bank` | `economy:sa` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ae_member_of_world_bank` | `economy:ae` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:qa_member_of_world_bank` | `economy:qa` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kw_member_of_world_bank` | `economy:kw` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ir_member_of_world_bank` | `economy:ir` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:il_member_of_world_bank` | `economy:il` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:za_member_of_world_bank` | `economy:za` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:eg_member_of_world_bank` | `economy:eg` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ng_member_of_world_bank` | `economy:ng` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ma_member_of_world_bank` | `economy:ma` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:kz_member_of_world_bank` | `economy:kz` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:pk_member_of_world_bank` | `economy:pk` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:bd_member_of_world_bank` | `economy:bd` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:ua_member_of_world_bank` | `economy:ua` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:cz_member_of_world_bank` | `economy:cz` → `alliance_org:world_bank` | blocked | source_conflict：目标联盟官方 formal-active 全集尚未冻结，不得猜测 keep/inactivate |
| `relationship:au_member_of_oecd` | `economy:au` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:nz_member_of_oecd` | `economy:nz` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:jp_member_of_oecd` | `economy:jp` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:kr_member_of_oecd` | `economy:kr` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:gb_member_of_oecd` | `economy:gb` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:de_member_of_oecd` | `economy:de` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:fr_member_of_oecd` | `economy:fr` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:it_member_of_oecd` | `economy:it` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:es_member_of_oecd` | `economy:es` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:nl_member_of_oecd` | `economy:nl` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:ch_member_of_oecd` | `economy:ch` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:se_member_of_oecd` | `economy:se` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:no_member_of_oecd` | `economy:no` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:dk_member_of_oecd` | `economy:dk` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:pl_member_of_oecd` | `economy:pl` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:tr_member_of_oecd` | `economy:tr` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:ca_member_of_oecd` | `economy:ca` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:mx_member_of_oecd` | `economy:mx` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:cl_member_of_oecd` | `economy:cl` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:il_member_of_oecd` | `economy:il` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:cz_member_of_oecd` | `economy:cz` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:us_member_of_oecd` | `economy:us` → `alliance_org:oecd` | inactivate | alliance_identity_convergence |
| `relationship:cz_member_of_eu` | `economy:cz` → `alliance_org:eu` | keep | formal_active |
| `relationship:dk_member_of_eu` | `economy:dk` → `alliance_org:eu` | keep | formal_active |
| `relationship:de_member_of_eu` | `economy:de` → `alliance_org:eu` | keep | formal_active |
| `relationship:fr_member_of_eu` | `economy:fr` → `alliance_org:eu` | keep | formal_active |
| `relationship:it_member_of_eu` | `economy:it` → `alliance_org:eu` | keep | formal_active |
| `relationship:es_member_of_eu` | `economy:es` → `alliance_org:eu` | keep | formal_active |
| `relationship:nl_member_of_eu` | `economy:nl` → `alliance_org:eu` | keep | formal_active |
| `relationship:pl_member_of_eu` | `economy:pl` → `alliance_org:eu` | keep | formal_active |
| `relationship:se_member_of_eu` | `economy:se` → `alliance_org:eu` | keep | formal_active |
| `relationship:br_member_of_brics` | `economy:br` → `alliance_org:brics` | keep | formal_active |
| `relationship:ru_member_of_brics` | `economy:ru` → `alliance_org:brics` | keep | formal_active |
| `relationship:in_member_of_brics` | `economy:in` → `alliance_org:brics` | keep | formal_active |
| `relationship:cn_member_of_brics` | `economy:cn` → `alliance_org:brics` | keep | formal_active |
| `relationship:za_member_of_brics` | `economy:za` → `alliance_org:brics` | keep | formal_active |
| `relationship:sa_member_of_brics` | `economy:sa` → `alliance_org:brics` | keep | formal_active |
| `relationship:eg_member_of_brics` | `economy:eg` → `alliance_org:brics` | keep | formal_active |
| `relationship:ae_member_of_brics` | `economy:ae` → `alliance_org:brics` | keep | formal_active |
| `relationship:id_member_of_brics` | `economy:id` → `alliance_org:brics` | keep | formal_active |
| `relationship:ir_member_of_brics` | `economy:ir` → `alliance_org:brics` | keep | formal_active |

## 5. Counts、Checksums 与人工出口

```text
alliance input v1 checksum = 4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a
membership model rows = 45 (10 resolved formal + 13 blocked formal + 1 term-bound blocked + 21 non-formal models，其中 20 not-applicable、1 WBG aggregate blocked)
economy target rows = 79 (35 reuse + 44 create)
economy target checksum = 95613a931adf3d7231cbb1d311e5051f3695d9da40c60bbeeccb39d006118cb3
formal-active member_of candidates = 133 (31 keep + 102 create)
member_of candidate checksum = c3d652571fa93307088633cbfe06dce18b1257d8ac2b3ee2ae88c5a27e69fcf7
existing active member_of dispositions = 223 (31 keep + 32 inactivate + 160 blocked)
existing disposition checksum = 6be2a8659257f321613feaf1ff5bfec81f4f2ce899af4263ba587698796f73c9
duplicate candidate tuples = 0
orphan candidate endpoints = 0 after candidate economy creates
wrong direction/type = 0
observer/partner/applicant/suspended/former admitted = 0
optional led_by/part_of candidates = 0 (excluded)
```

唯一人工出口是 Package 2.1 Review：逐项批准/修订 membership model、79 条 economy、133 条 formal-active tuple、32 条 proposed inactivate，并决定 160 条 blocked source-conflict 的处理方向。未通过前不得进入 Package 3，不得把本文件转换为 seed 或数据库输入。
