## Context

观潮家当前处于从产品资料和高保真原型进入源码工程的起点。`dev` 是源码工程根目录和 OpenSpec 根目录，目前只包含 OpenSpec 配置与 TRAE skills，尚未包含微信小程序代码。

项目架构已明确 MVP 首端为微信小程序，并采用前后端分离。小程序端负责展示、交互、轻状态和 API 调用；AI 推理、RAG、知识图谱、事件采集、支付密钥、模型密钥和数据存储均位于后端或后续服务层。

## Goals / Non-Goals

**Goals:**

- 创建原生微信小程序 + TypeScript 的工程骨架。
- 建立五个 tab 页面目录：行情、指数、AI 助手、板块、订阅。
- 建立组件、领域模型、mock 数据、services、store、utils、constants、styles、assets 等基础目录。
- 通过 services 层预留前后端分离 API 边界。
- 将 prototype 作为页面职责和视觉参考，后续逐步迁移为小程序数据驱动页面。

**Non-Goals:**

- 不引入 Taro、uni-app 或其他跨平台框架。
- 不实现真实后端接口、AI 推理、RAG、知识图谱、支付、推送或登录体系。
- 不完整迁移 prototype 的所有 UI 和交互。
- 不修改 `prototype` 目录或 `doc` 目录内容。

## Decisions

### Decision 1: Use native WeChat Mini Program with TypeScript

采用原生微信小程序结构，并使用 TypeScript 编写页面逻辑、服务封装和领域模型。

Rationale:
- 当前首要交付端是微信小程序，原生方案对订阅消息、支付、授权、分享和调试链路支持最直接。
- prototype 视觉高度定制，原生 WXSS 更利于精细还原。
- TypeScript 可为事件、指标、板块、图谱、订阅和 AI 消息建立早期数据契约。

Alternatives considered:
- Taro / uni-app: 更适合明确多端同步交付的阶段，但 MVP 当前更需要微信生态稳定性和低复杂度。
- Plain JavaScript: 初始成本更低，但领域数据结构会快速复杂化，后续接口联调和重构风险更高。

### Decision 2: Keep frontend source under `dev/apps/miniprogram`

在 `dev/apps/miniprogram` 下放置小程序源码，`dev/openspec` 继续作为规格与变更管理目录。

Target structure:

```text
dev/
├── openspec/
├── .trae/
├── apps/
│   └── miniprogram/
│       ├── app.ts
│       ├── app.json
│       ├── app.wxss
│       ├── project.config.json
│       ├── sitemap.json
│       ├── pages/
│       ├── components/
│       ├── services/
│       ├── data/
│       ├── models/
│       ├── store/
│       ├── utils/
│       ├── constants/
│       ├── styles/
│       └── assets/
├── package.json
├── tsconfig.json
├── .eslintrc.cjs
└── .prettierrc
```

Rationale:
- 保持源码工程和 OpenSpec artifacts 同属 `dev` 根目录。
- 为未来增加后端服务、管理台或 shared packages 留出 monorepo 演进空间。

### Decision 3: Use five tab pages as the first navigation shell

创建五个一级页面：`feed`、`index`、`ai`、`sectors`、`subscribe`。

Mapping:
- `feed`: 行情页，对应 prototype 的今日事件和长期跟踪。
- `index`: 指数页，对应全球定价锚和市场传导。
- `ai`: AI 助手页，对应模拟问答入口。
- `sectors`: 板块页，对应热力图和事件图谱。
- `subscribe`: 订阅页，对应主题和企业订阅。

Rationale:
- 与现有 prototype 的五个一级 screen 保持一致。
- 先建立导航壳，后续每个页面可单独通过 OpenSpec change 迁移实现。

### Decision 4: Use mock data and services as replaceable boundaries

首阶段 services 可以读取 `data/mock-*` 模块返回 Promise 或同步结构化结果，后续替换为真实 request 调用。

Rationale:
- 小程序 UI 可独立于后端建设推进。
- API DTO、领域模型和页面数据结构可提前稳定。
- 后续接入 BFF/API 时不需要大规模改页面。

### Decision 5: Convert prototype behavior to Mini Program data-driven patterns

prototype 中的 DOM 操作、`innerHTML`、`onclick`、`document`、`window`、SVG DOM 操作不能直接进入生产小程序源码。

Migration approach:
- 展开/折叠状态放入 page data 或组件 data。
- 列表使用 `wx:for` 渲染。
- 条件区域使用 `wx:if` 或 `hidden`。
- 点击事件使用 `bindtap`。
- 详情面板由状态驱动，而不是动态插入 HTML。
- 图谱能力首阶段只预留数据结构和容器，复杂绘制后续单独设计。

## Risks / Trade-offs

- [Risk] 原生小程序首版不支持跨平台复用 → [Mitigation] 通过端无关 services、models、DTO 和设计系统预留跨端迁移基础。
- [Risk] TypeScript 增加初始配置成本 → [Mitigation] 首阶段只配置必要 tsconfig 和基础类型，避免过度工程化。
- [Risk] 过早定义目录可能与未来后端联调不完全匹配 → [Mitigation] services 层只定义稳定领域边界，真实 API 契约通过后续 OpenSpec change 补充。
- [Risk] prototype 图谱和复杂动效迁移难度高 → [Mitigation] 本 change 只建立容器和数据模型，不承诺完整视觉迁移。

## Migration Plan

1. 创建小程序基础配置和 TypeScript 工程配置。
2. 创建五个 tab 页面空壳，保证小程序可启动和导航可识别。
3. 创建基础目录和 placeholder 模块，承接后续页面实现。
4. 添加 mock 数据、领域模型和 services 边界的最小样例。
5. 后续通过独立 OpenSpec change 逐页迁移 prototype 功能。

Rollback strategy: 删除本 change 新增的小程序工程文件和目录即可回退到当前 OpenSpec-only 状态。

## Open Questions

- 微信小程序 AppID 是否已确定，若未确定，`project.config.json` 使用开发占位值。
- 是否需要在首个实现 change 中立即接入 miniprogram-ci，还是仅保留配置入口。
- 图谱页面后续使用 Canvas、原生 view 绝对定位还是第三方可视化方案，需要单独设计。
