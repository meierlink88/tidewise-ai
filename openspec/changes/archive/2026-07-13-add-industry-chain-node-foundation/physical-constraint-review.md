# Layer 5 Physical Constraint 候选审查

> **SUPERSEDED：** 本文基于已取消的 industry_chain 容器与 membership 主体边界，只保留历史证据与迁移谱系；未晋级候选全部取消，已写PG的4条事实由后续统一节点架构 change 通过 forward migration 处理。

## 1. 边界与实时状态

- 审查对象仅为 `backend/data/entity_foundation/review/industry_chain_candidates_v1.json` 中15条review-only候选；本文不是正式seed、approval gate或写入授权。
- 2026-07-13实时只读核对：PostgreSQL有2条active+approved chain、26个pilot node、27条active membership、24条active topology，physical constraint为0。
- 候选涉及10个唯一node subject，10/10均为对应chain的active membership；没有edge subject、非法类型或完全重复机制。
- 全部保持`generated_by_ai=true`、`review_status=candidate`。“直接证据闭合”仅表示当前candidate provenance可进入下一轮逐项人工Review；“机制认可但provenance必须校正”不得原样晋级。任何分类都不能据此改状态、进入正式seed或写PG。
- Serenity只提供“系统变化→必需部件→稀缺物理约束→证据→风险”的识别启发；正式事实仍以Tidewise枚举、权威技术证据和人工逐项Review为准。

## 2. 补充权威来源

| 代码 | 来源与URL | 类型及用途 |
|---|---|---|
| P1 | IEA, Energy demand from AI — https://www.iea.org/reports/energy-and-ai/energy-demand-from-ai | 国际机构报告；AI服务器功率密度、数据中心供电与冷却构成 |
| P2 | IEA, Energy and AI Executive Summary — https://www.iea.org/reports/energy-and-ai/executive-summary | 国际机构报告；grid connection queue、输电建设周期、选址与能效缓解 |
| P3 | ASHRAE Datacom Series — https://www.ashrae.org/technical-resources/bookstore/datacom-series | 标准组织资料目录；证明液冷/高密度热管理方向，但当前页面不足以给出参数 |
| P4 | Micron, Integrating and Operating HBM2E Memory — https://assets.micron.com/adobe/assets/urn:aaid:aem:275edf31-79e3-4b6c-8bbd-a233babe9281/renditions/original/as/micron-hbm2e-memory-wp.pdf | 厂商技术白皮书；HBM接口与带宽参数，不单独证明工作负载瓶颈 |
| P5 | TSMC CoWoS — https://3dfabric.tsmc.com/english/dedicatedFoundry/technology/cowos.htm | Foundry技术资料；interposer面积、互连pitch及logic/HBM集成边界 |
| P6 | TSMC/ECTC, Reliability Evaluation of a CoWoS-enabled 3D IC Package — https://3dfabric.tsmc.com/schinese/dedicatedFoundry/technology/del/cowos_publications_ECTC2013.htm | 论文与官方镜像；热机械应力及C4/BGA/micro-bump可靠性 |
| P7 | ASML TWINSCAN NXE:3400C — https://www.asml.com/en/products/euv-lithography-systems/twinscan-nxe3400c | 设备商规格；单机曝光throughput与可用性 |
| P8 | KLA BBP Wafer Inspection — https://bbp.kla.com/ | 技术资料与论文入口；缺陷、process feedback与yield management |

官方技术资料可证明机制和参数，但厂商营销、行业新闻或聚合页不能单独证明行业唯一性、当前短缺、供应商集中或投资方向。补充来源只登记于本文，不回写fixture。

## 3. 逐条审查

### 3.1 AI算力基础设施（7条）

| Candidate ID | Subject与类型 | 机制／物理边界／工程缓解 | Provenance与状态 | Serenity映射 | 审核结论 |
|---|---|---|---|---|---|
| `constraint:ai:data_center:power_capacity` | AI算力基础设施／数据中心；`power_capacity`（供电容量） | IT与冷却功率密度提升增加供电需求；局部电网和站点可用功率限制部署规模；扩容供电、提高能效或调整选址。 | `generated_by_ai=true`; `candidate`; IEA Energy and AI；https://www.iea.org/reports/energy-and-ai/energy-demand-from-ai；`2026-07-12T00:00:00Z` | 能源与基础设施：必需能源输入受站点容量约束 | **直接证据闭合**。P1直接支持AI服务器功率密度、数据中心供电构成与grid connection；不得外推为任何地区当前短缺。晋级direct source见4.1。 |
| `constraint:ai:data_center:infrastructure_access` | AI算力基础设施／数据中心；`infrastructure_access`（基础设施接入） | 上线依赖物理并网和供电设施接入；接入容量与物理建设时序限制上线；新建或扩建连接设施。 | `generated_by_ai=true`; `candidate`; IEA Energy and AI；https://www.iea.org/reports/energy-and-ai/energy-demand-from-ai；`2026-07-12T00:00:00Z` | 扩容与接入：需求设施必须连接既有电网 | **机制认可，但provenance必须校正**。当前fixture页面侧重需求和设施构成；connection queue及输电物理建设周期的直接证据在P2。晋级前必须将source指向P2并排除审批、融资和政策周期。 |
| `constraint:ai:ai_server:thermal_dissipation` | AI算力基础设施／AI服务器；`thermal_dissipation`（散热能力） | 高功率加速器产生高热流密度；冷却能力需匹配机架热负载；液冷、热交换与系统热设计。 | `generated_by_ai=true`; `candidate`; NVIDIA MGX; ASHRAE；https://www.ashrae.org/technical-resources/bookstore/datacom-series；`2026-07-12T00:00:00Z` | 能源与基础设施：高密度计算的必需热移除能力 | **需补权威证据**。P3只证明指南方向，尚缺可核验热负载/冷却适用边界或标准正文。 |
| `constraint:ai:scale_up_interconnect:bandwidth` | AI算力基础设施／规模扩展互连；`bandwidth`（带宽） | 多GPU并行需要高吞吐通信；互连带宽限制扩展效率；更高代际互连与拓扑优化。 | `generated_by_ai=true`; `candidate`; NVIDIA NVLink；https://www.nvidia.com/en-eu/data-center/nvlink/；`2026-07-12T00:00:00Z` | 传输与时延：计算部件间数据搬运能力 | **需补权威证据**。产品页证明NVLink能力，不足以证明一般工作负载稳定瓶颈；需独立论文或开放标准。 |
| `constraint:ai:scale_up_interconnect:latency` | AI算力基础设施／规模扩展互连；`latency`（时延） | 集体通信对链路时延敏感；时延限制同步与扩展效率；减少hop并优化fabric。 | `generated_by_ai=true`; `candidate`; NVIDIA NVLink；同上；`2026-07-12T00:00:00Z` | 传输与时延：同步路径的物理时延 | **需补权威证据**。现有页面未把latency与collective scaling边界建立为可复核事实。 |
| `constraint:ai:hbm:bandwidth` | AI算力基础设施／高带宽内存；`bandwidth`（带宽） | 加速器需要高带宽供给数据；内存带宽可能限制计算利用率；升级HBM与封装互连。 | `generated_by_ai=true`; `candidate`; Micron HBM；https://www.micron.com/products/memory/hbm；`2026-07-12T00:00:00Z` | 传输与时延：memory-to-compute数据供给 | **需补权威证据**。P4支持带宽参数，但利用率边界依赖模型/算子；需独立性能研究并保留适用条件。 |
| `constraint:ai:advanced_packaging:packaging_density` | AI算力基础设施／先进封装；`packaging_density`（封装密度） | logic/HBM集成受interposer面积和互连密度限制；封装尺寸/pitch构成边界；更大interposer、RDL或3D integration。 | `generated_by_ai=true`; `candidate`; TSMC CoWoS；https://3dfabric.tsmc.com/english/dedicatedFoundry/technology/cowos.htm；`2026-07-12T00:00:00Z` | 可靠性与密度：多die集成的面积/pitch边界 | **直接证据闭合**。P5给出面积、pitch与更大封装路径；正式事实不得写当前产能短缺。晋级direct source见4.1。 |

### 3.2 半导体制造（8条）

| Candidate ID | Subject与类型 | 机制／物理边界／工程缓解 | Provenance与状态 | Serenity映射 | 审核结论 |
|---|---|---|---|---|---|
| `constraint:semi:lithography:equipment_capacity` | 半导体制造／光刻机；`equipment_capacity`（设备产能） | 原机制混合曝光throughput、设备数与供应能力；原边界写成设备生产/安装能力；缓解为增加设备、改善throughput与uptime。 | `generated_by_ai=true`; `candidate`; ASML Products；https://www.asml.com/en/products；`2026-07-12T00:00:00Z` | 制造与良率：关键工序单机节拍与可用性 | **删除或改写**。P7只支持单机曝光throughput/availability，不能推出ASML生产/交付能力；应删设备供应量表述后再审。 |
| `constraint:semi:wafer_fabrication:process_yield` | 半导体制造／晶圆制造；`process_yield`（工艺良率） | 多步骤制造中的物理缺陷影响合格die产出；良率决定有效产出；以工艺控制、检测与优化缓解。 | `generated_by_ai=true`; `candidate`; SEMI; KLA；https://bbp.kla.com/；`2026-07-12T00:00:00Z` | 制造与良率：缺陷使名义产能不能完全转为合格产出 | **需补权威证据**。KLA BBP入口及“论文入口”不是被锁定的直接文献。需具体技术文档、标准或同行评审论文，直接建立“缺陷密度/检测与合格die yield、有效产出”的关系，并登记稳定URL/DOI与核验日期。 |
| `constraint:semi:photoresist:material_purity` | 半导体制造／光刻胶；`material_purity`（材料纯度） | 图形材料对污染和一致性敏感；材料规格影响缺陷与工艺窗口；提纯与质量控制。 | `generated_by_ai=true`; `candidate`; SEMI Manufacturing Process；https://www.semi.org/en/ehs_PFAS/mfg_markets_overview；`2026-07-12T00:00:00Z` | 材料与工艺：材料质量进入光刻窗口 | **需补权威证据**。当前PFAS/market overview不证明纯度规格；需SEMI标准、材料商规格或论文。 |
| `constraint:semi:gases:material_purity` | 半导体制造／电子特气；`material_purity`（材料纯度） | 气体杂质可能影响成膜、刻蚀或掺杂；电子级纯度为质量边界；提纯、输送与现场质量控制。 | `generated_by_ai=true`; `candidate`; SEMI Manufacturing Process；同上；`2026-07-12T00:00:00Z` | 材料与工艺：反应/掺杂输入的杂质边界 | **需补权威证据**。当前来源不含具体纯度/污染标准；需SEMI/CGA/ISO标准、气体商规格及工艺证据。 |
| `constraint:semi:wafer_fabrication:process_cycle_time` | 半导体制造／晶圆制造；`process_cycle_time`（工艺周期） | 原机制为串联步骤形成周期；原边界混入排队和调度；缓解混入自动化和运营调度。 | `generated_by_ai=true`; `candidate`; SEMI Manufacturing Process；同上；`2026-07-12T00:00:00Z` | 制造与良率：不可省略工序与设备驻留时间 | **删除或改写**。只保留工艺步骤、设备处理/驻留及必要返工的物理周期；排队和调度属动态运营。 |
| `constraint:semi:advanced_packaging:production_capacity` | 半导体制造／先进封装；`production_capacity`（生产能力） | 多die/HBM集成需要专用线；有效产出受线体、设备节拍与良率限制；扩建线体并改善节拍/良率。 | `generated_by_ai=true`; `candidate`; TSMC CoWoS；https://3dfabric.tsmc.com/english/dedicatedFoundry/technology/cowos.htm；`2026-07-12T00:00:00Z` | 制造与良率：专用工艺线物理处理能力 | **需补权威证据**。P5证明工艺结构，不证明line capacity；需正式产能/节拍披露，不得写当前供需紧张。 |
| `constraint:semi:advanced_packaging:reliability` | 半导体制造／先进封装；`reliability`（可靠性） | 高密度互连和热机械应力需可靠性验证；C4/BGA/micro-bump寿命是交付边界；材料、结构与测试改进。 | `generated_by_ai=true`; `candidate`; TSMC CoWoS；https://3dfabric.tsmc.com/english/dedicatedFoundry/technology/cowos.htm；`2026-07-12T00:00:00Z` | 可靠性与密度：热机械载荷下的封装寿命 | **机制认可，但provenance必须校正**。P6直接支持失效机制与工程缓解，但fixture仍指向泛CoWoS页面。晋级前必须把source_name/source_url/verified_at改为P6具体论文或同等直接证据，不能沿用泛页面。 |
| `constraint:semi:wafer_fabrication:physical_expansion_cycle` | 半导体制造／晶圆制造；`physical_expansion_cycle`（物理扩产周期） | fab建设、设备安装、调试与良率爬坡需要物理时间；只保留建设/安装/调试/爬坡；模块化建设与提前导入设备。 | `generated_by_ai=true`; `candidate`; SEMI; Applied Materials；https://www.appliedmaterials.com/eu/en/semiconductor/products.html；`2026-07-12T00:00:00Z` | 扩容与接入：新增制造能力经过物理建设与爬坡 | **需补权威证据**。产品页不能证明完整周期；需项目级construction、tool install/qualification/ramp资料，排除审批融资。 |

## 4. 汇总

| 结论 | 数量 | Candidate IDs |
|---|---:|---|
| 直接证据闭合 | 2 | `ai:data_center:power_capacity`、`ai:advanced_packaging:packaging_density` |
| 机制认可但provenance必须校正 | 2 | `ai:data_center:infrastructure_access`、`semi:advanced_packaging:reliability` |
| 需补权威证据 | 9 | `ai:ai_server:thermal_dissipation`、`ai:scale_up_interconnect:bandwidth`、`ai:scale_up_interconnect:latency`、`ai:hbm:bandwidth`、`semi:wafer_fabrication:process_yield`、`semi:photoresist:material_purity`、`semi:gases:material_purity`、`semi:advanced_packaging:production_capacity`、`semi:wafer_fabrication:physical_expansion_cycle` |
| 删除或改写 | 2 | `semi:lithography:equipment_capacity`、`semi:wafer_fabrication:process_cycle_time` |

### 4.1 晋级时必须使用的direct source

| Candidate ID | 当前provenance是否可原样使用 | 晋级时正式seed必须使用的direct source |
|---|---|---|
| `constraint:ai:data_center:power_capacity` | 是 | P1：IEA, *Energy demand from AI*，https://www.iea.org/reports/energy-and-ai/energy-demand-from-ai；`verified_at`须更新为实际晋级复核时间。 |
| `constraint:ai:advanced_packaging:packaging_density` | 是 | P5：TSMC CoWoS官方技术页，https://3dfabric.tsmc.com/english/dedicatedFoundry/technology/cowos.htm；正式事实应引用具体interposer面积/pitch边界。 |
| `constraint:ai:data_center:infrastructure_access` | 否，当前URL不直接闭合connection queue/physical build论证 | P2：IEA, *Energy and AI Executive Summary*，https://www.iea.org/reports/energy-and-ai/executive-summary；晋级前同步更新`source_name/source_url/verified_at`。 |
| `constraint:semi:advanced_packaging:reliability` | 否，当前URL是泛CoWoS页面 | P6：TSMC/ECTC, *Reliability Evaluation of a CoWoS-enabled 3D IC Package*，https://3dfabric.tsmc.com/schinese/dedicatedFoundry/technology/del/cowos_publications_ECTC2013.htm；晋级前同步更新`source_name/source_url/verified_at`。 |

直接证据闭合与provenance校正项仍需用户逐项Review并在未来写入上下文提供显式approval gate；本轮没有修改fixture、正式seed或数据库状态。

## 5. 替代性与供给刚性的支持边界

- 可以支持：识别功能位置受哪些物理参数约束、工程缓解路径是否存在、扩容是否需要不可省略的建设/安装/工艺爬坡，以及替代方案必须满足哪些带宽、热、纯度、良率、密度或可靠性门槛。
- 不能直接回答“是否可替代”的商业结论；还需要功能等价、兼容性、认证、成本、时间和动态observation，当前也没有获批`substitutes_for`事实。
- 经批准的物理扩容周期、设备节拍、工艺良率和材料纯度可作为供给响应存在工程摩擦的输入，但不能直接推出当前供给刚性程度。
- 不能回答供应商集中度、垄断、市场份额、价格、当前供需缺口、政策、融资或“没人看见”。这些不是physical constraint，也不能由Serenity启发或AI候选推断为事实。

## 6. 下一门禁

主对话须逐项决定批准、补证或改写。只有逐项批准、来源闭合且未来写入上下文携带显式人工approval gate的条目，才可另行修改正式seed并申请PG写入授权；该授权不得从本文推定。

2026-07-13后续结果：用户批准`power_capacity`、`packaging_density`、完成P2校正的`infrastructure_access`、完成P6校正的`reliability`共4条；随后在独立Write授权与备份后仅执行一次constraint scope并通过只读验收。其余11条仍为review-only。
