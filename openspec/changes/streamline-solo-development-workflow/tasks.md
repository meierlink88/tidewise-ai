## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review；批准后连续执行内聚 Apply package | R1 | yes | SPEC_SEMANTICS | Proposal 批准后仅限本 change 的 workflow 规则、OpenSpec artifacts、现有 lint 的最小调整和稳定资源路径迁移评估；不得执行数据库/图谱写入、UAT/prod 或修改 backend 业务代码 |
| 2 | Apply-final Review；通过后才可 Sync/Archive/Deliver | R1 | yes | APPLY_FINAL | 仅限受影响 workflow/architecture 规则边界的完整验证、OpenSpec strict、diff/secret 检查及本 change scoped 交付；不得扩大到未批准环境或状态写入 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 0 |
| continuous_automation_scope | packages:none |

说明：`checkpoints=2` 仅指需要停顿并重新验收的 Proposal Review 与 Apply-final Review。Proposal 批准后的 Package 1 子项及其 commit 是连续执行证据；Archive/Deliver 只是生命周期记录，在输入指纹不变时不新增人工停顿或重复全量验证。`continuous_automation_scope=packages:none` 是因为两行 package 均为 `Human=yes`；它不否定 Proposal 批准后 Package 1 内子项连续执行，也不表示可跳过人工起始 gate。

## 1. Workflow contract and stable-resource Apply Package

- [x] 1.1 [RED] 为 package/gate 聚合、local-only 快速模式、stateful 一次 preflight/Write/verify、checkpoint 指纹与证据复用、stable resource path 及全局测试选择策略建立 Go architecture fixture/table-driven tests；确认测试不访问网络、凭证、生产数据库或运行时 manifest。
- [x] 1.2 [GREEN] 按 delta spec 最小修改 `.agents/openspec-workflow.md`、`.agents/testing-tdd.md`、`.agents/git-workflow.md` 或根规则中实际命中的重复/冲突条目；全局策略必须按真实影响区分 OpenSpec/workflow/architecture、局部 coding、数据-only 与 runtime repo-wide；保留 OpenSpec 生命周期、Superpowers、TDD、CI、UAT/prod 与 R2/R3 安全门禁，不新增 runner 或独立 lint 工具。
- [x] 1.3 [GREEN] 仅在真实消费者需要时，将 merged manifest 运行时输入迁移到稳定 backend `data/` 或 `resource/` 路径；OpenSpec change 仅保存 review snapshot/hash/evidence，完成只读路径、hash、schema 和 archive 后消费验证；若无真实 active-path 消费者则移除迁移项。
- [x] 1.4 [REFACTOR] 运行 targeted architecture/lint tests，修复失败并保持同一 package；将 package scope、non-goals、风险、证据、未验证项、阻断项、停止条件和下一步写入 checkpoint，失败不得新增人工 gate或隐式扩大授权。
- [x] 1.5 运行 package checkpoint 验证、`git diff --check`、scoped diff/secret/链接检查，复读结果并完成 self-review；只提交本 change 相关规则、spec、测试和稳定资源文件，不包含 backend 业务代码、数据库/图谱状态、UAT/prod 或 secret。

### Package 1 Evidence

- 阶段：Package 1 已完成 RED/GREEN/REFACTOR/checkpoint 验证；范围仅为 `.agents/openspec-workflow.md`、`.agents/testing-tdd.md`、`.agents/git-workflow.md`、`backend/internal/architecture/workflow_rules_test.go` 和本 change artifacts，non-goals 为 backend 业务逻辑、DB/schema、Neo4j、部署、UAT/prod/shared、secret 与运行时 manifest。
- Commit：输入 Proposal commit `e9047f73cdb5a42fd70949c107ada2f6f2e37e89`，此前 Apply commit `4fc962d`；本次全局策略修正的 scoped commit 待 Package 2 复核完成后创建。
- 已通过验证：RED 时 `TestStreamlinedSoloWorkflowContract` 与 `TestValidationScopeSelectionContract` 按预期暴露旧的机械全仓测试断言；GREEN 后 `GOCACHE=/tmp/tidewise-go-cache go test ./internal/architecture -count=1` 与 explicit task-design lint 通过。当前 change 属于 OpenSpec/workflow/architecture targeted 类；历史 `go test ./...` 仅为附加证据，不是本次要求。
- 输入状态指纹：accepted manifest SHA-256 `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268`；`frozenChainNodeRelationArtifactPath` 实际消费 `frozenChainNodeRelationArchiveRelativePath`，active 路径文件不存在，未发现 active runtime consumer。
- 下一步：完成 Package 2 fresh targeted validation 与独立复核、创建 scoped Apply commit/push，并仅等待 Apply-final Review。
- 真实 blocker：无。若 scope 扩展、manifest/hash 漂移、测试失败或发现 active runtime consumer，立即停止并重新审阅迁移范围；当前按条件范围跳过迁移，不制造 `data/resource` 副本或新增 path test。

## 2. Apply-final verification Package

- [x] 2.1 按全局测试选择策略运行当前 workflow-only change 的受影响 architecture/workflow 完整验证：`go test ./internal/architecture -count=1`、OpenSpec strict validation、精确 task-design lint、规则/路径/hash、scoped diff/secret/链接检查；只有 runtime repo-wide 触发条件成立时才运行 `go test ./...`。
- [x] 2.2 记录测试选择边界、命令、结果、未验证项、输入指纹和 checkpoint 下一步；历史 `go test ./...` 只作为附加证据，不能作为本 change 的验证要求。任一失败、漂移或边界不清立即停止并扩大验证或等待处理。
- [x] 2.3 完成 Apply-final self-review/code review，确认 proposal、design、delta spec、tasks、实现和验证证据一致；仅在用户通过 Apply-final Review 后执行 Sync、Archive、`openspec validate --all` 和 Deliver。

### Package 2 Evidence

- 阶段：Package 2 Apply-final targeted validation 已完成；依据全局策略，本 change 仅改 OpenSpec/workflow/agent-rule/architecture contract，未触发共享运行时代码、跨模块运行时契约、公共运行时基础设施或边界不清条件，因此不运行 repo-wide Go suite。
- Commit：输入 Proposal commit `e9047f73cdb5a42fd70949c107ada2f6f2e37e89`，此前 Apply commit `4fc962d`；本次 scoped correction commit 待独立 review 通过后创建。
- 已通过验证：`GOCACHE=/tmp/tidewise-go-cache go test ./internal/architecture -count=1`（exit 0）、explicit task-design lint（exit 0）、`openspec validate streamline-solo-development-workflow --strict`（valid）与 `git diff --check`（exit 0）均在本轮 fresh verification 通过；scoped diff 仅含本 change 的 7 个 workflow/OpenSpec/architecture 文件，secret scan 与 Markdown 链接 scan 均无匹配。历史 `GOCACHE=/tmp/tidewise-go-cache go test ./...`（exit 0）仅为更正前的附加证据，不能作为本 change 的要求或本轮 fresh evidence；未验证项为无。
- 输入状态指纹：`.agents/openspec-workflow.md` `18743c715987308f9d2b996d664f9ca29a06e84436a1f7efca711d4df6381c21`；`.agents/testing-tdd.md` `d3c05e6098c56586e5aa219e3fd1bf4c3261ab2bfd673831ec31c0d29a84bfab`；`.agents/git-workflow.md` `25292dac3f52815a8996b8e0a2785b876f670f09bd36e2fe23570fd7270ed167`；`workflow_rules_test.go` `3bb5a9b5361c9c916b805047828ba91ac306468913d8afc9fc7a9e5fa9d9d81f`；accepted manifest `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268`。
- 下一步：完成独立 review，创建 scoped correction commit/push，并仅等待 Apply-final Review。
- 真实 blocker：无；独立 reviewer 已复核并确认先前 Package 2 evidence P1 已关闭、无 blocking issue。未执行数据库、图谱、UAT/prod/shared、部署或 secret 操作。若 validation、scope、fingerprint 或 review 发生 drift，停止并重验受影响层。
