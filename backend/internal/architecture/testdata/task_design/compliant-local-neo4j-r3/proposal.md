## Why

镜像受阻 change 的真实 Stateful Layer Map。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Local Neo4j cleanup and rebuild | R3 | yes | R3_OPERATION | 两个独立 local Neo4j R3 layer |
| 2 | PostgreSQL write and Neo4j sync | R3 | yes | SPEC_SEMANTICS | PG-first 后独立同步 Neo4j |
| 3 | Apply-final Review | R1 | yes | APPLY_FINAL | 后续生命周期 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 4 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| local-neo4j-foundation-cleanup | 1 | local | 1 | 仅清空 local Neo4j 的 Tidewise namespace 节点与关系，保留 database、约束、索引和配置 | PG 业务数据、UAT、prod、shared、其他 namespace | approved-disposable-recovery | new:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG projection baseline identity、scope、count、hash、schema 已冻结；环境确认为批准的 disposable local Neo4j | Tidewise namespace 节点与关系为零；database、约束、索引、配置和 PG 业务数据不变 | workflow schema blocker 未解决、环境不符、PG baseline 漂移、越过 namespace 或断言失败立即停止 |
| local-neo4j-foundation-rebuild | 1 | local | 2 | 从已验收 PG baseline 重建 active alliance_org、economy、chain_node 及符合端点边界的 entity_edges 与 chain_node_relations | market/index/benchmark、事件、观测、推理、未批准数据、UAT、prod、shared | approved-disposable-recovery | reuse:foundation-pg-projection-baseline | counts=review-gated;hash=review-gated;schema=review-gated | 复验 PG baseline identity、scope、count、hash、schema 未漂移；cleanup Query 与 projector targeted tests 通过 | 分实体类型节点与分关系类型 counts 与 PG 一致；missing、duplicate、orphan、legacy 均为零 | workflow schema blocker 未解决、PG baseline 漂移、projector failed/skipped 或完整性断言失败立即停止 |
| all-chain-node-relations-postgres-write | 2 | local | 1 | 仅写入审核完成并经 Review 冻结的 final manifest | Neo4j、UAT、prod、shared | backup | new:all-chain-node-relations-pg-backup | counts=review-gated;hash=review-gated;schema=review-gated | identity、scope、count、hash、schema 已冻结；backup 可恢复 | 写后 Query 全部通过 | 基线漂移、部分写入或断言失败立即停止 |
| all-chain-node-relations-neo4j-sync | 2 | local | 2 | 仅从已验收 PG accepted baseline 同步 842 全量审核批准的 chain_node 与四类关系 | physical constraints、反写 PG、查询 API、派生关系、UAT、prod、shared | approved-disposable-recovery | new:all-chain-node-relations-pg-accepted-baseline | counts=review-gated;hash=review-gated;schema=review-gated | PG 写后 Query 已验收；accepted baseline 已冻结；取得单独 Neo4j R3 授权 | Neo4j 与 PG accepted baseline counts/type 一致；missing、duplicate、orphan、legacy 为零 | workflow schema blocker 未解决、PG baseline 漂移、未单独授权、同步或 Query 失败立即停止 |

## What Changes

- 验证 local Neo4j R3 recovery contract。
