# OpenSpec Workflow

OpenSpec 是正式工程 change 的唯一生命周期和 artifacts 来源。正式变更必须先创建 change，再实现代码；Skill 映射见 `.agents/skill-routing.md`，Git 交付见 `.agents/git-workflow.md`。

## Language And Artifact Rules

- OpenSpec 内容默认使用中文；仅固定标题、关键字、命令、路径、代码标识和协议字段保留英文。
- 主规格 `openspec/specs/` 是已生效能力的事实来源；新 change 必须基于主规格和现有代码增量设计。
- Proposal、design、delta specs、tasks 和归档历史都归 OpenSpec 所有，不得建立平行长期事实来源。

## Lifecycle

```text
Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver
```

阶段不得跳过或调换：

1. **Explore**：读取相关主规格、现有 change 和代码，确认当前状态、范围、非目标和复用点；只读探索不得推定实现或写入授权。
2. **Propose**：创建中文 `proposal.md`、`design.md`、delta specs 和 `tasks.md`，明确影响范围、风险和验证方式。
3. **Review**：用户人工确认全部 artifacts。未获明确批准不得进入 Apply；Skill 默认流程或自动化不得替代人工 Review。
4. **Apply**：读取当前 change 全部 artifacts、受影响主规格、相关代码和命中任务规则，严格按 tasks 顺序实施；每完成一项立即更新 checkbox。
5. **Validate**：运行 OpenSpec CLI 与任务相关验证，读取新鲜结果；失败或未验证项必须明确报告。
6. **Sync**：tasks 全部完成且第二次人工 Review 通过后，才可将 delta specs 同步到主规格。
7. **Archive**：Sync 后归档 change，运行 `openspec validate --all` 并提交 scoped archive checkpoint；archive 不等于 delivered。
8. **Deliver**：Archive commit 存在且工作区无当前 change 未提交文件后，才可按 `.agents/git-workflow.md` push、PR/merge 和 cleanup。完成 Deliver 前不得宣称 change 关闭，也不得启动依赖其产物或接续其工作的 sequential successor change；用户明确批准且无依赖、无共享写状态的 independent parallel change 可按 Git 门禁独立启动。

## Review And Stateful Operation Gates

- Propose 后必须停在人工 Review；批准前不得修改生产规则、源码、数据库或图谱。
- Apply 完成后必须提供 scoped diff 与验证证据并再次等待人工 Review；批准前不得 Sync、Archive 或 Deliver。
- 数据库 migration/apply、seed、业务写入、图谱关系写入、图谱投影重建或清理等有状态操作，必须先展示范围、顺序、预期影响和回滚边界，再取得用户明确批准。
- 分层数据或图谱工作必须逐层执行 `Review -> Write -> Rebuild -> Query`；上一层未验收不得进入下一层。
- 只读审计、设计批准、Apply 批准或某一层写入批准都不得被解释为其他有状态操作的授权。

## Starting Or Continuing A Change

开始新 change 前必须：

- 读取 `openspec/config.yaml`、受影响主规格、相关代码及命中的 `.agents` 规则。
- 确认 scope、non-goals、复用点和影响区域，禁止创建平行结构。
- 通过 `.agents/git-workflow.md` 的 New Change Gate；OpenSpec 不重复维护 branch/worktree 操作细节。

继续已有 change 时只读取：

- 该 change 的 proposal、design、tasks 和 delta specs。
- 受影响主规格和相关代码。
- 由实际生命周期、Git、后端、前端或测试动作命中的对应 `.agents` 文件。

不得因为开始任务而无差别读取全部规则、全部主规格或项目外文档。

## Apply Rules

- 实现前说明复用哪些已有页面、组件、services、models、data、store、repository 或配置。
- 不得在未检查现有实现前生成平行页面、service、model、store、data 或 config 层。
- Go 后端功能、bugfix 或重构必须服从 `.agents/testing-tdd.md`，先测试再实现。
- 若 design/spec/tasks 与代码现实冲突，立即暂停；先更新 artifacts 或征求用户确认，不得在过期设计上继续。
- 用户要求暂停时保留 change 和 tasks 状态，使其可恢复。

## Design Requirements

复杂后端 change 的 `design.md` 必须包含真实边界图示：

- 后端流程、跨模块调用、外部 API、scheduler、connector、事件抽取、图谱写入、异步任务或部署边界：Mermaid sequence diagram。
- 新增核心类型、接口、adapter、repository、service、parser、connector、worker 或跨包依赖：Mermaid class 或 component diagram。
- 简单配置、文案或小范围测试修复可以不画图，但 design 必须说明原因。

## Completion Gates

进入 Sync 前必须同时满足：

- tasks 全部完成并有对应验证证据。
- design、delta specs 和实现一致。
- 用户已完成 Apply 后人工 Review。

进入 Archive 前必须完成 Sync；进入 Deliver 前必须完成 Archive、`openspec validate --all` 和 archive commit。Git 提交、PR、merge、branch/worktree cleanup 的完整操作只在 `.agents/git-workflow.md` 维护。
