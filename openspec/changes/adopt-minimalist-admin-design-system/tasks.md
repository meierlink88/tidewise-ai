## 1. 设计系统和 agent 规则

- [x] 1.1 将 `Minimalist/` 原始设计系统资料移动到 `../prototype/.design_library/minimal-dashboard/`。
- [x] 1.2 将 Minimal Dashboard 安装为 repo-local skill：`.codex/skills/minimal-dashboard-design/`。
- [x] 1.3 新增 `.agents/frontend-boundaries.md`，固化 admin 前端设计系统、skill 使用和生产代码边界。
- [x] 1.4 更新 `AGENTS.md`，把前端和 admin 设计系统规则路由到 `.agents/frontend-boundaries.md`。

## 2. Admin Minimal Dashboard 基础层

- [ ] 2.1 阅读 `.codex/skills/minimal-dashboard-design/SKILL.md`、`library-consumption.json`、tokens、组件 JSON 和 dashboard kit，确认生产可转译内容。
- [ ] 2.2 为 `frontend/admin` 增加 Minimal Dashboard tokens 样式入口。
- [ ] 2.3 新建 `frontend/admin/src/components/ui/` 基础组件，覆盖当前登录页和调度器页面需要的 button、card、field、input、select、switch、status badge。
- [ ] 2.4 新建 `frontend/admin/src/layouts/AdminShell.tsx`，实现 sidebar、顶部登录状态、退出入口和内容布局。

## 3. 登录页和 token 会话

- [ ] 3.1 先更新或补充前端测试，覆盖未登录时展示登录页、输入 Admin Token 后进入后台、本地测试 token 提示和退出登录。
- [ ] 3.2 新建 `frontend/admin/src/pages/AdminLogin.tsx`，使用 Minimal Dashboard 风格实现 Admin Token 登录页。
- [ ] 3.3 将 `App` 改为登录态控制：未登录显示 `AdminLogin`，已登录显示 `AdminShell` 和调度器设置页面。
- [ ] 3.4 在本地测试阶段于登录输入框下方展示测试 token 提示，默认提示 `local-admin-token`，并确保该提示不参与后端鉴权。
- [ ] 3.5 在后台 shell 提供退出登录入口，清除本地保存 token 并返回登录页。

## 4. 调度器设置页迁移

- [ ] 4.1 先更新或补充前端测试，覆盖 Minimal Dashboard 迁移后仍能加载配置、保存配置和展示最近运行摘要。
- [ ] 4.2 将 `SchedulerSettings` 从 Ant Design 组件迁移到自有 UI primitives。
- [ ] 4.3 将 `App` 从 Ant Design layout/menu/input 迁移到 `AdminLogin`、`AdminShell` 和自有输入组件。
- [ ] 4.4 保持现有 scheduler API client、字段语义、Admin Token header 和 localStorage token 行为不变。

## 5. 依赖和验证

- [ ] 5.1 检查 `frontend/admin` 是否仍引用 Ant Design；如无引用，从 `package.json` 和 lockfile 移除。
- [ ] 5.2 运行 `npm test` 验证 admin 前端行为。
- [ ] 5.3 运行 `npm run build` 验证 admin 前端构建。
- [ ] 5.4 运行 `openspec validate adopt-minimalist-admin-design-system`。
- [ ] 5.5 人工通过浏览器检查登录页和 admin 页面是否符合 Minimal Dashboard 的密集、克制、运营后台风格，且移动和桌面视口无明显溢出。
