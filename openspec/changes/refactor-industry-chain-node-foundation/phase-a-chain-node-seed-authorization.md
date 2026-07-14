# Phase A chain_node seed R2 execution record

- 命名操作：`phase-a-chain-node-seed`；仅限 `tidewise-local-postgres` / `tidewise_local` / PostgreSQL 16.14。
- 主对话已批准 task 1.16 的 commodity→产业重命名与此命名 R2 层的自动调度；仅允许写入本文件所列精确 manifest 的 842 个 `chain_node` 与其 profile。
- 输入：[node/profile seed manifest](final-seed-candidate-artifacts/node-profile-seed-manifest.json)，SHA-256 `9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e`；候选审阅 manifest 指纹为 `14b9f1e1b4c5cb1925f33e6788eb755e89a24bdb8c590abfadad47956d1d6e3f`。
- 允许的唯一 Write 路径：`go run ./cmd/entity-seed -manifest-file <approved exact manifest>`；禁止手工 SQL、default seed、mapping、theme、relation/constraint、migration、Neo4j、UAT/prod。

## 条件式 preflight

同一维护窗口必须全部满足：Goose=16；`entity_nodes=466`、`entity_edges=331`、12 类非目标 count/checksum 与 migration 16 写后 evidence 一致；`chain_node_profiles=0`、`theme_profiles=0`、`entity_external_identifiers=0`；无 chain_node、无候选 UUID/key/canonical/alias 冲突、无 writer/长事务/锁冲突；稳定 custom backup 的 size/SHA-256/archive integrity/full decode 仍有效。任一漂移立即停止，不执行 Write。

## Write / Query 断言

Write 前以 manifest SHA-256 与候选指纹精确匹配为硬门禁。成功后立即只读 Query/assert：`entity_nodes=1308`、active `chain_node=842`、`chain_node_profiles=842`、`entity_edges=331`、external identifier rows=0、theme rows=0；466 个非目标 entity 和其 checksums 不变；842 个新 UUID/key/canonical 唯一；aliases 无跨节点或跨现存实体冲突；definition 非空；79 条宽边界均有 boundary；orphans/duplicate/blank key=0。Go seed report 必须只含 entity_nodes=842 created、chain_node_profiles=842 created，0 updated/unchanged/failed，且不包含其他表。

任一 assertion 失败，立即停止并保存脱敏 evidence；不执行 Down、cleanup 或下一层 Write，只能另行提交 forward-fix Review。完成 Query/assert 后等待主对话验收，不自动进入 mapping。

## 首次执行结果（fail-closed）

同一窗口 preflight 满足 Goose=16、466 entities、331 edges、0 chain_node/profile/external identifier、0 writer/long transaction，以及 stable backup SHA-256/size。标准 `entity-seed -manifest-file` 在首个 entity upsert 被 PostgreSQL 报错阻断：`entity_convergence_alias_moves` 不存在。立即只读 Query 确认 Goose=16、`entity_nodes=466`、`entity_edges=331`、`chain_node=0`、`chain_node_profiles=0`、external identifier=0，未发生部分写入。根因与 R1 最小修复见 [forward-fix Review](phase-a-chain-node-seed-forward-fix-review.md)。本记录不因修复自动恢复 R2 授权：不得重试、不得手工 SQL，必须重新提交并获批刷新后的 preflight -> Write -> Query 包。
