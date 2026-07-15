# 全量 chain_node_relations PostgreSQL Write R2 执行证据

## 结果

- 命名操作：`all-chain-node-relations-postgres-write`。
- 环境：local `tidewise_local`，PostgreSQL `16.14`，Goose `18 / 19 applied rows`。
- 单次 approved Write 已完成：`created=4`、`updated=0`、`unchanged=96`。
- 写后 accepted baseline：100 条 active `chain_node_relations`，其中 `is_subcategory_of=95`、`is_component_of=1`、`input_to=3`、`depends_on=1`。
- 未执行第二次 Write、手工 SQL、restore、Neo4j sync 或其他有状态操作。

## Recovery Evidence

Write 前创建 full custom-format PostgreSQL backup：

| 项目 | 值 |
|---|---|
| Path | `/private/tmp/tidewise_local_pre_all_chain_node_relations_20260714T172549Z.dump` |
| Bytes | `1088896` |
| SHA-256 | `f5e57999948f42f646553e04eb0302f1310d82e6d3ad517b80e59e5a82b22313` |
| TOC entries | `185` |
| Validation | PostgreSQL 16.14 `pg_restore --list` 非空，full schema/data decode 成功；Write 前 size/hash 复验一致 |

该 backup 是完整 `tidewise_local` recovery baseline；本层未执行 restore。

## Write 前断言

- Git HEAD/upstream 均为 `71f78c659542d1d1e187541ec6cee1afaa565f02`，worktree clean。
- manifest file SHA-256 为 `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268`，baseline artifact SHA-256 为 `a5475719cd874360116ba7e226d048c4ae9bc06006e1b4c23515198616120edb`。
- active chain_node/profile `842/842`，external identifiers `1169`，entity_edges `241`，physical constraints `0`。
- 写前关系 `96=95/1/0/0`；identity/content SHA-256 分别为 `9b0cd3cae6ca1f33cd5383fd653009fe43aedd18fffe0d7665c4496b373f0eec`、`c64289129b97edf6dbb5eda1bd6713ccb28536187c97d63c9ae294c18ba86a77`。
- fresh read-only dry-run 为 `created=4 / updated=0 / unchanged=96`。

## Write 后断言

- manifest tuple 与数据库集合逐行 diff 为 0。
- target identity SHA-256：`b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6`。
- target content SHA-256：`18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7`。
- orphan/invalid endpoint、duplicate tuple、duplicate ID、self-loop、legacy type、incomplete evidence/provenance/verified_at 均为 0。
- chain_node identity MD5 `d6b53dce56fb5ca72ec77eef816f0a4b`、profile MD5 `2876324fb6bffa41967812702c6bc038` 未变；842 node/profile、1169 mappings、241 entity_edges、constraints 0、Goose 18/19、schema contract 均未变。
- `graph_projection_runs=17`，证明本 R2 层未执行 projector。

## Fail-closed recovery

初次写后只读 dry-run 因 frozen batch 仍硬编码“必须为 96 条”而退出，数据库 Query 与 target hashes 当时均已正确。按 stop condition 未 retry、restore、forward-fix 或进入下一层。

用户随后明确批准最小 R1 recovery。targeted RED test 先证明原契约不能接受正确的 100 条写后状态；修复只允许以下两个 frozen 状态：

- 写前：`96=95/1/0/0`；
- 写后：`100=95/1/3/1`。

任何其他 total 或 type distribution 继续 fail-closed。修复后对当前 live PG 的唯一一次只读 dry-run 为 `created=0 / updated=0 / unchanged=100`，四类 report 为 `95/1/3/1`；随后重新执行的全部只读写后断言通过。

## 边界

本 checkpoint 只完成 task 2.5 的 R1 failure recovery、只读复验与证据冻结。未准备或执行 `all-chain-node-relations-neo4j-sync`，未进入 Package 3；下一层仍需独立 Review 与授权。

## Checkpoint 验证

- `GOCACHE=/tmp/tidewise-go-cache go test ./cmd/entity-seed ./internal/apps/entityfoundation/seed -count=1`：通过。
- `openspec validate rebuild-foundation-graph-and-enrich-chain-data --strict`：通过。
- explicit `TestOpenSpecTaskDesignLint`：通过。
- `git diff --check`、scoped file list 与 secret scan：通过。
