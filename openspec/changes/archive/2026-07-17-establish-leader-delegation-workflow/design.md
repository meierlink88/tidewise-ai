## Context

当前 `engineering-change-workflow` 已规定 Eino-first、OpenSpec、独立 `codex/<change-name>` 分支/worktree、TDD、两次人工评审和 PR 交付，但默认由同一个 Agent 贯穿全部阶段。该模型没有约束主 Codex 任务是否亲自实施，也没有定义新 change 如何获得独立 Desktop 任务，更未覆盖 PR merge 后的 worktree 和分支清理。

本 change 面向开发 Leader、执行 Agent 和最终控制 PR merge 的用户。Leader 位于主 Codex 任务，掌握全局编排与验收上下文；执行 Agent 位于由 Codex Desktop 创建的独立任务和 worktree，承担 change 的全部具体产出。现有 `adopt-openspec-tdd-workflow` worktree 是已合并的独立资源，不属于本 change 的修改或清理范围。

Eino 三仓审计提供了职责和门禁设计参考：

- `eino-ext` 的 `skills/eino-agent/reference/agent-as-tool.md`、`deep-agents.md` 与 `human-in-the-loop.md` 展示了只向子 Agent 传递明确任务、隔离上下文、Interrupt/Checkpoint/Resume 人工确认；这些模式用于提炼“委派隔离”和“显式批准”语义，不作为仓库工作流运行依赖。
- `eino-examples` 的 `adk/multiagent/supervisor`、`adk/human-in-the-loop/1_approval` 和 `5_supervisor` 展示了主管只协调专业 Agent、敏感动作先中断再由人批准，以及嵌套子 Agent 中断后定向恢复。
- `eino` 的 `adk/prebuilt/supervisor`、`adk/agent_tool.go`、`adk/interrupt.go`、`adk/deterministic_transfer.go` 及聚焦测试确认：Supervisor transfer 共享完整上下文且官方不推荐用于多数场景；AgentTool 默认隔离消息历史、共享受控 SessionValues；CompositeInterrupt 可跨嵌套边界传播审批；恢复必须匹配 checkpoint 和 interrupt target。
- Codex `create_thread`、Desktop-managed worktree、Git 分支和 post-merge cleanup 均不属于 Eino 运行时能力，继续由项目文档、OpenSpec requirements 和 Go 策略测试治理。

## Goals / Non-Goals

**Goals:**

- 建立 Leader 与执行 Agent 的强制职责分离，禁止 Leader 亲自实施 change。
- 要求每个新 change 通过 Codex `create_thread` 获得独立执行任务和 Desktop-managed worktree。
- 保持执行 Agent 对 Eino audit、OpenSpec、TDD、实现、验证、Sync、Archive、Git 提交、推送和 PR 的端到端责任。
- 将 Proposal Review 和 Apply-final Review 明确为 Leader 门禁，禁止执行 Agent 自我批准，并保留用户对 PR merge 的控制。
- 定义 merge 后可证明、安全、失败可见的 worktree/branch 清理顺序。
- 用现有 workflow policy Go 测试保护稳定的职责、委派、门禁、生命周期和清理标记。

**Non-Goals:**

- 不实现新的 Eino Agent、Supervisor、AgentTool、Interrupt 或 CheckPointStore。
- 不自动创建、批准或合并 Pull Request，不改变用户对 merge 的控制。
- 不新增守护进程、Git wrapper、数据库、队列、模型或外部服务。
- 不改变三个运行时任务 Agent 的业务行为或架构边界。
- 不修改、清理、暂存或提交已有 `adopt-openspec-tdd-workflow` worktree。

## Decisions

### 以 Codex 任务作为 change 的执行隔离单元

Leader 对每个新 change 使用 `create_thread` 创建独立执行任务，并由 Codex Desktop 配置独立 worktree。执行任务使用 `codex/<change-name>` 分支；若 Desktop 初始为 detached HEAD，执行 Agent在首次写入 change 文件前从最新 `origin/main` 安全创建并切换该分支。

选择该方案是因为 Codex 任务同时隔离对话上下文、Agent 职责和文件系统工作区，且用户可在 Desktop 中直接观察。备选方案是 Leader 在主任务中直接运行 `git worktree add` 后亲自实施，但它无法强制职责分离，也弱化了任务级监控和审批边界，因此不采用。

### Leader 只编排，执行 Agent 端到端实施

Leader 仅执行 Explore、`create_thread`/委派、Proposal Review、实施监控、Apply-final Review/验收，以及 PR merged 后从主项目发起清理。具体代码、测试、配置、文档和 OpenSpec 产物全部由执行 Agent完成。执行 Agent在两道门禁处提交材料并暂停，获得 Leader 明确批准后才继续。

该划分借鉴 Eino Supervisor“协调专业子 Agent”和 HITL“中断后显式恢复”的语义，但不采用 Eino Supervisor 的 full-context transfer。替代方案是允许 Leader 在执行 Agent受阻时直接修改文件；这会破坏责任归属和审计链，因此应改为补充委派指令或重新委派，而不是 Leader 接管实施。

### 固定扩展后的生命周期和批准主体

正式生命周期为：

`Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup`

`Leader Review` 对应 Proposal Review，`Leader Acceptance` 对应 Apply-final Review。执行 Agent不能自行批准；Leader 的批准必须在主任务与执行任务的协作记录中可识别。`Merge` 始终由用户控制，不因 Leader Acceptance 自动发生。

本 change 的 OpenSpec checkbox 在执行 Agent完成 Deliver 和 cleanup handoff 后闭环；`Merge` 与 `Cleanup` 仍是完整生命周期的规范性阶段，但属于 PR 交付后的 Leader operational state，由主 Codex 任务/线程状态持续跟踪。归档的 tasks 不因用户后续 merge 或 Leader cleanup 被重新打开、回写或要求在已合并分支上补充修改。用户 merge 后，Leader按正式 spec 的证明链执行 Cleanup 并在主任务报告结果；失败同样保留在主任务中处理和报告。

采用显式阶段名称便于策略测试稳定检查，也避免把“验证通过”误当成人工批准。替代方案是在现有阶段旁增加备注，但难以保护顺序和职责，故不采用。

### 执行 Agent负责 Deliver 前的全部 change 工作

执行 Agent按现有技能路由独立完成 Eino audit、Propose、TDD Apply、Validate，并在 Leader Acceptance 后完成 Sync、Archive、最终验证、选择性暂存、提交、推送和 PR。Leader只监控状态和审核证据，不代替执行 Agent运行实施或交付命令。

该决策保持单一执行责任链，避免 change 在多个 worktree 之间漂移。出现执行任务不可恢复时，应由 Leader明确终止或重新委派，并保留已产生状态，而不是静默换手。

### 合并后按证明链执行清理

Leader 仅在确认 PR merged 后，从主项目执行以下顺序：

1. 获取最新 `origin/main` 并识别 change branch、commit 与 Desktop worktree 的确切路径。
2. 检查 change worktree clean；存在 tracked 或 untracked 变更即停止并报告。
3. 验证 change 的必要提交已进入 `origin/main`；验证失败即停止并报告。
4. 删除已验证的 change worktree。
5. 删除本地 `codex/<change-name>` 分支。
6. 若远端 change 分支仍存在，则删除远端分支。
7. 执行 `git worktree prune`，并再次报告 worktree/branch 状态。

每一步仅在前一步成功后进行。任何失败都必须报告失败命令、错误和已完成步骤，不能宣称 cleanup 完成。该顺序优先保护未提交工作和未进入主线的提交。备选方案是 merge 后直接强制删除分支/worktree，存在数据丢失风险，因此禁止。

### 扩展现有文本策略测试

继续扩展 `internal/architecture/workflow_policy_test.go`，检查稳定标记而非完整段落：扩展生命周期字符串、`create_thread`、Desktop-managed worktree、Leader/执行 Agent职责、禁止自我批准、用户控制 merge、clean/`origin/main`/远端分支/prune 的清理规则。测试按 RED -> GREEN -> REFACTOR 进行，先证明当前文档缺少规则，再更新最小文档和配置。

不新增复杂 Markdown parser，因为当前策略文件结构简单，字符串策略测试已是仓库既有契约。若后续规则数量显著增长，再单独提出结构化 policy schema change。

## Risks / Trade-offs

- [Leader 与执行 Agent边界可能降低临时修复速度] → Leader通过补充委派、重新委派或明确终止任务处理阻塞，不直接接管实施，以保持审计链。
- [Codex Desktop 可能创建 detached HEAD] → 执行 Agent在写入前核验 worktree，并从最新 `origin/main` 安全建立 `codex/<change-name>`，绝不覆盖同名已有分支。
- [策略测试依赖 Markdown 稳定词汇] → 仅检查生命周期、命令名和安全前置条件等稳定标记，不比较整段文本。
- [误判 merged 或提交可达性导致数据丢失] → cleanup 同时要求 PR merged、worktree clean 和提交已进入 `origin/main`，任一条件失败即停止。
- [远端分支策略可能已自动删除分支] → 删除前先检查远端引用；不存在视为已满足，其他错误必须报告。
- [清理部分成功后重试状态复杂] → 每一步幂等核验当前资源状态，并报告已完成与待完成步骤。

## Migration Plan

1. RED：扩展 workflow policy Go 测试，使其因缺少新生命周期、职责、`create_thread` 和 cleanup 规则而失败。
2. GREEN：更新 `AGENTS.md`、`.agents/` 工作流/Git 文档和必要的 OpenSpec 配置，使聚焦测试通过。
3. REFACTOR：消除重复或歧义，运行聚焦测试、`go test ./...`、strict OpenSpec validation 与 diff/安全检查。
4. 在 Leader Acceptance 后同步 `engineering-change-workflow` delta spec、归档并由执行 Agent完成 Deliver。
5. 执行 Agent在 PR 创建后向 Leader交付 cleanup handoff；至此该 change 的 OpenSpec tasks 可完成。
6. 用户 merge PR 后，由 Leader从主项目按安全顺序完成 Cleanup，并将结果或失败保留在主 Codex 任务的 operational state 中，不回写已归档 tasks。

回滚时回退本 change 的文档、配置、策略测试和正式规格；不会影响运行时 Agent或生产数据。已经安全清理的 Git worktree/branch 不需要恢复，如清理未完成则按状态报告后人工处理。

## Open Questions

无。Codex Desktop API 的具体实现由产品提供，本 change 只规定仓库侧可观察职责和安全约束。
