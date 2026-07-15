# 全量 chain_node_relations PostgreSQL Write R2 Review 包

## 状态与授权边界

- 当前状态：`prepared_for_r2_review`；本 R1 checkpoint 未创建 backup、未写 PostgreSQL、未访问或写入 Neo4j。
- 命名操作：`all-chain-node-relations-postgres-write`。
- 风险：R2，local PostgreSQL curated data write。
- 本包只允许在后续独立授权后，单次把已冻结 100 条关系 manifest 应用到 `tidewise_local.chain_node_relations` 并立即 Query。
- 不授权 migration、节点/profile/external identifier/entity_edges/physical constraint 写入、Neo4j cleanup/rebuild/sync、自动 retry 或扩大范围。

## 冻结 manifest

| 项目 | 冻结值 |
|---|---|
| Path | `openspec/changes/rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-r0/approved-candidate-manifest.json` |
| File SHA-256 | `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268` |
| Approved semantic SHA-256 | `b578e957df6e6249f745f2661f11a2d03c73434dab85fe8e2fb35f33bf14f2d9` |
| Baseline | 842 个既有 active chain_node；identity MD5 `d6b53dce56fb5ca72ec77eef816f0a4b`；profile MD5 `2876324fb6bffa41967812702c6bc038` |
| Target | 100：`is_subcategory_of=95`、`is_component_of=1`、`input_to=3`、`depends_on=1` |
| Target identity SHA-256 | `b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6` |
| Target content SHA-256 | `18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7` |

checkpoint `1e31493` 的人工冻结不改写 immutable JSON 内原始 `review_state/ready_for_write/write_authorized` 字段，否则会破坏已批准 SHA。R2 执行授权只由本命名层的新鲜明确指令和 CLI 的 approved-write flag 共同表达。

## 当前只读基线

冻结时间：2026-07-15。本 R1 checkpoint 的 fresh read-only Query 为：

| 项目 | 当前值 |
|---|---|
| Database / user / server | `tidewise_local` / `tidewise` / PostgreSQL `16.14` |
| Goose | max applied `18`；applied rows `19` |
| active chain_node / profiles | `842 / 842` |
| external identifiers / entity_edges | `1169 / 241` |
| chain_node_relations | `96`：分类 `95`、组成 `1`、投入 `0`、依赖 `0` |
| physical constraints | `0` |
| 当前 relation identity SHA-256 | `9b0cd3cae6ca1f33cd5383fd653009fe43aedd18fffe0d7665c4496b373f0eec` |
| 当前 relation content SHA-256 | `c64289129b97edf6dbb5eda1bd6713ccb28536187c97d63c9ae294c18ba86a77` |
| columns MD5 / constraints MD5 | `30989050ddac02d7b70f0eeb8c510d19` / `a3779c06528cfb2fbf469d7ced849199` |
| orphan / duplicate tuple / self-loop | `0 / 0 / 0` |

显式只读 CLI dry-run 结果：`created=4`、`updated=0`、`unchanged=96`，四类 counts 精确为 95/1/3/1。任何后续 fresh preflight 与以上值不一致时必须停止并重新 Review。

## Recovery Evidence

`Recovery Evidence=backup`。获得 R2 授权后、Write 前必须创建 fresh full local PostgreSQL custom-format backup，并记录：

- 环境 identity、备份绝对路径、创建时间、文件 bytes、SHA-256。
- `pg_restore --list` 可读且 TOC 非空。
- 创建后、Write 前再次校验文件 size/hash 未变化。
- recovery 是从该 backup 恢复整个 `tidewise_local`；不得把 Neo4j 当作 PostgreSQL 恢复来源。

本 Review 包不提前创建 backup。backup 不存在、不可读、hash 漂移或恢复边界不清楚时不得 Write。

## Fresh Preflight

执行前必须全部成立：

1. branch、HEAD、worktree 与已验收 R1 checkpoint 一致且 clean；manifest path、file SHA、semantic SHA、100/95/1/3/1、842 baseline 全部通过 fail-closed loader。
2. database 必须是 local `tidewise_local`；Goose、server、schema hashes、842 identity/profile、保护表 counts、当前 96 relation hashes 与 integrity 必须等于本包。
3. 再运行一次 relation dry-run，结果必须仍为 `created=4 / updated=0 / unchanged=96`；任何 update、额外 create、端点失效或 schema drift 都停止。
4. fresh backup 已创建并完成 size/hash/TOC 复验。
5. 授权文本必须精确命名 `all-chain-node-relations-postgres-write`，不得从本 R1 checkpoint、dry-run 或 manifest 冻结推定。

## 唯一 Write 入口

只允许从 `backend/` 执行一次：

```sh
APP_ENV=local DATABASE_PASSWORD="$DATABASE_PASSWORD" go run ./cmd/entity-seed \
  -chain-node-relation-manifest ../openspec/changes/rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-r0/approved-candidate-manifest.json \
  -chain-node-relation-approved-data-write
```

该入口复用现有 serializable transaction batch：整批规划、端点锁、tuple/ID 冲突检查、单事务写入、提交前逐 ID 复验与最终 aggregate 断言。禁止手工 SQL、其他 seed mode、第二次执行或自动 retry。

## Write 后 Query / Assert

单次命令成功后立即只读证明：

- CLI report 为 `created=4 / updated=0 / unchanged=96`，四类 counts 95/1/3/1。
- `chain_node_relations=100`，identity/content SHA-256 分别等于本包 target；manifest tuple 集合与 DB 集合完全相等。
- 两端均为冻结 842 active profiled chain_node；orphan、duplicate tuple、duplicate ID、self-loop、非法/legacy type、incomplete evidence/provenance/verified_at 均为 0。
- active chain_node/profile、external identifiers、entity_edges、physical constraints、Goose、schema hashes 与本包保护基线不变。
- 再做一次只读 dry-run 必须为 `created=0 / updated=0 / unchanged=100`；不得用第二次 approved Write 验证幂等。

## Stop Conditions

manifest/path/hash/count/type/endpoint/842 baseline、PG identity、Goose、schema、保护 counts/hash、当前 96 tuple/content、backup 或 dry-run 任一漂移立即停止。Write 非零退出、超时、连接中断、结果不确定、部分写、report/Query/hash 不一致时立即停止并报告；不得自动 retry、forward-fix、restore、Neo4j sync 或扩大 scope。

## R1 验证证据

- RED：新 100 条 manifest 被旧通用 loader 以 unknown `artifact_type` 拒绝，旧 frozen contract 符号缺失测试失败，证明 CLI 仍绑定历史 96 条格式/契约。
- GREEN：`go test ./cmd/entity-seed ./internal/apps/entityfoundation/seed -count=1` 通过。
- live read-only dry-run 精确为 4 created、96 unchanged、0 updated；未提交 PostgreSQL 写入。
- 本 change 的 OpenSpec strict、task-design lint、diff/scope/secret 将在 R1 checkpoint 前刷新。
