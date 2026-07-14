# Apply-final Review package

## 结论与当前 gate

- Change：`refactor-industry-chain-node-foundation`，schema=`spec-driven`。
- 风险基线：R3；实际操作按各命名层分别执行 R0/R1/R2/R3 gate。
- Apply tasks 1.1—2.9 已完成；本 package 仅请求主对话进行 Apply 后人工 Review。
- 本 checkpoint 不执行 PostgreSQL/Neo4j 操作，不包含 Sync、Archive、Deliver、PR merge 或 branch/worktree cleanup。Apply Review 获批前不得进入下一生命周期。

## 实际交付范围

- PostgreSQL 旧 sector、industry_chain、旧 chain_node、membership、topology、physical constraint、相关审计与旧引用已按受控 migration 15 清理。
- 产业概念统一为 `entity_type=chain_node`，复用 `entity_nodes` 的身份、名称、aliases、状态；`chain_node_profiles` 仅含 `entity_id`、非空 `definition` 与可空 `boundary_note`。
- 新增 `entity_type=theme` 与最小 `theme_profiles` 能力，但未创建任何 theme 实例或 theme-node link。
- 新增通用 `entity_external_identifiers`；首批写入 1,169 条已批准 chain_node 外部标识。
- 新增独立 `chain_node_relations` 与 physical constraint schema；首批只写入 96 条静态分类/组成关系，不复用 `entity_edges`。
- 默认生产入口拒绝旧 sector/industry_chain、membership/topology/source mapping 与旧关系类型；migration、seed、mapping 与 relation runner 均有 fail-closed、幂等和事务测试。

## Scoped diff 与验证边界

最终 change diff 只位于 `backend/` 与 `openspec/changes/refactor-industry-chain-node-foundation/`：

- `backend/`：单一 Go module 的 entity foundation domain/repository/seed/CLI、dbmigration executor/locking、migration 15—17 与对应 tests。
- `openspec/`：本 change 的中文 proposal/design/delta specs/tasks、候选 manifest/validation、授权包与脱敏执行证据。
- 未修改 `frontend/`、`doc/`、`prototype/`、共享 `.agents` 规则或其他 active change。

Apply-final 选择完整运行 `backend/go test -count=1 ./...`。理由是全部生产代码变更均在该 Go module 内，该命令已经覆盖受影响的 entity foundation、migration、dbmigration、CLI、domain、repository 和共享 `internal/architecture` / contract tests；本 change 没有 frontend、第二 Go module、共享规则或 repo-wide 基础设施变更，因此不再存在另一个需要追加执行的 repo-wide test module。

## 新鲜验证（2026-07-14T07:22Z）

| 验证 | 结果 |
| --- | --- |
| `cd backend && go test -count=1 ./...` | exit 0；全部 backend packages 通过，含 `internal/apps/entityfoundation/seed`、`internal/domain`、`internal/platform/dbmigration`、`migrations`、`cmd/entity-seed`、`cmd/dbmigrate` 与 `internal/architecture` |
| 两项 relation generator pytest | `2 passed`；候选分层与 96 条 write manifest 的 counts、hash input、确定性、blocked/rejected 排除契约通过 |
| `openspec validate refactor-industry-chain-node-foundation --strict` | 通过 |
| scoped diff / `git diff --check` | 仅 backend 与本 change artifacts；无 whitespace error |
| connection-string / secret scan | 无完整 PostgreSQL URL、密码、token、API key 或 private key material；命令证据只保留 runtime-only placeholder |

标准 Go suite 中由专用环境变量控制的真实 PostgreSQL integration tests未连接测试数据库；本轮没有取得新的数据库写授权，因此未为它们注入 local DSN。其风险由对应 sqlmock/静态 tests，以及各命名 R2/R3 层已完成的真实 local preflight、单事务 Write 和独立写后 Query/assert 覆盖；这不等同于声称这些条件式 integration tests 已执行。

## 已执行 R2/R3 命名层

| 命名操作 | 风险 / checkpoint | Write 前 | 实际 Write | 写后 Query/assert |
| --- | --- | --- | --- | --- |
| `phase-a-legacy-industry-cleanup` | R3 / `f2bc90a` | Goose14；634 entities；168 targets；466 protected；稳定 archive/hash/full decode | target-version 15，仅 migration 15 | Goose15；旧三类实体与专属结构为0；466 protected checksums不变；`entity_edges=331`；orphans/duplicates=0 |
| `phase-a-external-identifier-schema` | R2 / `ce2136d` | Goose15；migration16 pending；466/331；目标表不存在 | target-version 16，仅 migration 16 | Goose16；空表、PK/FK/三列 unique/5 CHECK/普通索引精确；466/331与保护 checksum 不变 |
| `phase-a-chain-node-seed` | R2 / `e058a42` | Goose16；chain_node/profile=0；manifest SHA=`9141571579ce2eee4b7524f686c6244572be6dd1bce2a1ee0494bcafd6aef05e` | 创建 842 node + 842 profile | 842/842；definition全非空；79 wide-boundary非空、其余763按 nullable 契约；身份/孤儿/非目标保护通过 |
| `phase-a-external-identifier-mapping` | R2 / `b94b189` | Goose16；842/842；mapping=0；manifest SHA=`05539cd9f940cfcc5ec67cde5c395563b672ffa52d56090da0a83bd0d5997658` | 单事务创建1,169 | 818 eastmoney、351 ths、241双来源、13双taxonomy；重复/孤儿/错误绑定=0；dry-run=`0/0/1169` |
| `phase-b-relation-schema` | R2 / `a903e1e` | Goose16；migration17 pending；842/842/1,169/331；目标表不存在 | target-version 17，仅 migration 17 | Goose17；relation/constraint表schema精确且rows=0；保护基线不变 |
| `phase-b-relation-data` | R2 / `e2becc2` | Goose17；relation/constraint=0；manifest SHA=`7651e0b591df1e03838df00ebc9acd6101ebcc76da18a6a314ff478c9f42990e`；dry-run=`96/0/0` | 首次 precommit 假不匹配完整rollback；R1 `ee0379e` 后获批唯一重试，单事务创建96 | 95 `is_subcategory_of` + 1 `is_component_of`；input/depends/constraint=0；self-loop/evidence/tuple/endpoint异常=0；dry-run=`0/0/96`；842/842/1,169/331不变 |

详细证据入口：

- [Phase A Acceptance](phase-a-acceptance-review.md)
- [legacy cleanup package / execution record](phase-a-legacy-industry-cleanup-authorization.md)
- [external identifier schema package / execution record](phase-a-external-identifier-schema-authorization.md)
- [chain_node seed evidence](phase-a-chain-node-seed-execution-evidence.md)
- [external identifier mapping evidence](phase-a-external-identifier-mapping-execution-evidence.md)
- [relation schema evidence](phase-b-relation-schema-execution-evidence.md)
- [relation data evidence](phase-b-relation-data-execution-evidence.md)

命名 R2 `phase-a-backup-restore-rehearsal` 曾获授权后被主对话撤销，未创建目标恢复库、未执行 restore/assertion，临时资源零残留且 `backup_verified=false`；它不是已执行写层，也未被以上表格伪装为恢复验证。local cleanup 使用的是主对话明确接受的 archive integrity/hash、full decode、冻结基线与 forward-fix 风险边界。

## 最终已验收 local 状态

主对话对 checkpoint `e2becc2` 的独立 fresh readback 已确认：

| 断言 | 状态 |
| --- | --- |
| Goose | 17 |
| active chain_node / profiles | 842 / 842 |
| external identifiers | 1,169 |
| chain_node relations | 96；95 subcategory + 1 component；input/depends=0 |
| physical constraints | 0 |
| entity_edges | 331 |
| self-loop / blank evidence / duplicate tuple / bad endpoint | 均为0 |
| relation-only dry-run | `created=0, updated=0, unchanged=96` |

## Non-goals、未验证项与延后项

- Neo4j projection 在 PostgreSQL cleanup 后仍是陈旧投影；本 change 未查询、清理、写入或 rebuild Neo4j。后续必须以独立 change 和独立 R3 授权处理。
- UAT、prod、shared 环境未执行 migration、seed 或数据写入；所有实际有状态操作只发生在获批 local `tidewise_local`。
- 未创建 theme 实例或 theme-node link；未实现事件提取/推理、动态传导、观测数据、股票推荐或证券成分。
- 未调整 alliance、economy/country、market、benchmark/index；12 类非目标实体 counts/checksums 在各写层保持不变。
- 51 条 blocked relation、7 条 rejected relation、4 条 blocked physical constraint 均未写入。`input_to=0`、`depends_on=0`、physical constraint=0；本次96条只能解释为静态分类/组成，不是上下游、瓶颈或事件传导网络。
- physical constraint semantic identity 仍延后，当前没有 write-ready constraint，不构成本 change Apply-final blocker。

## Blockers 与下一步边界

没有已知实现或 local 数据完整性 blocker。唯一当前 gate 是本 package 的 Apply 后人工 Review：

1. 若主对话批准，下一阶段只能按 OpenSpec 生命周期进入 Sync；该批准本身不等于 Sync、Archive 或 Deliver。
2. 未获批准前不得修改主规格、archive change、创建完成态 PR、merge 或 cleanup branch/worktree。
3. Neo4j rebuild、physical constraint、blocked/rejected relation 与其他环境操作均不因 Apply Review 获得授权。
