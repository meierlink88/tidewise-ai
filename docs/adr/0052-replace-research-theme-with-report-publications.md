---
status: accepted
date: 2026-09-01
issue: 367
supersedes: 0003-research-theme-batch-snapshots.md
superseded_in_part_by: 0053-model-report-publication-as-optional-analysis-sections.md, 0054-select-one-latest-report-for-miniapp-home.md
---

# 用不可变 Report 发布替换 Research Theme 与 Reason Tree

## 背景

旧 Research Theme/Reason Tree 把一次完整推理拆成 Theme、Tree、Event 和 Signal 快照，并让
Miniapp 围绕“今日/历史 Theme”与推理树组织页面。AgentOS 现在每次推理会产生一份固定结构的
完整报告；产品原型也按同一报告的地缘政治、宏观经济、产业链和公司层投影展示。继续兼容旧域会
同时保留两套身份、表、API 和产品语言，并诱导 Data 重新查询 Event 或正式产业链来补造报告。

## 决策

- Data 用不可变 `Report` 作为一次 AgentOS 推理发布的完整根对象，保存全部成功发布结果；已发布
  Report 不更新、删除或按名称合并，纠错发布新 Report。
- AgentOS 负责把 Markdown 转换为 `report-publication.v1` 固定包并通过版本化 Data API 发布。
  Data 不读取 AgentOS Artifact，也不在本仓库实现转换器。
- 固定包直接发布全部产品展示内容、报告局部键、有序数组和显式图边。Data 只验证结构、包内引用、
  幂等和 Atomic Evidence 存在性，不判断研究结论，也不比较分析窗口、知识截止时间或生成时间。
- Report 唯一允许的跨领域关系是到既有 Atomic Evidence 的直接 `EVD` 引用；Data 为其建立
  restrictive 外键。Report 不关联 Event、Event Evidence Link、IndustryChain、ChainNode、Company
  或其它正式对象，也不通过这些对象动态重建展示。
- Miniapp Backend 而非 Data 决定首页 Report 集合：上海自然日当日有发布时返回当日全部 Report；
  当日为空时只回退到全部历史最后发布的一份。Miniapp 取消旧历史 Theme 产品入口，但 Data 继续
  保存并可稳定列出全部 Report。
- Miniapp Frontend 保留应用外壳，按 Report 分组展示每份报告持久化的卡片，并使用 Taro 注册页
  承载上层/产业链详情和 Evidence 列表；所有导航都携带所属 `report_id`。
- Migration `000078` 只物理删除旧 Theme/Reason Tree 九表、旧数据、trigger 与函数；后续
  migration 再创建 Report schema。这是零 mixed-version 的 forward-only 切换，不提供别名、
  双读、双写或 down migration。

## PostgreSQL 物理模型

Report 领域只新增 `reports` 与 `report_evidence_links` 两张表；既有 `evidences` 不属于新增表。

`reports` 是发布事实根表：

- `id (RPT...)` 为 Data 生成的正式身份；`source_report_id` 是 AgentOS 为一次推理结果生成的
  全局唯一、重试稳定身份；纠错必须使用新的 source ID；
- `contract_version` 固定为 `report-publication.v1`，`content_hash` 是 Data 对版本与 canonical
  content 计算的 lowercase SHA-256；
- `content JSONB` 保存完整固定报告包的 typed snapshot，包括四层入口、上层推导、全部产业链、
  持久化 `ReportCard`、节点、显式边、反证、Gap、停止条件与只含 ID 的 Evidence refs；
- `ReportCard` 是 AgentOS 显式选择并排序的首页投影，不由读取端扫描产业链后生成；它以结构化
  `detail_ref` 指向层或产业链，以影响项引用锚点或链节点，并拥有独立 `report_card` Evidence
  作用域。卡片展示快照必须与引用对象一致；地缘政治与宏观经济各有一张固定卡片，
  产业链卡片是本 Report 全部产业链的显式首页子集，不强制每条产业链都生成卡片；
- 传导路径使用可包含多个目标的结构化 `target_refs`，每个目标独立保存结果；页面导航不得从
  目标名称或 `CHN-xx` 文本解析。Miniapp v1 只对层与产业链 target 提供独立详情路由，
  锚点和链节点 target 仅作结构化展示。Report-local key 统一使用 lowercase ASCII 稳定键；
- `published_at` 是 Data 生成的唯一发布时间事实。当前/回退选择与历史分页都只使用该字段，
  不使用 `generated_at`，也不增加 `known_at_cutoff`；
- `source_report_id` 唯一；相同 source ID 与 hash 是安全重放并返回原 Report，相同 source ID 与
  不同 hash 返回冲突；
- `(published_at DESC, id ASC)` 索引支持当日全部 Report、历史回退和稳定游标分页。

`report_evidence_links` 是唯一跨领域关系表：

- `id (RPE...)`、`report_id`、`evidence_id (EVD...)`、`scope_type`、`scope_key`、`role`、
  `display_order`；
- 两端均为 `ON DELETE RESTRICT`，其中 `evidence_id` 只引用 `evidences.id`，不存在 Event 外键；
- `scope_type` 支持 `report_card`、`layer`、`anchor`、`reasoning_step`、`transmission_path`、
  `candidate_mechanism`、`industry_chain` 与 `industry_chain_node`；
- `(report_id, scope_type, scope_key, evidence_id)` 防止同一作用域重复 Evidence，
  `(report_id, scope_type, scope_key, display_order)` 同时保证顺序唯一并承担作用域查询索引；
- Evidence 读取严格保持关系的 `display_order`，不按 Evidence 发布时间动态重排；
- `evidence_id` 反向索引用于 Evidence 生命周期保护与诊断。包内 scope/key 闭包、顺序连续性和
  EVD 存在性，以及 JSON refs 与关系行逐项一致性，由严格 Wire/Biz 边界在同一发布事务中校验；
- Evidence 不从子对象自动聚合到卡片、层或产业链，每个可点击入口只能读取自己的显式 scope。

两张表全部由 statement trigger 禁止 `UPDATE`、`DELETE`、`TRUNCATE`。纠错只能发布新 Report；
Evidence 被 Report 引用后也不能被删除。旧九表按子表到父表顺序显式删除，不使用 `CASCADE`。

不创建 Receipt 表，也不注册 `RPR`。认证主体只属于请求审计，不成为 Report 事实；发布者、
独立 publication key、标题和生成时间不重复保存为列，标题与生成时间直接从 `content` 投影。

## 影响

ADR-0003 的 Theme 不可变快照决定不再是当前架构；历史 ADR 和 Goose migration 继续作为决策与账本
记录，但当前 Context、OpenAPI、运行时代码和最终数据库不得暴露 Theme/Reason Tree。发布前必须停止
旧写入并取得 PostgreSQL 恢复点；迁移后的回滚只能恢复快照并同时回退 Data、Miniapp 与 AgentOS
publisher。Research Graph、Event、Raw Evidence 和 Atomic Evidence 不属于退役范围并保持现有合同。
