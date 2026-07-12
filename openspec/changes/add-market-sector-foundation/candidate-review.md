# 市场板块 MVP 候选 Review 清单

## Review 状态与证据边界

- 状态：待用户逐项 Review；本文件不是正式 seed，不触发 migration、PostgreSQL 写入或 Neo4j 投影。
- 候选快照事实源：`backend/data/entity_foundation/sectors.json`，快照日期均为 `2026-07-08`；评估日期为 `2026-07-12`。
- `concept_###`、`industry_###`、`index_###` 是仓库现有 `sector_code`，本文件不把它们宣称为同花顺官方代码。
- 仓库未提供来源 URL、官方方法、成分样本、连续行情或事件窗口统计；不得用名称或排名替代这些外部证据。
- semantic classification、canonical key、英文 alias、传导簇、去重、benchmark 和运行层级均为 **Review recommendation**，未经批准不进入主数据。
- `override` 全部为“无”；低于 70 分的候选只有在用户批准并记录 approver、reason、替代证据和覆盖缺口后才能补位。

## 评分方法

总分公式：`事件可解释性×5 + 传导独立性×4 + 行情敏感度×3 + 数据完整性×3 + 长期稳定性×3 + 市场代表性×2`。

证据标记：

- `E3-N`：仅依据候选名称可列出至少一种事件类型和影响通道，属于分析建议；没有外部定义证明。
- `E0-B`：名称表达宽基/市场指数，按已批准边界不作为事件暴露 sector 评分。
- `T3-D`：仅依据候选名称和本清单比较，可提出与相邻候选不同的暴露边界；待成分证据复核。
- `T2-P`：可能具有独立暴露，但与宽基、行业或主题存在明显交叉，边界待核验。
- `T1-O`：与另一候选名称高度重叠，缺少成分或定义证明独立性。
- `T0-B`：benchmark-first 对象不评估 sector 传导独立性。
- `Q0-M`：仓库无连续行情和事件窗口证据，行情敏感度按缺失证据记 0。
- `D2-L`：仓库仅有名称、来源类型、内部代码、scope、排名和日期；缺 URL、官方代码证明、方法/成分和行情。
- `L2-U`：名称看似可持续，但仓库无跨年度定义或变更记录，只能记中间档 2。
- `M2-S`：仓库只有 `exchange_scope` 和单次排名，缺成分规模、市场覆盖或多来源采用证据。

## 概念板块候选（20）

每行 `证据` 顺序为 `E/T/Q/D/L/M`，source 均为 `sectors.json#<旧 key>`。

| # | 候选与来源 | semantic / canonical / alias 建议 | 分数 E/T/Q/D/L/M = 总分 | 证据与缺失 | 传导簇 | Review 建议 | benchmark 建议 | 层级 / override |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| C01 | 人工智能；`concept_001`；rank 1 | `theme_sector`; `sector:theme_artificial_intelligence`; `Artificial Intelligence` | 3/3/0/2/2/2 = 43 | `E3-N/T3-D/Q0-M/D2-L/L2-U/M2-S`; src `#sector:ths_concept_ai` | AI软件通信 | 保留候选；定义与算力、计算机应用边界 | 待核验是否存在正式板块行情 benchmark | 核心 / 无 |
| C02 | 算力；`concept_002`；rank 2 | `theme_sector`; `sector:theme_compute_power`; `Computing Power` | 3/3/0/2/2/2 = 43 | 同上；src `#sector:ths_concept_compute_power` | AI软件通信 | 保留候选；作为 AI 基础设施暴露建议 | 待核验 | 核心 / 无 |
| C03 | 数据中心；`concept_003`；rank 3 | `industry_sector`; `sector:industry_data_center`; `Data Center` | 3/3/0/2/2/2 = 43 | 同上；src `#sector:ths_concept_data_center` | AI软件通信 | 保留候选；与算力部分重叠但粒度不同 | 待核验 | 核心 / 无 |
| C04 | 半导体概念；`concept_004`；rank 4 | `theme_sector`; `sector:theme_semiconductor`; `Semiconductor Theme` | 3/1/0/2/2/2 = 35 | `E3-N/T1-O/Q0-M/D2-L/L2-U/M2-S`; src `#sector:ths_concept_semiconductor` | 半导体电子 | 合并候选；优先并入 I01，待成分范围证明 | 待核验 I01 的共同 benchmark | 扩展 / 无 |
| C05 | 芯片概念；`concept_005`；rank 5 | `theme_sector`; `sector:theme_chip`; `Chip Theme` | 3/1/0/2/2/2 = 35 | `E3-N/T1-O/Q0-M/D2-L/L2-U/M2-S`; src `#sector:ths_concept_chip` | 半导体电子 | 合并候选；与 C04/I01 高重叠待核验 | 待核验 I01 的共同 benchmark | 扩展 / 无 |
| C06 | 机器人概念；`concept_006`；rank 6 | `theme_sector`; `sector:theme_robotics`; `Robotics` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_robot` | 工业基建 | 保留候选 | 待核验 | 核心 / 无 |
| C07 | 低空经济；`concept_007`；rank 7 | `theme_sector`; `sector:theme_low_altitude_economy`; `Low-altitude Economy` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_low_altitude` | 政策主题 | 保留候选；政策与交通暴露边界待定义 | 待核验 | 核心 / 无 |
| C08 | 商业航天；`concept_008`；rank 8 | `theme_sector`; `sector:theme_commercial_space`; `Commercial Space` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_commercial_space` | 国防航天卫星 | 保留候选 | 待核验 | 核心 / 无 |
| C09 | 卫星导航；`concept_009`；rank 9 | `theme_sector`; `sector:theme_satellite_navigation`; `Satellite Navigation` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_satellite_nav` | 国防航天卫星 | 保留候选；与商业航天部分交叉不合并 | 待核验 | 核心 / 无 |
| C10 | 军工；`concept_010`；rank 10 | `industry_sector`; `sector:industry_defense`; `Defense Industry` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_defense` | 国防航天卫星 | 保留候选；需稳定行业定义 | 待核验 | 核心 / 无 |
| C11 | 新能源汽车；`concept_011`；rank 11 | `theme_sector`; `sector:theme_new_energy_vehicle`; `New Energy Vehicle` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_nev` | 汽车新能源 | 保留候选；与汽车整车非同义 | 待核验 | 核心 / 无 |
| C12 | 固态电池；`concept_012`；rank 12 | `theme_sector`; `sector:theme_solid_state_battery`; `Solid-state Battery` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_solid_state_battery` | 汽车新能源 | 保留候选；作为电池下位主题建议 | 待核验 | 核心 / 无 |
| C13 | 储能；`concept_013`；rank 13 | `theme_sector`; `sector:theme_energy_storage`; `Energy Storage` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_energy_storage` | 能源电力 | 保留候选 | 待核验 | 核心 / 无 |
| C14 | 光伏概念；`concept_014`；rank 14 | `theme_sector`; `sector:theme_photovoltaic`; `Photovoltaic Theme` | 3/2/0/2/2/2 = 39 | `E3-N/T2-P/Q0-M/D2-L/L2-U/M2-S`; src `#sector:ths_concept_photovoltaic` | 能源电力 | 保留候选；与 I11 建议上下位而非直接合并 | 待核验 | 核心 / 无 |
| C15 | 风电；`concept_015`；rank 15 | `industry_sector`; `sector:industry_wind_power`; `Wind Power` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_wind_power` | 能源电力 | 保留候选 | 待核验 | 核心 / 无 |
| C16 | 氢能源；`concept_016`；rank 16 | `theme_sector`; `sector:theme_hydrogen_energy`; `Hydrogen Energy` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_hydrogen` | 能源电力 | 保留候选 | 待核验 | 核心 / 无 |
| C17 | 核电；`concept_017`；rank 17 | `industry_sector`; `sector:industry_nuclear_power`; `Nuclear Power` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_nuclear_power` | 能源电力 | 保留候选 | 待核验 | 核心 / 无 |
| C18 | 数字货币；`concept_018`；rank 18 | `theme_sector`; `sector:theme_digital_currency`; `Digital Currency` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_digital_currency` | 政策主题 | 保留候选；明确不等于加密资产价格 benchmark | 待核验 | 核心 / 无 |
| C19 | 跨境电商；`concept_019`；rank 19 | `theme_sector`; `sector:theme_cross_border_ecommerce`; `Cross-border E-commerce` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_cross_border_ecommerce` | 消费农业 | 保留候选 | 待核验 | 核心 / 无 |
| C20 | 国企改革；`concept_020`；rank 20 | `theme_sector`; `sector:theme_state_owned_enterprise_reform`; `SOE Reform` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_concept_soe_reform` | 政策主题 | 保留候选；需政策主题稳定定义 | 待核验 | 扩展 / 无 |

## 行业板块候选（20）

| # | 候选与来源 | semantic / canonical / alias 建议 | 分数 E/T/Q/D/L/M = 总分 | 证据与缺失 | 传导簇 | Review 建议 | benchmark 建议 | 层级 / override |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| I01 | 半导体及元件；`industry_001`；rank 1 | `industry_sector`; `sector:industry_semiconductor_components`; `Semiconductors and Components` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_semiconductor_components` | 半导体电子 | 保留；建议承接 C04/C05 合并映射 | 待核验；可与 SOX 区分来源后关联 | 核心 / 无 |
| I02 | 通信设备；`industry_002`；rank 2 | `industry_sector`; `sector:industry_communication_equipment`; `Communication Equipment` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_communication_equipment` | AI软件通信 | 保留候选 | 待核验 | 核心 / 无 |
| I03 | 计算机应用；`industry_003`；rank 3 | `industry_sector`; `sector:industry_computer_applications`; `Computer Applications` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_software` | AI软件通信 | 保留候选；英文 alias 不沿用旧 key 的 software 推断 | 待核验 | 核心 / 无 |
| I04 | 传媒；`industry_004`；rank 4 | `industry_sector`; `sector:industry_media`; `Media` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_media` | 消费农业 | 保留候选 | 待核验 | 核心 / 无 |
| I05 | 证券；`industry_005`；rank 5 | `industry_sector`; `sector:industry_securities`; `Securities` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_securities` | 金融地产 | 保留候选 | 待核验 | 核心 / 无 |
| I06 | 银行；`industry_006`；rank 6 | `industry_sector`; `sector:industry_banking`; `Banking` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_bank` | 金融地产 | 保留候选 | 待核验 | 核心 / 无 |
| I07 | 保险及其他；`industry_007`；rank 7 | `industry_sector`; `sector:industry_insurance_and_other_finance`; `Insurance and Other Finance` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_insurance` | 金融地产 | 保留候选；“其他”范围必须补定义 | 待核验 | 核心 / 无 |
| I08 | 汽车整车；`industry_008`；rank 8 | `industry_sector`; `sector:industry_automobile_manufacturers`; `Automobile Manufacturers` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_auto` | 汽车新能源 | 保留候选 | 待核验 | 核心 / 无 |
| I09 | 汽车零部件；`industry_009`；rank 9 | `industry_sector`; `sector:industry_auto_parts`; `Auto Parts` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_auto_parts` | 汽车新能源 | 保留候选 | 待核验 | 核心 / 无 |
| I10 | 电力设备；`industry_010`；rank 10 | `industry_sector`; `sector:industry_power_equipment`; `Power Equipment` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_power_equipment` | 能源电力 | 保留候选 | 待核验 | 核心 / 无 |
| I11 | 光伏设备；`industry_011`；rank 11 | `industry_sector`; `sector:industry_photovoltaic_equipment`; `Photovoltaic Equipment` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_pv_equipment` | 能源电力 | 保留候选；与 C14 上下位建议 | 待核验 | 核心 / 无 |
| I12 | 电池；`industry_012`；rank 12 | `industry_sector`; `sector:industry_battery`; `Battery` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_battery` | 汽车新能源 | 保留候选；C12 为潜在下位主题 | 待核验 | 核心 / 无 |
| I13 | 医疗服务；`industry_013`；rank 13 | `industry_sector`; `sector:industry_healthcare_services`; `Healthcare Services` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_medical_service` | 医药生科 | 保留候选 | 待核验 | 核心 / 无 |
| I14 | 化学制药；`industry_014`；rank 14 | `industry_sector`; `sector:industry_chemical_pharmaceuticals`; `Chemical Pharmaceuticals` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_chemical_pharma` | 医药生科 | 保留候选 | 待核验 | 扩展 / 无 |
| I15 | 中药；`industry_015`；rank 15 | `industry_sector`; `sector:industry_traditional_chinese_medicine`; `Traditional Chinese Medicine` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_tcm` | 医药生科 | 保留候选 | 待核验 | 扩展 / 无 |
| I16 | 工业金属；`industry_016`；rank 16 | `industry_sector`; `sector:industry_industrial_metals`; `Industrial Metals` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_industrial_metal` | 有色化工材料 | 保留候选；不得与商品价格实体混同 | 待核验行业 benchmark | 扩展 / 无 |
| I17 | 煤炭开采加工；`industry_017`；rank 17 | `industry_sector`; `sector:industry_coal_mining_processing`; `Coal Mining and Processing` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_coal` | 能源电力 | 保留候选；不得与煤炭 commodity 混同 | 待核验行业 benchmark | 扩展 / 无 |
| I18 | 石油加工贸易；`industry_018`；rank 18 | `industry_sector`; `sector:industry_petroleum_processing_trade`; `Petroleum Processing and Trade` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_oil_processing` | 能源电力 | 保留候选；不得与原油 commodity/price 混同 | 待核验行业 benchmark | 扩展 / 无 |
| I19 | 房地产开发；`industry_019`；rank 19 | `industry_sector`; `sector:industry_real_estate_development`; `Real Estate Development` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_real_estate` | 金融地产 | 保留候选 | 待核验 | 扩展 / 无 |
| I20 | 消费电子；`industry_020`；rank 20 | `industry_sector`; `sector:industry_consumer_electronics`; `Consumer Electronics` | 3/3/0/2/2/2 = 43 | 通用标记；src `#sector:ths_industry_consumer_electronics` | 半导体电子 | 保留候选 | 待核验 | 扩展 / 无 |

## 指数板块来源候选（20）

当前 manifest 的这 20 条主要是宽基、市场、规模或风格指数，不是用户此前举例的产业/主题型“指数板块”。仍按已批准规则完整进入候选池，但根据 semantic sector / benchmark 边界给出排除建议。

| # | 候选与来源 | semantic / canonical / alias 建议 | 分数 E/T/Q/D/L/M = 总分 | 证据与缺失 | 传导簇 | Review 建议 | benchmark 建议 | 层级 / override |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| X01 | 上证50；`index_001`；rank 1 | `market_sector` 草案；`sector:market_sse_50`; `SSE 50` | 0/0/0/2/2/2 = 16 | `E0-B/T0-B/Q0-M/D2-L/L2-U/M2-S`; src `#sector:ths_index_sse50` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用正式 benchmark/index | 观察 / 无 |
| X02 | 沪深300；`index_002`；rank 2 | `market_sector`; `sector:market_csi_300`; `CSI 300` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_csi300` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 观察 / 无 |
| X03 | 中证500；`index_003`；rank 3 | `market_sector`; `sector:market_csi_500`; `CSI 500` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_csi500` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 观察 / 无 |
| X04 | 中证1000；`index_004`；rank 4 | `market_sector`; `sector:market_csi_1000`; `CSI 1000` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_csi1000` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 观察 / 无 |
| X05 | 科创50；`index_005`；rank 5 | `market_sector`; `sector:market_star_50`; `STAR 50` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_star50` | 市场宽基 | 排除 sector，benchmark-first；是否保留市场暴露待 Review | 核验并复用 | 观察 / 无 |
| X06 | 创业板50；`index_006`；rank 6 | `market_sector`; `sector:market_chinext_50`; `ChiNext 50` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_chinext50` | 市场宽基 | 排除 sector，benchmark-first | 核验并复用 | 观察 / 无 |
| X07 | 深证100；`index_007`；rank 7 | `market_sector`; `sector:market_szse_100`; `SZSE 100` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_szse100` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 观察 / 无 |
| X08 | 北证50；`index_008`；rank 8 | `market_sector`; `sector:market_bse_50`; `BSE 50` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_bse50` | 市场宽基 | 排除 sector，benchmark-first；市场覆盖可由 market 表达 | 核验并复用 | 观察 / 无 |
| X09 | 中证A500；`index_009`；rank 9 | `market_sector`; `sector:market_csi_a500`; `CSI A500` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_csi_a500` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 观察 / 无 |
| X10 | 中证红利；`index_010`；rank 10 | `style_sector`; `sector:style_dividend`; `CSI Dividend` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_csi_dividend` | 市场风格 | 排除 sector 建议；若保留 style sector 需独立事件语义证据 | 核验并复用 | 观察 / 无 |
| X11 | 央视50；`index_011`；rank 11 | `market_sector`; `sector:market_cctv_50`; `CCTV 50` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_cctv50` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 扩展 / 无 |
| X12 | 中证全指；`index_012`；rank 12 | `market_sector`; `sector:market_csi_all_share`; `CSI All Share` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_csi_all` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 扩展 / 无 |
| X13 | 万得全A；`index_013`；rank 13 | `market_sector`; `sector:market_wind_all_a`; `Wind All A` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_wind_all_a` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 扩展 / 无 |
| X14 | 国证2000；`index_014`；rank 14 | `market_sector`; `sector:market_cni_2000`; `CNI 2000` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_cni2000` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 扩展 / 无 |
| X15 | 恒生指数；`index_015`；rank 15 | `market_sector`; `sector:market_hang_seng`; `Hang Seng Index` | 0/0/0/2/2/2 = 16 | 同上；src `#sector:ths_index_hsi` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 扩展 / 无 |
| X16 | 恒生科技指数；`index_016`；rank 16 | `theme_sector`; `sector:theme_hong_kong_technology`; `Hang Seng TECH` | 3/2/0/2/2/2 = 39 | `E3-N/T2-P/Q0-M/D2-L/L2-U/M2-S`; src `#sector:ths_index_hstech` | AI软件通信 | 保留候选待核验；名称兼具主题暴露和指数角色 | 必须核验正式 benchmark 后用 `tracked_by_benchmark` | 扩展 / 无 |
| X17 | 纳斯达克100；`index_017`；rank 17 | `market_sector`; `sector:market_nasdaq_100`; `Nasdaq-100` | 0/0/0/2/2/2 = 16 | 宽基标记；src `#sector:ths_index_ndx` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 扩展 / 无 |
| X18 | 标普500；`index_018`；rank 18 | `market_sector`; `sector:market_sp_500`; `S&P 500` | 0/0/0/2/2/2 = 16 | 宽基标记；src `#sector:ths_index_sp500` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 扩展 / 无 |
| X19 | 费城半导体指数；`index_019`；rank 19 | `industry_sector`; `sector:industry_us_semiconductor`; `PHLX Semiconductor` | 3/2/0/2/2/2 = 39 | 主题标记；src `#sector:ths_index_sox` | 半导体电子 | 保留候选待核验；与 I01 市场范围不同 | 必须核验正式 benchmark 后关联 | 扩展 / 无 |
| X20 | MSCI中国A50互联互通；`index_020`；rank 20 | `market_sector`; `sector:market_msci_china_a50_connect`; `MSCI China A 50 Connect` | 0/0/0/2/2/2 = 16 | 宽基标记；src `#sector:ths_index_msci_china_a50` | 市场宽基 | 排除 sector，benchmark-only 建议 | 核验并复用 | 扩展 / 无 |

## Source mapping identity 草案

每条 mapping identity 采用：

```text
sector_source_mapping:<canonical_key>:ths:<concept|industry|index_sector>:<现有 sector_code>
```

每一候选行的 canonical key 与现有 `sector_code` 按上式共同构成该行的完整 identity，来源类型映射固定为 `concept -> concept`、`industry -> industry`、`index -> index_sector`。例如 C01 为 `sector_source_mapping:sector:theme_artificial_intelligence:ths:concept:concept_001`，I01 为 `sector_source_mapping:sector:industry_semiconductor_components:ths:industry:industry_001`，X19 为 `sector_source_mapping:sector:industry_us_semiconductor:ths:index_sector:index_019`。这些 identity 使用仓库现有代码作为候选来源标识；若 Review 后确认代码不是稳定来源代码，Apply 后续阶段必须改用规范化来源名和市场 scope，不能把当前内部序号伪装成官方代码。

## 自检结果

- 数量：概念 20、行业 20、指数板块来源 20，总计 60。
- 分数分布：43 分 37 个；39 分 3 个；35 分 2 个；16 分 18 个；70 分以上 0 个。
- Review recommendation：保留 40 个、合并 2 个、排除 sector/benchmark-first 18 个。
- 运行层级建议：核心 30、扩展 20、观察 10；其中被建议排除但标为扩展的 X11-X15、X17-X18、X20 只表示 Review 排查顺序，不表示进入推理调度。
- 重复/交叉：C04、C05 与 I01 高重叠；C14 与 I11、C12 与 I12 建议上下位；X19 与 I01 跨市场交叉；均缺成分/定义证据，不能自动合并。
- canonical key：60 条草案无完全重复；所有 key 均不含 `ths`、排名或快照日期。
- 证据缺失：全部 60 条缺来源 URL、官方代码证明、行情窗口、方法/成分和跨年度变更证据；因此全部 Q=0，D/L/M 仅按仓库有限字段给中间档 2。
- aliases：每条均给出英文 alias 建议，但仓库当前没有 aliases 事实，必须由用户 Review 或可靠来源核验。
- 传导簇：候选覆盖金融地产、能源电力、有色化工材料、半导体电子、AI软件通信、汽车新能源、医药生科、消费农业、国防航天卫星、政策主题；缺少明确的交通公用、工业基建（除机器人相关暴露外）和消费农业中的农业骨架候选。

## 用户逐项 Review 决策

1. 是否接受 18 个宽基/市场指数的 `benchmark-only` 排除建议，并用真实产业/主题型指数板块候选替换，以恢复最终 50-60 个 semantic sector 目标。
2. 是否批准 C04/C05 合并到 I01；是否要求 C14-I11、C12-I12 保持上下位而非合并。
3. 是否把 X16、X19 同时保留为 semantic sector 候选，并在核验正式 benchmark 后建立 `tracked_by_benchmark`。
4. 哪些低于 70 分候选允许因传导簇覆盖进入补证流程；每个 override 必须补 approver、reason、替代证据和覆盖缺口。
5. 是否补充交通公用、工业基建、农业等当前候选池覆盖不足的板块，并相应替换而不是机械扩容。
