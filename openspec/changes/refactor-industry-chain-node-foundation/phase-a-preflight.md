# Phase A cleanup preflight 结构契约

状态：代码与静态/单元测试已准备，当前 structure implementation checkpoint 未连接或查询 PostgreSQL，未执行 migration、cleanup、seed 或任何 schema/data Write。

## 只读一致性边界

`entity-seed --phase-a-preflight` 使用 `REPEATABLE READ`、`READ ONLY` 事务，在同一快照内输出三个证据集合：

1. cleanup 指标：旧 `sector`、`industry_chain`、`chain_node` 实体，全部专属 profile/source mapping/membership/topology/physical constraint，`entity_edges`、`event_entity_links`、convergence/audit 子表及 orphan/重复计数。
2. catalog 引用：所有命中目标表的 FK，以及引用旧 sector/industry-chain/chain-node/convergence 结构的 trigger、function/procedure、view 和 rule 定义。
3. 非目标保护基线：按非目标 `entity_type` 输出 row count 与基于身份、名称、aliases、状态的稳定校验和；alliance、economy/country、market、benchmark/index 等必须在 cleanup 前后保持一致。

preflight 同时报告 `entity_key` 空值/重复组，但只作为全局唯一约束的条件门禁，不自动增加约束。`archive_mode` 仅作为环境信息，`backup_verified` 保持 false；外部备份、校验和和恢复演练必须在 Cleanup Review 单独提交。

## migration 对 preflight 的依赖

`000015` 在任何删除前先要求当前 PostgreSQL session 显式设置 `tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified`；未设置时立即失败。该技术标记不能替代备份证据或人工授权。通过门禁后，migration 冻结 `entity_type IN ('sector','industry_chain','chain_node')` 的旧 ID 集合，只删除明确引用该集合的 `event_entity_links` 与 `entity_edges`，按叶到根删除 convergence/audit 和旧产业专属表，清空旧 `chain_node_profiles` 后收敛为最小列。删除旧 `entity_nodes` 前动态扫描所有仍存在的单列 FK；发现任何残留目标引用即抛错并回滚，不使用 CASCADE、TRUNCATE、无谓词 DELETE 或手工清库。

## Cleanup Review 仍需真实运行证据

- 在获准环境运行只读 preflight，提交完整 metrics、catalog references 与非目标校验和。
- 提交可恢复备份文件/快照、校验和、恢复命令和实际恢复演练证据。
- 核对 migration 预计删除 counts 与 preflight 一致，说明锁、超时、事务和提交后 forward-fix。
- 单独取得 cleanup Write 授权；本 checkpoint 不隐含该授权。

Neo4j 不在 preflight、migration 或本 checkpoint 中查询、清理、写入或 rebuild。未来 PG cleanup 完成后，既有投影会暂时陈旧，留给独立 change。
