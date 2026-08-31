---
status: accepted
date: 2026-08-30
issue: 359
supersedes_in_part: 0047-publish-event-identity-semantics.md
---

# 将 Event 建模为事件级完整业务命题

## 背景

Atomic Evidence 已使用最小完整业务命题表达主体、动作、对象、阶段、情态、时间、辖区、原因、
执行方式、指标和来源归因。Event 仍只保存七键身份投影，并把情态和两个时间字段放在顶层，导致
Evidence 中支持 Signal 推导的原因、方式和指标在 Event 收敛时丢失，也形成重复时间 owner。

## 决策

- Event wire 只保留 `title`、`summary` 和 `semantic`；Data 返回对象另有生命周期 `status`。
- Event `semantic` 严格包含 `actors`、`action`、`objects`、`stage`、`modality`、`time`、
  `jurisdictions`、`reason`、`method`、`metrics`。
- `time` 严格包含可空 UTC `occurred_at`、`announced_at`、`effective_at`、`observed_at` 和受控
  `precision`；四种时间至少一项存在。`observed_at` 只在业务时间未知时保存来源观察时间，不替代
  业务发生、宣布或生效时间。Data 内部可继续使用现有 modality 和业务时间列作为查询投影，但 wire、
  publication hash 和 Biz 输入不允许双写。
- Event 是多个 Evidence 收敛后的单一现实动作，不能复制某一来源的 `attribution`。来源归因继续
  由 Atomic Evidence 与 Event Evidence Link 保持。
- Event Identity 仍只比较核心主体、动作、对象、阶段和兼容事件时间；原因、执行方式和指标用于
  事实表达、Graphiti Fact 与 Signal 推导，不扩大 SAME_EVENT 身份维度。

## 影响

Data、Admin consumer 与 AgentOS producer 必须协调停止旧写入后切换，禁止 mixed traffic。
Migration 000075 对非空 Event 历史 fail closed，不转换或删除旧事实。完整回滚必须恢复 migration
75 前恢复点，并同时回退 Data、Admin 与 AgentOS。

Issue #363 追加 `observed_at` 兼容扩展。Data 与 Admin 在切换期同时接受旧四键和新五键时间对象，
Data 输出新五键合同；业务时间 Event 的既有 publication hash 与存储 JSON 保持不变，observed-only
Event 才持久化第五个时间键。该扩展不新增 Event 置信度、时间来源枚举或查询投影。
