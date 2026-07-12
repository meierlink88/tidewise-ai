## Context

项目当前通过 `AGENTS.md` 和 `.agents/*.md` 约束 OpenSpec、Git、TDD、前后端边界，并安装了 OpenSpec 1.5.0 repo-local skills、Superpowers 6.1.1 plugin 和 GitHub plugin。现有规则已经声明 OpenSpec 优先，但仍大量使用自定义流程语言重复 Skill 行为，且只覆盖部分 Superpowers skills。

Superpowers 默认要求在 `docs/superpowers/specs/` 和 `docs/superpowers/plans/` 生成长期 artifacts，并可能在测试完成后直接进入分支收尾；这与本项目以 OpenSpec change 为唯一正式工作单元、先 sync/archive 再 PR/merge 的规则存在冲突。OpenSpec 配置还使用了 1.5.0 不支持的 `rules.language`，导致 `openspec instructions` 持续输出警告。

## Goals / Non-Goals

**Goals:**

- 建立一个集中、可扫描的 Skill 路由表，让 Agent 优先调用已安装 Skill，而不是临时发明流程。
- 保持 OpenSpec 为 change 生命周期、正式设计、规格、任务状态和系统事实的唯一来源。
- 将 Superpowers 收敛为需求澄清、计划质量、TDD、调试、验证、Review、并行协作和分支收尾的执行纪律。
- 将 GitHub plugin 明确用于远端 push、PR、CI 和 review comments 操作。
- 删除重复或失效的规则描述，并消除 OpenSpec 1.5.0 配置警告。

**Non-Goals:**

- 不修改任何业务代码、数据库、API、前端页面或部署配置。
- 不修改第三方 OpenSpec、Superpowers 或 GitHub plugin 自带的 Skill 文件。
- 不要求所有任务机械调用全部 Skills；只在 Skill 触发条件匹配时调用。
- 不引入新的 workflow 框架、CI 产品或项目管理平台。

## Decisions

### Decision: 新增集中式 Skill 路由文件

新增 `.agents/skill-routing.md`，集中定义触发条件、Skill 组合、artifact 去向、禁止事项和优先级。`AGENTS.md` 只保留硬规则和路由索引，其余规则文件保留项目差异，不复制 Skill 的完整流程。

不把全部路由继续堆入 `AGENTS.md`，避免总纲膨胀并降低后续 Agent 的扫描准确率。

### Decision: OpenSpec 是唯一正式生命周期和 artifact 所有者

正式 change 使用以下映射：

| 阶段 | 主 Skill/机制 | Superpowers 辅助 |
|---|---|---|
| Explore | `openspec-explore` | `brainstorming` |
| Propose | `openspec-propose` | `writing-plans` 的范围与任务拆分纪律 |
| Review | 用户人工确认 | 必要时 `requesting-code-review` 只做技术审查 |
| Apply | `openspec-apply-change` | TDD、debug、执行计划或 Subagent |
| Validate | OpenSpec CLI +项目验证命令 | `verification-before-completion` |
| Sync | `openspec-sync-specs` | verification |
| Archive | `openspec-archive-change` | verification |

`brainstorming` 的确认结果必须写入 OpenSpec `design.md`；`writing-plans` 的任务结果必须写入 OpenSpec `tasks.md`。默认不得创建 `docs/superpowers/specs/` 或 `docs/superpowers/plans/`。只有用户明确要求额外计划文档时，才允许创建后者，且不得成为正式状态来源。

### Decision: 项目规则覆盖 Skill 默认流程，但不替代 Skill 方法

优先级固定为：

```text
用户当前明确指令
> AGENTS.md 与 .agents 项目规则
> 已批准 OpenSpec artifacts
> 已安装 Skills
> Agent 临时判断
```

项目规则只覆盖 artifact 路径、生命周期顺序、分支命名、安全边界和项目技术差异。TDD、调试、验证和 Review 的执行方法继续由对应 Skill 提供。

### Decision: Git 隔离和远端操作使用明确 Skill 路由

- 一个正式 OpenSpec change 对应一个 `codex/<change-name>` branch。
- `using-git-worktrees` 只在并行 change、长期隔离或多线程协作时使用，worktree 不能替代 branch。
- `finishing-a-development-branch` 只能在 tasks 完成、sync、archive 和 `validate --all` 后进入。
- push 和创建 PR 优先使用 `github:yeet`；CI 失败使用 `github:gh-fix-ci`；PR 意见处理使用 `github:gh-address-comments` 和 `receiving-code-review`。

### Decision: 并行 Agent 默认只处理独立边界

当存在两个以上不共享写状态的调研、测试、日志分析或独立模块任务时，可以使用 `dispatching-parallel-agents`。正式 OpenSpec artifacts、tasks checkbox、数据库写入和同一文件修改仍由主 Agent 统一执行。只有任务边界、文件所有权和合并策略明确时，才使用 `subagent-driven-development`。

### Decision: 修复 OpenSpec 配置兼容性和过期上下文

移除不受支持的 `rules.language`，将中文要求保留在 `context` 和项目 workflow 规则中。同步更新已失效的“暂不使用 Neo4j”等上下文，避免 OpenSpec skills 基于旧架构生成 artifacts。保留 `proposal/design/specs/tasks` 四个 OpenSpec 1.5.0 支持的规则键。

```mermaid
flowchart LR
    U[用户需求] --> E[openspec-explore]
    E --> B[superpowers:brainstorming]
    B --> P[openspec-propose]
    P --> R[用户 Review]
    R --> A[openspec-apply-change]
    A --> T[Superpowers TDD / Debug / Subagent]
    T --> V[verification-before-completion]
    V --> S[openspec-sync-specs]
    S --> AR[openspec-archive-change]
    AR --> F[finishing-development-branch]
    F --> G[GitHub plugin PR / Merge]
```

## Risks / Trade-offs

- [Risk] Skill 名称或插件版本升级后路由文件可能过期。 → 路由只引用稳定 Skill 名称，升级插件时必须复核可用 Skills。
- [Risk] 过度强制 Skill 会让简单问答变得繁琐。 → 只在 Skill 描述的触发条件匹配时调用，纯解释性问答不创建 change 或额外 artifacts。
- [Risk] 将 Superpowers 默认 artifacts 映射到 OpenSpec 可能偏离插件原始流程。 → Superpowers 明确允许用户和项目规则覆盖默认路径，且项目只调整产物位置与生命周期，不削弱其方法检查。
- [Risk] 精简规则时可能误删项目特有安全约束。 → 实施时逐文件建立保留清单，并用静态测试/搜索验证关键规则仍存在。

## Migration Plan

1. 新增 `.agents/skill-routing.md` 并在 `AGENTS.md` 注册为必读路由。
2. 精简 OpenSpec、Git 和 TDD 规则，将通用方法替换为 Skill 引用，保留项目差异。
3. 修复 `openspec/config.yaml` 的无效规则键和过期上下文。
4. 增加规则静态测试或验证脚本，检查关键 Skill 路由、artifact 唯一性和分支收尾顺序。
5. 运行规则测试、`openspec instructions`、`openspec validate --all` 和 Git diff 审查。

回滚时恢复原规则文件和 OpenSpec 配置即可，不涉及业务数据迁移。

## Open Questions

无。
