## Why

管理后台已经具备调度器设置页面，但当前实现以 Ant Design 为默认 UI 体系，尚未把用户提供的 Minimal Dashboard 设计系统固化为长期标准。随着后续采集源管理、原始数据列表、事件列表等后台能力增加，需要先建立稳定的 admin 设计系统和 agent 使用规则，避免每个页面各自发散。

## What Changes

- 将 Minimal Dashboard 作为 `frontend/admin/` 的正式设计系统来源。
- 将设计系统原始资料归档到 `../prototype/.design_library/minimal-dashboard/`，并在 repo 内安装工作副本 `.codex/skills/minimal-dashboard-design/` 供 Codex 稳定读取。
- 新增 `.agents/frontend-boundaries.md`，规定 admin 开发必须使用 Minimal Dashboard skill、tokens、组件模式、图标和 dashboard kit。
- 修改管理后台主规格：Ant Design 不再是长期标准设计系统，后续 admin 页面默认使用 Minimal Dashboard 自有样式和组件边界。
- 在本 change 中迁移当前调度器设置页面，使它成为 Minimal Dashboard 在 admin 后台的第一个生产实现样板。
- 新增管理后台登录页，使用 Admin Token 登录后进入后台，不再要求用户在后台页面右上角输入 token。
- 在本地测试阶段，登录输入框下方可以显示当前测试 Admin Token 提示，便于人工验证；该提示不得改变后端鉴权协议。
- 补充调度器设置页面的后续体验需求，包括运行状态表达、最近运行记录展示、保存反馈、登录状态和基础响应式布局。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `admin-console`：将管理后台设计系统从 Ant Design 默认实现调整为 Minimal Dashboard 标准，并要求当前调度器设置页完成迁移。

## Impact

- 影响 repo 内 agent 规则：`AGENTS.md`、`.agents/frontend-boundaries.md`。
- 影响 Codex repo-local skill：`.codex/skills/minimal-dashboard-design/`。
- 影响原始设计资料归档：`../prototype/.design_library/minimal-dashboard/`。
- 影响管理后台前端：`frontend/admin/` 的样式、组件结构、依赖和测试。
- 影响管理后台前端交互：新增登录页和登出入口，替换当前右上角 token 输入方式。
- 不影响 `frontend/miniapp/` 小程序工程。
- 不影响后端调度器 API、数据库 migration、采集运行逻辑和 Admin Token 鉴权协议。
