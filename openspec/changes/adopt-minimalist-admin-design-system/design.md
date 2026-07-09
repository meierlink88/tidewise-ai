## Context

当前 `frontend/admin/` 已有 Vite + React + TypeScript 工程和调度器设置页面，页面通过 Ant Design 的 `Layout`、`Menu`、`Form`、`Card`、`Button`、`Input` 等组件实现。用户提供的 `Minimalist/` 目录是一套完整的 Minimal Dashboard 设计系统，包含 `SKILL.md`、tokens、组件 JSON、preview HTML、图标和 dashboard kit。

本 change 已将原始设计资料移动到 `../prototype/.design_library/minimal-dashboard/`，并把工作副本安装到 `.codex/skills/minimal-dashboard-design/`。后续 agent 开发 admin 前端时应优先读取 repo 内 skill，避免依赖跨工作区路径。

## Goals / Non-Goals

**Goals:**

- 固化 Minimal Dashboard 为管理后台标准设计系统。
- 建立 `frontend/admin` 自有 tokens、基础 UI 组件和后台布局边界。
- 将当前调度器设置页从 Ant Design 迁移为 Minimal Dashboard 风格。
- 保持调度器 API、数据模型和保存逻辑不变，只调整前端展示和组件实现。
- 补充前端测试，验证迁移后仍能加载配置、保存配置、显示最近运行和处理 token 缺失。

**Non-Goals:**

- 不实现采集源管理、原始数据列表、事件列表或其他新后台菜单。
- 不修改后端调度器、repository、数据库 migration 或采集执行逻辑。
- 不把 Minimal Dashboard preview HTML 直接复制为生产页面。
- 不要求本 change 完成所有未来 admin 页面设计。
- 不把 prototype 目录作为运行时依赖。

## Decisions

### Decision: Minimal Dashboard 使用双位置模型

设计系统原始资料保存在 `../prototype/.design_library/minimal-dashboard/`，repo 内工作副本保存在 `.codex/skills/minimal-dashboard-design/`。

原因：prototype 是设计资产归档空间，但不在当前 Codex 工作区根目录内。repo-local skill 能保证新会话和后续 agent 在 `tidewise-ai` 中稳定读取设计系统。

备选方案：

- 只放在 prototype：路径更纯粹，但 agent 使用不稳定。
- 只放在 repo 根目录：读取方便，但容易被误认为生产源码。
- 直接放进 `frontend/admin`：会污染生产源码边界。

### Decision: 生产代码只消费转译后的 tokens 和组件

`frontend/admin` 不直接依赖 preview HTML。实现时从 `.codex/skills/minimal-dashboard-design/colors_and_type.css`、`components/*.json`、`components.css` 和 dashboard kit 中提炼生产用样式与组件。

目标结构：

```text
frontend/admin/src/styles/
├── tokens.css
└── app.css

frontend/admin/src/components/ui/
├── Button.tsx
├── Card.tsx
├── Field.tsx
├── Input.tsx
├── Select.tsx
├── Switch.tsx
└── StatusBadge.tsx

frontend/admin/src/layouts/
└── AdminShell.tsx
```

### Decision: Ant Design 作为历史实现迁移掉

本 change 完成后，`frontend/admin` 不再把 Ant Design 作为默认 UI 依赖。调度器页面应优先使用自有 UI primitives、CSS tokens 和 Minimal Dashboard 图标。

如果迁移后 `antd` 不再被引用，应从 `frontend/admin/package.json` 和 lockfile 移除。若因为过渡原因保留，必须在 tasks 中说明原因，并不得用于新增页面。

### Decision: 调度器设置页是第一个样板页面

调度器设置页需要保留已有功能：Admin Token 输入、启停调度、interval/fixed time 模式、并发数、batch size、超时秒数、保存设置、最近运行摘要。视觉结构改为 Minimal Dashboard 的 sidebar、card、button、table/status 模式。

## Risks / Trade-offs

- [Risk] Minimal Dashboard README 与 `colors_and_type.css` 对品牌色描述存在差异。→ 以可执行 tokens、`css.json` 和组件 preview 为生产实现依据，README 只作辅助解释。
- [Risk] 移除 Ant Design 会增加自有表单组件维护成本。→ 本 change 只实现当前页面需要的最小 UI primitives，避免一次性造完整组件库。
- [Risk] preview HTML 直接复制会引入 DOM/样式污染。→ agent 规则明确禁止复制 preview HTML，只允许转译为 React 和 CSS。
- [Risk] Codex 新会话可能不知道 prototype 中的源资料。→ repo-local skill 和 `.agents/frontend-boundaries.md` 固化读取入口。

## Migration Plan

1. 安装并验证 `.codex/skills/minimal-dashboard-design/`。
2. 更新 agent 前端规则，要求 admin 开发使用 Minimal Dashboard skill。
3. 建立 admin tokens、UI primitives 和后台 shell。
4. 迁移调度器设置页，保持 API 行为和测试语义不变。
5. 移除或停用 Ant Design 默认依赖。
6. 运行 `npm test`、`npm run build` 和 `openspec validate adopt-minimalist-admin-design-system`。
