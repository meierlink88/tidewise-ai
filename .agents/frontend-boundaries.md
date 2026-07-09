# Frontend Boundaries

本文件定义前端工程边界、设计系统使用规则和设计稿转生产实现规则。

## 前端子系统

- `frontend/miniapp/` 是跨平台小程序工程，使用 Taro + React + TypeScript。
- `frontend/admin/` 是独立 Web 管理后台工程，使用 Vite + React + TypeScript。
- 小程序和管理后台是两个独立前端子系统，不共享页面、路由、平台 API 或构建入口。
- 小程序不得包含后台管理能力；管理后台不得混入小程序平台 API。

## Admin 设计系统

管理后台标准设计系统是 Minimal Dashboard。

Codex 使用入口：

```text
.codex/skills/minimal-dashboard-design/
```

设计系统原始资料归档入口：

```text
../prototype/.design_library/minimal-dashboard/
```

使用规则：

- 处理 `frontend/admin` 设计、页面、组件、样式或交互时，必须先读取 `.codex/skills/minimal-dashboard-design/SKILL.md`。
- 需要更完整的设计系统上下文时，按 `library-consumption.json` 的顺序读取 `README.md`、`css.json`、`components/index.json`、相关 `components/*.json`、相关 preview 和 `ui_kits/dashboard/index.html`。
- 后续 admin 新页面必须基于 Minimal Dashboard 的 token、组件模式、图标、sidebar、card、table、button 和 dashboard kit 组合方式实现。
- `.codex/skills/minimal-dashboard-design/` 是当前 repo 内 agent 可稳定读取的工作副本；不要让后续 agent 依赖跨工作区路径才能完成 admin 开发。
- `../prototype/.design_library/minimal-dashboard/` 是设计系统源资料；设计系统升级时，应先更新该目录，再同步到 `.codex/skills/minimal-dashboard-design/`。

## Admin 生产实现规则

- 不得把 Minimal Dashboard preview HTML、DOM 操作、内联脚本或示例页面直接搬进 `frontend/admin`。
- 生产实现必须转译为 React 组件、CSS tokens、CSS modules 或普通 CSS 文件。
- 设计 tokens 应进入 `frontend/admin/src/styles/` 下的稳定样式入口。
- 可复用 UI 基础组件应进入 `frontend/admin/src/components/ui/`。
- 可复用后台布局应进入 `frontend/admin/src/layouts/`。
- 新页面不得继续把 Ant Design 作为默认 UI 依赖；既有 Ant Design 实现应通过 OpenSpec change 逐步迁移到 Minimal Dashboard 自有组件。
- 图标优先使用 Minimal Dashboard skill 中提供的 `assets/icons/`，并通过 React 组件或构建可接受的 SVG 引用方式使用。

## 设计稿转实现

- 原型或设计系统资料只作为输入，不是生产源码。
- 开发前必须识别设计系统、页面范围、当前代码结构和非目标。
- 页面实现要复用已有 layout、UI primitives、API service 和测试工具，不得为单个页面创建平行前端工程。
- 管理后台 UI 文案应保持短、准、运营后台化，避免营销式说明。
