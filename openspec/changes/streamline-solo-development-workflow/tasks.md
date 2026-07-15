## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review；批准后连续执行内聚 Apply package | R1 | yes | SPEC_SEMANTICS | 仅限本 change 的 workflow 规则、OpenSpec artifacts、现有 lint 的最小调整和稳定资源路径迁移设计；不得执行 Apply、数据库/图谱写入、UAT/prod 或修改 backend 业务代码 |
| 2 | Apply-final Review；通过后才可 Sync/Archive/Deliver | R1 | yes | APPLY_FINAL | 仅限受影响 workflow/architecture 规则边界的完整验证、OpenSpec strict、diff/secret 检查及本 change scoped 交付；不得扩大到未批准环境或状态写入 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

说明：`checkpoints=2` 仅指需要停顿并重新验收的 Proposal Review 与 Apply-final Review。Proposal 批准后的 Package 1 子项及其 commit 是连续执行证据；Archive/Deliver 只是生命周期记录，在输入指纹不变时不新增人工停顿或重复全量验证。`continuous_automation_scope=packages:none` 是因为两行 package 均为 `Human=yes`；它不否定 Proposal 批准后 Package 1 内子项连续执行，也不表示可跳过人工起始 gate。

## 1. Workflow contract and stable-resource Apply Package

- [x] 1.1 [RED] 为 package/gate 聚合、local-only 快速模式、stateful 一次 preflight/Write/verify、checkpoint 指纹与证据复用、stable resource path 建立 Go architecture fixture/table-driven tests；确认测试不访问网络、凭证、生产数据库或运行时 manifest。
- [x] 1.2 [GREEN] 按 delta spec 最小修改 `.agents/openspec-workflow.md`、`.agents/skill-routing.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md` 或根规则中实际命中的重复/冲突条目；保留 OpenSpec 生命周期、Superpowers、TDD、CI、UAT/prod 与 R2/R3 安全门禁，不新增 runner 或独立 lint 工具。
- [x] 1.3 [GREEN] 仅在真实消费者需要时，将 merged manifest 运行时输入迁移到稳定 backend `data/` 或 `resource/` 路径；OpenSpec change 仅保存 review snapshot/hash/evidence，完成只读路径、hash、schema 和 archive 后消费验证；若无真实 active-path 消费者则移除迁移项。
- [x] 1.4 [REFACTOR] 运行 targeted architecture/lint tests，修复失败并保持同一 package；将 package scope、non-goals、风险、证据、未验证项、阻断项、停止条件和下一步写入 checkpoint，失败不得新增人工 gate或隐式扩大授权。
- [x] 1.5 运行 package checkpoint 验证、`git diff --check`、scoped diff/secret 检查，复读结果并完成 self-review；只提交本 change 相关规则、spec、测试和稳定资源文件，不包含 backend 业务代码、数据库/图谱状态、UAT/prod 或 secret。

### Package 1 Evidence

- 阶段：Package 1 已完成 REFACTOR/checkpoint 验证；范围仅为 `.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`backend/internal/architecture/workflow_rules_test.go` 和本 change tasks，未修改 backend 业务逻辑、DB/schema、Neo4j、部署、UAT/prod/shared 或 secret。
- Commit：输入 Proposal commit `e9047f73cdb5a42fd70949c107ada2f6f2e37e89`；本 Package 的 scoped Apply commit 在所有 Package 2 证据与 review 通过后创建。
- 已通过验证：RED 时 `TestStreamlinedSoloWorkflowContract` 因缺少新 contract 按预期失败；GREEN 后该测试、`go test ./internal/architecture -count=1`、frozen chain-node relation targeted tests、OpenSpec strict、explicit task-design lint 与 diff/secret 检查通过。
- 输入状态指纹：accepted manifest SHA-256 `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268`；`frozenChainNodeRelationArtifactPath` 实际消费 `frozenChainNodeRelationArchiveRelativePath`，active 路径文件不存在，未发现 active runtime consumer。
- 下一步：完成 Package 2 复核、创建 scoped Apply commit/push，并仅等待 Apply-final Review。
- 真实 blocker：无。若 scope 扩展、manifest/hash 漂移、测试失败或发现 active runtime consumer，立即停止并重新审阅迁移范围；当前按条件范围跳过迁移，不制造 `data/resource` 副本或新增 path test。

## 2. Apply-final verification Package

- [x] 2.1 运行受影响 workflow/architecture 边界的完整验证：`go test ./...`、OpenSpec strict validation、现有 task-design lint、规则/路径/hash 检查及 scoped diff/secret 检查。
- [x] 2.2 记录 repo-wide full validation 的触发理由、命令、结果、未验证项、输入指纹和 checkpoint 下一步；任一失败、漂移或边界不清立即停止并扩大验证或等待处理。
- [x] 2.3 完成 Apply-final self-review/code review，确认 proposal、design、delta spec、tasks、实现和验证证据一致；仅在用户通过 Apply-final Review 后执行 Sync、Archive、`openspec validate --all` 和 Deliver。

### Package 2 Evidence

- 阶段：Package 2 Apply-final validation 已完成；repo-wide 触发理由为本 change 修改共享 workflow 规则与 `backend/internal/architecture` contract test，完整边界因此为 backend 全 module 和 active OpenSpec change。
- Commit：输入 Proposal commit `e9047f73cdb5a42fd70949c107ada2f6f2e37e89`；Apply checkpoint commit 待 final review 通过后创建。
- 已通过验证：最终 tasks 内容下重新运行 `GOCACHE=/tmp/tidewise-go-cache go test ./...`（exit 0）、`openspec validate streamline-solo-development-workflow --strict`（valid）、explicit task-design lint（exit 0）、`git diff --check`（exit 0）和 scoped secret scan（无匹配）；未验证项为无。
- 输入状态指纹：`.agents/openspec-workflow.md` `f584bcf59dc3c7e2168c7f4f70c1aa873f090545d16f3b980cd75f75fc85eb8a`；`.agents/git-workflow.md` `25292dac3f52815a8996b8e0a2785b876f670f09bd36e2fe23570fd7270ed167`；`workflow_rules_test.go` `dfecb2a76d48fb7394591e87373ff8a28b2ef407252114461c6a887c7b97f26b`；accepted manifest `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268`。
- 下一步：创建 scoped Apply commit/push，并仅等待 Apply-final Review。
- 真实 blocker：无；独立 code review 已复核且无阻断项。未执行数据库、图谱、UAT/prod/shared、部署或 secret 操作；若 validation、scope、fingerprint 或 review 发生 drift，停止并重验受影响层。
