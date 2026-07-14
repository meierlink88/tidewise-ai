# Phase A cleanup preflight 结构契约

状态：task 1.13 已在当前本地开发 PostgreSQL 真实运行标准只读入口；完整快照、备份与冻结集合见 [cleanup-review.md](cleanup-review.md)。未执行 migration、cleanup、seed 或任何 schema/data Write。

## 只读一致性边界

`entity-seed --phase-a-preflight` 使用 `REPEATABLE READ`、`READ ONLY` 事务，在同一快照内输出三个证据集合：

1. cleanup 指标：旧 `sector`、`industry_chain`、`chain_node` 实体，全部专属 profile/source mapping/membership/topology/physical constraint，`entity_edges`、`event_entity_links`、convergence/audit 子表及 orphan/重复计数。
2. catalog 引用：所有命中目标表的 FK，以及引用旧 sector/industry-chain/chain-node/convergence 结构的 trigger、function/procedure、view 和 rule 定义。
3. 非目标保护基线：按非目标 `entity_type` 输出 row count 与基于身份、名称、aliases、状态的稳定校验和；alliance、economy/country、market、benchmark/index 等必须在 cleanup 前后保持一致。

preflight 同时报告 `entity_key` 空值/重复组，但只作为全局唯一约束的条件门禁，不自动增加约束。`archive_mode` 仅作为环境信息，`backup_verified` 保持 false；外部备份、校验和和恢复演练必须在 Cleanup Review 单独提交。

## migration 对 preflight 的依赖

`000015` 在任何删除前先要求当前 PostgreSQL session 显式设置 `tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified`；未设置时立即失败。该技术标记不能替代备份证据或人工授权。通过门禁后，migration 冻结 `entity_type IN ('sector','industry_chain','chain_node')` 的旧 ID 集合，只删除明确引用该集合的 `event_entity_links` 与 `entity_edges`，按叶到根删除 convergence/audit 和旧产业专属表，清空旧 `chain_node_profiles` 后收敛为最小列。删除旧 `entity_nodes` 前动态扫描所有仍存在的单列 FK；发现任何残留目标引用即抛错并回滚，不使用 CASCADE、TRUNCATE、无谓词 DELETE 或手工清库。

## Cleanup Review 当前证据

- 已在 `REPEATABLE READ READ ONLY` 事务运行标准 preflight，提交完整 metrics、catalog references 与非目标校验和。
- 已提交 custom-format backup 文件位置、校验和、archive 解码与恢复命令；实际 restore rehearsal 会创建 schema/data，本轮未获授权，因此 `backup_verified=false`。
- 已核对 migration 预计删除 counts 与冻结目标集合一致，并记录锁、超时、事务和提交后 forward-fix。
- task 1.13 仍等待主对话验收；本 checkpoint 不隐含 cleanup Write 授权。

Neo4j 不在 preflight、migration 或本 checkpoint 中查询、清理、写入或 rebuild。未来 PG cleanup 完成后，既有投影会暂时陈旧，留给独立 change。
