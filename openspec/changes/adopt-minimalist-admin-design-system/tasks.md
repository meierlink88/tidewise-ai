## 1. 设计系统和 agent 规则

- [x] 1.1 将 `Minimalist/` 原始设计系统资料移动到 `../prototype/.design_library/minimal-dashboard/`。
- [x] 1.2 将 Minimal Dashboard 安装为 repo-local skill：`.codex/skills/minimal-dashboard-design/`。
- [x] 1.3 新增 `.agents/frontend-boundaries.md`，固化 admin 前端设计系统、skill 使用和生产代码边界。
- [x] 1.4 更新 `AGENTS.md`，把前端和 admin 设计系统规则路由到 `.agents/frontend-boundaries.md`。

## 2. Admin Minimal Dashboard 基础层

- [ ] 2.1 阅读 `.codex/skills/minimal-dashboard-design/SKILL.md`、`library-consumption.json`、tokens、组件 JSON 和 dashboard kit，确认生产可转译内容。
- [ ] 2.2 为 `frontend/admin` 增加 Minimal Dashboard tokens 样式入口。
- [ ] 2.3 新建 `frontend/admin/src/components/ui/` 基础组件，覆盖当前调度器页面需要的 button、card、field、input、select、switch、status badge。
- [ ] 2.4 新建 `frontend/admin/src/layouts/AdminShell.tsx`，实现 sidebar、顶部 token 输入区和内容布局。

## 3. 调度器设置页迁移

- [ ] 3.1 先更新或补充前端测试，覆盖 Minimal Dashboard 迁移后仍能加载配置、保存配置、展示 token 缺失和展示最近运行摘要。
- [ ] 3.2 将 `SchedulerSettings` 从 Ant Design 组件迁移到自有 UI primitives。
- [ ] 3.3 将 `App` 从 Ant Design layout/menu/input 迁移到 `AdminShell` 和自有输入组件。
- [ ] 3.4 保持现有 scheduler API client、字段语义、Admin Token header 和 localStorage token 行为不变。

## 4. 依赖和验证

- [ ] 4.1 检查 `frontend/admin` 是否仍引用 Ant Design；如无引用，从 `package.json` 和 lockfile 移除。
- [ ] 4.2 运行 `npm test` 验证 admin 前端行为。
- [ ] 4.3 运行 `npm run build` 验证 admin 前端构建。
- [ ] 4.4 运行 `openspec validate adopt-minimalist-admin-design-system`。
- [ ] 4.5 人工通过浏览器检查 admin 页面是否符合 Minimal Dashboard 的密集、克制、运营后台风格，且移动和桌面视口无明显溢出。
