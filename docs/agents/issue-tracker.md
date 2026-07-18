# Issue Tracker: GitHub

本仓库的 Issues 和 PRD 使用 GitHub Issues 管理，仓库由当前 git remote 确定。

## Operations

- 创建：`gh issue create --title "..." --body "..."`
- 查看：`gh issue view <number> --comments`
- 列表：`gh issue list --state open --json number,title,body,labels,comments`
- 评论：`gh issue comment <number> --body "..."`
- 标签：`gh issue edit <number> --add-label "..." --remove-label "..."`
- 关闭：`gh issue close <number> --comment "..."`

当 Skill 要求发布到 Issue Tracker 时，创建 GitHub Issue。当 Skill 要求读取 ticket 时，使用 `gh issue view` 获取正文、评论和标签。

## Pull Requests As A Request Surface

PRs as a request surface: no.

PR 不进入 Issue triage 队列；PR Review 使用独立的 GitHub Review 流程。
