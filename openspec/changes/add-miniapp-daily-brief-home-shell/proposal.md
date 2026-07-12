## Why

现有 `frontend/miniapp/` 已具备 Taro + React + TypeScript 工程壳和 mock-first service 边界，但 index tab 仍是“指数”占位页，不能呈现观潮家的核心首页体验。现在需要先交付一个微信小程序可运行的“今日观潮”纯前端 Mock 首页，为页面视觉验收和后续正式 API integration 提供稳定基线。

## What Changes

- 将现有 `pages/index` 增量演进为默认首页“今日观潮”，把它调整为 pages 与 tabBar 第一项并将 tab 文案改为“首页”；其他 tab 页面与源码保持不变。
- 为首页定义 loading、empty、error、ready 四态；错误态可重试，空态不伪造市场结论。
- 建立版本化前端候选契约、service port、mock adapter、dedicated mock data、页面 view model/template 与 section registry；页面不直接读取 fixture，未来 HTTP adapter 可替换 mock adapter而无需重写页面。
- 评估 mock-only 的 `DailyBriefV1`、`ReasoningConclusionV1`、`ImpactAssessmentV1`、`EvidenceItemV1` 及首页确有需要的引用结构，明确它们不是冻结的后端 API 契约。
- 将 `/Users/meierlink/Documents/david/创业项目/观潮家/prototype2/miniprogram.html` 首页最终渲染确立为本 change 的 page-level canonical visual/interaction source，以 375×812 viewport 截图基线进行视觉对比验收；小程序平台差异只允许做有记录的等价工程转译。
- 保留设计稿中的“看图谱”视觉入口，但本期不创建图谱页、图谱 DTO、graph components 或图谱交互。推荐点击后展示轻提示“推导图谱即将开放”，既保持 canonical 入口和点击反馈，也不误导用户认为能力已经开放；该行为作为 proposal Review 决策点。
- 在现有 `frontend/miniapp/src/styles` 与 components 边界内提炼轻量 tokens、primitives 和 page compositions；不复制原始 HTML、DOM/内联脚本、prototype shell、annotation/debug 资产或整套 `.design_library`。
- 逐项盘点原型图片、字体和内联 SVG；只有生产运行确实需要且授权明确的资产才可复制到 `frontend/miniapp/src/assets`，并记录来源与 SHA-256。
- 将原型中的个股推荐、公司排序和买卖暗示替换为市场、板块、benchmark、商品、经济体或产业链实体，并持续标注“不构成投资建议”。
- 在 Apply 更新 `.agents/frontend-boundaries.md`：旧 `ganchaojia-design` skill 保留为历史和基础 token 参考，但不再拥有生产小程序 page-level 最终视觉裁决权。
- 补充微信构建、本地微信开发者工具防错导入/预览和截图视觉验收任务；本 change 暂不把抖音构建或预览作为验收目标。

## Capabilities

### New Capabilities

- `miniapp-daily-brief-home-shell`: 定义“今日观潮”Mock 首页、四态、首页交互、候选契约、可替换 adapter、canonical 视觉基线和微信运行验收。

### Modified Capabilities

- `skill-driven-development-workflow`: 补充生产小程序 page-level canonical source、历史 design skill 路由、原型只读和视觉对比验收要求。

## Impact

- Apply 源码范围：`frontend/miniapp/` 内现有应用配置、index 首页、共享组件、styles、contracts/types、models、services、mock adapter、dedicated mock data、页面 template/section registry、必要且获授权的生产资产及测试配置；不创建平行工程。
- Apply 项目规则范围：`.agents/frontend-boundaries.md`。proposal checkpoint 只提交 OpenSpec artifacts，不提前修改该规则文件。
- canonical source：`/Users/meierlink/Documents/david/创业项目/观潮家/prototype2/miniprogram.html`，SHA-256 `ad90bcc8942cf30cdcd730134361e596e86b581c93e0a28087df2eb00d43f69a`；关联 `colors_and_type.css` SHA-256 `3605c97214cbc09d66c270a9280a3373251a2f2ad8c653850f3cb3680c23a889`；关联 `components.css` SHA-256 `f0c32a315a1c3e96355dcc7b9efee9650144a4db0b5a39d6317916bad43757c1`；目标 viewport 为 375×812。
- 页面/状态范围：首页 ready 的展开/折叠、主线切换、“看图谱”占位反馈，以及 loading、empty、error。实现阶段通过固定截图基线与视觉对比验收，不以复制 HTML 达成一致。
- 资产盘点：`assets/home-header-sea.jpg` SHA-256 `667dcd64bcfb7c3d40e4f5f5a6d0b9be1f88a90824e5e3db88527f08703b6fdc` 是首页 canonical 候选资产，复制前必须确认授权；`assets/nav-avatar.png` 属 prototype shell，排除；远程 Google Fonts 不直接带入小程序；内联 SVG 转译为 Taro 可用图标/样式。
- 非目标：不实现推导图谱页或图谱数据结构，不修改其他 tab、Go 后端、数据库、Neo4j、Agent/RAG、真实 HTTP API、鉴权、订阅、支付、推送、`doc/` 或任何 prototype 源文件，不创建 repo-local miniapp design skill，不触碰 `add-ai-event-extraction-pipeline`。
- 平台范围：本 change 仅验收微信小程序。既有 Taro 抖音依赖和 `build:tt` 脚本保留以避免无关破坏，但不要求本 change 构建、预览或声明抖音兼容性。
- 依赖：不新增 UI 框架，不复制整套 design library；测试工具只允许 miniapp workspace 内的最小增量，并由 Apply Review 决定。
