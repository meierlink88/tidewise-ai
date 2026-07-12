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
