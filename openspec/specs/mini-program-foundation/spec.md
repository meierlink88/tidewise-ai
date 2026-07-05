## Purpose

定义 Taro 跨平台小程序工程壳、一级页面、源码目录边界、mock-first 数据边界和前端安全边界，作为后续小程序功能开发的当前系统事实。

## Requirements

### Requirement: Taro 跨平台小程序工程壳
系统 SHALL 在源码工程根目录下提供 Taro 跨平台小程序工程壳，让后续产品功能可以在一致位置实现，并支持微信和抖音小程序目标。

#### Scenario: 工程壳存在
- **WHEN** 开发者打开源码工程根目录
- **THEN** Taro 小程序源码壳位于 `frontend/miniapp`

#### Scenario: 核心应用配置存在
- **WHEN** 开发者检查 Taro 小程序源码壳
- **THEN** 必须存在应用级配置、应用入口、全局样式、项目配置和多端构建配置

### Requirement: 一级 Tab 导航
系统 SHALL 定义五个一级 Taro 小程序 tab 页面，并与 MVP 产品导航匹配。

#### Scenario: 配置 tab 页面
- **WHEN** Taro 小程序应用配置被加载
- **THEN** tab 导航必须包含 feed、index、AI assistant、sectors 和 subscribe 页面

#### Scenario: tab 页面拥有页面文件
- **WHEN** 开发者检查每个一级 tab 页面目录
- **THEN** 每个页面都必须包含 Taro 页面组件、样式文件和页面配置

### Requirement: 工程目录边界
系统 SHALL 将页面 UI、可复用组件、领域模型、mock 数据、service 访问、共享状态、工具函数、常量、样式和资源分离到专门目录。

#### Scenario: 共享源码区域存在
- **WHEN** 开发者检查 Taro 小程序源码壳
- **THEN** 必须存在 components、models、data、services、store、utils、constants、styles 和 assets 专用目录

#### Scenario: 原型保持只读参考
- **WHEN** Taro 小程序源码壳被创建
- **THEN** prototype 文件必须保留在源码壳之外，并且不被本 change 修改

### Requirement: Mock-First 数据边界
系统 SHALL 通过领域数据模块和 service wrappers 支持 mock-first 开发，并允许后续替换为真实 API 调用。

#### Scenario: service 边界存在
- **WHEN** 页面需要事件、市场、板块、AI、报告或订阅数据
- **THEN** 页面可以依赖 service 模块，而不是读取 prototype 文件或硬编码浏览器 DOM 状态

#### Scenario: mock 数据被隔离
- **WHEN** 为 MVP 页面添加 mock 内容
- **THEN** mock 内容必须放在专门 data modules 中，而不是直接嵌入页面 markup

### Requirement: 前端安全边界
系统 SHALL 确保小程序源码不包含后端密钥、模型凭证、支付凭证、直接数据库访问、RAG 逻辑和 Agent 编排。

#### Scenario: 需要敏感后端能力
- **WHEN** Taro 小程序需要 AI 分析、报告生成、支付、订阅或事件智能能力
- **THEN** 小程序必须调用 service/API 边界，而不是在客户端嵌入凭证、Agent 平台调用或后端执行逻辑

#### Scenario: 展示分析内容
- **WHEN** 页面展示 AI 或市场分析内容
- **THEN** UI 必须包含或保留安全定位，说明内容是决策辅助信息而不是直接投资建议
