# 产业链候选冻结 Review

## 1. 状态与边界

- 本文是 Apply 第 1 阶段候选清单，不是正式 seed、PostgreSQL 或 Neo4j 事实。
- 2026-07-12 用户已批准：4 张新表、`chain_node_profiles` 增量、`supplies_to/depends_on/substitutes_for`、13 类 physical constraint、`mapped_to_sector`、AI 算力基础设施与半导体制造两条试点，以及逐层 stateful 门禁。
- “复用”表示 `backend/data/entity_foundation/chain_nodes.json` 已有 stable key；本轮未查询 PostgreSQL，PG 实际存在与状态必须在后续获准的只读检查中确认。
- “改进”表示复用 stable key，但建议补齐 definition、node_category、unit_of_analysis、granularity_note；不改变节点身份。
- “新增”仅是候选；缺少足够权威来源时明确标记，不为满足数量虚构来源。
- 所有 physical constraint 均为 `candidate`，不得声称 reviewed/approved。
- 第二批机器人、新能源汽车/储能、创新药/生物制造不进入本文 seed 候选。

## 2. 来源登记

| 代码 | 来源 | 类型 | URL | 用途 |
|---|---|---|---|---|
| S1 | IEA, Energy and AI | 国际机构研究 | https://www.iea.org/reports/energy-and-ai/energy-demand-from-ai | 数据中心、加速服务器、电力和冷却边界 |
| S2 | NVIDIA Data Center Architecture | 官方技术文档 | https://docs.nvidia.com/ncx/ncp-software-reference-guide/latest/data-center-architecture.html | GPU rack、网络、存储、NVLink 架构 |
| S3 | NVIDIA NVLink and NVLink Switch | 官方产品技术页 | https://www.nvidia.com/en-eu/data-center/nvlink/ | scale-up interconnect、bandwidth、latency |
| S4 | NVIDIA MGX Platform | 官方平台页 | https://www.nvidia.com/en-us/data-center/products/mgx/ | GPU/CPU/networking/power/cooling 的 rack-scale 组成 |
| S5 | Micron HBM | 官方产品技术页 | https://www.micron.com/products/memory/hbm | HBM 与 AI accelerator 的带宽/功耗边界 |
| S6 | TSMC CoWoS | 官方技术页 | https://3dfabric.tsmc.com/english/dedicatedFoundry/technology/cowos.htm | advanced packaging、HBM、interposer |
| S7 | ASHRAE Data Center Resources | 标准组织技术资料 | https://www.ashrae.org/technical-resources/bookstore/datacom-series | thermal、liquid cooling、reliability |
| S8 | Open Compute Project Rack & Power | 开放标准组织 | https://www.opencompute.org/index.php/community/rack-and-power | rack power、conversion、storage、control |
| S9 | Synopsys EDA | 官方技术页 | https://www.synopsys.com/silicon-design.html | IC design、verification、manufacturing tools |
| S10 | SEMI Semiconductor Manufacturing Process | 行业组织技术资料 | https://www.semi.org/en/ehs_PFAS/mfg_markets_overview | 半导体设备、材料和制造流程总览 |
| S11 | ASML Products | 官方技术页 | https://www.asml.com/en/products | EUV/DUV lithography、metrology/inspection |
| S12 | Applied Materials Semiconductor Products | 官方技术页 | https://www.appliedmaterials.com/eu/en/semiconductor/products.html | deposition、CMP、etch、ion implant、analysis |
| S13 | KLA BBP Wafer Inspection | 官方技术资料 | https://bbp.kla.com/ | wafer inspection 与 yield management |
| S14 | TSMC 2024 Annual Report / 3DFabric | 公司正式披露 | https://investor.tsmc.com/static/annualReports/2024/english/ebook/files/basic-html/page102.html | wafer/advanced packaging/HBM integration |
| S15 | 已批准 canonical sector review | repo 内已批准来源映射 | https://github.com/meierlink88/tidewise-ai/blob/03273effecb946ba21c953f6d12165d65b3dee88/openspec/changes/add-market-sector-foundation/candidate-review.md | `mapped_to_sector` 候选 |

来源说明：厂商官方资料可证明技术组成和产品位置，但不单独证明整个行业唯一性、当前短缺或投资结论。需要行业级证据的 physical constraint 均保持 candidate，并在“缺口”列说明。

## 3. 去重节点主清单

### 3.1 AI 算力基础设施（13 个）

| # | 处理 | 中文名 | English canonical name | stable key | 定义 | node_category | unit_of_analysis | granularity | 来源/缺口 |
|---:|---|---|---|---|---|---|---|---|---|
| 1 | 复用并改进 | 电网 | Power Grid | `chain_node:power_grid` | 向数据中心提供外部输配电与并网能力的基础设施 | infrastructure | infrastructure | 与发电、电力商品及机架内供电分开 | S1；需补中国电网接入技术来源 |
| 2 | 复用并改进 | 电力 | Electric Power | `chain_node:electric_power` | 数据中心 IT 与辅助设施消耗的电能 | resource | material | 作为能源投入，不等同于电网设施 | S1 |
| 3 | 复用并改进 | 数据中心 | Data Center | `chain_node:data_center` | 容纳服务器、存储、网络及供电冷却设施的物理场所 | infrastructure | system | 保持设施级，不拆到单栋园区 | S1、S2 |
| 4 | 新增 | 机架级供电系统 | Rack-scale Power System | `chain_node:rack_power_system` | 从设施输入到 AI rack 的变换、配电、备电与控制系统 | equipment | system | 与外部电网、发电及芯片电源器件分开 | S4、S8 |
| 5 | 新增 | 液冷系统 | Liquid Cooling System | `chain_node:liquid_cooling_system` | 面向高功率密度 IT 设备的液体传热、换热与循环系统 | infrastructure | system | 以 rack/facility cooling system 为分析单位 | S7；具体 CDU/冷板细分待后续 |
| 6 | 新增 | AI 存储系统 | AI Storage System | `chain_node:ai_storage_system` | 支撑训练数据摄取、checkpoint 与推理数据访问的存储系统 | infrastructure | system | 不与 HBM 或服务器内存合并 | S2；需补独立存储标准/厂商交叉来源 |
| 7 | 新增 | 数据中心网络 | Data Center Networking | `chain_node:data_center_networking` | 连接 rack、存储和集群的 scale-out/front-end 网络系统 | infrastructure | system | 与 rack 内 scale-up interconnect 分开 | S2、S4 |
| 8 | 新增 | 光互连 | Optical Interconnect | `chain_node:optical_interconnect` | 数据中心内以光器件和光链路承载高速互连的技术单元 | component | component | 保持光链路/模块层，不下钻激光器/DSP | S2；需补 OIF/IEEE 标准来源 |
| 9 | 新增 | 规模扩展互连 | Scale-up Interconnect | `chain_node:scale_up_interconnect` | rack 内 GPU/accelerator 之间的高带宽低时延互连 | component | system | 与 scale-out data center network 分开 | S3 |
| 10 | 复用并改进 | GPU | Graphics Processing Unit | `chain_node:gpu` | 执行并行 AI 训练与推理计算的加速处理器 | component | component | 芯片级，不绑定单一厂商或型号 | S2、S4 |
| 11 | 新增 | 高带宽内存 | High-bandwidth Memory | `chain_node:hbm` | 与 AI accelerator 近封装、提供高带宽访问的堆叠内存 | component | component | 与普通 DRAM、SSD 分开 | S5、S6 |
| 12 | 新增（两链共享） | 先进封装 | Advanced Packaging | `chain_node:advanced_packaging` | 通过 interposer、3D/2.5D integration 等集成 logic、HBM 与互连的制造环节 | process | process | 制造过程级，不绑定 CoWoS 单一品牌 | S6、S14 |
| 13 | 新增 | AI 服务器 | AI Server | `chain_node:ai_server` | 集成 accelerator、CPU、memory、networking、storage 和 power/cooling interface 的计算系统 | product | system | 服务器/compute tray 级，不等同数据中心 | S2、S4 |

### 3.2 半导体制造（14 个）

| # | 处理 | 中文名 | English canonical name | stable key | 定义 | node_category | unit_of_analysis | granularity | 来源/缺口 |
|---:|---|---|---|---|---|---|---|---|---|
| 1 | 复用并改进 | EDA 软件 | Electronic Design Automation | `chain_node:eda` | 用于集成电路设计、验证、实现和制造准备的软件工具体系 | service | service | 领域工具体系级，不拆单个软件模块 | S9 |
| 2 | 新增 | 半导体硅片 | Semiconductor Silicon Wafer | `chain_node:semiconductor_silicon_wafer` | 承载晶圆制造工艺的半导体级硅衬底 | material | material | 不复用光伏 `polysilicon`，纯度与用途不同 | S10；需补硅片厂商/SEMI wafer 标准来源 |
| 3 | 新增 | 光刻胶 | Photoresist | `chain_node:photoresist` | 在光刻曝光和显影中形成图形的感光材料 | material | material | 保持材料级，不拆配方 | S10；需补 SEMI/材料厂技术来源 |
| 4 | 新增 | 电子特气 | Electronic Specialty Gases | `chain_node:electronic_specialty_gases` | 用于沉积、刻蚀、掺杂等晶圆工艺的高纯气体集合 | material | material | 按电子级特气集合，后续可再拆具体气体 | S10；需补气体厂/SEMI 纯度标准 |
| 5 | 复用并改进 | 光刻机 | Lithography System | `chain_node:lithography_machine` | 将芯片图形投射到晶圆上的 DUV/EUV 制造系统 | equipment | equipment | 设备平台级，不拆 EUV/DUV 型号 | S11 |
| 6 | 新增 | 薄膜沉积设备 | Thin-film Deposition Equipment | `chain_node:deposition_equipment` | 在晶圆表面形成介质、金属或外延薄膜的设备 | equipment | equipment | 按工艺设备族，不拆 ALD/CVD/PVD | S12 |
| 7 | 新增 | 刻蚀设备 | Etch Equipment | `chain_node:etch_equipment` | 选择性移除晶圆材料、形成结构的制造设备 | equipment | equipment | 按设备族，不拆介质/导体/选择性刻蚀 | S12 |
| 8 | 新增 | 离子注入设备 | Ion Implantation Equipment | `chain_node:ion_implantation_equipment` | 向晶圆材料引入受控掺杂离子的设备 | equipment | equipment | 单一关键工艺设备族 | S12 |
| 9 | 新增 | CMP 设备 | Chemical Mechanical Planarization Equipment | `chain_node:cmp_equipment` | 通过化学机械作用实现晶圆表面平坦化的设备 | equipment | equipment | 设备级；CMP 材料后续另审 | S12 |
| 10 | 新增 | 量测与缺陷检测设备 | Metrology and Inspection Equipment | `chain_node:metrology_inspection_equipment` | 测量关键尺寸并发现晶圆缺陷、为工艺控制提供反馈的设备 | equipment | equipment | 合并 metrology/inspection，首版不再细拆 | S11、S13 |
| 11 | 新增 | 晶圆制造 | Wafer Fabrication | `chain_node:wafer_fabrication` | 在晶圆上重复执行成膜、光刻、刻蚀、掺杂、平坦化与检测的前道制造过程 | process | process | fab process 总体环节，具体设备仍保留独立节点 | S10、S14 |
| 12 | 新增（两链共享） | 先进封装 | Advanced Packaging | `chain_node:advanced_packaging` | 集成多 die、interposer、HBM 与高密度互连的后道制造环节 | process | process | 与普通封装区分，但不绑定厂商品牌 | S6、S14 |
| 13 | 新增 | 半导体测试 | Semiconductor Testing | `chain_node:semiconductor_testing` | 对晶圆或封装后器件执行电性、功能和可靠性测试的制造环节 | process | process | 合并 wafer/final test，后续按证据再拆 | S10；需补 SEMI/测试设备来源 |
| 14 | 新增 | 集成电路成品 | Integrated Circuit | `chain_node:integrated_circuit` | 完成制造、封装和测试后可进入电子系统的芯片产品 | product | product | 通用产出，不用 GPU 代表全部半导体产品 | S10 |

### 3.3 去重统计

- AI 算力基础设施：13 个。
- 半导体制造：14 个。
- 共享：`chain_node:advanced_packaging` 1 个。
- 去重合计：26 个。
- repo seed 已有并建议复用：`power_grid`、`electric_power`、`data_center`、`gpu`、`eda`、`lithography_machine`，共 6 个；均需 profile 改进。
- 建议新增：20 个。
- 明确不复用：`chain_node:polysilicon` 不能代替半导体硅片；`chain_node:semiconductor_equipment` 粒度过宽，不与具体设备节点同时进入两条试点 membership，保留其现有身份但本批不使用。

## 4. Membership Review 清单

所有条目均为本次建议，status 候选；来源引用节点清单。

### 4.1 AI 算力基础设施 membership

| order | node key | stage | role | core | 理由 |
|---:|---|---|---|---|---|
| 10 | `chain_node:power_grid` | infrastructure | infrastructure | yes | 数据中心外部供电和并网基础 |
| 20 | `chain_node:electric_power` | upstream | resource | yes | AI 设施持续运行的能源投入 |
| 30 | `chain_node:rack_power_system` | infrastructure | equipment | yes | 设施电力到 rack 的转换、分配和备电 |
| 40 | `chain_node:liquid_cooling_system` | infrastructure | infrastructure | yes | 高功率密度服务器散热 |
| 50 | `chain_node:data_center` | infrastructure | infrastructure | yes | 服务器、网络、存储、电力和冷却承载设施 |
| 60 | `chain_node:advanced_packaging` | midstream | process | yes | 集成 accelerator 与 HBM |
| 70 | `chain_node:hbm` | midstream | component | yes | accelerator 近封装高带宽内存 |
| 80 | `chain_node:gpu` | midstream | component | yes | AI 计算核心 accelerator |
| 90 | `chain_node:scale_up_interconnect` | midstream | component | yes | rack 内 accelerator 高带宽互连 |
| 100 | `chain_node:data_center_networking` | infrastructure | infrastructure | yes | rack 间和存储/服务网络 |
| 110 | `chain_node:optical_interconnect` | midstream | component | no | 高速长距离数据中心链路候选 |
| 120 | `chain_node:ai_storage_system` | infrastructure | infrastructure | no | 数据集、checkpoint、推理数据访问 |
| 130 | `chain_node:ai_server` | downstream | product | yes | 集成上述部件的交付系统 |

### 4.2 半导体制造 membership

| order | node key | stage | role | core | 理由 |
|---:|---|---|---|---|---|
| 10 | `chain_node:eda` | upstream | service | yes | 芯片设计、验证和制造准备 |
| 20 | `chain_node:semiconductor_silicon_wafer` | upstream | material | yes | 前道制造衬底 |
| 30 | `chain_node:photoresist` | upstream | material | yes | 光刻图形形成材料 |
| 40 | `chain_node:electronic_specialty_gases` | upstream | material | yes | 成膜、刻蚀、掺杂材料输入 |
| 50 | `chain_node:lithography_machine` | midstream | equipment | yes | 图形转移设备 |
| 60 | `chain_node:deposition_equipment` | midstream | equipment | yes | 薄膜形成设备 |
| 70 | `chain_node:etch_equipment` | midstream | equipment | yes | 材料移除和结构形成设备 |
| 80 | `chain_node:ion_implantation_equipment` | midstream | equipment | no | 掺杂设备 |
| 90 | `chain_node:cmp_equipment` | midstream | equipment | no | 平坦化设备 |
| 100 | `chain_node:metrology_inspection_equipment` | midstream | equipment | yes | 工艺量测、缺陷和良率反馈 |
| 110 | `chain_node:wafer_fabrication` | midstream | process | yes | 前道工艺总环节 |
| 120 | `chain_node:advanced_packaging` | downstream | process | yes | 多 die/HBM/互连后道集成 |
| 130 | `chain_node:semiconductor_testing` | downstream | process | yes | 电性、功能和可靠性验证 |
| 140 | `chain_node:integrated_circuit` | downstream | product | yes | 制造链最终通用产品 |

## 5. Canonical Topology Review 清单

所有关系均为 candidate；方向遵循 canonical edge，不以反向 `depends_on` 重复供应事实。

### 5.1 AI 算力基础设施 topology

| from | type | to | 来源 | 理由/证据缺口 |
|---|---|---|---|---|
| `advanced_packaging` | `supplies_to` | `gpu` | S6、S14 | 先进封装集成 logic 与 HBM；候选 |
| `hbm` | `supplies_to` | `gpu` | S5、S6 | HBM 作为 accelerator 近封装 memory；候选 |
| `gpu` | `supplies_to` | `ai_server` | S2、S4 | GPU 是 AI server compute component |
| `scale_up_interconnect` | `supplies_to` | `ai_server` | S3、S4 | rack/server scale-up fabric |
| `data_center_networking` | `supplies_to` | `data_center` | S2、S4 | 网络是 data center system component |
| `optical_interconnect` | `supplies_to` | `data_center_networking` | S2；缺 OIF/IEEE | 光链路承载网络互连，需补标准来源 |
| `ai_storage_system` | `supplies_to` | `data_center` | S2；需补交叉来源 | 存储是数据中心组成 |
| `rack_power_system` | `supplies_to` | `ai_server` | S4、S8 | rack power 转换/分配给 IT gear |
| `data_center` | `depends_on` | `power_grid` | S1 | 设施级并网/供电依赖，非反向供应重复 |
| `ai_server` | `depends_on` | `liquid_cooling_system` | S4、S7 | 对高功率密度系统的热管理依赖；具体适用机型需 Review |

`electric_power` 是能源投入，不用 topology 表达；待 commodity/能源模型具备适当实体后再评估 `uses_commodity`。

### 5.2 半导体制造 topology

| from | type | to | 来源 | 理由/证据缺口 |
|---|---|---|---|---|
| `eda` | `supplies_to` | `wafer_fabrication` | S9 | EDA/制造工具支持芯片设计和制造准备 |
| `semiconductor_silicon_wafer` | `supplies_to` | `wafer_fabrication` | S10 | 晶圆是制造衬底 |
| `photoresist` | `supplies_to` | `wafer_fabrication` | S10；需补材料标准 | 光刻材料输入 |
| `electronic_specialty_gases` | `supplies_to` | `wafer_fabrication` | S10；需补纯度标准 | 多工艺材料输入 |
| `lithography_machine` | `supplies_to` | `wafer_fabrication` | S11 | 图形转移设备能力 |
| `deposition_equipment` | `supplies_to` | `wafer_fabrication` | S12 | 薄膜沉积设备能力 |
| `etch_equipment` | `supplies_to` | `wafer_fabrication` | S12 | 刻蚀设备能力 |
| `ion_implantation_equipment` | `supplies_to` | `wafer_fabrication` | S12 | 离子注入设备能力 |
| `cmp_equipment` | `supplies_to` | `wafer_fabrication` | S12 | CMP 设备能力 |
| `metrology_inspection_equipment` | `supplies_to` | `wafer_fabrication` | S11、S13 | 工艺控制和缺陷反馈能力 |
| `wafer_fabrication` | `supplies_to` | `advanced_packaging` | S14 | 前道 die 进入先进封装 |
| `advanced_packaging` | `supplies_to` | `semiconductor_testing` | S6、S10 | 封装后进入测试；需补测试流程权威资料 |
| `semiconductor_testing` | `supplies_to` | `integrated_circuit` | S10；需补测试标准 | 测试合格后形成可交付产品 |

首批不建议 `substitutes_for`：当前资料不足以证明节点级可替代关系，避免凭方法论猜测。

## 6. Physical Constraint Review 清单

全部状态：`candidate`。以下只是 AI 基于技术资料生成的建议，必须补齐权威技术证据并由用户逐项批准后才能进入正式 seed。

| chain | subject | constraint_type | mechanism | physical_limit_note | mitigation_path | 来源 | 缺口 |
|---|---|---|---|---|---|---|---|
| AI 算力 | `data_center` | `power_capacity` | IT 与 cooling power density 提升增加设施供电需求 | 局部电网与站点可用功率限制部署规模 | 扩容站点供电、提高能效、调整选址 | S1 | 需具体地区/项目证据 |
| AI 算力 | `data_center` | `infrastructure_access` | 数据中心上线依赖物理并网和供电设施接入 | 接入容量与建设时序限制上线 | 新建/扩建连接设施 | S1 | 仅保留物理接入，不含审批周期 |
| AI 算力 | `ai_server` | `thermal_dissipation` | 高功率 accelerator 产生高热流密度 | 风冷/液冷能力需匹配 rack heat load | 液冷、热交换和系统热设计 | S4、S7 | 需平台级热设计参数 |
| AI 算力 | `scale_up_interconnect` | `bandwidth` | 多 GPU 并行需要高吞吐通信 | interconnect 带宽限制整体扩展效率 | 更高代际 interconnect、拓扑优化 | S3 | 厂商资料，需跨厂商标准对照 |
| AI 算力 | `scale_up_interconnect` | `latency` | 集体通信对链路时延敏感 | 时延限制同步与扩展效率 | 减少 hop、优化 fabric | S3 | 需独立技术论文/标准 |
| AI 算力 | `hbm` | `bandwidth` | accelerator 需要高带宽供给数据 | memory bandwidth 限制 compute utilization | 更高代际 HBM、封装互连优化 | S5 | 厂商资料，需 JEDEC 标准可访问证据 |
| AI 算力 | `advanced_packaging` | `packaging_density` | logic/HBM integration 受 interposer 面积与互连密度限制 | package/interposer 尺寸和 density 是工程边界 | 更大 interposer、3D integration | S6、S14 | candidate，不代表当前短缺 |
| 半导体 | `lithography_machine` | `equipment_capacity` | 晶圆产出受曝光设备 throughput 和可用设备数限制 | 设备生产/安装能力限制新增线体 | 新设备、throughput 和 uptime 改进 | S11 | 需交付周期/产能权威披露 |
| 半导体 | `wafer_fabrication` | `process_yield` | 多步骤制造中缺陷影响合格 die 产出 | yield 决定有效产能 | process control、inspection、工艺优化 | S10、S13 | 需具体工艺/节点证据 |
| 半导体 | `photoresist` | `material_purity` | 先进图形材料对污染和一致性敏感 | 材料规格影响缺陷和工艺窗口 | 提纯与质量控制 | S10 | 缺具体标准，证据不足 |
| 半导体 | `electronic_specialty_gases` | `material_purity` | 工艺气体杂质可能影响成膜/刻蚀/掺杂 | 电子级纯度是物理质量边界 | 提纯、供应和现场质量控制 | S10 | 缺具体 SEMI 标准/厂商资料 |
| 半导体 | `wafer_fabrication` | `process_cycle_time` | 大量串联工艺步骤形成制造周期 | 工序数量、排队和设备节拍限制 cycle time | 自动化、调度、设备 throughput 提升 | S10 | 不包含审批或融资 |
| 半导体 | `advanced_packaging` | `production_capacity` | 多 die/HBM integration 需要专用封装产线 | 有效封装产出受线体与良率限制 | 扩建封装线、提升良率 | S6、S14 | 需独立产能数据 |
| 半导体 | `advanced_packaging` | `reliability` | 高密度互连和热机械应力需要可靠性验证 | package reliability 是交付边界 | 材料、结构和测试改进 | S6 | 需标准/可靠性报告 |
| 半导体 | `wafer_fabrication` | `physical_expansion_cycle` | fab 建设、设备安装、调试与良率爬坡需要物理时间 | 仅物理建设/安装/调试/爬坡 | 模块化建设、提前设备导入 | S10、S12 | 缺项目级时间证据，不含审批融资 |

13 类中本批未提出独立 `reliability` 以外的全部类型覆盖情况：`power_capacity`、`thermal_dissipation`、`bandwidth`、`latency`、`production_capacity`、`process_yield`、`material_purity`、`reliability`、`process_cycle_time`、`packaging_density`、`equipment_capacity`、`infrastructure_access`、`physical_expansion_cycle` 均至少有一条 candidate。覆盖不代表必须批准。

## 7. 跨实体关系 Review 清单

### 7.1 Economy

两条产业链均建议 `scope_type=global`，本批不创建 `scoped_to_economy`。现有 `economy:cn/us/tw/jp/kr/nl` 可作为未来事件或公司分析实体，但用多条 scope 关系会把全球链误写成属地链。若后续建立“中国半导体制造链”版本，再单独 Review `industry_chain → economy:cn`。

### 7.2 Commodity

本批不建议正式写入 commodity 关系：

- 现有 `commodity:polysilicon` 面向多晶硅商品，不能替代半导体级 silicon wafer。
- 电子特气、photoresist 没有对应 commodity 主数据。
- 铜等已有 commodity 与本批节点存在物理使用可能，但当前节点粒度和来源不足以形成精确 `uses_commodity`。

结论：commodity 清单为空，待补主数据与权威投入证据后再 Review，不为凑关系写入。

### 7.3 Benchmark

现有 `benchmarks.json` 只有国债收益率、原油、黄金和加密参考率，没有直接观测 AI 算力或半导体制造的 benchmark。本批 `observed_by_benchmark` 清单为空。

注意：`index:sox` 是 index，不得误写为 benchmark。未来如新增数据中心电力、半导体设备交付或制造产能 benchmark，必须先通过独立 benchmark 主数据 Review。

### 7.4 `mapped_to_sector`

以下均为分析映射 candidate，不表示身份、法定覆盖或影响方向；来源 S15 只能证明 sector 已批准存在，chain/node 到 sector 的映射理由仍需用户 Review。

| from | to sector | 方向 | 理由 | 来源/状态 |
|---|---|---|---|---|
| `industry_chain:ai_compute_infrastructure` | `sector:theme_computing_infrastructure` | chain → sector | 主题范围直接对应算力基础设施 | S15，candidate |
| `industry_chain:ai_compute_infrastructure` | `sector:theme_data_centers_cloud` | chain → sector | 数据中心是 AI 算力物理承载 | S1、S15，candidate |
| `chain_node:data_center` | `sector:theme_data_centers_cloud` | node → sector | 节点与 canonical theme 定义一致 | S15，candidate |
| `chain_node:power_grid` | `sector:industry_power_utilities` | node → sector | 电网基础设施映射电力公用事业 | S15，candidate；需定义范围核验 |
| `chain_node:rack_power_system` | `sector:industry_power_equipment_batteries` | node → sector | rack power 属电力设备分析范围 | S8、S15，candidate |
| `chain_node:data_center_networking` | `sector:industry_software_communications` | node → sector | 数据中心网络属于通信基础设施分析范围 | S2、S15，candidate；范围较宽 |
| `industry_chain:semiconductor_manufacturing` | `sector:industry_semiconductors_electronics` | chain → sector | 制造链映射半导体与电子行业骨架 | S10、S15，candidate |
| `industry_chain:semiconductor_manufacturing` | `sector:theme_semiconductor_resilience` | chain → sector | 自主可控主题用于中国市场分析映射 | S15，candidate；不表达全球链身份 |
| `chain_node:lithography_machine` | `sector:industry_star_semiconductor_materials_equipment` | node → sector | 光刻设备属于半导体材料设备范围 | S11、S15，candidate |
| `chain_node:deposition_equipment` | `sector:industry_star_semiconductor_materials_equipment` | node → sector | 沉积设备属于材料设备范围 | S12、S15，candidate |
| `chain_node:etch_equipment` | `sector:industry_star_semiconductor_materials_equipment` | node → sector | 刻蚀设备属于材料设备范围 | S12、S15，candidate |
| `chain_node:eda` | `sector:industry_star_chip_design` | node → sector | EDA 支撑芯片设计，但不等同芯片设计公司 | S9、S15，candidate；建议重点 Review |

任何海外 market 不得通过 `covers_sector` 指向上述中国 sector。

## 8. 分层审批顺序

以下每层均独立执行 `Review → Write → Rebuild → Query`，上一层批准不推定下一层：

1. 两条 industry chain + 26 个去重 node profile。
2. 两条链 membership。
3. Canonical topology。
4. Physical constraints（全部当前为 candidate）。
5. `mapped_to_sector`；economy、commodity、benchmark 当前为空清单。
6. 每层 PostgreSQL 验收后，只有用户单独批准才执行 Neo4j rebuild；physical constraints 永不进入当前 Neo4j projection。

本 checkpoint 到此暂停，等待用户逐项 Review；不产生正式 seed 或任何有状态写入。
