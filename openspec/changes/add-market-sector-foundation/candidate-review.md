# 市场板块 MVP 候选 Review 清单（返工版）

## 状态与口径

- 状态：待用户逐项 Review；task 1.4 保持未完成。本文件不是正式 seed，不触发 migration、PostgreSQL 写入或 Neo4j 投影。
- 本轮从事件推理角度推荐 `industry`、`concept`、`index_sector` 各 20 个原始候选，共 60 个；顺序是 Review 优先级，不是同花顺、东方财富或 PG 热度排名。
- 旧 `backend/data/entity_foundation/sectors.json` 的 60 条只作为迁移输入，逐项列于“旧 PG 迁移对照”，不再决定新候选池。
- 建议正式入选 57 个 canonical sector：行业骨架 20、概念来源主导 20、指数板块来源主导 17；X03/C06、X05/C04、X09/C01 三组同义候选合并为同一 canonical sector 并保留多来源 mapping。
- 未核验任何同花顺/东方财富代码、Top 排名或历史行情；官方指数代码仅在本轮官方资料明确出现时才可引用，本清单主表仍不以代码作为身份。
- 所有结论均为 Review recommendation，不表达投资建议。

### Candidate 与 source mapping identity 边界

主表中的 `I01-I20`、`C01-C20`、`X01-X20` 是本次版本化 Review identity。canonical key 草案已逐项给出，但 source mapping identity 必须包含真实来源系统和稳定来源代码，或规范化来源名加 market scope；本轮未核验同花顺/东方财富的具体分类记录，因此 industry/concept 不生成伪造 mapping identity。X 项在后续取得逐条官方编制方案、正式名称和代码后，才生成对应官方 source mapping。此边界不妨碍用户先 Review semantic sector 与 canonical key。

## 来源登记

| ID | 来源 | 用途 |
| --- | --- | --- |
| S1 | [国家统计局：国民经济行业分类](https://www.stats.gov.cn/zs/tjws/tjbz/202301/t20230101_1903769.html) | 稳定行业骨架与经济活动同质性原则 |
| S2 | [国家统计局：战略性新兴产业界定](https://www.stats.gov.cn/zs/tjws/tjbz/202301/t20230101_1903710.html) | 新一代信息技术、高端装备、新材料、生物、新能源汽车、新能源等长期分类 |
| S3 | [国家发展改革委：产业结构调整指导目录（2024年本）](https://zfxxgk.ndrc.gov.cn/web/iteminfo.jsp?id=20305) | 政策、技术、产能与产业升级事件主题依据 |
| S4 | [中国人大网：2026年国民经济和社会发展计划报告](https://www.npc.gov.cn/c2/c30834/202603/t20260316_453271.html) | AI、卫星互联网、低空经济、氢能、储能、新一代电池等政策触发依据 |
| S5 | [上交所：科创板芯片设计、半导体材料设备主题指数公告](https://www.sse.com.cn/market/sseindex/diclosure/c/c_20240715_10759821.shtml) | 两条半导体产业主题指数的官方定义 |
| S6 | [中证指数：中证卫星产业指数编制方案](https://oss-ch.csindex.com.cn/static/html/csindex/public/uploads/indices/detail/files/zh_CN/931594_Index_Methodology_cn.pdf) | 卫星制造、发射、通信、导航、遥感产业定义 |
| S7 | [中证指数：中证卫星导航产业指数编制方案](https://oss-ch.csindex.com.cn/static/html/csindex/public/uploads/indices/detail/files/zh_CN/931585_Index_Methodology_cn.pdf) | 卫星导航产业定义 |
| S8 | [国证指数：机器人、风电光伏装备、新能源电池修订公告](https://www.cnindex.com.cn/zh_information/notices_news/2023/202302/t20230228_18049.html?act_menu=2) | 三条产业指数定义与样本边界 |
| S9 | [国证指数：人工智能、机器人、新能源电池、化肥农药、通用航空指数公告](https://www.cnindex.com.cn/zh_information/notices_news/2024y/202412/P020241222692311613993.pdf) | 产业主题指数存在性与名称依据 |
| S10 | [上交所：信息安全、高端制造、基建、银发经济、汽车等指数](https://www.sse.com.cn/aboutus/mediacenter/hotandd/c/c_20221012_5709968.shtml) | 产业/主题指数体系依据 |
| S11 | [上交所：科创新能源、工业机械等指数](https://www.sse.com.cn/aboutus/mediacenter/hotandd/c/c_20230313_5717726.shtml) | 科创新能源与工业机械指数定义 |
| S12 | [上交所：科创板指数体系简介](https://www.sse.com.cn/market/sseindex/overview/) | 生物医药、芯片、高端装备、新材料主题指数体系依据 |
| S13 | [深交所：先进制造、数字经济、绿色低碳指数体系](https://www.szse.cn/aboutus/trends/news/t20230803_602459.html) | 生物医药、机器人、芯片、算力设施、新能源车、绿色电力等指数体系依据 |

## 评分规则与共性证据

总分：`E×5 + T×4 + Q×3 + D×3 + L×3 + M×2`，对应事件可解释性、传导独立性、行情敏感度、数据完整性、长期稳定性、市场代表性。

- 行业骨架统一基线 `5/4/0/3/5/4 = 73`：S1/S2 支撑稳定分类与长期性；事件与差异路径为 Review 分析；未做历史事件窗口检验，Q=0；没有具体来源 mapping/成分，D=3。
- 概念统一基线 `5/4/0/3/4/3 = 68`：S2/S3/S4 支撑跨周期政策或技术主题；事件与差异路径明确；未做行情窗口检验，Q=0；具体 source taxonomy mapping 待核验，D=3。
- 官方定义较完整的指数板块 X01-X10 为 `4/4/0/5/4/4 = 71`：官方公告/编制方案证明定义和可观测 benchmark 角色；未在本轮核验历史行情敏感度，Q=0。
- 官方指数体系已确认但本轮未逐条取得编制方案的 X11-X20 为 `4/4/0/4/4/4 = 68`：存在性与产业方向有官方来源，具体方法、成分和代码待核验。
- 每行“证据”给出来源 ID 与该候选的独立事件/传导判断；所有英文 alias、canonical key 和层级均待 Review。

## 原始候选：industry 20

| ID | 中文主名 / English alias | semantic / canonical | 主要事件触发与传导簇 | 评分 | 分项证据 | benchmark / 层级 | 建议结果 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| I01 | 银行 / Banking | `industry_sector`; `sector:industry_banking` | 利率、信贷、资本与地产周期；金融地产 | 5/4/0/3/5/4=73 | S1；E利率信用明确，T区别非银，Q0，D3，L5，M4 | 行业指数待核验；核心 | 入选 |
| I02 | 证券与保险 / Securities and Insurance | `industry_sector`; `sector:industry_securities_insurance` | 资本市场制度、保费与风险事件；金融地产 | 5/4/0/3/5/4=73 | S1；E监管/成交/灾害风险明确，T区别银行，其余按行业基线 | 待核验；核心 | 入选，后续可拆分 |
| I03 | 房地产与物业 / Real Estate and Property Services | `industry_sector`; `sector:industry_real_estate_property` | 利率、供地、销售、城市政策；金融地产 | 5/4/0/3/5/4=73 | S1；E地产政策明确，T连接信用与建设链，其余按基线 | 待核验；核心 | 入选 |
| I04 | 煤炭 / Coal | `industry_sector`; `sector:industry_coal` | 供给、安监、进口与电力需求；能源电力 | 5/4/0/3/5/4=73 | S1；E供需/安全触发明确，T区别油气电力，其余按基线 | 可关联煤炭行业指数，非煤价 commodity；核心 | 入选 |
| I05 | 石油天然气与炼化 / Oil Gas and Refining | `industry_sector`; `sector:industry_oil_gas_refining` | 地缘、产量、运输、炼化价差；能源电力 | 5/4/0/3/5/4=73 | S1；E地缘供需明确，T上游/炼化链独立，其余按基线 | 行业指数待核验，原油价格仅 benchmark；核心 | 入选 |
| I06 | 电力与公用事业 / Power and Utilities | `industry_sector`; `sector:industry_power_utilities` | 电价、燃料、负荷、容量与监管；能源电力/交通公用 | 5/4/0/3/5/4=73 | S1；E监管和供需明确，T区别设备制造，其余按基线 | 待核验；核心 | 入选 |
| I07 | 有色金属与新材料 / Nonferrous Metals and New Materials | `industry_sector`; `sector:industry_nonferrous_new_materials` | 矿供给、出口管制、制造需求；有色化工材料 | 5/4/0/3/5/4=73 | S1/S2；E资源与技术触发明确，T区别商品价格，其余按基线 | 行业指数待核验；核心 | 入选 |
| I08 | 基础化工 / Basic Chemicals | `industry_sector`; `sector:industry_basic_chemicals` | 原料、环保、产能、农业需求；有色化工材料 | 5/4/0/3/5/4=73 | S1/S3；E供需/监管明确，T工艺链独立，其余按基线 | 待核验；核心 | 入选 |
| I09 | 高端机械与工业自动化 / Advanced Machinery and Automation | `industry_sector`; `sector:industry_advanced_machinery_automation` | 制造资本开支、更新改造、自动化；工业基建 | 5/4/0/3/5/4=73 | S2/S3；E资本开支明确，T设备周期独立，其余按基线 | 可关联高端制造指数；核心 | 入选 |
| I10 | 建筑与基础设施 / Construction and Infrastructure | `industry_sector`; `sector:industry_construction_infrastructure` | 财政、专项债、项目开工、基建投资；工业基建 | 5/4/0/3/5/4=73 | S1/S10；E财政项目明确，T区别地产/机械，其余按基线 | 可关联基建指数；核心 | 入选 |
| I11 | 半导体与电子 / Semiconductors and Electronics | `industry_sector`; `sector:industry_semiconductors_electronics` | 技术迭代、出口管制、库存与资本开支；半导体电子 | 5/4/0/3/5/4=73 | S2/S5；E技术/监管明确，T产业链独立，其余按基线 | 多个细分 benchmark 待 Review；核心 | 入选 |
| I12 | 软件与通信 / Software and Communications | `industry_sector`; `sector:industry_software_communications` | AI、云、运营商资本开支与数据监管；AI软件通信 | 5/4/0/3/5/4=73 | S2/S13；E技术/监管明确，T区别硬件制造，其余按基线 | 待核验；核心 | 入选 |
| I13 | 汽车与零部件 / Automobiles and Components | `industry_sector`; `sector:industry_automobiles_components` | 销量、补贴、关税、智能化与供应链；汽车新能源 | 5/4/0/3/5/4=73 | S1/S2；E政策/需求明确，T整车链独立，其余按基线 | 可关联全指汽车类指数；核心 | 入选 |
| I14 | 电力设备与电池 / Power Equipment and Batteries | `industry_sector`; `sector:industry_power_equipment_batteries` | 电网投资、新能源装机、电池技术与原料；汽车新能源/能源电力 | 5/4/0/3/5/4=73 | S2/S8；E资本开支/技术明确，T区别公用事业，其余按基线 | 可关联新能源电池/装备指数；核心 | 入选 |
| I15 | 医药与生物科技 / Pharmaceuticals and Biotechnology | `industry_sector`; `sector:industry_pharma_biotech` | 审批、医保、研发突破、公共卫生；医药生科 | 5/4/0/3/5/4=73 | S1/S2/S12；E监管/技术明确，T研发链独立，其余按基线 | 可关联生物医药指数；核心 | 入选 |
| I16 | 消费与零售 / Consumer and Retail | `industry_sector`; `sector:industry_consumer_retail` | 收入、促消费、渠道、价格与消费信心；消费农业 | 5/4/0/3/5/4=73 | S1；E需求/政策明确，T区别农业上游，其余按基线 | 待核验；核心 | 入选 |
| I17 | 农业与食品 / Agriculture and Food | `industry_sector`; `sector:industry_agriculture_food` | 天气、种植、疫病、粮价与食品安全；消费农业 | 5/4/0/3/5/4=73 | S1/S9；E供给/监管明确，T农业链独立，其余按基线 | 可关联农业主题指数；核心 | 入选 |
| I18 | 交通运输与物流 / Transportation and Logistics | `industry_sector`; `sector:industry_transportation_logistics` | 运价、油价、贸易、港口航空与物流需求；交通公用 | 5/4/0/3/5/4=73 | S1；E贸易/成本明确，T运输链独立，其余按基线 | 待核验；核心 | 入选 |
| I19 | 环保与水务 / Environmental Services and Water Utilities | `industry_sector`; `sector:industry_environment_water` | 环保监管、水价、治理投资与排放标准；交通公用 | 5/4/0/3/5/4=73 | S1/S2；E监管/投资明确，T区别电力公用，其余按基线 | 待核验；核心 | 入选 |
| I20 | 国防与航空航天 / Defense and Aerospace | `industry_sector`; `sector:industry_defense_aerospace` | 地缘、安全预算、装备采购与航天任务；国防航天卫星 | 5/4/0/3/5/4=73 | S2/S6；E地缘/采购明确，T安全链独立，其余按基线 | 可关联军工/航天指数；核心 | 入选 |

## 原始候选：concept 20

| ID | 中文主名 / English alias | semantic / canonical | 主要事件触发与传导簇 | 评分 | 分项证据 | benchmark / 层级 | 建议结果 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| C01 | 人工智能 / Artificial Intelligence | `theme_sector`; `sector:theme_artificial_intelligence` | 模型突破、算力政策、应用监管；AI软件通信 | 5/4/0/3/4/3=68 | S2/S4；E技术监管明确，T跨软硬件，Q0，D3，L4，M3 | 待核验；核心 | 入选；X09 合并至此 |
| C02 | 算力基础设施 / Computing Infrastructure | `theme_sector`; `sector:theme_computing_infrastructure` | 芯片供给、数据中心资本开支、电力约束；AI软件通信 | 5/4/0/3/4/3=68 | S3/S13；E资本开支明确，T基础设施链独立，其余按概念基线 | 待核验；核心 | 入选 |
| C03 | 数据中心与云 / Data Centers and Cloud | `theme_sector`; `sector:theme_data_centers_cloud` | 云需求、能耗政策、网络建设；AI软件通信 | 5/4/0/3/4/3=68 | S3/S13；E需求/监管明确，T运营暴露独立，其余按基线 | 待核验；核心 | 入选 |
| C04 | 机器人与具身智能 / Robotics and Embodied AI | `theme_sector`; `sector:theme_robotics_embodied_ai` | 机器人技术、自动化投资、劳动力变化；工业基建 | 5/4/0/3/4/3=68 | S3/S8；E技术/投资明确，T整机零部件链独立，其余按基线 | 国证机器人类 benchmark 待关联；核心 | 入选；X05 合并至此 |
| C05 | 低空经济 / Low-altitude Economy | `theme_sector`; `sector:theme_low_altitude_economy` | 空域、适航、基础设施与场景开放；政策主题/交通公用 | 5/4/0/3/4/3=68 | S4/S9；E政策事件明确，T区别商业航天，其余按基线 | 通用航空 benchmark 仅交叉；核心 | 入选 |
| C06 | 商业航天与卫星产业 / Commercial Space and Satellite Industry | `theme_sector`; `sector:theme_commercial_space_satellite` | 发射、组网、订单、出口管制；国防航天卫星 | 5/4/0/3/4/3=68 | S4/S6；E任务/采购明确，T覆盖完整卫星链，其余按基线 | 中证卫星产业待关联；核心 | 入选；X03 合并至此 |
| C07 | 卫星通信与导航 / Satellite Communications and Navigation | `theme_sector`; `sector:theme_satellite_communications_navigation` | 星座组网、北斗应用、频谱与终端；国防航天卫星 | 5/4/0/3/4/3=68 | S4/S7；E组网/应用明确，T区别发射制造，其余按基线 | 中证卫星导航待关联；核心 | 入选；与C06上下位 |
| C08 | 网络与数据安全 / Cybersecurity and Data Security | `theme_sector`; `sector:theme_cyber_data_security` | 安全事件、合规、国产化与预算；AI软件通信/政策主题 | 5/4/0/3/4/3=68 | S3/S10；E监管/安全事件明确，T安全支出独立，其余按基线 | 上证信息安全类待关联；核心 | 入选 |
| C09 | 半导体自主可控 / Semiconductor Supply-chain Resilience | `theme_sector`; `sector:theme_semiconductor_resilience` | 出口管制、国产替代、晶圆厂资本开支；半导体电子 | 5/4/0/3/4/3=68 | S3/S5；E地缘/技术明确，T政策暴露区别行业骨架，其余按基线 | 材料设备/芯片设计 benchmark；核心 | 入选 |
| C10 | 新型储能 / Advanced Energy Storage | `theme_sector`; `sector:theme_advanced_energy_storage` | 电力市场、装机、系统安全与成本；能源电力 | 5/4/0/3/4/3=68 | S4/S8；E政策/技术明确，T区别电池制造，其余按基线 | 新能源电池指数仅交叉；核心 | 入选 |
| C11 | 固态与新一代电池 / Solid-state and Next-generation Batteries | `theme_sector`; `sector:theme_next_generation_batteries` | 技术突破、量产、材料供给与安全标准；汽车新能源 | 5/4/0/3/4/3=68 | S4；E技术/量产明确，T电池路线独立，其余按基线 | 待核验；扩展 | 入选 |
| C12 | 氢能 / Hydrogen Energy | `theme_sector`; `sector:theme_hydrogen_energy` | 电解槽、绿氢项目、储运与补贴；能源电力 | 5/4/0/3/4/3=68 | S4；E项目/政策明确，T能源载体链独立，其余按基线 | 待核验；扩展 | 入选 |
| C13 | 智能电网 / Smart Grid | `theme_sector`; `sector:theme_smart_grid` | 电网投资、消纳、配网数字化与电价；能源电力 | 5/4/0/3/4/3=68 | S2/S3；E资本开支明确，T电网环节独立，其余按基线 | 待核验；扩展 | 入选 |
| C14 | 核电与先进核能 / Nuclear Power and Advanced Nuclear Energy | `theme_sector`; `sector:theme_nuclear_advanced_energy` | 核准、建设、设备订单与技术路线；能源电力 | 5/4/0/3/4/3=68 | S2/S4；E核准/项目明确，T基荷技术独立，其余按基线 | 待核验；扩展 | 入选 |
| C15 | 风电与光伏装备 / Wind and Solar Equipment | `theme_sector`; `sector:theme_wind_solar_equipment` | 装机、招标、贸易、原料与产能；能源电力 | 5/4/0/3/4/3=68 | S8；E招标/贸易明确，T装备链独立，其余按基线 | 国证风电光伏装备待关联；扩展 | 入选 |
| C16 | 智能网联新能源汽车 / Intelligent Connected New Energy Vehicles | `theme_sector`; `sector:theme_intelligent_connected_nev` | 补贴、销量、自动驾驶监管与电池技术；汽车新能源 | 5/4/0/3/4/3=68 | S2/S13；E政策/技术明确，T跨整车电子链，其余按基线 | 新能源车类指数待核验；扩展 | 入选 |
| C17 | 创新药 / Innovative Drugs | `theme_sector`; `sector:theme_innovative_drugs` | 临床结果、审批、医保与授权交易；医药生科 | 5/4/0/3/4/3=68 | S2/S12；E研发/监管明确，T区别传统制药，其余按基线 | 待核验；扩展 | 入选 |
| C18 | 高端医疗器械 / Advanced Medical Devices | `theme_sector`; `sector:theme_advanced_medical_devices` | 注册审批、采购政策、技术替代；医药生科 | 5/4/0/3/4/3=68 | S2/S12；E审批/采购明确，T器械链独立，其余按基线 | 待核验；扩展 | 入选 |
| C19 | 数字经济与数据要素 / Digital Economy and Data Elements | `theme_sector`; `sector:theme_digital_economy_data_elements` | 数据制度、云服务、数字基础设施与监管；AI软件通信/政策主题 | 5/4/0/3/4/3=68 | S2/S3；E制度/投资明确，T数据资产链独立，其余按基线 | 待核验；扩展 | 入选 |
| C20 | 国企改革与央企科技 / SOE Reform and Central SOE Technology | `theme_sector`; `sector:theme_soe_reform_technology` | 国资改革、重组、考核与科技投入；政策主题 | 5/4/0/3/4/3=68 | S3/S10；E制度事件明确，T所有制与科技投入暴露独立，其余按基线 | 央企科技类指数待核验；扩展 | 入选 |

## 原始候选：index_sector 20

`index_sector` 是 source taxonomy；每项 semantic classification 仍为 `industry_sector` 或 `theme_sector`。每项都建议关联同名官方 benchmark，但不得复制 benchmark 职责。

| ID | 中文主名 / English alias | semantic / canonical | 主要事件触发与传导簇 | 评分 | 分项证据 | benchmark / 层级 | 建议结果 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| X01 | 科创半导体材料设备 / STAR Semiconductor Materials and Equipment | `industry_sector`; `sector:industry_star_semiconductor_materials_equipment` | 晶圆厂资本开支、出口管制、材料供给；半导体电子 | 4/4/0/5/4/4=71 | S5；E/T产业环节明确，Q0，D5官方定义，L4，M4 | 同名官方指数；扩展 | 入选 |
| X02 | 科创芯片设计 / STAR Chip Design | `industry_sector`; `sector:industry_star_chip_design` | 芯片迭代、终端需求、EDA/IP与监管；半导体电子 | 4/4/0/5/4/4=71 | S5；E/T设计环节独立，Q0，D5，L4，M4 | 同名官方指数；扩展 | 入选 |
| X03 | 中证卫星产业 / CSI Satellite Industry | `theme_sector`; `sector:theme_commercial_space_satellite` | 发射、组网、订单与遥感应用；国防航天卫星 | 4/4/0/5/4/4=71 | S6；E/T完整产业链明确，Q0，D5，L4，M4 | 中证卫星产业；扩展 | 合并至 C06，保留 mapping |
| X04 | 中证卫星导航产业 / CSI Navigation Satellite Industry | `theme_sector`; `sector:theme_satellite_navigation_index_exposure` | 北斗应用、终端、频谱与导航服务；国防航天卫星 | 4/4/0/5/4/4=71 | S7；E/T导航链明确，Q0，D5，L4，M4 | 中证卫星导航；扩展 | 入选；与C07交叉但范围待核验 |
| X05 | 国证机器人产业 / CNI Robot Industry | `theme_sector`; `sector:theme_robotics_embodied_ai` | 自动化投资、机器人技术与零部件；工业基建 | 4/4/0/5/4/4=71 | S8；E/T机器人链明确，Q0，D5，L4，M4 | 国证机器人产业；扩展 | 合并至 C04，保留 mapping |
| X06 | 国证风电光伏装备 / CNI Wind and Solar Equipment | `industry_sector`; `sector:industry_wind_solar_equipment_index_exposure` | 装机、招标、贸易与材料成本；能源电力 | 4/4/0/5/4/4=71 | S8；E/T装备定义明确，Q0，D5，L4，M4 | 国证风电光伏装备；扩展 | 入选；与C15交叉待核验 |
| X07 | 国证新能源电池 / CNI New Energy Battery | `industry_sector`; `sector:industry_new_energy_battery_index_exposure` | 储能、动力电池、系统集成与安全；汽车新能源 | 4/4/0/5/4/4=71 | S8；E/T电池链明确，Q0，D5，L4，M4 | 国证新能源电池；扩展 | 入选 |
| X08 | 国证通用航空产业 / CNI General Aviation Industry | `theme_sector`; `sector:theme_general_aviation_industry` | 空域、适航、机场与航空器需求；交通公用/政策主题 | 4/4/0/5/4/4=71 | S9；E/T通航链明确，Q0，D5公告确认，L4，M4 | 国证通用航空产业；扩展 | 入选；与C05交叉不等同 |
| X09 | 创业板人工智能 / ChiNext Artificial Intelligence | `theme_sector`; `sector:theme_artificial_intelligence` | AI技术、算力、应用与监管；AI软件通信 | 4/4/0/5/4/4=71 | S9；E/T主题明确，Q0，D5公告确认，L4，M4 | 创业板人工智能；扩展 | 合并至 C01，保留 mapping |
| X10 | 国证化肥农药 / CNI Fertilizer and Pesticide | `industry_sector`; `sector:industry_fertilizer_pesticide` | 农资价格、种植周期、环保与粮食安全；消费农业/有色化工材料 | 4/4/0/5/4/4=71 | S9；E/T农化链明确，Q0，D5公告确认，L4，M4 | 国证化肥农药；扩展 | 入选 |
| X11 | 上证信息安全 / SSE Information Security | `theme_sector`; `sector:theme_information_security_index_exposure` | 网络安全事件、合规与政企预算；AI软件通信 | 4/4/0/4/4/4=68 | S10；E/T安全支出明确，Q0，D4方法待取，L4，M4 | 上证信息安全；观察 | 入选；与C08交叉待核验 |
| X12 | 中证高端制造 / CSI Advanced Manufacturing | `industry_sector`; `sector:industry_advanced_manufacturing_index_exposure` | 设备更新、制造投资、技术突破；工业基建 | 4/4/0/4/4/4=68 | S10；E/T制造链明确，Q0，D4，L4，M4 | 中证高端制造；观察 | 入选 |
| X13 | 中证基建 / CSI Infrastructure | `industry_sector`; `sector:industry_infrastructure_index_exposure` | 财政、专项债、项目开工；工业基建 | 4/4/0/4/4/4=68 | S10；E/T基建周期明确，Q0，D4，L4，M4 | 中证基建；观察 | 入选；与I10交叉待核验 |
| X14 | 中证银发经济 / CSI Silver Economy | `theme_sector`; `sector:theme_silver_economy` | 人口、养老政策、医疗与消费服务；医药生科/消费农业 | 4/4/0/4/4/4=68 | S10；E/T人口政策明确，Q0，D4，L4，M4 | 中证银发经济；观察 | 入选 |
| X15 | 中证全指汽车 / CSI All Share Automobiles | `industry_sector`; `sector:industry_automobile_index_exposure` | 销量、关税、技术与供应链；汽车新能源 | 4/4/0/4/4/4=68 | S10；E/T汽车链明确，Q0，D4，L4，M4 | 中证全指汽车；观察 | 入选；与I13交叉待核验 |
| X16 | 科创新能源 / STAR New Energy | `theme_sector`; `sector:theme_star_new_energy` | 装机、技术、贸易与资本开支；能源电力 | 4/4/0/4/4/4=68 | S11；E/T新能源明确，Q0，D4，L4，M4 | 科创新能源；观察 | 入选 |
| X17 | 科创工业机械 / STAR Industrial Machinery | `industry_sector`; `sector:industry_star_industrial_machinery` | 制造资本开支、工控与设备更新；工业基建 | 4/4/0/4/4/4=68 | S11；E/T工业机械明确，Q0，D4，L4，M4 | 科创工业机械；观察 | 入选 |
| X18 | 科创生物医药 / STAR Biomedicine | `industry_sector`; `sector:industry_star_biomedicine` | 审批、临床、医保与研发；医药生科 | 4/4/0/4/4/4=68 | S12；E/T生物医药明确，Q0，D4，L4，M4 | 科创生物医药；观察 | 入选 |
| X19 | 科创新材料 / STAR New Materials | `industry_sector`; `sector:industry_star_new_materials` | 技术突破、国产化、资源与制造需求；有色化工材料 | 4/4/0/4/4/4=68 | S12；E/T材料链明确，Q0，D4，L4，M4 | 科创新材料；观察 | 入选 |
| X20 | 科创高端装备 / STAR Advanced Equipment | `industry_sector`; `sector:industry_star_advanced_equipment` | 设备投资、国产化与重大工程；工业基建/国防安全 | 4/4/0/4/4/4=68 | S12；E/T装备链明确，Q0，D4，L4，M4 | 科创高端装备；观察 | 入选 |

## 建议正式入选结果

- 原始候选：60。
- 建议正式 canonical sector：57；`industry` 来源主导 20、`concept` 来源主导 20、`index_sector` 来源主导 17。
- 合并但保留多来源 mapping：X03→C06、X05→C04、X09→C01，共减少 3 个重复 canonical sector。
- 70 分及以上：30 个（行业 20、官方定义较完整指数板块 10）；低于 70 分：30 个。低分主要因为 Q=0 和具体 source mapping/方法证据不足，不因目标数量提升分数。
- 本轮没有 benchmark-only 的新原始候选；所有 X 项均为产业/主题指数板块。对应指数实体仍应作为 benchmark，通过 `tracked_by_benchmark` 关联，而不是替代 sector。

## 重复、上下位与交叉关系建议

| 关系 | 建议 |
| --- | --- |
| C01 ↔ X09、C04 ↔ X05、C06 ↔ X03 | 完全同义方向，合并 canonical sector，保留 concept/index_sector 两条 source mapping |
| C06 > C07；X03 > X04 | 卫星产业为上位，卫星通信导航为下位；本 change 暂不写正式层级边 |
| I11 > X01/X02；C09 ↔ I11/X01/X02 | 行业骨架、产业环节、政策主题分别保留，建立交叉关系建议，不强制合并 |
| I14 > C10/C11/C13/C15；X06/X07/X16 交叉 | 电力设备与电池为行业骨架，储能/新电池/智能电网/风光为主题或指数暴露 |
| I09 > C04/X05/X17/X20 | 高端机械为行业骨架，机器人、工业机械、高端装备是不同粒度暴露 |
| I15 > C17/C18/X18；I17 ↔ X10 | 医药与农业行业骨架分别承载细分主题/指数暴露 |

## 传导簇覆盖统计（按原始 60，可多选）

| 传导簇 | 候选数 | 代表候选 |
| --- | ---: | --- |
| 金融地产 | 3 | I01-I03 |
| 能源电力 | 12 | I04-I06、I14、C10-C15、X06-X07/X16 |
| 有色化工材料 | 5 | I07-I08、X10、X19、C09交叉 |
| 工业基建 | 9 | I09-I10、C04、X05、X12-X13、X17、X20 |
| 半导体电子 | 5 | I11、C09、X01-X02、X19交叉 |
| AI软件通信 | 8 | I12、C01-C03/C08/C19、X09/X11 |
| 汽车新能源 | 5 | I13-I14、C11/C16、X07/X15交叉 |
| 医药生科 | 5 | I15、C17-C18、X14、X18 |
| 消费农业 | 4 | I16-I17、X10、X14 |
| 交通公用 | 5 | I06、I18-I19、C05、X08 |
| 国防航天卫星 | 7 | I20、C06-C07、X03-X04/X08/X20交叉 |
| 政策主题 | 7 | C05/C08/C19-C20 等 |

统计用于覆盖检查，不是互斥分类；同一候选可以进入多个传导簇。

## 旧 PG 60 条迁移对照

### 旧 concept 20

| 旧候选 | 动作 | 新去向 |
| --- | --- | --- |
| 人工智能 | retain | C01 |
| 算力 | retain | C02 算力基础设施（改名） |
| 数据中心 | merge | C03 数据中心与云 |
| 半导体概念 | merge | C09 + I11，待定义范围 |
| 芯片概念 | merge | C09 + I11，待定义范围 |
| 机器人概念 | retain | C04（改名） |
| 低空经济 | retain | C05 |
| 商业航天 | merge | C06 |
| 卫星导航 | merge | C07 |
| 军工 | merge | I20 |
| 新能源汽车 | retain | C16（改名） |
| 固态电池 | merge | C11 |
| 储能 | retain | C10 新型储能（改名） |
| 光伏概念 | merge | C15/X06，待范围核验 |
| 风电 | merge | C15/X06 |
| 氢能源 | retain | C12 |
| 核电 | retain | C14（改名） |
| 数字货币 | replace | C19 数字经济与数据要素；旧对象是否另留待 Review |
| 跨境电商 | merge | I16 消费与零售；若有独立定义可候补 |
| 国企改革 | retain | C20（改名） |

### 旧 industry 20

| 旧候选 | 动作 | 新去向 |
| --- | --- | --- |
| 半导体及元件 | merge | I11 |
| 通信设备 | merge | I12 |
| 计算机应用 | merge | I12 |
| 传媒 | merge | I16 消费与零售；独立保留待 Review |
| 证券 | merge | I02 |
| 银行 | retain | I01 |
| 保险及其他 | merge | I02 |
| 汽车整车 | merge | I13 |
| 汽车零部件 | merge | I13 |
| 电力设备 | merge | I14 |
| 光伏设备 | merge | I14/C15 |
| 电池 | merge | I14 |
| 医疗服务 | merge | I15；服务子行业待 Review |
| 化学制药 | merge | I15 |
| 中药 | merge | I15；独立保留待 Review |
| 工业金属 | merge | I07 |
| 煤炭开采加工 | retain | I04（改名） |
| 石油加工贸易 | merge | I05 |
| 房地产开发 | merge | I03 |
| 消费电子 | merge | I11/I16，按业务边界 Review |

### 旧 index 20

| 旧候选 | 动作 | 新去向 |
| --- | --- | --- |
| 上证50 | benchmark-only | 不进入 sector 60 |
| 沪深300 | benchmark-only | 不进入 sector 60 |
| 中证500 | benchmark-only | 不进入 sector 60 |
| 中证1000 | benchmark-only | 不进入 sector 60 |
| 科创50 | benchmark-only | 不进入 sector 60 |
| 创业板50 | benchmark-only | 不进入 sector 60 |
| 深证100 | benchmark-only | 不进入 sector 60 |
| 北证50 | benchmark-only | 不进入 sector 60 |
| 中证A500 | benchmark-only | 不进入 sector 60 |
| 中证红利 | benchmark-only | 不进入 sector 60；style sector 另行 Review |
| 央视50 | benchmark-only | 不进入 sector 60 |
| 中证全指 | benchmark-only | 不进入 sector 60 |
| 万得全A | benchmark-only | 不进入 sector 60 |
| 国证2000 | benchmark-only | 不进入 sector 60 |
| 恒生指数 | benchmark-only | 不进入 sector 60 |
| 恒生科技指数 | benchmark-only | 不进入 sector 60；如需港股科技 sector 另行候补 |
| 纳斯达克100 | benchmark-only | 不进入 sector 60 |
| 标普500 | benchmark-only | 不进入 sector 60 |
| 费城半导体指数 | benchmark-only | 不进入新 60；可作为 I11 的海外 benchmark 候选 |
| MSCI中国A50互联互通 | benchmark-only | 不进入 sector 60 |

## 待用户 Review

1. 批准或调整建议正式 57 个 canonical sector，以及三组同义合并。
2. 决定低于 70 分的 30 项是否进入补证流程；未经 override 不因数量目标自动批准。
3. 确认 I02、I07、I12-I16 等较宽行业骨架是否需要拆分；拆分会改变最终总数。
4. 对 X04、X06-X08、X10-X20 获取逐条编制方案后，再确认 `tracked_by_benchmark` 和 source mapping。
5. 确认旧数字货币、跨境电商、传媒、中药、医疗服务等对象是合并、独立保留还是作为高质量候补；若拆分后仍保持证据充分，最终总数可在 50-60 内调整。
