# Skill 路由

## 固定前置步骤

任何仓库写入任务都先调用 `$eino-reference-first`，完成三仓核验并记录引用的
上游实现或明确说明没有可复用实现。该步骤完成前不得写入 proposal 或实现文件。

## 委派职责

开发 Leader 只负责探索、任务委派、评审、批准、监控、验收和合并后清理。每个新
change 由 Leader 使用 Codex `create_thread` 委派给独立执行 Agent；执行 Agent负责
完成技能路由要求的审计、OpenSpec、TDD、实现、验证和交付，且不得自行批准 Leader
Review 或 Leader Acceptance。

“执行 Agent”仅指 `create_thread` 创建的用户可见独立 Codex 任务；禁止使用内部 sub-agent
或 `multi_agent` 承载、替代正式 change 的任何生命周期阶段。用户提出需要新 OpenSpec
change 的修改请求即构成创建该独立任务的授权。`create_thread`、Desktop-managed
worktree、默认模型组合或返回的任务标识任一不可用时，Leader 必须停止并报告。

工作流规则修改是唯一例外：当变更仅涉及工程协作规则、正式工作流规格、策略测试，
以及直接支撑该流程的 `.agents/skills/` 项目级工程协作 skill 或脚本时，由 Leader 在
当前对话直接实施，无需委派、无需 OpenSpec change。Leader 仍须完成 Eino audit、
策略测试的 RED/GREEN/REFACTOR、全量验证，并通过独立分支和 Pull Request 交付。
运行时 Agent skill 或任何非工作流规则内容不属于例外，必须恢复普通 change 的技能路由。

## OpenSpec 路由

| 任务意图 | 必须使用的 skill | 结果 |
|---|---|---|
| 调研问题、梳理范围但暂不承诺修改 | `$openspec-explore` | 探索结论，不直接实施 |
| 创建新的正式 change | `$openspec-propose` | proposal、design、specs、tasks |
| 用户已批准提案并要求实施 | `$openspec-apply-change` | 按 tasks 和 TDD 实施 |
| 用户已批准完成评审，需要同步正式规格 | `$openspec-sync-specs` | 更新 `openspec/specs/` |
| 规格已同步且准备结束 change | `$openspec-archive-change` | 归档 change 并最终验证 |

除上述工作流规则修改外，不得因为改动较小而跳过 OpenSpec。纯只读问答或状态检查
不产生仓库写入时，不需要新建 change。

## 语言规则

所有 change 产物中文优先。OpenSpec 要求的固定结构标记（例如 `Requirement`、
`Scenario`、`WHEN/THEN`、`SHALL/MUST`）、命令、路径、代码标识符和 capability
标识保持规范形式，其余标题和正文使用中文。

## GitHub 交付

完成 Sync 和 Archive 后，使用当前可用的 GitHub PR 发布能力；若专用 skill
不可用，则使用 `gh` CLI。只能提交当前 change 范围内的文件，并在提交前执行
密钥和禁止路径检查。
