# Skill-Driven Development Workflow Specification

## Purpose

定义观潮家正式研发 change 如何组合 OpenSpec、Superpowers、GitHub plugin、Git branch/worktree 和项目规则完成全生命周期交付。
## Requirements
### Requirement: 生产小程序视觉事实源路由
系统 SHALL 在 `.agents/frontend-boundaries.md` 中规定生产小程序页面的 visual/interaction source 路由：由已批准 OpenSpec change 指定的 page-level canonical prototype 最终渲染拥有页面视觉裁决权，旧 `ganchaojia-design` skill 只保留为历史和基础 token/component 参考。

#### Scenario: change 指定 canonical 页面
- **WHEN** 已批准 OpenSpec design 为生产小程序页面指定固定 prototype 路径、版本指纹和视觉验收范围
- **THEN** Agent 必须以该页面最终渲染裁决页面效果，不能让旧 design skill 的冲突规则覆盖它

#### Scenario: 没有指定 canonical 页面
- **WHEN** 小程序设计 change 没有已批准的 page-level canonical source
- **THEN** Agent 可以读取旧 `ganchaojia-design` skill 作为历史和基础 token 参考，但不得把它自动宣称为当前生产页面事实源

#### Scenario: 使用原型作为生产输入
- **WHEN** Agent 将 canonical prototype 转译为 Taro/React
- **THEN** 原型目录必须保持只读，生产源码只能提炼必要 tokens/primitives/compositions 和经授权资产，不得复制 HTML、DOM/内联脚本、整套 design library 或 prototype 辅助资产

#### Scenario: 更新前端边界规则
- **WHEN** 本 change 进入 Apply
- **THEN** `.agents/frontend-boundaries.md` 必须记录上述 miniapp 路由，同时保留 admin 的 Minimal Dashboard 路由和其他既有前端边界

### Requirement: 正式研发必须由已安装 Skill 驱动
系统 SHALL 在任务与已安装 Skill 的触发条件匹配时优先调用该 Skill，不得使用临时自定义流程替代已有能力。

#### Scenario: 开始正式功能 change
- **WHEN** 用户提出需要设计和实现的正式工程变更
- **THEN** Agent 必须通过 OpenSpec Explore/Propose skills 建立 change，并按项目 Skill 路由进入后续阶段

#### Scenario: 纯解释性问答
- **WHEN** 用户只要求解释概念且不要求修改工程行为
- **THEN** Agent 不得机械创建 OpenSpec change 或无关长期 artifacts

### Requirement: OpenSpec 必须拥有唯一正式 artifacts
系统 MUST 将 proposal、design、requirements、tasks、当前主规格和归档历史保存在 OpenSpec 目录中，不得默认创建平行的 Superpowers 设计或计划事实来源。

#### Scenario: Brainstorming 完成设计
- **WHEN** `superpowers:brainstorming` 已获得用户设计确认
- **THEN** 设计结果必须进入 OpenSpec `design.md`，不得默认创建 `docs/superpowers/specs/`

#### Scenario: 复杂任务完成计划拆分
- **WHEN** 使用 `superpowers:writing-plans` 的任务拆分方法
- **THEN** 可执行计划必须进入 OpenSpec `tasks.md`，除非用户明确批准，否则不得创建 `docs/superpowers/plans/`

### Requirement: OpenSpec 生命周期必须映射到明确 Skills
系统 SHALL 使用 repo-local OpenSpec skills 驱动 Explore、Propose、Apply、Sync 和 Archive，并使用人工 Review 与 OpenSpec CLI 完成 Review 和 Validate。

#### Scenario: 执行 change 全生命周期
- **WHEN** 正式 change 从需求进入实现和交付
- **THEN** Agent 必须依次使用或遵循 `openspec-explore`、`openspec-propose`、人工 Review、`openspec-apply-change`、Validate、`openspec-sync-specs` 和 `openspec-archive-change`

### Requirement: 工程纪律必须使用对应 Superpowers Skills
系统 MUST 在功能实现、缺陷、完成验证和代码审查场景中调用对应 Superpowers skills，并将结果更新到当前 OpenSpec change。

#### Scenario: 实现功能或修复缺陷
- **WHEN** Apply 阶段涉及功能、bugfix、重构或可观察行为变更
- **THEN** Agent 必须使用 `test-driven-development`，遇到异常时必须先使用 `systematic-debugging`

#### Scenario: 声明完成或执行 Git 交付
- **WHEN** Agent 准备声明完成、commit、push、创建 PR、sync 或 archive
- **THEN** Agent 必须使用 `verification-before-completion` 并读取新鲜验证结果

#### Scenario: 收到代码审查意见
- **WHEN** 用户或 GitHub 提供 review feedback
- **THEN** Agent 必须使用 `receiving-code-review` 验证意见后再修改，不得盲目接受

### Requirement: Git 隔离和收尾必须服从 change 边界
系统 MUST 为正式 OpenSpec change 使用独立 `codex/<change-name>` branch；在 Codex Desktop 可用时，所有新 change 必须通过 Desktop 新任务创建独立受管 worktree，并按 sequential successor 或 explicitly approved independent parallel change 的关系门禁执行。

#### Scenario: 从最新主分支开始 change
- **WHEN** Agent 准备创建新的正式 OpenSpec change
- **THEN** Agent 必须先更新远端引用，并在 Desktop-managed worktree 中基于最新 `origin/main` 创建或切换匹配的 `codex/<change-name>` branch，保持工作区 clean 且 scoped

#### Scenario: 启动 sequential successor change
- **WHEN** 新 change 依赖当前 change 的产物或接续同一产品、数据或交付链工作
- **THEN** Agent 必须等待前序 change 完成 archive commit、Deliver 和隔离清理后再启动，不得用新 worktree 绕过依赖顺序

#### Scenario: 启动 explicitly approved independent parallel change
- **WHEN** 另一 change 仍 active 且用户明确批准启动无依赖的独立 parallel change
- **THEN** Agent 必须使用独立 Desktop-managed task/worktree 和 branch，记录文件、OpenSpec artifacts/tasks 与数据库状态边界，并确保不共享源码文件或写状态

#### Scenario: 并行边界发生变化
- **WHEN** 并行 change 执行中出现产物依赖、文件重叠或共享数据库写状态
- **THEN** Agent 必须立即暂停相关工作并重新排序，不得继续并行

#### Scenario: Desktop 机制不可用时使用 fallback
- **WHEN** Codex Desktop 受管机制不可用且用户明确批准 fallback
- **THEN** Agent 才可按 `.agents/git-workflow.md` 创建路径和所有权明确的 project-owned worktree

#### Scenario: 准备 PR 或 merge
- **WHEN** 实现、测试和 Apply 后人工 Review 已经完成
- **THEN** Agent 只有在 OpenSpec Sync、Archive、`openspec validate --all` 和 archive commit 完成后才能进入 PR 或 merge

### Requirement: 并行 Agent 必须限制共享写状态
系统 SHALL 只对两个以上无共享写状态、无顺序依赖的任务使用并行 Agent，并由主 Agent 统一维护正式 artifacts 和数据库写入。

#### Scenario: 并行只读分析
- **WHEN** 多个来源核验、测试分析或日志排查可以独立完成
- **THEN** Agent 可以使用 `dispatching-parallel-agents`，等待结果后由主 Agent 汇总

#### Scenario: 多 Agent 可能修改同一文件
- **WHEN** 候选子任务会修改同一个 OpenSpec artifact、tasks 文件或数据库状态
- **THEN** Agent 不得并行执行这些写任务

### Requirement: OpenSpec 配置必须兼容当前 schema
系统 MUST 只在 `openspec/config.yaml` 的 `rules` 中使用当前 spec-driven schema 支持的 artifact IDs，并保持工程上下文与已生效架构一致。

#### Scenario: 获取 artifact instructions
- **WHEN** Agent 运行 `openspec instructions <artifact> --change <name> --json`
- **THEN** 命令不得输出未知 artifact rule key 警告，且上下文不得声明与主规格相冲突的过期架构

### Requirement: Agent 规则必须采用分层单一事实来源
系统 MUST 将跨任务硬门保留在根 `AGENTS.md`，并将 OpenSpec 生命周期、Skill 映射、Git 交付和领域边界的完整规则分别维护在对应 `.agents` 专责文件中，不得在多个文件复制同一操作流程。

#### Scenario: Agent 查找生命周期规则
- **WHEN** Agent 需要确定 OpenSpec 阶段顺序或阶段门禁
- **THEN** 根 `AGENTS.md` 必须提供不可绕过的摘要和路由，`.agents/openspec-workflow.md` 必须是完整生命周期的唯一详述来源

#### Scenario: Agent 查找 Git 交付规则
- **WHEN** Agent 需要创建 branch/worktree、提交、推送、合并或清理已交付 change
- **THEN** `.agents/git-workflow.md` 必须提供唯一完整操作规则，其他规则文件只保留必要硬门或引用

### Requirement: Agent 必须按任务范围读取规则
系统 SHALL 要求 Agent 先读取根 `AGENTS.md`，再按照任务类型读取对应 `.agents` 规则、当前 change artifacts、受影响主规格和相关代码，不得要求所有任务无差别读取全部规则和全部规格。

#### Scenario: 只处理前端任务
- **WHEN** Agent 处理前端页面、组件或设计系统 change
- **THEN** Agent 必须读取前端与生命周期所需规则，但不得仅因开始新任务而被要求读取无关后端边界全文

#### Scenario: 处理已有 change
- **WHEN** Agent 继续一个已有 OpenSpec change
- **THEN** Agent 必须读取该 change artifacts、受影响主规格与相关代码，并依据实际 Git 或领域动作加载对应规则

### Requirement: 规则精简必须保留关键工程硬门
系统 MUST 在规则精简后继续明确约束 OpenSpec 唯一生命周期、Codex Desktop 受管任务/worktree 强制入口、change branch/worktree 隔离、人工 Review 与有状态操作审批、Sync/Archive/Deliver 顺序、按 worktree 所有权交付清理、数据事实源和通用安全边界。

#### Scenario: 未完成人工 Review
- **WHEN** change 的 proposal artifacts 尚未获得人工确认
- **THEN** Agent 不得进入 Apply，也不得以自动化或 Skill 默认流程替代该确认

#### Scenario: 执行数据库或图谱有状态操作
- **WHEN** Agent 准备写入数据库、写入图谱关系层或重建图谱投影
- **THEN** Agent 必须按层展示拟执行范围并取得明确批准，不得从只读审计推定写入授权

#### Scenario: 在 Codex Desktop 中启动新 change
- **WHEN** Agent 在 Codex Desktop 可用的环境中启动任何新 change
- **THEN** Agent 必须先通过 Desktop 新任务创建受管 worktree，再在该受管任务内创建或切换 `codex/<change-name>` branch，不得把手工创建 branch/worktree 作为等价默认路径

#### Scenario: 清理 Desktop-managed worktree
- **WHEN** 使用 Desktop-managed worktree 的 change 已完成 PR merge 且默认分支已验证包含最终 commit
- **THEN** Agent 必须依次删除远端 change branch、归档或关闭对应 Desktop 任务、等待并验证 Desktop 已释放托管 worktree，再删除仍存在的本地 change branch

#### Scenario: Desktop 尚未释放托管 worktree
- **WHEN** 对应 Desktop 任务已请求归档或关闭但托管 worktree 仍未释放
- **THEN** Agent 不得执行 `rm` 或 `git worktree remove`，不得声明 cleanup 完成，并必须记录待清理状态

#### Scenario: 清理 project-owned fallback worktree
- **WHEN** 使用用户批准 fallback 创建的项目自有 worktree 的 change 已完成 PR merge 且默认分支已验证包含最终 commit
- **THEN** Agent 必须依次删除远端 change branch、仅对所有权与路径均确认的项目自有 worktree 执行 `git worktree remove`，再删除本地 change branch

#### Scenario: 维护图数据边界
- **WHEN** Agent 设计或修改实体、事件或关系存储和图投影
- **THEN** 规则必须继续声明 PostgreSQL 是事实源、Neo4j 是可重建投影

#### Scenario: 处理生产资料与输出
- **WHEN** Agent 修改生产代码或生成投研与 AI 分析内容
- **THEN** Agent 不得直接复制 prototype 代码、提交或打印 secret，也不得把内容表达为直接投资建议

### Requirement: 规则精简必须可量化验证
系统 SHALL 在 Apply 中记录精简前后根 `AGENTS.md` 的行数、字符数与字节数，维护关键规则覆盖矩阵，并验证 Desktop 强制受管、sequential/parallel 分流、两类 cleanup 顺序、重复/冲突、文件链接和 OpenSpec artifacts。

#### Scenario: 精简结果进入人工 Review
- **WHEN** Agent 完成规则正文修改并准备请求人工验收
- **THEN** Agent 必须提供前后行数、字符数与字节数、压缩率、覆盖矩阵、重复/冲突扫描、链接检查、scoped diff 和 OpenSpec validate 的新鲜结果

### Requirement: Worktree Skill 路由必须保留 Desktop 与 fallback 条件语义
系统 MUST 将 Codex Desktop 新任务机制作为 Desktop 可用时创建受管 worktree 的唯一默认入口，并且仅在 Desktop 受管机制不可用且用户明确批准 fallback 时，才将 `superpowers:using-git-worktrees` 路由到 project-owned fallback worktree。

#### Scenario: Desktop 可用时启动 change
- **WHEN** Agent 在 Codex Desktop 可用的环境中启动正式 OpenSpec change
- **THEN** Skill 路由必须要求通过 Desktop 新任务机制创建受管 worktree，不得调用 `superpowers:using-git-worktrees` 将手工或 project-owned worktree 作为等价默认路径

#### Scenario: 使用 approved fallback
- **WHEN** Codex Desktop 受管机制不可用且用户已经明确批准 fallback
- **THEN** Skill 路由必须允许调用 `superpowers:using-git-worktrees`，并继续服从 `.agents/git-workflow.md` 对路径、所有权、branch 和 scoped changes 的约束

#### Scenario: fallback 条件不完整
- **WHEN** Desktop 受管机制仍可用或用户尚未明确批准 fallback
- **THEN** Agent 不得调用 `superpowers:using-git-worktrees` 创建 project-owned worktree

### Requirement: 工作流架构测试必须验证当前规则语义
系统 SHALL 通过自动化架构测试验证 OpenSpec 生命周期顺序与人工 Review、Desktop-managed 默认入口、approved fallback 双条件、sequential/parallel 分流、两类 cleanup 顺序以及 Archive 后进入 finishing branch，不得依赖已被当前规则替代的旧叙述性短语。

#### Scenario: 规则语义完整
- **WHEN** 开发者运行 `go test ./internal/architecture -count=1`
- **THEN** 测试必须验证当前工作流契约并通过，且不得要求恢复默认手工 worktree 路径

#### Scenario: fallback Skill 映射被删除
- **WHEN** `.agents/skill-routing.md` 不再包含 approved fallback 对 `superpowers:using-git-worktrees` 的条件映射
- **THEN** 架构测试必须失败并指出缺失的 worktree Skill 路由语义

#### Scenario: Desktop 强制受管或 fallback 双条件被削弱
- **WHEN** 规则允许在 Desktop 可用时绕过 Desktop 新任务，或在未同时满足 Desktop 不可用与用户明确批准时创建 project-owned fallback worktree
- **THEN** 架构测试必须失败

### Requirement: 正式研发必须使用统一风险等级
系统 MUST 为每个正式 change 声明 R0—R3 基线风险，并按具体阶段或命名操作上调：R0 为文档、调研和只读审计；R1 为源码或测试变更但无有状态写入；R2 为 migration、seed 或本地/UAT 数据变更；R3 为生产、不可逆清理、Neo4j rebuild 或敏感部署。混合风险 change MUST 按当前操作的最高适用等级执行 gate。

#### Scenario: 静态 migration 代码与实际 apply 风险不同
- **WHEN** Agent 只修改并验证 migration 代码而不执行 apply
- **THEN** 该阶段可以按 R1 管理，但实际在本地或 UAT apply migration 必须上调为 R2

#### Scenario: cleanup 属于高风险操作
- **WHEN** cleanup 被判定为不可逆或高破坏性
- **THEN** 系统必须将其标为 R3 并要求独立授权，不得与普通 schema 或 seed 写入合并

### Requirement: 普通任务不得自动成为人工门禁
系统 SHALL 将普通 task checkbox 作为 package 内可验证工作单元，而不是自动人工门禁。人工 gate MUST 在 Gate Map 标注风险等级、合法风险原因、所需证据、通过后允许的下一步和明确不授权的操作；一级 task MUST 表达内聚交付 package。

#### Scenario: 完成微型实现任务
- **WHEN** Agent 完成一个不跨越授权边界的微型 task
- **THEN** Agent 必须将其证据汇入所属 package，不得仅因 checkbox 完成而要求一次独立人工 Review

#### Scenario: 设置人工 gate
- **WHEN** 某一步需要用户人工 Review、Authorization 或 Acceptance
- **THEN** Gate Map 必须说明该 gate 所控制的合法风险原因与授权边界，不能只写通用“等待确认”

#### Scenario: 普通操作被单独包装为 gate
- **WHEN** tasks 将测试、修复、dry-run、validate、diff/secret check、commit 或 push 单独标为人工 gate
- **THEN** task-design lint 必须失败，Agent 必须把该操作合并回所属 package

### Requirement: 阶段 Review package 必须聚合一致风险边界内的证据
系统 SHALL 允许 contract、实现、测试、修复、dry-run、只读 preflight、validate、diff/secret check、commit/push 和异常清单组成同一风险边界的 package。package MUST 说明 scope、non-goals、风险等级、证据、未验证项、阻断项、停止条件和下一步授权边界，且不得绕过 Proposal 后或 Apply 后人工 Review。

#### Scenario: R1 阶段形成 Review package
- **WHEN** contract、实现和 targeted tests 已在无状态写入的同一阶段完成
- **THEN** Agent 必须用一个 package 提交验收，而不是为每个微型 task 分别 commit、push 或 Review

#### Scenario: package 内测试失败并修复
- **WHEN** package 内 targeted test、validate 或 lint 失败且修复不扩大风险与 scope
- **THEN** Agent 必须在同一 package 内修复并刷新证据，不得为失败与修复新增人工 gate

#### Scenario: package 涉及状态写入
- **WHEN** package 包含 R2 或 R3 有状态操作
- **THEN** 普通阶段 Review 不得隐含写入授权，Agent 必须另行提交满足对应风险等级的明确授权对象

### Requirement: 验证深度必须随风险与生命周期递增
系统 MUST 以 package 为单位组织验证：开发中运行与当前工作匹配的 targeted tests；package checkpoint 运行一次范围匹配验证；Apply final 运行一次受影响 app/module/package 的完整 suite 与共享 architecture/contract tests。只有共享规则、跨模块契约、公共基础设施或 repo-wide 变更才 MUST 运行 repo-wide full validation。验证选择 MUST 记录受影响交付边界、共享 tests 与 repo-wide 判定理由；边界、理由或 suite 不清楚时 MUST fail-closed。R2 与 R3 MUST 额外提供执行前后状态断言，任何失败或未验证项必须明确报告。

#### Scenario: 开发中运行 targeted tests
- **WHEN** Agent 在 package 内实现、调试或修复
- **THEN** Agent 可以按需重复 targeted tests，但不得把每次测试运行升级为 checkpoint 或人工 Review

#### Scenario: R1 package checkpoint
- **WHEN** Agent 准备提交无有状态写入的 package checkpoint
- **THEN** Agent 必须运行一次与整个 package 范围匹配的验证，不需要在每个微型 task 后重复完整验证

#### Scenario: Apply final 验证
- **WHEN** change 完成 Apply 并准备请求 Apply 后人工 Review
- **THEN** Agent 必须运行一次受影响交付边界完整 suite、共享 architecture/contract tests、OpenSpec strict validation、diff/scope/secret 检查，并汇总所有 R2/R3 pre/post evidence

#### Scenario: 共享规则 change 的 Apply final
- **WHEN** change 修改共享规则、跨模块契约、公共基础设施或其他 repo-wide 行为
- **THEN** Agent 必须运行 repo-wide full validation；本 workflow change 修改全项目规则与 architecture tests 时必须运行 `go test ./...` 和相关 OpenSpec/规则检查

#### Scenario: 验证边界无法明确
- **WHEN** Agent 无法明确受影响交付边界、完整 suite 或是否触发 repo-wide 条件
- **THEN** Agent 必须 fail-closed，扩大到 repo-wide full validation 或停止等待澄清，不得自行省略测试

#### Scenario: R2 操作断言失败
- **WHEN** R2 命名操作的 post-state、counts、保护或幂等断言任一失败
- **THEN** Agent 必须立即停止，不得继续后续层，也不得用旧验证证据替代失败结果

### Requirement: Task agent 必须先自审再通知主对话
系统 MUST 要求 task agent 在通知主对话验收前完成内部 self-review 或适用的 code review，复读测试结果并执行 diff、scope、secret 与需求覆盖检查；阻断问题 MUST 先自行整改并刷新验证。该自审不得替代用户的 Proposal 或 Apply 后人工 Review。

#### Scenario: 自审发现阻断问题
- **WHEN** task agent 的内部 review 发现规格遗漏、测试失败、越界 diff 或 secret 风险
- **THEN** task agent 必须先整改并重新验证，不能把已知阻断问题直接作为待验收结果交给主对话

#### Scenario: 自审只有非阻断风险
- **WHEN** 自审只发现无法在当前 scope 内消除的非阻断风险
- **THEN** Agent 必须在 Review package 中明确风险与未验证项，再通知主对话验收

### Requirement: 候选数据必须采用规则、抽样和异常聚焦审阅
系统 SHALL 要求规模化候选数据 package 提供生成规则与输入指纹、总体 counts、全量机器校验、异常/冲突清单、宽边界清单、低置信度清单、用户明确指定项和 fail-closed 条件。正常项不得被机械要求全部逐条审阅；异常、冲突、宽边界、低置信度及用户明确指定的清单 MUST 逐项人工审阅。

#### Scenario: 大量正常候选通过全量校验
- **WHEN** 候选由固定规则生成且全部正常项通过机器校验与总体断言
- **THEN** 用户可以审阅生成规则、counts 和例外清单，不需要机械逐条确认全部正常记录

#### Scenario: 候选存在异常或低置信度
- **WHEN** 候选包含异常、冲突、宽边界或低置信度项
- **THEN** 系统必须把这些项列入人工清单逐项审阅，未决项必须阻断后续写入

#### Scenario: 用户指定人工审阅项
- **WHEN** 用户明确指定某些候选或类别必须人工确认
- **THEN** 系统必须将其加入人工清单，不得因机器校验通过而跳过

#### Scenario: 业务契约要求 final manifest 逐项确认
- **WHEN** 已批准规格明确要求某个 final manifest 由用户逐项确认
- **THEN** 全量机器校验不得取消该人工决策，只能用于其余正常候选的证据组织

### Requirement: R2 条件式执行包必须逐层显式授权并声明 recovery evidence
系统 SHALL 允许用户在一次明确授权中预授权多个 Spec 已批准且精确匹配的 local-only R2 命名层，但执行包 MUST 逐层列出每个命名操作、环境、顺序、范围、排除范围、recovery evidence、预期 counts、before/after assertions 和停止条件。每层 MUST 严格执行 `preflight -> Write -> Query/assert`；只有当前层全部断言通过，才可自动进入包内已逐名授权的下一层。每层 recovery evidence MUST 明确选择可恢复备份或经批准的 disposable recovery。同一环境、同一维护窗口且基础状态未变化时，可以复验并引用同一 recovery baseline；复验不一致 MUST 立即停止。shared local、开发主数据、UAT 或任何不可替代数据 MUST 提供可恢复备份，当前 tidewise 本地 curated PostgreSQL MUST NOT 自动视为 disposable。普通 Apply 批准、旧批准或上一层批准 MUST NOT 被解释为该执行包授权。

#### Scenario: 用户一次授权多个 local-only R2 命名层
- **WHEN** 执行包逐名列出 Layer A 和 Layer B 的全部授权字段、每层 recovery evidence 且用户明确批准整个包
- **THEN** Agent 可以严格执行 `preflight A -> Write A -> Query/assert A -> preflight B -> Write B -> Query/assert B`

#### Scenario: 同一维护窗口复用 recovery baseline
- **WHEN** 下一命名层与上一层处于同一环境和维护窗口，且环境身份、scope、count/hash/schema 的复验与 recovery baseline 一致
- **THEN** Agent 可以引用已验证 baseline，不需要为该层重新制造完整 backup package

#### Scenario: recovery baseline 复验漂移
- **WHEN** 环境身份、scope、count/hash/schema 与已验证 recovery baseline 不一致
- **THEN** Agent 必须立即停止，未执行层授权失效，并重新建立 recovery evidence 后请求必要授权

#### Scenario: disposable local test 层使用重建证据
- **WHEN** 某 local/test 层被用户逐层批准为 disposable，且环境没有不可替代数据并提供确定性 recreate/reseed 命令、预计耗时和验证断言
- **THEN** Agent 可以以 approved disposable recovery 作为该层 recovery evidence，而不是物理备份

#### Scenario: shared 或不可替代数据层
- **WHEN** R2 层涉及 shared local、开发主数据、UAT 或任何不可替代数据
- **THEN** Agent 必须在该层执行前提供可恢复备份，不得使用 baseline 简化取消 recovery evidence 或使用 disposable recovery

#### Scenario: 上一层断言通过
- **WHEN** Layer A 的全部自动断言通过且 Layer B 已在同一执行包中被逐名明确授权
- **THEN** Agent 可以进入 Layer B，因为 Layer B 已被显式授权，而不是从 Layer A 的批准推定

#### Scenario: 任一断言、范围或 recovery evidence 失败
- **WHEN** 某层断言失败、实际范围漂移、recovery evidence 不成立、出现冲突或触发停止条件
- **THEN** Agent 必须立即停止，所有未执行层的剩余授权自动失效，重新执行必须取得新授权

#### Scenario: 执行包使用概括性后续范围
- **WHEN** 执行包只写“其余层”“后续数据”或其他未逐名范围
- **THEN** 这些未命名操作不在授权范围内，Agent 不得执行

### Requirement: R3 操作必须保持独立授权和恢复证据
系统 MUST 默认禁止 R3 跨层批量执行。生产、不可逆清理、Neo4j rebuild 和敏感部署 MUST 分别获得独立明确授权；被定义为 R3 的 cleanup MUST 单独成包。R3 MUST 提供备份/恢复或等价灾难恢复证据，且不得使用 disposable recovery 例外。

#### Scenario: PostgreSQL R2 层完成后准备 Neo4j rebuild
- **WHEN** PostgreSQL 各命名层已经通过 Query 验收
- **THEN** Agent 仍必须为 Neo4j rebuild 提交独立 R3 授权，不得从 PostgreSQL 执行包推定

#### Scenario: 生产操作与 UAT 操作相似
- **WHEN** 相同命令已在 UAT 作为 R2 获批并成功
- **THEN** 生产环境仍属于独立 R3 授权对象，UAT 授权不得扩展到生产

### Requirement: Commit 与 Review 必须采用阶段级 checkpoint
系统 SHALL 为内聚且可独立验证的阶段创建 commit/checkpoint，并禁止把每个微型 task 自动升级为 commit、push 或人工 Review。Proposal checkpoint、Apply 阶段 package、Apply final 与 Archive checkpoint仍必须服从现有 Git 和生命周期门禁。

#### Scenario: 同一阶段包含多个微型任务
- **WHEN** 多个 task 属于同一风险边界并共同形成可验证结果
- **THEN** Agent 必须在阶段完成后创建一个 scoped checkpoint，而不是为每个 checkbox 分别 commit 和 push

### Requirement: 新规则必须通过显式 adoption 应用于 active change
系统 MUST 让本规则 Deliver 后创建的新 change 默认使用新流程；active change MUST 保持历史 artifacts、任务包装和授权不变，并在 `.agents/openspec-task-lint-baseline.tsv` 中以 change name 和 reason 显式登记，直到其 branch 更新最新 `origin/main`、提交 scoped workflow-adoption tasks diff并通过一次用户人工 Review。adoption diff MUST 移除对应 baseline 行，使后续 active lint 生效。adoption 只能合并未来 gate，不能追认历史操作、取消已开始写操作的验收或扩大既有授权。

#### Scenario: Active change 尚未 adoption
- **WHEN** 本规则已经 Deliver 但某 active change 尚未提交并通过 adoption Review
- **THEN** 该 active change 必须继续按原 tasks 与授权边界执行，并只能通过 baseline 中可见的有效行被 active lint 跳过；explicit lint 仍可用于审阅

#### Scenario: Adoption 合并未来 R2 gates
- **WHEN** active change 的未开始 schema、seed 和 mapping 层被 scoped tasks diff 逐名组织为 R2 条件包并通过用户 Review
- **THEN** 这些未来层可以使用新条件式执行包，adoption diff 必须移除 baseline 行，但已开始的写操作仍按原验收完成

#### Scenario: Adoption 试图扩大旧授权
- **WHEN** adoption diff 把既有批准扩展到新环境、新层、新范围或 Neo4j rebuild
- **THEN** 用户必须拒绝该 adoption 范围，系统不得把旧批准解释为新授权

### Requirement: 风险工作流规则必须保持分层单一事实来源
系统 MUST 让根 `AGENTS.md` 只保留简短总原则与路由，并把风险等级、Review package、条件式执行包、候选审阅、自审和 adoption 的详细唯一规则维护在 `.agents/openspec-workflow.md`；Git checkpoint、测试验证和 Skill 路由只在各自专责文件维护，不得复制完整流程。

#### Scenario: Agent 查找条件式执行包规则
- **WHEN** Agent 需要确定 R2 多层执行包的授权与停止语义
- **THEN** `.agents/openspec-workflow.md` 必须提供完整唯一规则，根 `AGENTS.md` 和其他 `.agents` 文件只保留硬门摘要或引用

#### Scenario: Apply 验证规则分层
- **WHEN** 本 change 修改规则正文
- **THEN** 自动化架构测试必须验证 R0—R3、Proposal/Apply Review、R2 逐层显式授权、R3 独立授权、阶段 checkpoint 和 active adoption 边界

### Requirement: Proposal 必须声明任务包装契约
系统 MUST 要求正式 change 的 Proposal 与 tasks 开头提供可机器读取且内容一致的 Gate Map 与 Complexity Budget。Proposal 的 `## Gate Map` MUST 是 `## Why` 后第一个二级 heading，tasks 的 `## Gate Map` MUST 是第一个二级 heading；其固定列 MUST 依次为 `Package`、`Gate`、`Risk`、`Human`、`Reason Code`、`Allowed Scope`。`Package` MUST 是与 `## <Package>. <name> Package` 一级 heading 一一对应的不带前导零正整数；`Risk` MUST 精确为 `R0`、`R1`、`R2` 或 `R3`；`Human` MUST 是小写 `yes` 或 `no`；`Human=yes` 的 Reason Code MUST 精确为 `SPEC_SEMANTICS`、`R3_OPERATION`、`NEO4J`、`SHARED_ENV`、`DEPLOYMENT_SECURITY`、`DRIFT_RECOVERY`、`APPLY_FINAL` 或 `GIT_COMPLETION`，`Human=no` MUST 使用 `NONE`。linter MUST 通过 Package ID 关联 Gate Map 与一级 package，不得从自然语言猜测。

Complexity Budget MUST 紧跟 Gate Map，固定为 `Key`、`Value` 两列并依次包含 `human_gates`、`stateful_layers`、`checkpoints`、`full_test_runs`、`continuous_automation_scope`。前四个值 MUST 是允许零值的无符号十进制整数；连续执行范围 MUST 使用 `packages:<selector>`，selector 只能引用存在且 `Human=no` 的升序无重复 package ID/闭区间，空范围为 `packages:none`。Proposal 与 tasks 的两张机器表 MUST 经空白规范化后逐行相同。

#### Scenario: 创建新的 R0 或 R1 change
- **WHEN** Agent 为无有状态写入的正式 change 编写 Proposal 与 tasks
- **THEN** artifacts 必须按固定 heading、表头、枚举、整数和 selector schema 提供一致的 Gate Map 与 Complexity Budget，并把实现、测试、修复、验证和 Git 操作组织为对应 package 子项

#### Scenario: Gate Map 与一级 package 对应
- **WHEN** tasks 包含编号为 1、2、3 的三个一级 package
- **THEN** Gate Map 必须按相同顺序各包含唯一 Package 1、2、3，且 linter 必须对缺失、重复、多余或顺序不一致 fail

#### Scenario: Complexity Budget 可机器解析
- **WHEN** Agent 声明零个 stateful layer、两个人工 gate 和 package 2 连续自动执行
- **THEN** 固定值必须分别为 `stateful_layers=0`、`human_gates=2`、`continuous_automation_scope=packages:2`，不得混入单位或说明文字

#### Scenario: 复杂度预算超过常见阈值
- **WHEN** Proposal 声明的人工 gate、checkpoint 或完整测试次数高于同风险 change 的建议阈值但字段完整且风险理由合法
- **THEN** lint 必须输出 warning 供 self-review，不得仅因启发式数量阻断 proposal

### Requirement: Stateful Layer Map 必须完整映射有状态层
系统 MUST 在 `stateful_layers>0` 时要求 Proposal 与 tasks 在 Complexity Budget 后、首个编号 package 前提供内容一致的 `## Stateful Layer Map`。固定列 MUST 依次为 `Layer`、`Package`、`Environment`、`Order`、`Scope`、`Exclusions`、`Recovery Evidence`、`Recovery Baseline`、`Expected Counts/Hash/Schema`、`Before Assertions`、`After Assertions`、`Stop Conditions`。`Layer` MUST 是唯一 kebab-case；`Package` MUST 引用 Risk 为 R2/R3 的 Gate Map package；`Environment` MUST 是 `local`、`shared-local`、`uat` 或 `prod`；`Order` MUST 在 package 内从 1 连续且唯一；空 exclusions MUST 写 `none`。Recovery Evidence MUST 是 `backup` 或 `approved-disposable-recovery`，Recovery Baseline MUST 是 `new:<kebab-id>` 或引用同环境更早 order 的 `reuse:<kebab-id>`；expected state MUST 同时提供 `counts=<value>;hash=<value>;schema=<value>`，不适用值写 `na`。其余范围与断言字段 MUST 单行非空。

`approved-disposable-recovery` MUST 保留对符合既有 disposable 条件的 local R2 layer 的支持。它也 MUST 仅在以下条件全部满足时允许 local R3 layer 使用：Scope 以具有 ASCII token 边界的 `Neo4j` 与 `cleanup`、`rebuild` 或 `sync` 中一个封闭 operation token 共同明确为 Neo4j projection operation，不要求额外出现字面量 `projection`；Layer 名称 MUST NOT 代替 Scope 证明业务范围；Before Assertions 以同样有边界的 `PG` 或 `PostgreSQL` 与 `baseline` 明确引用事实源 baseline；上述匹配 MUST 大小写不敏感且不得把 cleanupSuffix、resync 或 neo4jBackup 视为合法 token；PostgreSQL 是冻结、已验收且可完整重建该投影的唯一事实源；Gate 的 Risk 为 R3 且 Human 为 yes；Before/After Assertions、Stop Conditions 与 expected state 完整；用户在运行前对该命名 layer 单独明确授权。该 recovery 只表达恢复策略，不构成操作授权。`shared-local`、UAT、prod/shared、生产、非 Neo4j R3、非 projection、operation 不在封闭集合或无法从 PostgreSQL 完整重建的状态 MUST 拒绝 disposable recovery，并要求 `backup` 或等价正式灾备。Neo4j R3 的独立授权、逐层 `preflight -> Write -> Query/assert`、失败立即停止、未执行授权失效和禁止跨层批量 MUST 保持不变。本 requirement 不定义或引入 UAT Neo4j recovery、backup、deployment、adoption 或验收能力。

当 `stateful_layers=0` 时系统 MUST 允许省略 Stateful Layer Map；若仍提供该表，则 MUST 使用固定表头且不得包含数据行。linter MUST 校验数据行数等于预算值，并只通过 Package ID 关联 layer 与 package。

#### Scenario: 无有状态层省略 Stateful Layer Map
- **WHEN** Complexity Budget 声明 `stateful_layers=0`
- **THEN** Proposal 与 tasks 可以省略 `## Stateful Layer Map`，且 lint 不得要求空占位表

#### Scenario: 多层条件式执行包完整映射
- **WHEN** 一个 R2 package 包含两个已命名 local layer
- **THEN** Stateful Layer Map 必须有两行、引用同一 Package、使用连续 Order 1 和 2，并完整提供环境、范围、排除、recovery、expected state、前后断言和停止条件

#### Scenario: local Neo4j R3 disposable projection 合法表达
- **WHEN** 一个 local R3 layer 的 Scope 以有边界 token 明确包含 Neo4j 与 `cleanup`、`rebuild` 或 `sync`，Before Assertions 以有边界 token 引用已冻结验收且可完整重建投影的 PG/PostgreSQL baseline，Gate Human 为 yes，且全部字段与命名 layer 独立授权完整
- **THEN** task-design lint 必须允许 `approved-disposable-recovery`，但该结果不得被解释为已授权执行该 layer

#### Scenario: disposable R3 环境或状态越界
- **WHEN** `approved-disposable-recovery` 用于 shared-local、UAT、prod/shared、生产、非 Neo4j R3、非 projection、非 cleanup/rebuild/sync 或未引用 PG/PostgreSQL baseline 的 layer
- **THEN** task-design lint 必须 fail-closed，并要求 backup 或等价正式灾备

#### Scenario: local R2 disposable recovery 保持兼容
- **WHEN** 一个 local R2 layer 满足既有 approved disposable recovery 条件
- **THEN** task-design lint 必须继续允许该 recovery，不得因新增 Neo4j R3 例外改变原有行为

#### Scenario: Stateful layer 无法对应 package
- **WHEN** layer 引用不存在的 Package，或引用 Risk 为 R0/R1 的 package
- **THEN** task-design lint 必须 fail，不得根据 layer 名称或 scope 文案猜测归属

#### Scenario: 复用 recovery baseline
- **WHEN** 某行使用 `reuse:<baseline-id>`
- **THEN** 同一 Environment 的更早 Order 必须已有 `new:<baseline-id>`，before assertions 必须复验 identity、scope、count/hash/schema；否则 lint 必须 fail

#### Scenario: R3 recovery 不降低执行门禁
- **WHEN** 同一 package 声明多个获准表达 disposable recovery 的 local Neo4j R3 layer
- **THEN** 每个命名 layer 仍必须分别取得明确授权并逐层执行，任何失败或中止必须停止且不得把授权推定到下一层

### Requirement: 人工 gate 必须使用限定风险原因
系统 MUST 只允许以下人工 gate 原因：Spec/业务语义、R3、Neo4j、UAT/prod/shared、部署/secret/权限、scope/count/hash/schema 漂移或失败恢复、Apply-final、PR merge/cleanup。普通源码实现、测试/修复、dry-run、validate、diff/secret check、commit/push MUST NOT 单独成为人工 gate。

#### Scenario: 普通实现与验证属于同一 package
- **WHEN** Agent 在同一 R0/R1 风险边界内执行源码实现、测试、dry-run、validate、diff/secret check、commit 或 push
- **THEN** 这些步骤必须作为 package 内子项连续执行，不得分别请求人工 Review

#### Scenario: Gate Map 使用非法原因
- **WHEN** 人工 gate 的原因不是规范允许的语义、安全、环境、漂移恢复、Apply-final 或 Git 完成边界
- **THEN** task-design lint 必须失败并指出该 gate 的非法原因

### Requirement: Task-design lint 必须可靠且保持轻量
系统 SHALL 复用 `backend/internal/architecture/` 的 Go 标准库测试模式实现无新依赖的 task-design lint。active mode MUST 在 `OPENSPEC_TASK_LINT_CHANGE` 未设置时扫描 active changes；explicit mode MUST 在该变量设置为单段 kebab-case change name 时只检查指定 active change并忽略 baseline 跳过。Proposal/package checkpoint 的精确 explicit 命令 MUST 是从 `backend/` 运行 `OPENSPEC_TASK_LINT_CHANGE=<change-name> go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1`；现有 CI 的 `go test ./...` MUST 自动运行 active mode。

legacy baseline MUST 使用 repo-local `.agents/openspec-task-lint-baseline.tsv`，固定 UTF-8 TSV header 为 `change_name<TAB>reason`，只列规则 Deliver 时仍 active 的 kebab-case change name 与非空且不含 TAB/CR/LF 的单行 reason。其语义 MUST 归 `.agents/openspec-workflow.md` 所有，并只能由 scoped OpenSpec workflow/adoption change 维护；linter MUST NOT 自动改写。archive MUST 在 baseline 前自动排除；每个仍有效的 active skip MUST 输出包含 reason 与 adoption 移除提示的 warning；已归档、未知、重复或 adoption 后未移除的条目 MUST 至少 warning 且不得产生新的静默跳过能力；非法 header、空字段、非法 name 或多余列 MUST fail。

lint MUST 校验固定 Markdown schema、Gate Map/package 对应、Complexity Budget、Stateful Layer Map、合法人工 gate 与两份 artifacts 一致性；可静态可靠判断的违规 MUST fail，启发式复杂度 MUST 只 warning。fixture MUST 覆盖合规 zero-stateful、合规 multi-layer、schema/mapping 违规、baseline stale/duplicate/unknown/archived、explicit mode 与 warning。

#### Scenario: tasks 缺少确定性必填结构
- **WHEN** active 或显式传入 change 的 artifacts 缺固定 Gate Map/Complexity Budget schema、合法人工 gate 原因、package 对应，或 stateful package 缺固定 layer 字段
- **THEN** lint 必须失败并返回包含 artifact、section 与字段的可定位错误

#### Scenario: tasks 疑似过度拆分
- **WHEN** lint 发现重复微型 Review/checkpoint 或把测试、dry-run、commit/push 疑似提升为一级 package，但无法仅靠结构可靠判定违规
- **THEN** lint 必须输出 warning 和复核依据，不得使验证失败

#### Scenario: lint 运行作用域
- **WHEN** CI 运行 active mode 或开发者明确传入一个 change
- **THEN** active mode 必须排除 archive 并仅跳过 baseline 中仍 active 的 change，explicit mode 必须只校验指定 change且不得因 baseline 跳过

#### Scenario: baseline 包含过期或重复条目
- **WHEN** baseline 的 change 已归档、目录未知、重复，或已 adoption 但条目未移除
- **THEN** lint 必须输出 warning 并停止对无效条目提供跳过能力，不能无限静默忽略

#### Scenario: explicit mode 指向 archive 或未知 change
- **WHEN** `OPENSPEC_TASK_LINT_CHANGE` 包含 archive、路径分隔符或不存在的 change name
- **THEN** lint 必须 fail，不得扩大扫描范围或回退到 active mode

#### Scenario: lint 接入现有 CI
- **WHEN** 仓库现有 backend CI 运行 `go test ./...`
- **THEN** task-design lint 必须随工作流架构测试执行，不新增第三方依赖或平行 CI job
