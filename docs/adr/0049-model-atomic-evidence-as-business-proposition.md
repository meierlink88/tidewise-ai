---
status: accepted
date: 2026-08-30
supersedes: 0021-simplify-atomic-evidence-semantics.md
issue: 351
---

# 将 Atomic Evidence 建模为最小完整业务命题

## 背景

严格单层 5W1H 会诱导提炼方按句子、三元组或单项指标机械拆分。一篇报道中的媒体归因可能被误当成
业务主体，同一次公司披露中的多个经营指标也可能被拆成大量失去业务语境的 Evidence，不利于后续
Event 提炼、实体关联和变量信号构建。同时 Keywords 仍属于整篇 Raw Evidence，无法准确描述拆分后的
每条命题。

## 决策

- Atomic Evidence 是一条“最小完整业务命题”，边界由同一主体、核心动作、作用对象、现实阶段和时间
  共同确定，而不是由句子数、三元组数或指标数确定。
- 同一业务披露中的同类指标可以聚合；已经发生的结果与未来指引允许拆分。转载或报道主体写入归因，
  不取代实施业务动作的主体。
- `semantic` 严格使用 `actors`、`action`、`objects`、`stage`、`modality`、`time`、
  `jurisdictions`、`reason`、`method`、`metrics` 和 `attribution`。阶段、情态和时间精度使用受控枚举；
  无法可靠换算的相对时间保留原文并让标准时间边界为空。
- Keywords 从 Raw Evidence 移到 Atomic Evidence。每条 Evidence 必须拥有一至五个有序、唯一且不超过
  六个 Unicode 字符的关键词；Data 只校验与原样保存，不生成、不规范化、不参与跨来源去重。
- Data 根据 Raw Evidence ID 与规范的 `summary + keywords + semantic` 派生 Evidence ID，只支持同一
  Raw Evidence 的确定性重试。跨来源相同事实继续保存为独立 Evidence，由后续 Event Resolution 收敛。
- Issue #351 授权零兼容切换。迁移在历史 Raw Evidence、Atomic Evidence、Event 或依赖关系非空时
  fail closed；历史清理由操作员在取得恢复点后显式执行，不由 migration 静默删除。

## 影响

Data Context、OpenAPI、DTO、Biz 不变量、Service 映射、Data Adapter、fixture 和测试同步切换。
AgentOS 负责业务命题拆分、规范化、同篇精确去重和时间换算；Data 保持验证、持久化和正式身份职责。
发布必须协调停止旧写入者，新旧 Data/AgentOS binary 不得承载 mixed traffic。完整回滚必须恢复
migration 74 前数据库恢复点并同时回退应用。
