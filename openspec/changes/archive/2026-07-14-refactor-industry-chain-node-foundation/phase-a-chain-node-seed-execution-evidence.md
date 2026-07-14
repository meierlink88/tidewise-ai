# Phase A chain_node seed 执行证据（R1 修复 + R2）

## 边界与结论

- 命名操作：`phase-a-chain-node-seed`；唯一目标为 local `tidewise_local` 中批准 manifest 的 842 个 `chain_node` 与 842 个 `chain_node_profiles`。
- 结果：R1 最小修复和 fresh 验证通过后，R2 唯一一次标准 seed Write 成功；task 1.17 等待主对话独立验收。
- 未执行：external identifier/source mapping、theme 实例、关系/physical constraint、migration 17、旧数据 cleanup、Neo4j、UAT/prod/shared、Sync/Archive/Deliver/PR merge。

## R1：post-cleanup preflight 修复

首次 R2 的遗留 convergence 引用已被前次 forward-fix 清除；本次 fresh preflight 又发现只读报告仍查询 migration 15 已删除的 `sector_profiles`。该错误 fail-closed，未发生 Write。

本次 TDD 最小修复将 `entity-seed -phase-a-preflight` 对齐 Goose=16 的 post-cleanup schema：

- 读取并报告当前 database、PostgreSQL version、Goose version；验证当前 target schema、旧对象已移除、catalog legacy 引用为零、profile/external identifier/event/edge orphan 为零。
- 强制 `-manifest-file`，计算并报告输入 SHA-256、entity、chain_node 与 profile 数；不再把旧 sector/industry_chain/membership/topology/convergence 表当作当前表查询。
- 保留环境身份、受保护实体 count/checksum、恢复证据、输入指纹、候选 identity、目标空集与锁/事务等门禁；没有删除断言、忽略错误或放宽范围。
- RED/GREEN 回归测试覆盖 Goose=16 post-cleanup report 与 explicit manifest proof；定向测试和 fresh `go test -count=1 ./...` 均通过。

## 输入与恢复证据

| 项目 | 结果 |
| --- | --- |
| 批准 manifest | `final-seed-candidate-artifacts/node-profile-seed-manifest.json` |
| manifest SHA-256 | `9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e` |
| manifest 内容 | 842 entities / 842 chain_node / 842 profiles |
| stable custom archive | `~/.local/share/tidewise-ai/backups/postgresql/phase-a/20260714T015901Z/tidewise_phase_a_post_cleanup_pre_schema16.dump` |
| archive size | 904,964 bytes |
| archive SHA-256 | `02b12193985b0fc88df8c04b3a8efe5c3312e2184513be6036676243f616a0e4` |

本窗口重新核验 archive 的 size/SHA-256，精确匹配已验收的 archive integrity/full-decode 证据对应字节。未进行 restore rehearsal，也没有把 `backup_verified` 改为 true；本次 local-only R2 沿用已批准的稳定 archive/hash、冻结基线与 forward-fix recovery evidence。

## Write 前只读 preflight

- 环境身份：`tidewise_local`、PostgreSQL 16.14、Goose=16；仅 migration 16 已应用。
- 写前目标状态：`entity_nodes=466`、`chain_node=0`、`chain_node_profiles=0`、`theme_profiles=0`、`entity_external_identifiers=0`、`entity_edges=331`。
- 12 类非目标实体的 counts 与 checksums 均与 cleanup/migration 16 写后冻结基线相同；候选 UUID/key/canonical/alias 冲突查询为零。
- retired sector/industry_chain/membership/topology/convergence relation 为零，legacy function/trigger/view/rule 引用为零；blank/duplicate key、profile/external/event/edge orphan 均为零。
- 维护窗口内 writer、long transaction、waiting lock 均为零。

## 单次 Write 与写后 Query/assert

唯一执行路径为标准 `entity-seed -manifest-file <approved exact manifest>`（runtime-only local credentials，不记录连接串）。Go report：

| 对象 | created | updated | unchanged | failed |
| --- | ---: | ---: | ---: | ---: |
| `entity_nodes` | 842 | 0 | 0 | 0 |
| `chain_node_profiles` | 842 | 0 | 0 | 0 |

随即只读 Query/assert 结果：

- Goose=16；`entity_nodes=1,308`、active `chain_node=842`、`chain_node_profiles=842`、`entity_edges=331`、external identifier=0、theme/theme profile=0。
- 主对话对 local PostgreSQL 的 fresh 只读读回确认：842 个 manifest UUID/entity_key/canonical 与数据库实际记录逐一精确匹配；UUID、entity_key、canonical 无重复，profile 无 orphan，842 条 `definition` 均非空。79 条 wide-boundary 节点的 `boundary_note` 均非空；其余 763 条按已批准的可空契约为 `NULL`/空值，未发生数据漂移。
- 16 个 commodity 同名冲突节点均使用批准的产业语义 canonical：动力煤产业、大豆产业、天然气产业、橡胶产业、焦炭产业、焦煤产业、玉米产业、白银产业、稀土产业、纯碱产业、钴产业、铁矿石产业、铜产业、铝产业、镍产业、黄金产业。
- 12 类非目标实体 count/checksum 与写前基线逐项相同；旧实体类型、retired schema、legacy catalog reference、blank/duplicate key 与所有已列 orphan 断言均为零。

本授权只允许一次 Write，因此没有以第二次写入验证幂等。幂等的本次证据是确定性 manifest、842/842 identity 精确匹配、零重复以及 repository 的回归测试；如需实际重复 Write，必须另获单独授权，预期只能为 unchanged 且不得改变保护基线。

## 新鲜验证

- `go test -count=1 ./cmd/entity-seed ./internal/apps/entityfoundation/seed`：通过。
- `go test -count=1 ./...`：通过。
- `openspec validate refactor-industry-chain-node-foundation --strict`：通过。
- `git diff --check`：通过。

主对话已在本证据更正后完成 task 1.17 技术验收。下一步仅允许 task 1.18 的 R0 mapping candidate Review；mapping 层仍受 13 个未消歧 taxonomy code 阻断，且本次未写入任何 mapping。
