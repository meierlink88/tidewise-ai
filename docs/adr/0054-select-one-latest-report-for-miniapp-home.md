---
status: accepted
date: 2026-09-03
issue: 387
supersedes_in_part: 0052-replace-research-theme-with-report-publications.md
---

# Miniapp 首页只选择一份最新 Report

## 背景

ADR-0052 曾决定 Miniapp 首页展示上海当日全部 Report，并在前端用发布时间 Tab
切换。该多 Report 交互尚未完成产品设计，且同时暴露正式报告与同日较旧报告，与首页聚焦当前最新结论的
产品语义不符。

## 决策

- Data Domain Service 继续保留全部不可变 Report，不删除、覆盖或合并同日报告。
- Miniapp Backend 使用 `Asia/Shanghai` 计算当日时间范围，并按 Data 的
  `published_at DESC, id ASC` 权威顺序以 `limit=1` 读取当日最新 Report。
- 当日查询为空时，Miniapp Backend 以相同权威顺序和 `limit=1` 读取历史最新 Report。
- Miniapp Home API 保留 `reports` 集合形状表达正常空态，但成功语义的基数固定为零或一；
  `latest_fallback` 固定为一。
- Miniapp Frontend 不排序或选择 Report，不展示多 Report Tab；若 Backend 返回超过一份则按合同异常
  fail closed。

## 影响

首页始终只有一个发布时间和一份报告内容。同日较旧 Report 仍可由 Data Report 列表稳定查询，
但在未完成历史浏览产品设计前不由 Miniapp 首页暴露。该决定不改变 Report 发布、不可变性、Evidence 关系或
产业链分页合同。
