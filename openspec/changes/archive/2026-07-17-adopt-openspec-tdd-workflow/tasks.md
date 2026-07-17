## 1. 工作流策略测试（RED）

- [x] 1.1 新增一个预期失败的 Go 策略测试，要求存在 Eino-first 路由、OpenSpec 生命周期、中文优先、`codex/<change-name>` worktree 隔离、评审门禁、TDD 阶段和 PR 交付规则。
- [x] 1.2 执行聚焦的策略测试，记录测试因缺少新工作流文档和配置而失败的 RED 证据。

## 2. 仓库工作流（GREEN）

- [x] 2.1 更新 `AGENTS.md`，加入强制生命周期和技能路由表，同时保留 Eino 上游参考仓库只读规则。
- [x] 2.2 在 `.agents/` 下增加技能路由、OpenSpec 评审/apply/sync/archive、Git 分支/worktree/PR 和 TDD 证据规则文档。
- [x] 2.3 增加工程架构边界，覆盖公共运行时基础设施、AI 采集器、AI 事件提取器和 AI 投研报告分析师。
- [x] 2.4 定制 `openspec/config.yaml`，加入工程上下文、中文优先规则、可测试需求和 TDD tasks 约束。
- [x] 2.5 执行聚焦的策略测试，记录最小工作流配置完成后的 GREEN 证据。

## 3. 重构与验证（REFACTOR）

- [x] 3.1 消除各工作流文档之间的重复表述，同时不削弱已经被测试保护的规则。
- [x] 3.2 对策略测试执行 `gofmt`，然后运行 `go test ./...`。
- [x] 3.3 对 change 执行严格 OpenSpec 验证，并检查暂存差异中是否存在密钥或禁止提交的 `.reference/cloudwego/` 内容。

## 4. 评审与交付

- [x] 4.1 在执行 1—3 组任务前，提交 proposal、design、spec、风险和 TDD 计划，等待用户明确完成提案评审。
- [x] 4.2 在同步和归档前，提交实现差异及验证证据，等待用户明确完成实施评审。
- [x] 4.3 同步 capability specs、归档已批准的 change、重新执行最终验证、有选择地提交文件、推送 `codex/adopt-openspec-tdd-workflow`，并创建 GitHub Pull Request。
