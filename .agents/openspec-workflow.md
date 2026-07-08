# OpenSpec Workflow

本项目严格按照 OpenSpec 方法论执行工程开发。正式工程变更必须先创建 OpenSpec change，再实现代码。

## Language Rules

- 所有 OpenSpec 生成内容默认使用中文。
- 只有 OpenSpec 规范要求保留的框架性文案、固定标题、关键字、命令、文件名、路径、代码标识和协议字段可以保留英文。
- proposal、design、tasks、spec requirements 和 scenarios 的正文、说明、任务描述、风险、取舍、影响范围和验收内容都应使用中文。

## Standard Flow

```text
Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive
```

各阶段含义：

- Explore：讨论问题、架构、边界和取舍，不直接写实现代码。
- Propose：创建 `openspec/changes/<change-name>/`，生成 proposal、specs、design、tasks。
- Review：人工确认 artifacts 是否符合方向和范围。
- Apply：严格按 tasks 实现代码，并在完成后更新任务状态。
- Validate：运行可用验证命令，检查配置、类型、lint、测试或运行结果。
- Sync：将 delta specs 同步到 `openspec/specs/`，使其成为当前系统事实。
- Archive：完成后归档 change，保留历史决策与实现记录。

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

## Artifact Ownership

- OpenSpec artifacts 是正式工程事实来源。
- Superpowers 可以辅助澄清、TDD、debug 和验证，但不应默认产生与 OpenSpec 平行的长期设计事实。
- 如果用户要求暂停某个 change，不要删除 change；应保持 tasks 状态和 artifacts 可恢复。
