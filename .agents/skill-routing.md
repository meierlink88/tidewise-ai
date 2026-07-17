# Skill 路由

## 固定前置步骤

任何仓库写入任务都先调用 `$eino-reference-first`，完成三仓核验并记录引用的
上游实现或明确说明没有可复用实现。该步骤完成前不得写入 proposal 或实现文件。

## OpenSpec 路由

| 任务意图 | 必须使用的 skill | 结果 |
|---|---|---|
| 调研问题、梳理范围但暂不承诺修改 | `$openspec-explore` | 探索结论，不直接实施 |
| 创建新的正式 change | `$openspec-propose` | proposal、design、specs、tasks |
| 用户已批准提案并要求实施 | `$openspec-apply-change` | 按 tasks 和 TDD 实施 |
| 用户已批准完成评审，需要同步正式规格 | `$openspec-sync-specs` | 更新 `openspec/specs/` |
| 规格已同步且准备结束 change | `$openspec-archive-change` | 归档 change 并最终验证 |

不得因为改动较小而跳过 OpenSpec。纯只读问答或状态检查不产生仓库写入时，不需要
新建 change。

## 语言规则

所有 change 产物中文优先。OpenSpec 要求的固定结构标记（例如 `Requirement`、
`Scenario`、`WHEN/THEN`、`SHALL/MUST`）、命令、路径、代码标识符和 capability
标识保持规范形式，其余标题和正文使用中文。

## GitHub 交付

完成 Sync 和 Archive 后，使用当前可用的 GitHub PR 发布能力；若专用 skill
不可用，则使用 `gh` CLI。只能提交当前 change 范围内的文件，并在提交前执行
密钥和禁止路径检查。
