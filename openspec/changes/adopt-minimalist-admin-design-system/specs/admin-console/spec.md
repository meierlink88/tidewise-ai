## ADDED Requirements

### Requirement: Minimal Dashboard 管理后台设计系统
系统 SHALL 将 Minimal Dashboard 作为 `frontend/admin/` 的标准设计系统，并通过 repo-local Codex skill 固化后续 agent 的使用入口。

#### Scenario: 使用 repo-local skill
- **WHEN** agent 处理 `frontend/admin` 的页面、组件、样式或布局开发
- **THEN** agent 必须读取 `.codex/skills/minimal-dashboard-design/SKILL.md`，并按该 skill 的 tokens、组件模式、preview 和 dashboard kit 转译生产实现

#### Scenario: 区分设计资料和生产源码
- **WHEN** 开发者查看设计系统资产位置
- **THEN** 原始设计资料必须归档在 `../prototype/.design_library/minimal-dashboard/`，repo 内工作副本必须位于 `.codex/skills/minimal-dashboard-design/`

#### Scenario: 禁止直接复制 preview HTML
- **WHEN** agent 将 Minimal Dashboard 设计系统用于生产管理后台
- **THEN** 不得把 preview HTML、DOM 操作、内联脚本或示例页面直接复制到 `frontend/admin`

### Requirement: 管理后台自有 UI 基础层
系统 SHALL 在 `frontend/admin` 中提供自有的 Minimal Dashboard 风格 tokens、基础组件和后台布局，使后续页面不依赖 Ant Design 作为默认 UI 体系。

#### Scenario: 提供样式 tokens
- **WHEN** 管理后台页面渲染
- **THEN** 页面必须通过 `frontend/admin/src/styles/` 下的 Minimal Dashboard tokens 表达颜色、字体、间距、圆角、边框、状态色和基础阴影

#### Scenario: 提供基础组件
- **WHEN** 管理后台页面需要按钮、卡片、输入、选择、开关、状态标记或列表展示
- **THEN** 页面必须优先使用 `frontend/admin/src/components/ui/` 下的自有组件，而不是直接使用 Ant Design 组件

#### Scenario: 提供后台布局
- **WHEN** 管理后台新增或修改页面
- **THEN** 页面必须复用 `frontend/admin/src/layouts/` 下的后台 shell、sidebar 和内容布局模式

## MODIFIED Requirements

### Requirement: 独立 Web 管理后台
系统 SHALL 在 `frontend/admin/` 提供独立 Web 管理后台，用于承载运营和系统管理能力，并与跨平台小程序工程保持边界隔离。

#### Scenario: 独立前端工程
- **WHEN** 开发者查看前端源码结构
- **THEN** 管理后台源码必须位于 `frontend/admin/`，不得混入 `frontend/miniapp/`

#### Scenario: 管理后台技术栈
- **WHEN** 开发者初始化或运行管理后台
- **THEN** 管理后台必须采用 Vite + React + TypeScript 技术栈，并以 Minimal Dashboard 作为标准设计系统

#### Scenario: 不影响小程序
- **WHEN** 管理后台新增页面、依赖或构建脚本
- **THEN** 不得破坏 `frontend/miniapp/` 的 Taro 小程序构建和运行边界

### Requirement: 调度器设置页面
系统 SHALL 在管理后台提供符合 Minimal Dashboard 设计系统的调度器设置菜单，使管理员可以查看和修改全局采集调度配置。

#### Scenario: 仅包含调度器设置菜单
- **WHEN** 本 change 完成后用户打开管理后台
- **THEN** 页面必须只提供调度器设置相关菜单，不得提前实现采集源管理、原始数据列表或事件列表

#### Scenario: 配置 interval 模式
- **WHEN** 管理员选择 interval 调度模式
- **THEN** 页面必须允许配置启用状态、interval 分钟数、并发数、batch size 和超时时间

#### Scenario: 配置固定时间模式
- **WHEN** 管理员选择固定时间模式
- **THEN** 页面必须允许配置至少 5 个每日固定时间，并在保存前校验时间格式

#### Scenario: 查看最近运行摘要
- **WHEN** 管理员打开调度器设置页面
- **THEN** 页面必须以 Minimal Dashboard 的卡片、状态标记或表格模式展示最近调度 run 的状态、开始时间、结束时间、执行轮次成功失败统计和错误摘要

#### Scenario: 保存反馈
- **WHEN** 管理员保存调度器设置
- **THEN** 页面必须给出成功、失败或 token 缺失的明确反馈，并保持字段值不会因刷新丢失已保存配置
