# OpenSpec Workflow

本项目严格按照 OpenSpec 方法论执行工程开发。正式工程变更必须先创建 OpenSpec change，再实现代码。

## Language Rules

- 所有 OpenSpec 生成内容默认使用中文。
- 只有 OpenSpec 规范要求保留的框架性文案、固定标题、关键字、命令、文件名、路径、代码标识和协议字段可以保留英文。
- proposal、design、tasks、spec requirements 和 scenarios 的正文、说明、任务描述、风险、取舍、影响范围和验收内容都应使用中文。

## Standard Flow

```text
Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver
```

生命周期必须由 repo-local OpenSpec Skills 驱动：

- Explore：`openspec-explore`，需要创意澄清时结合 `superpowers:brainstorming`。
- Propose：`openspec-propose`，生成 proposal、specs、design、tasks。
- Review：用户人工确认 artifacts 是否符合方向和范围。
- Apply：`openspec-apply-change`，严格按 tasks 实现并即时更新状态。
- Validate：运行 OpenSpec CLI 和项目验证命令，并遵守 `superpowers:verification-before-completion`。
- Sync：`openspec-sync-specs`，将 delta specs 同步为当前系统事实。
- Archive：`openspec-archive-change`，完成后归档历史决策与实现记录。
- Deliver：按照 `.agents/git-workflow.md` 完成 archive commit、push、PR/merge 和已合并 change 的 branch/worktree cleanup；只有 Deliver 完成后 change 才可视为关闭。

详细 Skill 组合和 artifact 归属见 `.agents/skill-routing.md`。

## Directory Model

```text
openspec/
├── config.yaml   # 项目上下文、技术约束和 artifact 规则
├── specs/        # 当前系统已经生效的主规格
└── changes/      # 正在设计或实现的变更
```

每个 change 通常包含：

```text
openspec/changes/<change-name>/
├── proposal.md   # 为什么做、做什么、不做什么、影响范围
├── design.md     # 怎么做、技术选型、架构边界、风险取舍
├── tasks.md      # 可执行实现清单
└── specs/        # 本次变更的 delta requirements
```

主规格 `openspec/specs/` 是系统当前行为和能力的事实来源。新 change 必须基于主规格和现有代码增量设计。

## Before Starting A Change

开始任何新 change 前，必须：

- 读取 `openspec/config.yaml`。
- 读取相关主规格 `openspec/specs/**/spec.md`。
- 检查相关已有代码目录和文件。
- 总结当前系统状态，再提出增量方案。
- 优先复用和扩展已有模块，不要创建平行结构。
- 明确本次 change 的 scope、non-goals 和 impact。

## Before Applying A Change

实现任何 change 前，必须：

- 读取该 change 的 `proposal.md`、`design.md`、`tasks.md` 和相关 `specs/**/spec.md`。
- 读取受影响的现有代码文件。
- 说明将复用哪些已有页面、组件、services、models、data、store 或配置。
- 如果 change 涉及后端功能实现，必须先设计并实现 Go 单元测试或可自动化测试用例，再编写生产代码。
- 严格按照 tasks 顺序执行。
- 完成一个任务后立即把对应 checkbox 从 `- [ ]` 改为 `- [x]`。

## Design Diagram Rules

复杂后端 change 的 `design.md` 必须包含架构图示：

- 涉及后端流程、跨模块调用、外部 API、scheduler、connector、事件抽取、图谱写入、异步任务或部署边界时，必须包含 Mermaid sequence diagram。
- 涉及新增核心类型、接口、adapter、repository、service、parser、connector、worker 或跨包依赖关系时，必须包含 Mermaid class diagram 或 component diagram。
- 图中的节点名称必须尽量使用真实包名、类型名、接口名、connector key、parser key 或数据表名，不得只画抽象概念。
- 简单配置、文案、小范围测试修复或不涉及结构变化的小修可以不强制补图，但 design 中应说明原因。

## When Design And Code Diverge

实现过程中如果发现设计不匹配，必须：

- 暂停继续编码。
- 说明 design/spec/tasks 与现实代码的冲突。
- 先更新 OpenSpec artifacts 或征求用户确认。
- 不得在 artifacts 过期时继续盲目实现。

## Completing A Change

完成 change 后，必须：

- 运行适当验证。
- 确认 tasks 全部完成。
- 同步 delta specs 到 `openspec/specs/`。
- 归档 change 到 `openspec/changes/archive/`。
- 运行 `openspec validate --all`。
- 检查 `git status --short`，只暂存当前 change 的源码、测试、主规格和 archive 文件。
- 提交 `spec: archive <change-name>` 检查点，并按 `.agents/git-workflow.md` 完成 push、PR/merge 和 branch/worktree cleanup。
- 在 archive commit 存在且当前 change 不再有未提交文件前，不得声明 change 已关闭，也不得开始下一个 change。

## Starting The Next Change

开始新 change 前必须完成交付隔离检查：

- 上一个 change 必须已经完成 archive commit；工作区不得残留上一个 change 的未提交文件。
- 执行 `git fetch origin`，新 change 必须从最新 `origin/main` 创建 `codex/<change-name>` 分支。
- 当前 branch 名称必须与新 change 匹配；禁止在上一个 change 的 branch 中创建或实现新 change。
- 如果当前 worktree 不干净、已有其他 active change，或需要保留上一分支用于 PR review，必须为新 change 创建独立 worktree 并切换到对应 branch。
- 在 Codex Desktop 中优先使用新任务绑定的原生 worktree；无法使用时再按 `.agents/git-workflow.md` 创建 Git worktree。

## Artifact Ownership

- OpenSpec artifacts 是唯一正式工程事实来源。
- brainstorming 结论进入 `design.md`，writing-plans 结果进入 `tasks.md`；默认不得生成平行的 Superpowers 长期 artifacts。
- 如果用户要求暂停某个 change，不要删除 change；应保持 tasks 状态和 artifacts 可恢复。
