# Phase A PostgreSQL cleanup preflight（初步、只读）

执行时间：2026-07-13（Asia/Shanghai）
环境：本地 `tidewise_local` PostgreSQL 16
执行边界：两次 `BEGIN TRANSACTION READ ONLY ... COMMIT`，仅 `SELECT`；未运行 migration、seed 或任何 schema/data Write。

## Material Proposal Change 后的结论

本报告原先用于 legacy→target 收敛，现已改为 cleanup 输入。当前结果只能证明部分旧数据范围，**尚不足以申请 cleanup Write**：未统计 `event_entity_links`、convergence reference/alias moves 各表、所有 FK/trigger/function、逻辑引用代码路径及非目标校验和。任务 1.6 必须补齐后重新提交 Review。

- 旧产业实体目标基线：sector 112（active 52 / inactive 60）、industry_chain 2、chain_node 54；这些旧实体及 profiles 最终全部清除，不复用 ID/key。
- 旧专属/关系 facts：sector source mapping 89、membership 27、topology 24、physical constraint 4、至少 58 条 `entity_edges` 引用旧 sector/industry_chain。
- convergence：manifest version 1、convergence rows 60；完整 reference/alias move counts 尚未查询。默认目标为备份后删除全部仅服务 sector convergence 的表和记录。
- 当前已检查 orphan 均为 0；这不等于 cleanup 后引用安全。
- `entity_key` 空值 0、重复组 0；全局唯一约束仍不在默认 migration 中，必须独立 preflight/Review。
- backup：`archive_mode=off`，且没有恢复演练证据；`backup_verified=false`，构成 cleanup Write 硬阻断。
- Neo4j 未查询、未清理、未写入、未 rebuild。PG cleanup 后现有图投影会暂时陈旧，留给后续独立 change。

## 已取得的实际 counts

|指标|值|
|---|---:|
|entity_type.sector|112|
|sector.active / inactive|52 / 60|
|entity_type.industry_chain / active|2 / 2|
|entity_type.chain_node|54|
|profile.sector / industry_chain / chain_node|112 / 2 / 54|
|legacy.sector_source_mapping|89|
|legacy.membership / topology / physical_constraint|27 / 24 / 4|
|entity_edges.old_endpoint（仅 sector/industry_chain）|58|
|convergence.records / current_manifest|60 / 1|
|status.merged|0|
|entity_key.blank / duplicate_groups|0 / 0|
|membership.duplicate_groups / topology.duplicate_groups|0 / 0|
|已检查 profile/端点/constraint orphan|0|

非目标基线（仅 counts，仍需校验和/引用保护）：alliance_org 10、economy 50、market 47、benchmark 10、index 43。

## 精确 cleanup 依赖顺序（设计输入，待完整审计确认）

1. 冻结执行时旧产业 ID 集合：全部旧 `entity_type IN ('sector','industry_chain','chain_node')`。
2. 删除/验证指向该集合的逻辑引用：`event_entity_links`、`entity_edges` 两端及代码扫描发现的其他表；不做重定向。
3. 备份 sector convergence 审计快照；受控移除 append-only triggers/function，再删除 alias moves、reference moves、convergences、manifests。若发现非 sector 用途则停止并提交保留理由。
4. 按 FK 叶子到根删除 historical physical constraints → topology edges → memberships → industry_chain profiles。
5. 删除 sector source mappings、sector profiles、旧 chain_node profiles。
6. 删除冻结集合中的旧 entity_nodes。
7. 删除旧专属表；建立/验证最小 chain_node/theme schema。不得触碰非目标 profile/table。

所有删除必须在版本化 migration/受控命令中使用显式谓词或冻结 ID 集合；禁止手工 SQL、TRUNCATE 或无谓词 DELETE。

## Cleanup Review 前必须补齐

- `event_entity_links` 及所有 `entity_nodes` FK 引用的目标/非目标 counts。
- `entity_convergence_reference_moves`、`entity_convergence_alias_moves`、trigger/function 与 manifests 的完整依赖/counts。
- 生产 Go/SQL/seed/graph projection 对旧专属表的读取路径及切换/停用计划。
- 每表预计删除 counts、事务锁影响、超时、dry-run 与重复执行结果。
- 非目标实体/profile/关系的前后 counts 与校验和断言。
- 可恢复备份文件/快照、校验和、恢复命令和实际恢复演练证据。
- cleanup 提交后的 forward-fix 方案；不得依赖 down migration 或手工清库。
