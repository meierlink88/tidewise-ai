## 1. Review 与基线

- [ ] 1.1 取得用户对 `proposal.md`、`design.md`、delta spec 和本 tasks 的明确 Review 批准；批准前不得修改 `AGENTS.md` 或 `.agents/*.md`。
- [ ] 1.2 在当前 `codex/streamline-agent-rules` 独立分支/worktree 记录 `AGENTS.md` 与全部 `.agents/*.md` 的精简前行数、字符数、字节数、Git 状态和相关规则落点；根文件基线为 218 行、6,410 字符、11,308 字节，并确认 scoped diff 不含其他 active change。
- [ ] 1.3 建立关键规则覆盖矩阵，至少覆盖 OpenSpec 唯一生命周期、Desktop 可用时必须由新任务创建受管 worktree、禁止 agent 手工 `git worktree add`、仅经用户批准的不可用 fallback、新 change 隔离、Review/Apply/数据库写入/图谱分层审批、sync/archive/deliver、两类 cleanup 顺序与待清理状态、PostgreSQL/Neo4j、不混 change、prototype、secret 和投资建议边界。

## 2. 规则所有权重构

- [ ] 2.1 精简 `.agents/skill-routing.md`，只保留规则优先级、阶段/场景到 Skill 的唯一映射和 artifact 所有权，并将生命周期及 Git 细节链接到专责文件。
- [ ] 2.2 精简 `.agents/openspec-workflow.md`，使其成为 `Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver`、人工 Review 与有状态操作审批、sync/archive 顺序的唯一完整来源。
- [ ] 2.3 精简 `.agents/git-workflow.md`，使其成为最新 `origin/main`、受管任务内创建/切换 `codex/<change-name>`、Desktop 可用时禁止手工 worktree、用户批准 fallback、scoped commit、PR/merge 和两类 cleanup 完整顺序的唯一来源；明确禁止手工移除 Desktop-managed worktree，并规定 Desktop 未释放时记录待清理状态。
- [ ] 2.4 检查其余 `.agents/*.md`，仅删除跨领域重复并保留后端、TDD、前端的专属边界；不得借机改变业务架构或测试标准。

## 3. 根规则精简

- [ ] 3.1 将 `AGENTS.md` 重构为项目总纲、workspace 边界、规则优先级与按任务路由、生命周期硬门、架构不变量和通用安全规则，删除 `Useful Context Files` 的全量读取模式。
- [ ] 3.2 将根文件控制在约 90–110 行、5–6 KB；记录精简后行数、字符数与字节数，并计算相对 218 行、6,410 字符、11,308 字节基线的行数、字符数和字节压缩率，任何超窗都必须保留原因供 Review。
- [ ] 3.3 对照覆盖矩阵逐项标记根入口摘要、唯一详述文件和验证证据，确认所有不可删除硬规则仍有明确文本落点。

## 4. 自动与人工验证

- [ ] 4.1 扫描 `AGENTS.md` 与 `.agents/*.md` 的生命周期序列、完成条件、审批、worktree 和 cleanup 重复/冲突，区分必要入口摘要与应删除的流程副本，并记录剩余例外。
- [ ] 4.2 验证根硬门与 `.agents/git-workflow.md` 明确使用强制措辞：Desktop 可用时必须由新任务创建受管 worktree、不得手工 `git worktree add`；fallback 必须同时满足 Desktop 不可用与用户明确批准；Desktop cleanup 必须先释放 worktree 再删本地 branch，且不得手工移除托管目录。
- [ ] 4.3 检查根路由表、规则文件和 OpenSpec artifacts 中引用的 repo 内路径均存在，确认不存在失效链接或要求无差别读取全部规则的残留文本。
- [ ] 4.4 运行 `openspec validate streamline-agent-rules`，检查 `git diff --check`、`git status --short` 和 scoped diff，确认只包含本 change 允许的规则与 artifacts，且未触碰其他 active change/worktree。
- [ ] 4.5 提交精简后的规则实现前，将前后尺寸、覆盖矩阵、Desktop 强制受管与两类 cleanup 验证、重复/冲突扫描、链接检查、OpenSpec validate 和 scoped diff 交由用户人工 Review；未批准不得进入 Sync、Archive 或 Deliver。
