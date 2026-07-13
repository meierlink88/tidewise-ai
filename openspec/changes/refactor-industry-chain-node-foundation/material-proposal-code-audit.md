# Material Proposal Change：当前未验收代码差异审计

审计时间：2026-07-13
状态：Apply 已暂停；以下源码 diff 未经新 Proposal Review，不代表可执行实现。未执行 migration/seed/PG Write/Neo4j 操作。

## 可复用方向

|当前差异|可复用部分|重新 Apply 时的要求|
|---|---|---|
|`EntityTypeTheme`、`Theme` / `ThemeProfile`、最小 `ChainNodeProfile`|字段命名和非空校验符合已批准实体模型|保留 TDD，补充 cleanup 后新 schema 的集成测试|
|生产 manifest 拒绝 sector/industry_chain、旧 profile 字段、source mapping/membership/topology|符合“不得继续新建旧模型”|扩展为只接受主对话最终批准的 final seed，不得接受第一轮 955/34 工作簿直接写入|
|默认 seed paths 排除 sector、source mapping、industry_chain 容器及旧产业关系|可阻止普通 seed 恢复旧数据|cleanup 后验证全部生产入口；旧 fixture 只能作为只读审计输入|
|`entity-seed --phase-a-preflight` 与 SELECT-only SQL 骨架|可作为完整 cleanup preflight 的起点|补齐 event links、全部 FK/trigger/function、convergence 子表、非目标校验和和备份验证；保持只读|
|生产 `Service.Apply` 移除旧 source mapping/industry batch 写入分支，CLI 禁用旧 apply scopes/convergence flags|符合切断旧写路径|继续扫描其他 cmd/repository/graph projection 生产路径，未切换前不得 cleanup|
|候选非产业标签过滤测试|可作为基础拒绝规则|以最新工作簿 202 明确排除为 fixture，并补充 34 待复核不可自动准入测试|

## 与新目标冲突、必须重做

|当前差异|冲突|处理决定|
|---|---|---|
|`StableEntityIdentity(entityKey, approvedID)` 优先返回旧/批准 ID|新目标禁止复用旧 sector/industry_chain/chain_node UUID/key|删除旧 ID 分支；final seed 只从最终新 key 生成 deterministic UUID|
|`000015_refactor_industry_chain_node_phase_a.sql` 仅 expand/guard，保留旧列和全部旧数据/旧表|新目标要求备份/引用审计后完整 cleanup|不得 apply；以新的版本化 cleanup + target schema migration 重新设计，明确删除顺序、非目标保护和幂等|
|migration/测试围绕 legacy profile 收敛与旧 facts 继续存在|新目标不做 legacy→target 收敛|重写为 cleanup target-set、引用删除、表删除、全新 schema 和 already-clean 测试|
|`LegacyMigrationInputPaths` 及旧 fixture 仍可读取旧 seed|只能用于只读审计，不能成为迁移/新 seed 输入|重命名/隔离为 audit fixture，确保生产 service/CLI 无写入口|
|此前候选清单逐项“复用/合并旧 UUID/key”|已被 Material Proposal Change 覆盖|已替换为最新工作簿汇总与 34 待复核清单；不生成 final seed|
|preflight 只统计部分旧表与 orphan|未覆盖 event links、convergence reference/alias moves、全部 FK/逻辑引用、非目标校验和|任务 1.6 重新实现并运行；当前报告明确标记不完整和阻断|
|旧 repository/domain/convergence/graph projection 代码仍大量存在|cleanup 后可能读取已删除表或投影陈旧|在新 Apply 中逐路径测试并切换；Neo4j 代码本 change 不执行，PG cleanup 后陈旧状态写入 evidence|

## 未授权状态确认

- 当前 `git diff` 只包含源码、测试、migration 文件和 OpenSpec/evidence；尚未 commit 本次 Material Proposal update。
- 本轮只读 PG preflight 发生在策略变更前，使用显式 `READ ONLY` transaction；策略变更后没有再次访问或修改数据库。
- 未创建可执行 final chain_node seed，未提出 theme 实例，未创建 Phase B relation 候选。
- 在主对话重新批准 Material Proposal 前，不继续修改生产实现，也不标记 tasks 1.3–1.9 完成。
