## Why

当前正式研发 change 把普通任务、阶段验收与有状态操作授权混在同一粒度，容易造成微型 task 频繁 Review、正常候选逐条机械审阅，以及旧批准被误解为后续写入授权。需要在不削弱 OpenSpec 生命周期、Desktop/Git 边界和数据库/图谱/部署安全门禁的前提下，引入统一风险分级与阶段化验收，使低风险工作更顺畅、高风险操作更明确、可恢复且可审计。

## What Changes

- 为所有新正式研发 change 定义 R0—R3 风险等级，并要求人工 gate 标注风险理由；普通 task checkbox 不自动成为人工门禁。
- 引入阶段 Review package，将 contract、实现、测试、dry-run 和只读 preflight 组织为可一次验收的阶段证据；按风险等级规定 checkpoint 验证深度，Apply final 才执行按交付边界选择的完整验证。
- 要求 task agent 在通知主对话验收前完成内部 self-review/code review，并先自行整改阻断问题。
- 将候选数据审阅改为“生成规则 + 抽样 + 异常/冲突清单”，同时保留对高风险、宽边界和冲突项的逐项审阅。
- 引入条件式执行包：用户对包内每个命名操作、环境、顺序、范围、可验证 recovery evidence、预期 counts、断言和停止条件一次明确授权；R2 对明确为 disposable 的 local/test 可使用经批准的 deterministic recreate/reseed evidence 替代物理备份，shared local、开发主数据、UAT 或不可替代数据仍必须备份；R3 必须提供备份/恢复或等价灾难恢复证据，默认不得跨层批量执行。
- 将 Apply final 改为“受影响交付边界的完整验证”：完整运行受影响 app/module/package 的 suite 与共享 architecture/contract tests；仅共享规则、跨模块契约、公共基础设施或 repo-wide 变更强制 repo-wide full validation。本 workflow change 属于共享规则变更，仍运行 `go test ./...`。
- 保留 `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver`、Proposal/Apply 后人工 Review、Desktop worktree/branch、PR/cleanup，以及数据库、图谱和部署明确授权、可恢复、可审计等不可削弱边界。
- 建立 active change 显式 adoption 流程：新规则 Deliver 后，active branch 从最新 `origin/main` 更新并提交 scoped workflow-adoption tasks diff，经一次人工 Review 后仅合并未来 gate，不追认历史操作、不取消已开始的写操作验收、不扩大既有授权。
- 规则正文采用分层单一事实源：根 `AGENTS.md` 只保留简短总原则，详细规则进入 `.agents/openspec-workflow.md`、`.agents/git-workflow.md`、`.agents/testing-tdd.md`，必要的 Skill 路由仅进入 `.agents/skill-routing.md`；以架构测试验证关键语义。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `skill-driven-development-workflow`: 增加统一风险分级、阶段 Review package、agent 自审、候选数据审阅策略、阶段级 checkpoint、条件式执行包及 active change 显式 adoption 的行为要求。

## Impact

- 本 change 基线为 R1（规则正文与工作流架构测试变更，无有状态写入）；当前 Proposal checkpoint 为 R0 artifact checkpoint。
- 影响 `tidewise-ai` 内的 `AGENTS.md`、命中的 `.agents/*.md`、工作流架构测试、OpenSpec 主规格与后续正式 change 的 artifacts/tasks 编写方式。
- 本 change 的 Apply 不修改业务源码、API、依赖、migration、seed、PostgreSQL/Neo4j 状态或部署环境，也不执行任何有状态操作。
- 当前 tidewise 本地 curated PostgreSQL 不得自动视为 disposable；若未来 R2 包选择 disposable recovery，必须逐层提供明确环境身份、disposable 声明、确定性重建/重灌命令、预计耗时与验证断言。
- `prototype/` 仅保持只读且不作为本 change 输入；`doc/` 不更新；不新增平行流程文档或长期事实来源。
- 当前 active changes 不自动改写；只有本规则 Deliver 后按显式 adoption 流程分别迁移未来 gate。
