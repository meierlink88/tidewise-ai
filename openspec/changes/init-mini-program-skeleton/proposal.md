## Why

观潮家需要从当前原型和文档阶段进入可持续工程开发阶段，先建立微信小程序源码骨架和工程边界，避免后续页面、数据、服务和组件代码混杂在一起。

当前 `dev` 目录尚未包含小程序工程，本变更用于创建可承接 MVP 页面迁移、mock 数据、前后端分离接口和后续 OpenSpec 变更的基础结构。

## What Changes

- 在 `dev` 源码工程中初始化原生微信小程序工程骨架。
- 建立五个一级 tab 页面入口：行情、指数、AI 助手、板块、订阅。
- 建立公共组件、领域模型、mock 数据、服务封装、工具函数、常量、全局样式和资源目录。
- 建立 TypeScript、工程配置、代码规范和小程序基础配置文件。
- 预留前后端分离 API 边界，首阶段只使用 mock 数据，不接入真实后端、模型、支付或推送能力。
- 以 `prototype` 目录中的高保真原型作为页面职责和视觉参考，但不迁移原型文件本身。

## Capabilities

### New Capabilities

- `mini-program-foundation`: 定义微信小程序工程骨架、页面入口、目录分层、mock 数据边界、服务接口边界和基础工程配置要求。

### Modified Capabilities

- None.

## Impact

- Scope: `dev` source engineering root and OpenSpec artifacts.
- Prototype reference: `prototype` may be read as design reference, but files in that directory are not modified by this change.
- Documentation reference: `doc/architecture.md` provides architectural context, but documentation updates are not part of this change.
- Affected systems:微信小程序前端工程结构、后续页面实现方式、mock 数据组织方式、前后端 API 边界预留方式。
- Non-goals: 不实现真实业务接口、不实现真实 AI 推理、不接入 Neo4j/向量库/Redis/PostgreSQL、不实现微信支付、不实现订阅消息推送、不完整迁移 prototype UI。
