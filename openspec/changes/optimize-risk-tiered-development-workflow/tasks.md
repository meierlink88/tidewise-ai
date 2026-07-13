## 1. Proposal Review Gate（R1 change，风险：修改全项目正式研发流程）

- [ ] 1.1 主对话人工确认 proposal、design、delta spec 与 tasks，重点确认 R0—R3、阶段 Review package、R2 条件式执行包、R3 独立授权、active adoption 和不修改业务/数据库的边界；通过只授权进入 Apply，不授权任何数据库、Neo4j 或部署操作。

## 2. 工作流架构测试先行

- [ ] 2.1 在 `backend/internal/architecture/workflow_rules_test.go` 先新增或调整失败断言，覆盖 R0—R3、普通 checkbox 非人工 gate、阶段 Review package、R0/R1/按交付边界的 Apply final 验证、共享规则触发 repo-wide 验证、R2 pre/post assertions、backup 与 approved disposable recovery 的 fail-closed 选择、R2 逐层显式授权与失败后剩余授权失效、R3 独立恢复证据、阶段级 checkpoint、agent self-review 和 active adoption 边界。
- [ ] 2.2 运行 `go test ./internal/architecture -run Workflow -count=1`（或现有测试名匹配命令）并保留规则尚未实现时的预期 RED 证据。

## 3. 分层规则实现

- [ ] 3.1 更新 `.agents/openspec-workflow.md`，作为风险等级、gate 标注、阶段 Review package、候选数据审阅、条件式执行包、agent 自审和 active adoption 的详细唯一事实源；保留完整生命周期与两次人工 Review。
- [ ] 3.2 更新 `.agents/testing-tdd.md`，只维护 targeted/受影响交付边界完整/repo-wide 验证选择、R2/R3 pre/post evidence 和 self-review 测试证据边界，不复制完整授权流程。
- [ ] 3.3 更新 `.agents/git-workflow.md`，只维护阶段级 commit/checkpoint 与 active adoption 的 fetch、最新 `origin/main`、scoped tasks diff 和 Review 边界；保留 Desktop worktree、PR、merge、两类 cleanup 与 Delivered 条件。
- [ ] 3.4 更新 `.agents/skill-routing.md`，只补充阶段 Review package 与 self-review/code review 的 Skill 路由，不复制风险矩阵或执行包正文。
- [ ] 3.5 精简更新根 `AGENTS.md`，仅保留 R0—R3、人工授权不可推定、条件包/R3 硬门和详细规则路由的短摘要；不得扩写为第二份完整流程。
- [ ] 3.6 在工作流规则中实现 active change 显式 adoption 契约，并以 `refactor-industry-chain-node-foundation`、`reinitialize-alliance-economy-foundation` 的设计示例核对未来 gate 可迁移且历史授权不变；本 change 不修改两个 active change 的 artifacts。
- [ ] 3.7 完成架构测试 GREEN，并确认未修改业务源码、migration、seed、部署 workflow、其他 active change artifacts、`prototype/` 或 `doc/`。

## 4. 阶段 Review package 与内部自审

- [ ] 4.1 运行 targeted `go test ./internal/architecture -count=1`、`openspec validate optimize-risk-tiered-development-workflow --strict`、规则文件链接检查、重复/冲突扫描、`git diff --check`、scope 与 secret 检查；在 Review package 记录 affected delivery boundary、共享 tests、repo-wide 判定和 R2 backup/disposable recovery 的 fail-closed 选择规则。
- [ ] 4.2 task agent 对 requirements 覆盖、before/after gate 表、验证矩阵、状态机/sequence diagram、风险/回退和两个 active change 示例进行 self-review/code review；发现阻断问题先整改并刷新验证。
- [ ] 4.3 形成一个 R1 阶段 Review package，汇总规则 diff、架构测试、targeted 验证、未验证项和明确 non-goals；以一个 scoped 阶段 checkpoint 提交，不为 2.x/3.x 的微型 task 分别 commit、push 或 Review。

## 5. Apply Final 与人工 Review Gate（风险：新规则准备进入主规格）

- [ ] 5.1 本 workflow change 修改全项目规则与 architecture tests，故 Apply final 运行 repo-wide `go test ./...`、OpenSpec strict validation、规则链接/重复/冲突检查、`git diff --check`、scope/secret 检查，并确认没有 R2/R3 操作或状态变化；其他 change 按受影响交付边界完整验证，只有满足共享规则、跨模块契约、公共基础设施或 repo-wide 条件才运行 repo-wide full validation。
- [ ] 5.2 提交 scoped diff、Apply final 验证证据、self-review 结果与阶段 checkpoint，等待主对话 Apply 后人工 Review；通过前不得 Sync、Archive、Deliver，也不得修改任何 active change tasks。
