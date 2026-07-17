## Context

仓库当前以 OpenSpec 作为正式 change 生命周期，但 `.agents/skill-routing.md`、`.agents/testing-tdd.md`、`.agents/git-workflow.md` 和 workflow architecture assertions 仍把 Superpowers 的多个 skill 名称作为强制机制。主规格也把 Superpowers 写入 workflow 定义。目标是移除这条外部依赖，同时维持现有工程约束、人工 gates、风险模型和 Desktop-managed worktree 规则。

## Goals / Non-Goals

**Goals:**

- 让正式规则、主规格和 architecture assertions 在没有 Superpowers plugin 时仍自洽、可验证、可执行。
- 保留 OpenSpec 生命周期、Proposal Review 与 Apply-final Review、R1/R2/R3 边界、TDD、debug diagnosis、fresh verification、Desktop worktree、Git 交付和 cleanup 顺序。
- 以负向引用扫描和正向 workflow contract assertions 证明依赖已解除且约束未削弱。

**Non-Goals:**

- 不实现新的 Skill、workflow runner、插件替代品或全局响应拦截器。
- 不修改业务源码、API、数据库、部署、UAT/prod/shared 状态、prototype 或 doc。
- 不在 Proposal 阶段 Apply、Sync、Archive、Deliver、创建 PR 或卸载 plugin。

## Decisions

1. **OpenSpec 保持唯一生命周期入口。** OpenSpec Skills 继续负责 Explore/Propose/Apply/Sync/Archive；用户人工确认 Review，项目原生规则描述 TDD、debug、verification，GitHub plugin/`gh` 负责交付。这样删除的是辅助工具依赖，不是流程约束。
2. **项目规则承载可执行纪律。** `.agents/testing-tdd.md` 直接规定测试先行、RED→GREEN→REFACTOR、失败诊断、证据和验证边界；`.agents/git-workflow.md` 直接规定 Desktop-managed 默认入口、经批准 fallback 及交付清理顺序。
3. **architecture assertions 双向验证。** 测试同时拒绝 `superpowers:*`、`docs/superpowers` 和“必须安装 Superpowers”的正式约束，并继续要求生命周期、Review、TDD/debug/verification、worktree、branch/cleanup、风险/事实源/安全边界。只做精确 workflow assertion，不运行业务 `go test ./...`。
4. **不迁移历史描述为执行来源。** 仅修改 active 正式规则和长期主 spec；不新增 docs 目录，不把旧 Superpowers 文档转化为替代事实源。

## Risks / Trade-offs

- [Risk] 卸载后用户失去 Superpowers 的全局响应前置行为 → [Mitigation] 明确该行为属于 plugin 自身，不由本 change 模拟；正式工程约束改由仓库规则和 OpenSpec gate 保证。
- [Risk] 删除路由时误删工程门禁 → [Mitigation] 采用逐项 before/after 路由矩阵、正向 contract assertions、严格 OpenSpec validate 和精确 task-design lint。
- [Risk] 历史文档或非 active 内容仍出现旧词导致扫描误判 → [Mitigation] 将检查范围限定为 active `.agents`、主 workflow spec 和 architecture assertions，并单独报告历史/归档内容，不扩大 change 范围。

## Migration Plan

Apply-final Review 通过后才允许 Sync；Sync/Archive/Deliver/PR merge+cleanup 完成后，独立在 `main` 上核验正式规则和主 spec 不含 Superpowers 强制引用，随后由用户卸载 plugin。若 Apply 验证失败，在同一 package 内修复；若 scope、风险或输入状态漂移则停止，不进行替代写入。

## Open Questions

- 无。是否卸载 plugin 属于用户在本 change 完整交付后的独立环境操作，不纳入 Apply。
