## Agent skills

### Delivery workflow

Product changes use the repository workflow in `docs/agents/workflow.md`.

- The default lifecycle is `grill-with-docs -> to-spec -> implement`.
- OpenSpec is retired and must not be created, restored, validated, synced, or archived.
- The current task executes the work directly by default. Create or delegate another Codex task only when the user explicitly requests delegation.
- The user controls pull-request merge.

### Issue tracker

Issues and PRDs are tracked in this repository's GitHub Issues. See `docs/agents/issue-tracker.md`.

### GitHub CLI

- Run `gh` as a direct command. Do not wrap it in `bash -lc`, `zsh -lc`, command chains, substitutions, or inline multiline bodies.
- Write multiline Issue and PR bodies to a temporary file, then pass it with `--body-file` so `.codex/rules/github.rules` can match the command and allow macOS Keychain access.

### Triage labels

Use the five canonical triage labels defined for this repository. See `docs/agents/triage-labels.md`.

### Domain docs

This repository uses a single-context domain documentation layout. See `docs/agents/domain.md`.

### 观潮家开发规范

讨论、设计、实现、调试、迁移或审查任何观潮家工程需求前，必须先使用全局 Skill
`$ganchaojia-development-standard`，不得以熟悉现有代码为由跳过。

- Miniapp 前端工作执行其中的 Taro reference-first 分支。
- Backend Service 工作执行其中的 Kratos 分支。
- Eino/Agent 工作执行其中的 Eino reference-first 分支，并遵守本仓库
  `docs/agents/workflow.md` 的 Eino reference-first gate。
- 跨前端、Service 与 Agent 的需求同时执行所有适用分支，并先冻结 API、数据和所有权边界。
- 仅不改变系统行为或项目事实的纯解释、纯文案工作可以不触发。
