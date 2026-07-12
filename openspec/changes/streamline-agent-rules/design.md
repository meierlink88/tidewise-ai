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
| `.agents/git-workflow.md` | New Change Gate、Desktop worktree、commit、push/PR/merge、delivered cleanup | Explore/Apply 等阶段内容的重复解释，只保留 Git 所需检查点 |
| 其他 `.agents/*.md` | 领域专属边界和验证要求 | 跨领域生命周期重复，改为引用 |

### 3. 根文件不可删除的硬规则

根文件必须继续直接声明：

- OpenSpec 是正式工程 change 的唯一生命周期与 artifacts 来源，且 `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver` 不得跳步。
- 未获人工 Review 不得进入 Apply；数据库写入、图谱关系层写入/重建等有状态操作必须按层展示并取得明确批准。
- 新 change 从最新 `origin/main` 使用匹配的 `codex/<change-name>` 分支；并行 change、未合并 PR 或 dirty worktree 使用隔离 worktree，并优先 Codex Desktop 原生任务/worktree。
- 完成顺序必须是验证、sync、archive、archive commit、Deliver；PR merge 后验证主分支、删除远端和本地 branch、只清理项目自有 worktree，并归档对应 Desktop 任务。
- PostgreSQL 是实体、事件和关系事实源；Neo4j 只是可重建投影。
- 不混入并行 change，不创建平行结构，不复制 `prototype` 代码，不提交或打印 secret，不表达直接投资建议。

### 4. 按任务路由替代全量读取

新任务先读 `AGENTS.md`。只有命中任务类型才读取对应文件：正式研发/Skill 选择读 `skill-routing`；生命周期读 `openspec-workflow`；Git 操作读 `git-workflow`；后端、TDD、前端分别读各自边界。已有 change 只读该 change artifacts、受影响主规格与相关代码，不再默认遍历全部 specs 或 `../doc/architecture.md`。

### 5. 以覆盖矩阵约束精简

Apply 必须建立规则覆盖矩阵，至少包含：规则主题、根入口是否可见、唯一详述文件、验证证据。矩阵不得以“语义相近”掩盖缺失；上述不可删除硬规则每项都必须有明确文本落点。

## Risks / Trade-offs

- [风险] 根文件过短导致 agent 未路由到细则 → [缓解] 根文件保留硬门，并用动作关键词明确触发文件。
- [风险] 去重时删除了唯一约束 → [缓解] 修改前后覆盖矩阵逐项对照，人工 Review 作为 Apply 前和完成后的双重门。
- [风险] 多个文件仍以不同措辞表达同一生命周期 → [缓解] 重复/冲突扫描关注阶段序列、完成条件、worktree、cleanup 和审批词组，非所有关键词出现都视为重复。
- [风险] 5–6 KB 与 90–110 行目标互相拉扯 → [缓解] 两项都作为目标窗口；硬规则覆盖优先于机械压缩，超窗必须在 Review 中解释。
- [风险] `.agents` 链接或未来新增路由失效 → [缓解] 检查所有反引号路径存在，路由表新增文件时同步验证。
- [取舍] 根文件仍会摘要少量生命周期和 Git 门 → 这是为降低未路由时的高影响风险，完整操作文本仍只在专责文件中维护。

## Migration Plan

1. 人工 Review 本 proposal、design、delta spec 和 tasks，明确同意后才进入 Apply。
2. 记录 `AGENTS.md` 与 `.agents/*.md` 精简前行数、字符数、字节数及硬规则落点。
3. 先调整专责 `.agents` 文件的唯一所有权和交叉引用，再精简根 `AGENTS.md`，避免短暂丢失规则。
4. 运行覆盖矩阵、重复/冲突、链接、尺寸与 `openspec validate streamline-agent-rules` 检查。
5. 人工 Review 精简后的 scoped diff；未通过则回退本 change 的文档修改，不影响运行时或数据。

## Open Questions

- Review 决策点 1：是否接受根文件直接保留完整八阶段名称，但阶段解释只留在 `.agents/openspec-workflow.md`？推荐接受。
- Review 决策点 2：是否将“Desktop 任务归档”纳入 `.agents/git-workflow.md` 的 delivered cleanup，根文件只保留硬门摘要？推荐接受。
- Review 决策点 3：若硬规则完整但最终字节数略高于 6 KB，是否允许以覆盖完整优先？推荐允许，但应保持 110 行以内或说明原因。
