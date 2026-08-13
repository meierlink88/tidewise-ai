# AGENTS.md

## Agent skills

### Delivery workflow

产品变更使用 `$ganchaojia-development-standard` 发现的统一流程。OpenSpec 已退役；
除非用户明确要求，不创建独立任务或 worktree。实现默认在当前 checkout 的
`codex/*` feature branch 中完成，并由用户控制 PR 合并。

### Issue tracker

工作事项使用 GitHub Issues 管理。详见 `docs/agents/issue-tracker.md`。

### GitHub CLI

- `gh issue create` 和 `gh pr create` 是默认自动交付动作，无需用户逐次授权；允许
  Codex 自动在沙箱外读取 macOS Keychain 凭据并创建 Issue 或 PR。
- Codex Runtime 自动添加的 Shell 启动层不视为人工 Shell 包装。Agent 不应主动使用
  `bash -lc`、`zsh -lc`、命令串联或变量替换来隐藏 GitHub CLI 操作。
- PR 和 Issue 的多行正文优先写入临时文件，再通过 `--body-file` 传入，避免转义错误。
  这是一项正文格式约定，不是 PR 创建审批门禁。

### Triage labels

使用五种标准 triage 状态。详见 `docs/agents/triage-labels.md`。

### Domain docs

采用 Data、Miniapp、Admin Portal、AgentRun 四个上下文的 multi-context 布局。应用
源码分别位于 `analyse-data-service/`、`miniapp/`、`admin-portal/` 和
`agent-run/`；AgentRun 共仓后仍保持独立 Context、数据库、Artifact 与 API 边界。
详见 `docs/agents/domain.md`。

### Engineering standards

任何会改变系统行为、项目事实、API、数据、配置、运行时、部署或测试合同的任务，都必须
先使用 `$ganchaojia-development-standard` 发现共享规范，再按 behavior owner 读取对应
技术栈规范。全局 Skill 和外部参考提供路由与证据；Skill 发现的仓库内规范、Context、
ADR、OpenAPI 和当前数据约束才是项目权威。

### Testing

Backend Service 采用按风险边界测试，不要求每个源码文件或 Kratos 层级都拥有测试。
开发前先确认测试 seam，默认覆盖 Biz 行为与 API/HTTP 合同；Data、Migration、Conf、
Lifecycle 和 Architecture 仅在对应风险被本次修改触及时启用。具体测试规范由
`$ganchaojia-development-standard` 按任务发现。

### 观潮家开发规范

讨论、设计、实现、调试、迁移或审查任何观潮家工程需求前，运行环境提供全局 Skill
`$ganchaojia-development-standard` 时必须先使用，不得以熟悉现有代码为由跳过。
若该 Skill 不可用，必须立即报告，不得猜测规范路径、静默跳过或直接实施。

- Miniapp 前端工作执行 Skill 路由的 Taro reference-first 分支。
- Admin Portal 前端工作执行 Skill 路由的 shadcn 分支；不得为适配前端框架改造 Admin
  Backend API。
- Backend Service 工作执行 Skill 路由的 Kratos 分支；工程结构使用当前结构规范，
  运行时、API、数据与业务事实以 Context、ADR、OpenAPI 和当前数据约束为准。
- Eino/Agent 工作执行 Skill 路由的 Eino reference-first 分支。
- 跨前端、Service 与 Agent 的需求同时执行所有适用分支，并先冻结 API、数据和所有权边界。
- 仅不改变系统行为或项目事实的纯解释、纯文案工作可以不触发。
