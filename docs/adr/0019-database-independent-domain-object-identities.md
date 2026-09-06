---
status: accepted
date: 2026-08-15
supersedes_in_part: 0016-tidewise-ai-2-object-schema-and-independent-region.md, 0017-independent-country-and-economy-retirement.md, 0018-independent-organization-and-alliance-retirement.md
superseded_in_part_by: 0044-retire-legacy-industry-chain-tables.md
superseded_in_part_by_2: 0045-retire-entity-identifiers-and-redirects.md
superseded_in_part_by_3: 0052-replace-research-theme-with-report-publications.md
---

# 数据库无关的领域对象身份

## 背景

Data 各领域曾并存裸 UUID、`PREFIX_ + code`、固定短码和非 UUID 自然键。
这使身份格式与 PostgreSQL 类型或当前业务 code 耦合，也让跨 Context 消费者
无法根据身份判断所有领域。Issue #241 的人工设计审查明确要求将该机制作为
与 Kratos Biz/Data/Service 平行的技术组件，不归属任一业务领域。

## 决策

- 领域对象 ID 统一为“2–8 位大写 ASCII 领域前缀 + canonical lowercase
  RFC 4122 UUID”，中间不使用分隔符。
- Data Service 在 `backend/internal/core/id` 拥有与数据库无关的随机生成、
  确定性生成、旧 UUID 保留和解析校验原语。这是对固定 Kratos 布局的经审查
  技术机制例外，owner 为 Data Application，不是新业务 layer。
- `core/id` 只能依赖 Go 标准库和 UUID 库；不得依赖 Biz、Data Adapter、Service、
  Server、Conf 或数据库。它拥有关闭的 typed object-kind 注册表；任意字符串前缀不是
  公开 interface。普通写入由 Biz 决定生成时机；受控初始化发布也必须调用同一原语。
- 正常创建合同不接收系统主键。可重放发布接收调用方自然键，由 Biz 确定性生成正式
  主键；Data Adapter 只持久化 Biz command。Data Service 管理的业务表、目录表和关系表
  都使用名为 `id` 的唯一主键，自然键和关系端点另作唯一约束。
- 可移植目录以受控自然键确定性生成 UUID；旧 Entity/Entity Relation 在切换时
  保留原 UUID 作为后缀，避免无意义的身份重排。

## 受控对象前缀

`ENT` Entity、`IND` Industry、`CON` Concept、`CND` ChainNode、`ICH` IndustryChain、
`ERL` Entity Relation、`COU` Country、`REG` Region、`ORG` Organization、
`SUB` Subdivision、`MIN` Ministry、`INS` Institution、
`GPD` GeopoliticDomain、`GPR` GeopoliticRivalry、`MEC` MacroEconomic、
`COM` Company、`CIL` Company Industry Link、
`OCA` Organization Category、`OFN` Organization Function、`ODT` Organization Domain Tag、
`ODL` Organization Domain Tag Link、
`RAW` Raw Evidence、`EVD` Evidence、`EVC` Evidence Category、`RCL` Raw Evidence Category Link、
`CRL` Country Region Link、
`EVT` Event、`EEL` Event Evidence Link、`EAC` Event Actor Link、`EAS` Event Asset Link、
`IGE` Industry Chain Graph Edge、
`OMB` Organization Membership、`RPT` Report、`RPE` Report Evidence Link。

## 切换与回滚

Migration `000050`–`000052` 在 stop-write 窗口内改写所有独立业务主键、传递外键，以及
ID 数组和研究回执 map，并将 Country code 收敛为 ISO 3166-1 alpha-2。旧应用不兼容新身份，回滚必须同时恢复
迁移前 PostgreSQL 快照与上一版应用，不运行 down migration。

Issue #251 通过 forward-only migration `000053` 将 Organization Category、Organization Domain Tag、
Organization Domain Tag Link 和 Raw Evidence Category Link 补齐正式身份，并将 Raw Evidence
与 Atomic Evidence 的主键列及所有 Data/Biz/Service/API 宣言统一为 `id`。该切换不提供
旧数据回填或旧主键名兼容。

Issue #253 通过 forward-only migration `000054` 补齐 Organization Function 的 `OFN` 正式身份，
使其与其余 Organization 目录共享 `id` 主键、确定性目录发布和 API 身份合同。该修正不改变
既有表的物理列顺序，也不提供旧目录行回填或无 `id` 的旧 wire 兼容。

Issue #265 通过 forward-only migration `000058` 将已经独立的 Industry、Concept、ChainNode
和 IndustryChain 从历史共享 `ENT` 身份切换到 `IND`、`CON`、`CND` 和 `ICH`，保留 canonical
UUID 后缀并同步改写全部支持引用。普通 Entity 继续使用 `ENT`；四类独立对象不接受旧前缀。

Issue #267 通过 forward-only migration `000059` 退役 Data-owned Event Semantic 与
Variable Signal 持久化，因此从当前注册表删除 `DIA`、`ENL`、`ECS`、`SCL`、`ERB`、
`ERS`、`ESS`、`VSM` 和 `VSG`。历史 migration 中的旧前缀仍是不可改写的账本事实。

Issue #277 通过 forward-only migration `000060` 重建 Event 领域，保留 `EVT` 和
`EEL`，新增 `EAC` 与 `EAS`，并从当前注册表删除已退役的 `EER`、`ETD`、
`ETA` 和 `EPR`。这些旧前缀在历史 migration 中的出现仍保留。

Issue #298 通过 additive forward-only migration `000063` 增加独立 Ministry 与 Institution
事实及 `MIN`、`INS` 身份。由于第一阶段明确只交付 persistence 和公开 Data Adapter，且不建立
Biz UseCase，Adapter 的 Create input 不接收主键并在服务端调用同一随机生成原语；未来 Biz/API
接入仍不得允许调用方提交主键或把人工业务 code 嵌入身份。

Issue #300 通过 additive forward-only migration `000064` 增加独立 GeopoliticRivalry 与
MacroEconomic 静态叙事蓝图及 `GPR`、`MEC` 身份。第一阶段同样只交付 persistence 和公开
Data Adapter，因此 Adapter 的 Create input 不接收主键并调用共享随机生成原语；未来 Biz/API
接入必须把生成时机收敛到 owning Biz，且不得允许调用方提交主键或从名称、参与方文本派生身份。

Issue #302 通过 additive forward-only migration `000065` 增加独立 StorylineDomain 与
StorylineDomainTactic 静态目录事实及 `SLD`、`SDT` 身份。第一阶段同样由公开 Data Adapter
调用共享随机生成原语；未来 Biz/API 接入必须将生成时机收敛到 owning Biz，不得从 Domain
分类、Tactic key 或名称派生身份，也不得允许调用方提交主键。

Issue #304 通过 additive forward-only migration `000066` 增加独立 Storyline 与
Storyline Event Link 事实及 `STL`、`SLE` 身份。第一阶段同样由公开 Data Adapter 调用共享
随机生成原语；未来 Biz/API 接入必须将两个身份的生成时机收敛到 owning Biz，不得从类型、
锚点、Event 端点或名称派生身份，也不得允许调用方提交主键。

Issue #413 通过零兼容 forward-only migration `000082` 退役 StorylineDomain、
StorylineDomainTactic、Storyline 和 Storyline Event Link，因此从当前注册表删除
`SLD`、`SDT`、`STL`、`SLE`。新增 `GPD` GeopoliticDomain；受控发布以 domain code
确定性生成 `GPD`，以审阅后的唯一故事线中文名称确定性生成 `GPR`。普通
Adapter Create 仍不接收调用方主键。被退役前缀在历史 migration 中的出现不可改写。

Issue #306 通过 stop-write forward-only migration `000067` 将 Company 从共享 `ENT` 身份
切换到独立 `COM` 身份并保留 canonical UUID 后缀，新增 `CIL` Company Industry Link 身份。
普通 Company 写入由 owning Biz 随机生成 `COM`；完整 Industry 集合替换由 Biz 基于
Company/Industry 端点确定性生成 `CIL`。Company 不加入通用 Entity/Research Graph 引用合同。

Issue #332 通过 forward-only migration `000070` 退役 Chain Node Relation、Chain Node Physical
Constraint 与 Industry Relationship Import Receipt，因此从当前注册表删除 `CNR`、`CPC` 和
`IRI`。历史 migration 中的旧前缀仍是不可改写的账本事实。

Issue #334 通过 forward-only migration `000071` 退役 Entity External Identifier 与 Entity
Redirect，因此从当前注册表删除 `EEI`。Entity Redirect 没有独立 ID kind；两者的历史
migration 仍是不可改写的账本事实。

Issue #367 通过 forward-only migration `000078` 物理退役 Research Theme 与 Reason Tree，
因此从当前注册表删除 `RRI`、`RRN`、`RRT`、`RTI` 和 `RTH`，并新增 `RPT` Report 与 `RPE`
Report Evidence Link。历史 migration 中的旧前缀继续作为
不可改写的账本记录存在。
