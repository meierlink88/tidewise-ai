## 1. Architecture Artifact Review

- [x] 1.1 审阅 `proposal.md`，确认范围限定在 `tidewise-ai` OpenSpec artifacts、Taro 小程序源码骨架和未来源码架构边界。
- [x] 1.2 审阅 `design.md`，确认 frontend/backend 分离目录、Taro 前端、Go/Gin API/BFF、模块化单体、Agent 平台集成、采集、图谱、RAG 数据边界、存储、异步任务和部署决策符合项目方向。
- [x] 1.3 审阅 `specs/technical-architecture/spec.md` 和 `specs/mini-program-foundation/spec.md`，确认每条 requirement 可验证，并使用正确 scenario 格式。

## 2. Project Configuration

- [x] 2.1 创建 `package.json`，包含 Taro + React + TypeScript 开发脚本和开发依赖。
- [x] 2.2 创建用于 Taro 前端 TypeScript 编译的 `tsconfig.json`。
- [x] 2.3 在源码工程根目录创建 lint 和格式化配置文件。
- [x] 2.4 创建 `frontend/miniapp` Taro 工程配置，并包含微信和抖音小程序构建目标的安全开发默认值。

## 3. App Shell

- [x] 3.1 创建 `frontend/miniapp` 下的 Taro 应用入口、应用配置和全局样式文件。
- [x] 3.2 配置 feed、index、AI assistant、sectors 和 subscribe 五个 tabBar 入口。
- [x] 3.3 为 Taro 小程序壳添加全局样式导入和基础设计变量。

## 4. Page Skeletons

- [x] 4.1 创建 Taro feed 页面文件，包含最小页面标题和占位内容。
- [x] 4.2 创建 Taro index 页面文件，包含最小页面标题和占位内容。
- [x] 4.3 创建 Taro AI assistant 页面文件，包含最小页面标题和安全提示占位内容。
- [x] 4.4 创建 Taro sectors 页面文件，包含最小页面标题和占位内容。
- [x] 4.5 创建 Taro subscribe 页面文件，包含最小页面标题和占位内容。

## 5. Shared Source Directories

- [x] 5.1 创建 market card、event card、sector card、insight panel、confidence bar、tag list 和 empty state 的可复用组件目录。
- [x] 5.2 创建 event、market、sector、graph、report、subscription 和 AI message 的领域模型类型文件。
- [x] 5.3 创建 utility、constants、store、styles 和 assets 目录，并在需要时加入最小占位模块。

## 6. Mock Data and Service Boundary

- [x] 6.1 创建 events、markets、sectors、graph 和 subscriptions 的 mock 数据模块。
- [x] 6.2 创建 request wrapper，以及 event、market、sector、AI、report 和 subscription 领域 service 模块。
- [x] 6.3 确保页面骨架依赖 service 边界，而不是导入 prototype 文件或浏览器专属逻辑。

## 7. Validation

- [x] 7.1 运行 `openspec validate define-technical-architecture`，并修复所有验证错误。
- [x] 7.2 验证生成文件没有修改 `prototype` 或 `doc` 目录。
- [x] 7.3 验证 Taro 小程序配置引用了全部 tab 页面和实际存在的页面文件。
- [x] 7.4 运行可用的 Taro 微信/抖音构建、TypeScript、lint 或配置验证命令，并记录不可用的工具。
- [x] 7.5 确认前端源码中不存在密钥、模型凭证、Agent 平台凭证、支付凭证、直接数据库访问、后端执行逻辑、RAG 逻辑或 Agent 编排。
- [x] 7.6 确认后续 Go 后端骨架必须包含 local/uat/prod 配置加载设计，且 secret 不进入配置文件或 repo。
