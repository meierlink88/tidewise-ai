---
status: accepted
date: 2026-09-07
issue: 419
supersedes_in_part: 0053-model-report-publication-as-optional-analysis-sections.md
---

# Report 按故事线与 Concept 组织总结、详情

一份报告可能包含多条地缘/宏观故事线；产业链综合结论按 Concept 组织，Concept 下有多条链。
原来的单上层对象和单链卡片不足以表达这一结构。

保留不可变 Report JSONB 和 Evidence 关系表。新发布声明 report-publication/v3，以显式有序
分析单元表达 summary/detail；Concept 详情包含独立产业链分析，图拓扑和节点影响分开。
AgentOS 拥有综合结论和故事线分组，Data 只验证与持久化，Miniapp 后续拥有展示交互。

旧版发布、重放和读取继续保留，新旧快照不能混合。报告列表默认只返回旧版，显式
schema_version 参数选择新版，防止未升级的 Miniapp 意外选中新版报告。这是消费者迁移边界，
不改变 Miniapp 在选定集合中选择最新报告的语义。新增按单元分页和链详情接口。

只扩展 Evidence scope CHECK，不新增业务表或正式图对象外键，不迁移历史报告。
推理假设允许引用推导依据，但保留待验证性质。完整字段、查询和 rollout 见
[Report 发布合同](../contexts/data/report-publication-v3.md)。
