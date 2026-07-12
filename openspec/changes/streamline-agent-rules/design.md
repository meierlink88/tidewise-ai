## Context

根 `AGENTS.md` 当前 218 行、6,410 字符、11,308 字节。它既是入口，又重复描述 OpenSpec 生命周期、Git 隔离、前后端边界和安全细节；`.agents/skill-routing.md` 与 `.agents/openspec-workflow.md` 又各自完整列出相同生命周期，`.agents/git-workflow.md` 再以 18 步重复交付阶段。结果是新任务被要求一次性读取约 800 行规则，同一约束存在多个可被单独修改的副本。

本 change 只治理 agent 规则文档，不改变产品、运行时、数据库或 OpenSpec 工具行为。当前 proposal checkpoint 不修改规则正文；Apply 必须在人工 Review 后另行开始。

### 审计摘要

| 主题 | 当前重复或风险 | 目标单一事实来源 |
|---|---|---|
| 生命周期顺序 | 根文件、`skill-routing`、`openspec-workflow`、`git-workflow` 重复 | `.agents/openspec-workflow.md` 详述阶段与门禁 |
| Skill 映射 | 根文件概述，`skill-routing` 与 `openspec-workflow` 重复列举 | `.agents/skill-routing.md` 只维护阶段到 Skill 的映射 |
| branch/worktree/交付清理 | 根文件、`openspec-workflow`、`git-workflow` 重复 | `.agents/git-workflow.md` 详述 Git 操作和 cleanup |
| 任务上下文读取 | 根文件要求新任务读取全部规则 | 根文件路由表按任务触发对应规则 |
| 架构说明 | 根文件多次描述前端、后端和方向 | 根文件只保留不变量，领域细节下沉到边界规则与主规格 |
| 硬规则丢失 | 精简时易把审批、事实源、安全与清理门一并删除 | 覆盖矩阵逐条核验“入口摘要 + 唯一详述来源” |

## Goals / Non-Goals

**Goals:**

- 将根 `AGENTS.md` 压缩到约 90–110 行、5–6 KB，预计行数减少 50%–59%，字节数减少 47%–56%。
- 让根文件保留项目身份、workspace 边界、规则优先级、按任务路由、生命周期硬门、架构不变量和通用安全规则。
- 让每个可演化流程只有一个详述来源，其他文件仅保留必要入口或交叉引用。
- 将 Codex Desktop 受管任务/worktree 设为可用时的强制入口，手工 Git worktree 只作为经用户明确批准的不可用 fallback。
- 保留 Review、Apply、数据库写入和图谱分层操作的人工审批边界，不因“自动化流程”而推定授权。
- 用可重复扫描验证压缩量、覆盖、重复/冲突、链接和 OpenSpec 有效性。

**Non-Goals:**

- 本轮不进入 Apply，不修改 `AGENTS.md` 或 `.agents/*.md` 正文。
- 不改变现有 OpenSpec 生命周期、Skill 组合、Git 命名和清理义务。
- 不修改源码、主规格之外的产品行为、数据库、`doc` 或 `prototype`。
- 不触碰其他 active change 或其 worktree，不借机重写领域规则。

## Decisions

### 1. 采用“入口摘要 + 专责详述”三层结构

根文件只保留跨任务都必须可见的硬规则；`.agents` 文件按职责保存完整操作规则；OpenSpec 主规格保存已经生效的规范性能力。这样既避免纯索引导致关键门不可见，也避免根文件复制全部细节。

考虑过的替代方案：

- 只压缩措辞但保留所有章节：改动小，但生命周期和架构重复仍存在，无法解决漂移。
- 将根文件缩成纯路由索引：压缩率最高，但 agent 未读取细则时可能错过审批、安全和架构不变量。
- 推荐方案：根文件保留不可绕过的摘要，详述只存在于一个专责文件。

### 2. 明确文件所有权

| 文件 | 保留内容 | 下沉或删除内容 |
|---|---|---|
| `AGENTS.md` | 项目总纲、目录边界、优先级、路由、生命周期硬门、架构不变量、安全规则 | 完整阶段解释、Git 18 步、前后端目录细节、重复技术选型、全量读取清单 |
| `.agents/skill-routing.md` | 冲突优先级、阶段/场景到 Skill 的唯一映射、artifact 所有权 | 生命周期完成条件的重复长述，改为引用 workflow/git |
| `.agents/openspec-workflow.md` | 唯一完整生命周期、Review/Apply/Validate/Sync/Archive 门、审批边界 | Skill 说明与 Git cleanup 操作细节，改为引用专责文件 |
| `.agents/git-workflow.md` | New Change Gate、Desktop 受管 worktree 硬门、fallback 授权、commit、push/PR/merge、两类 delivered cleanup | Explore/Apply 等阶段内容的重复解释，只保留 Git 所需检查点 |
| 其他 `.agents/*.md` | 领域专属边界和验证要求 | 跨领域生命周期重复，改为引用 |

### 3. 根文件不可删除的硬规则

根文件必须继续直接声明：

- OpenSpec 是正式工程 change 的唯一生命周期与 artifacts 来源，且 `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver` 不得跳步。
- 未获人工 Review 不得进入 Apply；数据库写入、图谱关系层写入/重建等有状态操作必须按层展示并取得明确批准。
- 在 Codex Desktop 可用时，新 change 和并行 change 必须通过 Desktop 新任务创建受管 worktree，agent 不得手工执行 `git worktree add`；受管任务内再基于最新 `origin/main` 创建或切换匹配的 `codex/<change-name>` 分支。只有 Desktop 机制不可用且用户明确批准 fallback 时，才允许按 `.agents/git-workflow.md` 创建项目自有 worktree。
- 完成顺序必须是验证、sync、archive、archive commit、Deliver；根文件只摘要“按 worktree 所有权完成有序释放和 branch 清理，Desktop 未释放时不得宣称 cleanup 完成”，完整顺序唯一归属 `.agents/git-workflow.md`。
- PostgreSQL 是实体、事件和关系事实源；Neo4j 只是可重建投影。
- 不混入并行 change，不创建平行结构，不复制 `prototype` 代码，不提交或打印 secret，不表达直接投资建议。

### 4. 按任务路由替代全量读取

新任务先读 `AGENTS.md`。只有命中任务类型才读取对应文件：正式研发/Skill 选择读 `skill-routing`；生命周期读 `openspec-workflow`；Git 操作读 `git-workflow`；后端、TDD、前端分别读各自边界。已有 change 只读该 change artifacts、受影响主规格与相关代码，不再默认遍历全部 specs 或 `../doc/architecture.md`。

### 5. 以覆盖矩阵约束精简

Apply 必须建立规则覆盖矩阵，至少包含：规则主题、根入口是否可见、唯一详述文件、验证证据。矩阵不得以“语义相近”掩盖缺失；上述不可删除硬规则每项都必须有明确文本落点。

### 6. 按 worktree 所有权执行两条 cleanup 路径

`.agents/git-workflow.md` 必须唯一详述以下顺序：

1. Desktop-managed：PR merge 并验证 `origin/main` 包含最终 commit → 删除远端 change branch → 归档或关闭对应 Desktop 任务，由 Desktop 释放托管 worktree → 验证 worktree 已释放 → 删除仍存在的本地 change branch。
2. Project-owned fallback：PR merge 并验证 `origin/main` 包含最终 commit → 删除远端 change branch → 仅对所有权和路径均确认的项目自有 worktree 执行 `git worktree remove` → 删除本地 change branch。

Agent 不得对 Desktop-managed worktree 执行 `rm` 或 `git worktree remove`。若 Desktop 尚未释放托管 worktree，必须记录待清理状态，change 不得声明 cleanup 完成或 delivered。

## Risks / Trade-offs

- [风险] 根文件过短导致 agent 未路由到细则 → [缓解] 根文件保留硬门，并用动作关键词明确触发文件。
- [风险] 去重时删除了唯一约束 → [缓解] 修改前后覆盖矩阵逐项对照，人工 Review 作为 Apply 前和完成后的双重门。
- [风险] 多个文件仍以不同措辞表达同一生命周期 → [缓解] 重复/冲突扫描关注阶段序列、完成条件、worktree、cleanup 和审批词组，非所有关键词出现都视为重复。
- [风险] 5–6 KB 与 90–110 行目标互相拉扯 → [缓解] 两项都作为目标窗口；硬规则覆盖优先于机械压缩，超窗必须在 Review 中解释。
- [风险] `.agents` 链接或未来新增路由失效 → [缓解] 检查所有反引号路径存在，路由表新增文件时同步验证。
- [风险] agent 把 Desktop “优先”误读为可以自行手工 fallback → [缓解] 使用 MUST/不得措辞，并要求 fallback 同时满足机制不可用和用户明确批准。
- [风险] 本地 branch 仍被 Desktop worktree 占用时提前删除 → [缓解] Desktop cleanup 必须先归档任务并验证释放，再删除本地 branch；未释放时记录待清理状态。
- [风险] agent 误删 Desktop 托管目录 → [缓解] 明令禁止对 Desktop-managed worktree 执行 `rm` 或 `git worktree remove`。
- [取舍] 根文件仍会摘要少量生命周期和 Git 门 → 这是为降低未路由时的高影响风险，完整操作文本仍只在专责文件中维护。

## Migration Plan

1. 人工 Review 本 proposal、design、delta spec 和 tasks，明确同意后才进入 Apply。
2. 记录 `AGENTS.md` 与 `.agents/*.md` 精简前行数、字符数、字节数及硬规则落点。
3. 先调整专责 `.agents` 文件的唯一所有权和交叉引用，再精简根 `AGENTS.md`，避免短暂丢失规则。
4. 运行覆盖矩阵、重复/冲突、链接、尺寸与 `openspec validate streamline-agent-rules` 检查。
5. 人工 Review 精简后的 scoped diff；未通过则回退本 change 的文档修改，不影响运行时或数据。

## Review Decisions

- 根文件保留完整八阶段名称，阶段解释只留在 `.agents/openspec-workflow.md`。
- 根文件保留 Desktop 强制受管与按所有权清理的硬门摘要，两条 cleanup 完整顺序只留在 `.agents/git-workflow.md`。
- 硬规则完整性优先于机械字节目标；若超出 6 KB，必须保持约 90–110 行或在 Review 中说明原因。

## Apply Evidence

### 精简前基线

| 文件 | 行数 | 字符数 | 字节数 |
|---|---:|---:|---:|
| `AGENTS.md` | 218 | 6,410 | 11,308 |
| `.agents/backend-boundaries.md` | 216 | 5,319 | 8,091 |
| `.agents/frontend-boundaries.md` | 51 | 1,725 | 2,877 |
| `.agents/git-workflow.md` | 102 | 3,817 | 5,953 |
| `.agents/openspec-workflow.md` | 118 | 3,668 | 6,002 |
| `.agents/skill-routing.md` | 64 | 2,797 | 4,145 |
| `.agents/testing-tdd.md` | 38 | 1,352 | 2,458 |

基线分支为 `codex/streamline-agent-rules`，Apply 起点 commit 为 `e494d70`，起始工作区干净；其他 active change 均位于独立 worktree。

### 关键规则覆盖矩阵

| 规则主题 | 根入口必须直接可见 | 唯一完整来源 | Apply 验证 |
|---|---|---|---|
| OpenSpec 唯一八阶段生命周期 | 是 | `.agents/openspec-workflow.md` | 八阶段序列与人工 Review 门扫描 |
| Review、Apply、数据库/图谱分层审批 | 是 | `.agents/openspec-workflow.md` | Review 与有状态写入批准文本扫描 |
| Skill 映射和 artifact 所有权 | 路由可见 | `.agents/skill-routing.md` | 阶段到 Skill 表与平行 artifact 禁止项扫描 |
| Desktop 新任务受管 worktree | 是 | `.agents/git-workflow.md` | “必须”“不得手工 git worktree add”扫描 |
| fallback 授权与新 change 隔离 | 是 | `.agents/git-workflow.md` | “Desktop 不可用 + 用户明确批准”与 `codex/<change-name>` 扫描 |
| sequential successor 与 independent parallel 分流 | 硬门摘要 | `.agents/git-workflow.md` | 公共条件、顺序依赖、并行批准/所有权/写状态边界扫描 |
| Desktop-managed cleanup | 硬门摘要 | `.agents/git-workflow.md` | main → 远端 branch → Desktop 释放 → 本地 branch 顺序扫描 |
| project-owned fallback cleanup | 硬门摘要 | `.agents/git-workflow.md` | main → 远端 branch → 项目 worktree → 本地 branch 顺序扫描 |
| PostgreSQL/Neo4j 数据边界 | 是 | `AGENTS.md` 与相关主规格 | 事实源与可重建投影文本扫描 |
| 不混 change/不建平行结构 | 是 | `AGENTS.md` | 隔离与反重复文本扫描 |
| prototype/doc 边界 | 是 | `AGENTS.md`、`.agents/frontend-boundaries.md` | 路径和禁止复制文本扫描 |
| secret 与投资建议边界 | 是 | `AGENTS.md` | 禁止提交/打印与决策辅助文本扫描 |
| 后端/TDD/前端领域标准 | 按任务路由可见 | 对应领域 `.agents` 文件 | 文件 diff 与领域关键文本对照 |

### 精简后尺寸

| 文件 | 精简前 | 精简后 | 压缩率 |
|---|---:|---:|---:|
| `AGENTS.md` 行数 | 218 | 90 | 58.7% |
| `AGENTS.md` 字符数 | 6,410 | 3,477 | 45.8% |
| `AGENTS.md` 字节数 | 11,308 | 6,045 | 46.5% |

根文件同时满足约 90–110 行与 5–6 KB 目标，无超窗。覆盖矩阵各项均已在根入口或对应唯一完整来源找到明确文本落点；最终扫描结果在 tasks 4.x 验证后补充。

### 规则与链接扫描

- 生命周期完整序列只在 `.agents/openspec-workflow.md` 详述；根 `AGENTS.md` 仅保留硬门摘要。
- 生命周期到 Skill 的完整映射只在 `.agents/skill-routing.md`；Desktop-managed 与 project-owned fallback cleanup 标题及完整步骤只在 `.agents/git-workflow.md`。
- 未发现“Desktop 优先”“无法使用时自行回退”等弱化措辞；强规则同时覆盖 Desktop 新任务、禁止手工 `git worktree add`、双条件 fallback 和未释放待清理状态。
- New Change Gate 已拆为全部 change 的公共基线、sequential successor 和 explicitly approved independent parallel 两条路径；顺序后继受前序 Deliver 约束，独立并行需用户批准、独立 Desktop worktree、无依赖和无共享写状态，边界变化时必须暂停重排。
- 根文件与专责规则中不存在 `Useful Context Files` 标题或正向全量读取要求，仅保留“不得无差别读取”的否定约束。
- 根路由、OpenSpec artifacts、主规格目录、前后端目录与 Minimal Dashboard skill 等 repo 内引用均存在。
- scoped diff 仅包含 `AGENTS.md`、三个专责 `.agents` 文件及本 change 的 `design.md`、`tasks.md`；未触碰其他 active change/worktree。
- `openspec validate streamline-agent-rules`、`git diff --check`、根文件尺寸窗口与 scoped diff 检查均通过；Apply checkpoint 将只提交上述六个文件并停在第二次人工 Review。
