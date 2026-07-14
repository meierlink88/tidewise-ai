## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review 通过且前置 graph change 完整 Deliver 后连续执行 R1 整理 | R1 | no | NONE | 仅机械拆分 `backend/internal/repositories`、迁移有效测试并删除已确认无效测试；不得改变行为或执行有状态操作 |
| 2 | Apply-final Review | R1 | yes | APPLY_FINAL | 汇总 scoped diff、测试与验证证据；通过后仅允许 Sync、Archive |
| 3 | Git completion | R0 | yes | GIT_COMPLETION | Archive checkpoint 后才允许 PR merge 与 Desktop-owned cleanup；不得扩大实现范围 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 3 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:1 |

## 1. Repository 与 Migration Test 整理 Package

- [ ] 1.1 等 `rebuild-foundation-graph-and-enrich-chain-data` 完整 Deliver、PR merge 与 branch/worktree cleanup 后，fetch 并更新到最新 `origin/main`，对 repository、graph projector、domain、migration tests 和主规格执行 fresh overlap audit；逐项输出 `retain-and-move`、`already-removed`、`remove-as-orphan`，若最终契约超出已批准矩阵则先更新 artifacts 并回到 Review。
- [ ] 1.2 先运行并保存 `go test ./internal/repositories ./internal/platform/dbmigration ./migrations` 当前行为基线，再补充会在错误移动、目录不纯或安全契约丢失时失败的最小测试；不得设置真实数据库 DSN 或执行 migration/seed/PG/Neo4j write。
- [ ] 1.3 按 design 处置矩阵将仍有效的 migration 静态、Goose transaction/rollback 和 opt-in integration 契约迁入 `internal/platform/dbmigration`，删除有证据证明只保护已废止最终 schema 的断言，并使 `backend/migrations` 只保留 SQL 与 `README.md`；运行 `go test ./internal/platform/dbmigration`。
- [ ] 1.4 按 source catalog、raw document、benchmark observation、admin query、scheduler、ingestion run、graph projection、identity 文件映射先拆对应测试，再机械移动业务接口/DTO、`PostgresRepository` SQL/Scan 和 `InMemoryRepository` 方法；保留唯一共享 adapter、现有接口语义和参数化 SQL，每完成一个业务职责运行 `go test ./internal/repositories`。
- [ ] 1.5 将 `postgres.go` 与 `memory.go` 收敛到最小共享核心，删除空 `doc.go` 和已搬空旧文件；依据 overlap audit 处理 legacy graph/sector/industry-chain 条件项并完成 identity 命名/归属审计，未证明需要重命名时只移动不改签名。
- [ ] 1.6 运行 `go test ./internal/repositories`、`go test ./internal/platform/dbmigration`、`go test ./internal/apps/...`，检查业务接口/SQL/错误/结果/稳定 ID 行为、migration 目录纯度和矩阵覆盖；在同一 package 内连续修复，完成 self-review、scoped diff/scope/secret 检查后提交并 push R1 checkpoint，不创建 PR。

## 2. Apply-final、Sync 与 Archive Package

- [ ] 2.1 运行唯一一次 Apply-final `go test ./...`，再运行 OpenSpec strict、task-design lint、`git diff --check`、scope/secret scan；复读新鲜结果并完成需求覆盖自审，提交并 push scoped Apply-final Review package 后停止等待人工 Review。
- [ ] 2.2 仅在 Apply-final Review 明确通过后 Sync delta specs、Archive change、运行 `openspec validate --all` 和 archive scoped diff/secret 检查，并提交 archive checkpoint；不得创建或 merge PR。

## 3. Git Completion Package

- [ ] 3.1 仅在 Git completion 获得明确授权且 archive checkpoint 完整后，按 Git workflow push、创建/审阅 PR、merge，并由 Codex Desktop 按所有权顺序完成远端 branch、task/worktree 与本地 branch cleanup；全部条件满足前不得声明 Deliver。
