---
status: accepted
date: 2026-08-18
issue: 266
superseded_in_part_by: 0027-retire-agent-run.md
---

# 使用有界停写模式执行 UAT Tidewise AI 2.0 切换

## 背景

UAT Data database 当前 migration ledger 是 `44`，仓库当前结构是 `58`。Migration
`45`–`58` 包含 Data rewrite、旧事实删除和身份前缀切换，不支持旧、新服务并行写入；现有
UAT Action 只允许 schema migration，并会在候选失败时自动启动旧镜像，因此不能安全执行
本次零兼容切换。

UAT 的业务数据不要求保留，但 AgentRun database 中的模型与 Connector 配置、Artifact 和
独立 Qdrant 仍是运行时依赖，不能作为 Data schema 重建的附带清理对象。

## 决策

- 保留单一 `Deploy UAT` workflow，默认 `normal` 模式维持 schema-only、按变更服务构建和
  前一完整 release 自动回退。
- 增加显式 `tidewise_2_cutover` 模式，只接受 Data current `44` 和严格连续的 pending
  `45`–`58`，同时要求 AgentRun current `015` 且没有 pending migration。
- Cutover 要求已确认 RDS 恢复点和破坏性 Data 变更，强制构建五个服务，并在 Data 写入前
  完成 release state、RDS TLS、Artifact、Qdrant 和 migration ledger 预检。
- 执行 migration 前停止全部五个 Tidewise AI 服务并证明没有服务运行；候选 Data 镜像只以
  `dbmigrate -apply -target-version 58` 推进正式 ledger。
- 一旦开始 Data migration，失败路径不再启动旧镜像。脚本保留非敏感恢复标记，只允许相同
  release 对 ledger 的连续 pending 后缀 forward recovery，或由操作员恢复 RDS 后回到旧应用。
- 成功后记录当前五镜像 release，移除 cutover marker，并禁止把 pre-2.0 release 作为普通
  自动回退目标。后续所有迭代继续使用同一 workflow 的默认 `normal` 模式。
- 历史数据违反 fail-closed migration 前置条件时，只有已存在同 SHA cutover marker 且操作员
  额外选择 `rebuild_empty_data_schema`，候选 Data `dbmigrate` 才在 advisory lock 内重建 Data
  `public` schema 并推进到 `58`；不得清理 AgentRun database、Artifact、Qdrant 或其他基础设施。

## 影响

Cutover 是一次有界的停机发布。迁移前失败仍可恢复旧 release；迁移开始后的恢复只能前向完成
或同时恢复数据库与旧应用。普通部署的 schema-only 和自动回退合同不因本次例外而放宽。
