# Package 5 Apply-final Review Package

准备时间：2026-07-14。本 package 只汇总 Apply-final 人工 Review 所需证据；不执行 Sync、Archive、Deliver、PR、merge、branch/worktree cleanup、数据库/migration/Neo4j 或其他环境操作。

## 已验收范围

- 联盟输入与冻结 artifact：45 alliance，79 target economy，133 formal-active `economy -> alliance_org member_of`；冻结 manifest SHA-256 为 `118573006c341830f3c977a994d12a537fe2d1ac8174a818d8007fe98e87a55d`。
- 最小 schema：migration `000018` 仅将 `alliance_org_profiles` 收敛为 `abbreviation`、`leadership_summary`、`influence_scope_summary`；未引入 economy identity_kind、平行 profile 表或全局 entity_key 唯一约束。
- local PostgreSQL 已由独立执行包验收：Goose=18，alliance/profile=45/45，economy/profile=94/94（79 target + 15 non-target），member_of=133，has_market=40，dependency audit `blocked=false`。
- 同一 artifact 的幂等复跑返回 entity/profile/member writes=`0/0/0`；stable protected checksum 为 `388e2945842053f5eb0674e7901b2f8b351b743c21951b9f43cb314fea987c26`。
- PostgreSQL 是本 change 的事实源完成边界；Neo4j 仍明确排除。

## Requirements Coverage Self-review

| 已批准要求 | 覆盖证据 | 结论 |
|---|---|---|
| alliance 名称与 profile 三字段边界 | migration 000018、manifest validator、schema/seed tests | 覆盖 |
| 45/79/133 冻结输入、端点与 formal-active 方向 | frozen manifest、manifest/rebuild tests、4.2 exact Query evidence | 覆盖 |
| 15 non-target 与跨域事实保护 | dependency audit、cleanup/rebuild protection tests、4.1/4.2 evidence | 覆盖 |
| 单次 change-specific importer、fail-closed、幂等 | entity-seed isolated mode、repository tests、零写入 idempotency evidence | 覆盖 |
| 不扩展 categories、economy schema、通用 framework、Neo4j | proposal/design/delta specs 与 scoped diff self-review | 覆盖 |

## Fresh Apply-final Validation

从 `backend/` 运行：

```text
go test -count=1 ./cmd/entity-seed ./internal/apps/entityfoundation/seed ./internal/domain ./internal/repositories ./internal/platform/dbmigration ./migrations
PASS

OPENSPEC_TASK_LINT_CHANGE=reinitialize-alliance-economy-foundation go test -count=1 ./internal/architecture
PASS

openspec validate reinitialize-alliance-economy-foundation --strict
PASS
```

`git diff --check`、`git diff origin/main...HEAD --check`、范围清单与 secret 扫描已执行。扫描仅命中测试环境变量名和既有文档中的规则性描述；没有密码、token、连接串或 API key 被加入 diff。

未运行 `backend go test ./...`：本 change 的实际 diff 不涉及公共 platform、HTTP/API、采集、图投影或跨模块共享基础设施；完整验证已覆盖 entity-seed、entityfoundation、domain、repositories、migration runner/contract 和共享 architecture。该判定符合受影响交付边界规则，不是未验证项。

## Self-review / Code Review

- 对 `origin/main...HEAD` 的 35 个文件执行范围审查：修改仅位于 entity-seed、entity foundation seed、冻结 data、domain/migration contract、migration 000018 和本 change OpenSpec artifacts。
- 复核没有新增通用 runner/service/policy/report framework、无未批准 relation type、无 Neo4j 代码或配置、无未授权跨域写入入口。
- 未发现阻断问题；本 package 唯一待决事项是主对话的 Apply-final 人工 Review。

## Requested Decision And Stop Boundary

请主对话 Review 本 package。只有明确 Apply-final 批准后，才可启动 Sync；本次 checkpoint 保持 5.1 未完成，且不推定 Archive、Deliver 或任何后续 Git 生命周期授权。
