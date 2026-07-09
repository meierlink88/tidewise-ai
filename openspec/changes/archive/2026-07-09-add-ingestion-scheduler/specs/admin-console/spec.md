## ADDED Requirements

### Requirement: 独立 Web 管理后台
系统 SHALL 在 `frontend/admin/` 提供独立 Web 管理后台，用于承载运营和系统管理能力，并与跨平台小程序工程保持边界隔离。

#### Scenario: 独立前端工程
- **WHEN** 开发者查看前端源码结构
- **THEN** 管理后台源码必须位于 `frontend/admin/`，不得混入 `frontend/miniapp/`

#### Scenario: 管理后台技术栈
- **WHEN** 开发者初始化或运行管理后台
- **THEN** 管理后台必须采用 Vite + React + TypeScript + Ant Design 技术栈

#### Scenario: 不影响小程序
- **WHEN** 管理后台新增页面、依赖或构建脚本
- **THEN** 不得破坏 `frontend/miniapp/` 的 Taro 小程序构建和运行边界

### Requirement: 调度器设置页面
系统 SHALL 在管理后台提供调度器设置菜单，使管理员可以查看和修改全局采集调度配置。

#### Scenario: 仅包含调度器设置菜单
- **WHEN** 本 change 完成后用户打开管理后台
- **THEN** 页面必须只提供调度器设置相关菜单，不得提前实现采集源管理、原始数据列表或事件列表

#### Scenario: 配置 interval 模式
- **WHEN** 管理员选择 interval 调度模式
- **THEN** 页面必须允许配置启用状态、interval 分钟数、并发数、batch size、超时时间和 source filter

#### Scenario: 配置固定时间模式
- **WHEN** 管理员选择固定时间调度模式
- **THEN** 页面必须允许配置至少 5 个每日固定时间，并在保存前校验时间格式

#### Scenario: 查看最近运行摘要
- **WHEN** 管理员打开调度器设置页面
- **THEN** 页面必须展示最近调度 run 的状态、开始时间、结束时间、成功数、失败数和错误摘要

### Requirement: Admin Token 前端接入
系统 SHALL 允许管理后台通过 Admin Token 调用后端管理 API，并避免把 token 写入 repo 或前端源码。

#### Scenario: 输入 Admin Token
- **WHEN** 管理员首次访问管理后台或 token 失效
- **THEN** 页面必须允许管理员输入 Admin Token

#### Scenario: 请求携带 token
- **WHEN** 管理后台调用调度器管理 API
- **THEN** 前端必须在请求头中携带 `Authorization: Bearer <token>`

#### Scenario: 不提交真实 token
- **WHEN** 开发者查看 repo 中的前端源码、配置和示例文件
- **THEN** 不得出现真实 Admin Token、模型 API key、搜索 API key 或数据库密码
