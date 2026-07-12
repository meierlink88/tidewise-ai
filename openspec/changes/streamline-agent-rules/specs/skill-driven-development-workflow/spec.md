## ADDED Requirements

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
系统 MUST 在规则精简后继续明确约束 OpenSpec 唯一生命周期、Codex Desktop 受管任务/worktree 强制入口、change branch/worktree 隔离、人工 Review 与有状态操作审批、sync/archive/deliver 顺序、按 worktree 所有权交付清理、数据事实源和通用安全边界。

#### Scenario: 未完成人工 Review
- **WHEN** change 的 proposal artifacts 尚未获得人工确认
- **THEN** Agent 不得进入 Apply，也不得以自动化或 Skill 默认流程替代该确认

#### Scenario: 执行数据库或图谱有状态操作
- **WHEN** Agent 准备写入数据库、写入图谱关系层或重建图谱投影
- **THEN** Agent 必须按层展示拟执行范围并取得明确批准，不得从只读审计推定写入授权

#### Scenario: 启动并行 change
- **WHEN** 另一个 change 仍在 review、实现或未合并状态
- **THEN** 在 Codex Desktop 可用时，Agent 必须通过 Desktop 新任务创建受管 worktree，并在受管任务内从最新 `origin/main` 创建或切换匹配的 `codex/<change-name>` branch，不得手工执行 `git worktree add`，且不得混入其他 change 的文件

#### Scenario: 在 Codex Desktop 中启动新 change
- **WHEN** Agent 在 Codex Desktop 可用的环境中启动任何新 change
- **THEN** Agent 必须先通过 Desktop 新任务创建受管 worktree，再在该受管任务内创建或切换 `codex/<change-name>` branch，不得把当前 worktree 中手工创建 branch/worktree 作为等价默认路径

#### Scenario: Desktop 机制不可用时创建 fallback worktree
- **WHEN** Codex Desktop 受管任务/worktree 机制不可用且用户明确批准 fallback
- **THEN** Agent 才可按 `.agents/git-workflow.md` 创建项目自有 worktree，并在其中从最新 `origin/main` 创建或切换匹配的 `codex/<change-name>` branch

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
系统 SHALL 在 Apply 中记录精简前后根 `AGENTS.md` 的行数、字符数与字节数，维护关键规则覆盖矩阵，并验证 Desktop 强制受管与两类 cleanup 顺序、重复/冲突、文件链接和 OpenSpec artifacts。

#### Scenario: 精简结果进入人工 Review
- **WHEN** Agent 完成规则正文修改并准备请求人工验收
- **THEN** Agent 必须提供前后行数与字节数、压缩率、覆盖矩阵、重复/冲突扫描、链接检查、scoped diff 和 `openspec validate streamline-agent-rules` 的新鲜结果
