---
status: accepted
date: 2026-08-13
supersedes_in_part: 0002-backend-service-architecture.md, 0007-app-oriented-monorepo.md
---

# 退役仓库根共享测试 Fixture

## 背景

仓库根 `testdata/` 曾保存 Data、Miniapp Backend 与 Miniapp Frontend 共用的 Research JSON
样例。该目录把 provider wire contract、consumer mapping、页面展示和开发 mock 绑定到同一批
大文件；样例生命周期不属于任何应用，删除过时业务能力时也无法判断由谁维护。

Tidewise AI 2.0 已决定删除根 `testdata/`，不再以共享样例文件作为跨应用合同权威。

## 决策

- 仓库根不再建立跨应用共享 `testdata/`。
- Provider OpenAPI、稳定 wire DTO、错误码和 provider contract test 继续拥有远程合同。
- Consumer 通过自己的 typed Adapter、OpenAPI drift test 和必要 HTTP smoke 验证消费合同；
  provider/consumer 合同变化必须在同一变更中同步双方验证。
- 单应用测试样例和开发 mock 跟随其 behavior owner。它们只验证该应用行为，不晋升为
  provider 合同或正式业务数据。
- Data 与 Miniapp 不 import 对方运行时实现，也不通过共享测试文件建立运行时依赖。

## 影响

旧 `testdata/reasoning-tree-v1`、`testdata/research-analysis-context-v1` 和
`testdata/research-theme-analyst-snapshot-v3` 不再是当前合同来源。只验证这些文件内容的测试
按 `obsolete` 删除；已有更高行为 seam 覆盖的 prepared-UAT 测试按
`duplicated-by-stronger-seam` 删除。Miniapp 仍需的展示数据成为 Miniapp-owned mock asset。

本决策不改变任何生产 HTTP 路径或 DTO。回滚需要恢复根 fixture、其全部消费者以及 CI 风险
路由；仅恢复目录而不恢复双方验证不构成有效回滚。
