# Phase A chain_node seed forward-fix Review（R1）

## 事实与根因

- 首次命名 R2 `phase-a-chain-node-seed` 的只允许路径在第一个 `entity_nodes` upsert 即返回 PostgreSQL `42P01`：`entity_convergence_alias_moves` 不存在；写后只读断言确认 Goose=16、`entity_nodes=466`、`entity_edges=331`、`chain_node=0`、`chain_node_profiles=0`、`entity_external_identifiers=0`，没有部分写入。
- `000015_refactor_industry_chain_node_phase_a.sql` 已按获批 cleanup 范围删除 `entity_convergence_alias_moves`、`entity_convergence_reference_moves`、`entity_convergences` 与 `entity_convergence_manifests`。
- 但通用 `buildEntityUpsert()` 仍无条件组装 `entityAliasOwnershipCTEs()`，读取上述已删除表并把旧 alias move 追加到 seed aliases。这与 cleanup 后全新、由审核 manifest 唯一确定 aliases 的契约矛盾，且会使每次普通 seed 在首条写入前失败。

## R1 最小修复

1. `buildEntityUpsert()` 直接写入已在 Go 层稳定排序、去重和校验的 `$7::text[]` aliases；不再组装旧 convergence CTE。
2. 删除只服务该 CTE 的 alias merge helper 及要求已删除 convergence audit 表存在的集成测试；保留的 PostgreSQL `EXPLAIN` 测试改为 `chain_node` 输入，且不写入数据。
3. 新增回归断言：普通 entity upsert SQL 不得出现 `entity_convergence_alias_moves`、`entity_convergences`、`entity_convergence_manifests`、`owned_aliases` 或 `seed_aliases`，并必须直接使用 `$7::text[] AS aliases`。

## 验收与边界

- RED：新回归测试先因 SQL 含 `entity_convergence_alias_moves` 失败。
- GREEN：最小修复后，定向 `go test -count=1 ./internal/apps/entityfoundation/seed -run 'Test(EntityUpsertSQL|PostgresChainNodeUpsertSQL)'` 通过；完整验证与 checkpoint 前须重新运行 scoped suite、全量 Go 测试、OpenSpec strict、diff/scope/secret 检查。
- 本包是 R1 代码/测试与文档审阅，不重新执行、也不追认失败的 R2 Write。未获新的明确 R2 许可前，不得连接数据库执行 node/profile seed、不得 mapping、relation、migration、Neo4j、UAT/prod 或手工 SQL。
- 新 R2 包必须重新核对 approved manifest SHA-256、候选指纹、Goose=16、466 非目标保护基线、零 chain_node/profile、无 writer/long transaction/lock conflict、stable backup evidence，随后仅允许一次标准 `entity-seed -manifest-file` Write，并立即执行完整 Query/assert；任一漂移继续 fail-closed。
