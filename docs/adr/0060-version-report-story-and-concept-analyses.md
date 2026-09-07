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

## 故事线产业链详情扩展（Issue #426）

用户确认地缘、宏观详情也需要“受影响产业链 → 真实结构图 → 节点落点”。复用 v3 的
`detail.industry_chains`，链 source_id/name 与当前故事线产业链锚点匹配；总结不得引用嵌套
节点。地缘总结下挂宏观故事线/产业链，宏观下挂产业链，Concept 下挂链节点。
新增按 kind/analysis_key 读取单链的路径，旧 Concept 路径保留。沿用 JSONB、Evidence scopes、
不可变和幂等规则，不新增迁移；Event/Signal 与覆盖审计不属于发布内容。

## 总结影响度扩展（Issue #429）

在 v3 summary 中增加可选 impact_assessment，承载发布者推导的后果幅度等级、依据及核心
Evidence。旧对象省略字段以保留 canonical hash，新评级进入不可变内容身份；未知等级、
错 label、空依据及无支持证据的已评级结果拒绝。沿用故事线/Concept 总结 Evidence scope，
不新增迁移；列表与单元详情投影返回独立的 Evidence scope token，不由 Data 自动评级。
