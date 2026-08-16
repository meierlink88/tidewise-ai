---
status: accepted
date: 2026-08-15
supersedes_in_part: 0016-tidewise-ai-2-object-schema-and-independent-region.md, 0017-independent-country-and-economy-retirement.md, 0018-independent-organization-and-alliance-retirement.md
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

`ENT` Entity、`ERL` Entity Relation、`COU` Country、`REG` Region、`ORG` Organization、
`OCA` Organization Category、`OFN` Organization Function、`ODT` Organization Domain Tag、
`ODL` Organization Domain Tag Link、
`RAW` Raw Evidence、`EVD` Evidence、`EVC` Evidence Category、`RCL` Raw Evidence Category Link、`CPC` Chain Node Physical
Constraint、`CNR` Chain Node Relation、`CRL` Country Region Link、`DIA` Direct Impact
Assertion、`EEI` Entity External Identifier、`ENL` Event Entity Link、`EPR` Event
Publication Receipt、`ECS` Event Semantic Candidate Snapshot、`SCL` Event Semantic Context
Lease、`ERB` Event Semantic Resolution Binding、`ERS` Event Semantic Review Snapshot、`ESS`
Event Semantic Submission、`EEL` Event Evidence Link、`ETD` Event Tag Definition、`ETA`
Event Tag Assignment、`EVT` Event、`IGE` Industry Chain Graph Edge、`IRI` Industry Relationship
Import Receipt、`OMB` Organization Membership、`EER` Event Evidence Record、`RRI` Research
Reasoning Tree Import Receipt、`RRN` Research Reasoning Tree Node、`RRT` Research Reasoning Tree、
`RTI` Research Theme Import Receipt、`RTH` Research Theme、`VSM` Variable Signal Measurement、
`VSG` Variable Signal。

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
