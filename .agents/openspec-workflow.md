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

## 风险分级、阶段 Review package 与条件式执行包

本节是 R0—R3、人工 gate、阶段 Review package、候选数据审阅、R2/R3 recovery evidence、条件式执行包和 active change adoption 的详细唯一事实来源。根 `AGENTS.md` 只保留不可绕过摘要；Git、测试与 Skill 文件只维护各自专责规则。

### 风险等级和人工 gate

- R0：文档、调研、只读审计。artifact checkpoint 运行 OpenSpec validate、scoped diff 和 secret 检查。
- R1：源码或测试变更但无有状态写入。除 R0 证据外运行 targeted tests，并在 Apply final 运行受影响交付边界的完整验证。
- R2：migration、seed、本地/UAT 数据变更。必须具备明确授权、只读 preflight、可验证 recovery evidence 和 before/after state assertions。
- R3：生产、不可逆 cleanup、Neo4j rebuild 或敏感部署。必须独立明确授权及备份/恢复或等价灾难恢复证据；R3 不得跨层批量执行，R3 cleanup 必须单独成包。

change 在 Proposal 声明基线风险；阶段或命名操作可以上调，混合 change 按当前操作的最高风险执行。普通 task checkbox 不自动成为人工 gate。任何人工 Review、Authorization 或 Acceptance gate 必须记录风险等级、风险理由、所需证据、通过后允许的下一步和明确不授权的操作。

### 阶段 Review package 与验证选择

同一风险边界内的 contract、实现、测试、dry-run、只读 preflight、diff、候选异常清单和验证结果可以组成一个阶段 Review package。package 必须记录 scope、non-goals、风险等级、证据、未验证项、阻断项和下一步授权边界。阶段级 checkpoint 对应一个内聚 package；不得把每个微型 task 自动升级为 commit、push 或人工 Review。

阶段 checkpoint 运行与范围匹配的 targeted tests。Apply final 必须运行受影响交付边界的完整验证：受影响 app/module/package 的完整 suite 与共享 architecture/contract tests。只有共享规则、跨模块契约、公共基础设施或 repo-wide 变更才运行 repo-wide full validation。验证记录必须写明受影响边界、共享 tests 和 repo-wide 判定理由；边界、理由或 suite 不清楚时 fail-closed，必须扩大到 repo-wide full validation 或停止等待澄清。任何失败、环境限制或未验证项必须进入 package，不能用旧日志替代。

task agent 在通知主对话验收前必须完成 self-review/code review，复读测试结果并检查 diff、scope、secret 和需求覆盖；发现阻断问题必须先整改并刷新证据。该内部审查不替代 Proposal 后或 Apply 后人工 Review。

### 候选数据审阅

候选数据 package 必须包含生成规则与输入指纹、总体 counts、确定性抽样、异常/冲突清单、宽边界清单、预期动作分类和 fail-closed 条件。正常项使用规则、抽样和总体断言审阅；高风险、宽边界、身份/来源/映射冲突、异常 disposition 与用户明确要求的清单必须逐项审阅。已批准规格要求逐项确认的 final manifest 不得被抽样策略降级。

### R2 条件式执行包

条件式执行包是独立授权对象，不由普通 Apply、旧批准或上一层批准推定。用户必须一次明确授权包内每个命名操作的环境、顺序、范围、排除范围、recovery evidence、预期 counts、before/after assertions 和停止条件。

每个 R2 层必须逐一选择：

- `backup`：shared local、开发主数据、UAT 或任何不可替代数据必须提供可恢复备份；
- `approved disposable recovery`：仅限用户逐层批准且明确声明 disposable、没有不可替代数据、具备确定性 recreate/reseed 路径的 local/test；必须记录环境身份、声明、重建/重灌命令、预计耗时和验证断言。

当前 tidewise 本地 curated PostgreSQL 不得自动视为 disposable。未逐层声明 recovery evidence、证据不成立、重建/验证失败、范围漂移、断言失败或触发停止条件时必须 fail-closed：不得执行或继续该层，未执行层的剩余授权自动失效，重新执行必须取得新授权。

执行顺序严格为 `Write(layer N) -> Query/assert(layer N)`；只有本包内已逐名明确授权的下一层，且当前层全部自动断言通过，才可继续。下一层因执行包中被显式命名而获授权，不是从上一层结果推定。R3 操作不得放入 R2 条件式执行包。

### Active change adoption

本规则 Deliver 后创建的新 change 默认使用新规则。active change 不自动重写、不追认历史操作、不取消已开始写操作的验收、不扩大既有授权。

每个 active change 采用新规则前，必须在其 branch 执行 `git fetch origin` 并更新到最新 `origin/main`，检查共享规则和 tasks 冲突，提交仅包含未来 gate 的 scoped workflow-adoption tasks diff，并经用户一次人工 Review。adoption 只能合并尚未开始的未来 gate；未命名的新环境、新层、新范围或 Neo4j rebuild 仍需独立授权。

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
