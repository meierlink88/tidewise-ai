# NervJS 仓库对 Miniapp 首页重做的参考价值

日期：2026-07-18

## 结论

NervJS 当前有 81 个公开仓库。对观潮家 Miniapp 首页重做而言：

- 2 个仓库可直接作为工程依据：`taro`、`taro-docs`。
- 7 个仓库仅可局部参考：`taro-plugin-mock`、`taro-rfcs`、`taro-doctor`、`taro-user-cases`、`taro-test-utils`、`postcss-pxtransform`、`taro-benchmark`。
- 其余 72 个仓库不应成为本次实现基线，主要原因是版本陈旧、用途不同、绑定特定平台、面向框架开发者，或与 Taro 应用无关。

本项目已经安装 Taro、微信插件和字节插件 `4.2.0`。不应重新初始化技术栈，也不应复制旧示例工程。正确做法是保留当前 Taro 4 + React + TypeScript 构建外壳，以 Taro 4.x 文档为规范，重建 Miniapp 应用源码。

## 调查方法

1. 使用 GitHub GraphQL 枚举 `NervJS` 组织全部公开仓库，返回总数为 81。
2. 对 81 个仓库读取默认分支 README 标题、仓库描述和用途。
3. 对候选仓库进一步阅读全文、检查目录或 npm 包元数据。
4. 对照本项目实际依赖版本和目标：Taro 4.2、React、TypeScript、微信优先、未来支持字节小程序、自定义观潮家 UI。
5. 不以 GitHub 仓库的 `updatedAt` 判断代码新旧；该字段可能只是元数据变化。候选仓库以 README 声明、依赖约束和实际默认分支内容为准。

主要来源：

- [NervJS 组织](https://github.com/NervJS)
- [Taro 主仓库](https://github.com/NervJS/taro)
- [Taro 4.x 项目组织](https://docs.taro.zone/docs/spec-for-taro)
- [Taro React 概述](https://docs.taro.zone/docs/react-overall)
- [Taro 组件样式约束](https://docs.taro.zone/docs/component-style/)
- [Taro 主仓库 examples](https://github.com/NervJS/taro/tree/main/examples)

## 直接采用

### `taro`

用途：Taro 框架、平台插件、编译器和当前官方 examples 的源码仓库。

对本项目的价值：

- 核验 Taro 4 React API、微信与字节平台插件、编译配置和跨端限制。
- 只按需参考 `examples/` 中的单项能力，例如自定义 TabBar、分包和平台特性。
- 不复制整个仓库，也不把框架内部 monorepo 结构套到业务应用。

决定：直接作为技术事实来源，应用结构仍以本项目边界和 Taro 4.x 文档为准。

### `taro-docs`

用途：Taro 官方文档源仓库，对应 `docs.taro.zone`。

对本项目的价值：

- 冻结项目组织、页面和组件规则、样式隔离、路由、生命周期和平台差异。
- Taro 文档建议将页面私有组件放在对应页面目录下，将真正跨页面复用的组件放在 `src/components`。
- 自定义组件样式不能假设网页端 CSS 级联行为，应避免依赖页面样式穿透组件。

决定：本次实现的首要外部规范。

## 局部参考

### `taro-plugin-mock`

[README](https://github.com/NervJS/taro-plugin-mock) 展示了通过 Taro 插件启动 HTTP mock server 的方式，但仍明确面向 Taro 2/3，npm 最新版本仅为 `0.0.10`。

决定：只参考“mock 与真实数据访问使用同一合同”的思想，不安装该插件。本期单页 mock 使用项目内类型化 adapter，更轻、更稳定，也不会给微信开发链增加一个 HTTP 进程。

### `taro-test-utils`

[README](https://github.com/NervJS/taro-test-utils) 强调从用户行为和外部表现测试组件，并支持组件、应用生命周期和多端环境。这个测试理念值得保留。

风险：npm 最新 `@tarojs/test-utils-react@0.1.1` 的 peer dependencies 是 Taro `^3.6.0`，与本项目 Taro 4.2 不匹配。

决定：参考测试边界，不在本次变更中引入该依赖。继续用 Vitest 测试纯逻辑、筛选和 mock contract；通过 Taro 构建及产物检查验证平台编译。

### `taro-doctor`

[仓库](https://github.com/NervJS/taro-doctor) 是 Doctor 能力的底层实现，README 只提供插件身份，没有应用结构示例。

决定：可把当前 Taro CLI 提供的 doctor 命令作为故障诊断工具，但不复制源码、不把它作为每次测试的必跑项。

### `taro-rfcs`

[仓库](https://github.com/NervJS/taro-rfcs) 解释 Taro 重大框架变更的设计动机。

决定：只在遇到 Taro 4 平台行为不清时追溯设计原因；不用于业务应用目录设计。

### `taro-user-cases`

[README](https://github.com/NervJS/taro-user-cases) 是案例和二维码索引，不包含可复用的应用工程。

决定：只适合观察行业案例和视觉结果，不作为源码模板。

### `postcss-pxtransform`

[仓库](https://github.com/NervJS/postcss-pxtransform) 是 Taro 尺寸转换基础能力。

决定：用于理解 `designWidth` 和尺寸转换，不单独引入或复制。当前 Taro 构建已包含相应能力。

### `taro-benchmark`

[仓库](https://github.com/NervJS/taro-benchmark) 比较 Taro 与原生小程序性能。

决定：后续出现列表性能问题时可借鉴测量方法；它不是目录、组件或页面实现模板。

## 明确不采用的重点仓库

### `taro-project-templates`

[仓库](https://github.com/NervJS/taro-project-templates) 名称看起来最像项目模板，但默认分支只包含 MobX、Redux、微信云和微信插件等旧模板。浅克隆核验的 HEAD 为 `57d6c2a`，提交时间是 2022-02-10，主要为 JavaScript 旧工程。

决定：不作为 Taro 4 React TypeScript 初始化基线。

### `taro-sample-weapp`

[README](https://github.com/NervJS/taro-sample-weapp) 的目标是 Taro 与微信原生页面、wxParse、ECharts 原生组件混合。

决定：本项目需要未来支持字节小程序，不能把微信原生混编当默认架构。仅在未来确有单个平台原生组件需求时单独研究。

### `taro-ui-sample`

[README](https://github.com/NervJS/taro-ui-sample) 演示如何打包和发布一个多端 UI 组件库，不是如何构建业务应用。

决定：不采用。观潮家已有自己的设计系统，也不需要在本期发布独立 UI 包。

### 旧 React/Hooks 应用样例

- [`taro-v2ex-hooks`](https://github.com/NervJS/taro-v2ex-hooks) 要求 Taro 1.3。
- [`taro-todomvc-hooks`](https://github.com/NervJS/taro-todomvc-hooks) 要求 Taro 1.3。
- [`taro-zhihu-sample`](https://github.com/NervJS/taro-zhihu-sample) 描述的多端状态仍处于 Taro 早期阶段。

决定：这些项目只能证明历史能力，不能指导 Taro 4 的目录、状态管理、测试或跨端实现。

### 旧 UI 兼容项目

`taro-ui-demo`、`taro-weui`、`taro-antd-mobile`、`taro-vant`、`taro3-vant-sample`、`taro-calendar` 等用于旧 UI 库或原生组件兼容。

决定：本次页面严格实现观潮家设计稿，不引入第三方 UI 库，也不复制其组件结构。

## 对观潮家 Miniapp 的工程建议

1. 保留 `src/frontend/miniapp/package.json`、`config/`、Babel、TypeScript、发布脚本和 Taro 4.2 平台插件。
2. 删除旧业务页面、组件、model、service、mock、template 和专属测试后重新实现；不要运行 `taro init` 覆盖现有 workspace。
3. 首页页面放在 `src/pages/home/`，本页专属组件放在 `src/pages/home/components/`；只有跨页面复用后才提升到 `src/components/`。
4. 建立首页专用的类型化 view model 和数据 port。mock adapter 与未来 Miniapp Backend API adapter 实现同一 port。
5. 当前只有一个首页，不引入 Redux、MobX 或新的状态管理库；使用 React 本地状态和小型纯函数完成搜索、分类与跟踪筛选。
6. 不引入 Taro UI、Vant、WeUI 或其他视觉组件库。使用 Taro 基础组件和观潮家 design tokens 实现设计稿。
7. 避免微信原生组件混编。平台差异仅收敛在导航安全区、胶囊位置等平台 adapter 中，页面和卡片保持跨端。
8. 测试分三层：纯函数与数据合同测试、关键筛选交互测试、`weapp` 与 `tt` 构建检查。不要为本次页面运行无关 Backend 全量测试。
9. 当前不需要 shared runtime、分包、SSR、React Native、Harmony 或组件库发布能力；达到实际触发条件后再评估。

## 后续用户登录与微信支付的参考边界

NervJS 组织内没有可以直接套用的生产级用户注册、登录或微信支付业务仓库。可参考内容只有两类：

1. `taro` 与 `taro-docs`：用于确认 `Taro.login`、`Taro.checkSession`、`Taro.getUserProfile`、`Taro.requestPayment` 和网络请求等客户端 API 的当前合同与平台支持情况。它们不提供用户、会话、订单、支付回调或退款等后端业务实现。
2. `taro-project-templates/wxcloud`：包含一个调用微信云函数并返回 `OPENID`、`APPID`、`UNIONID` 的早期登录片段。该模板仍使用 Nerv、React 16 和旧版 Taro 工程结构，默认分支内容最后提交于 2022 年，只适合帮助理解微信身份获取链路，不适合作为本项目的登录架构或源码模板。

进一步核验：

- `taro-apis-sample` 基于 Taro 1.2.15，默认分支最后提交于 2019 年，且代码中没有登录或支付流程。
- `taro-sample-weapp` 已核验的应用源码中没有用户登录或支付流程；它的重点是微信原生组件混编。
- 其他历史业务样例也没有覆盖微信用户注册、服务端登录态、订单创建、支付通知验签与幂等更新的完整闭环。

因此，未来实现应只把 Taro 当作客户端平台适配层：Miniapp 调用 `Taro.login` 获取 code，由 Miniapp Backend 换取微信身份并创建或绑定 Tidewise 用户；支付由 Backend 创建业务订单和微信预支付单，Miniapp 仅使用服务端返回的参数调用 `Taro.requestPayment`，最终支付状态以后端接收并验证微信支付通知为准。该能力应按 Tidewise 的 User/Auth/Payment 领域设计，不应从 NervJS 的旧样例复制业务架构。

## 81 个公开仓库附录

标记：`D` 直接依据，`C` 条件参考，`N` 本次不采用。

| # | 仓库 | 标记 | 本次判断 |
|---:|---|:---:|---|
| 1 | [taro](https://github.com/NervJS/taro) | D | 框架、平台插件、当前 examples |
| 2 | [taro-harmony-project](https://github.com/NervJS/taro-harmony-project) | N | Harmony 项目 |
| 3 | [parse-css-to-stylesheet](https://github.com/NervJS/parse-css-to-stylesheet) | N | RN/Harmony CSS 转换 |
| 4 | [nerv](https://github.com/NervJS/nerv) | N | 旧 React 替代框架 |
| 5 | [taro-plugin-platform-xhs](https://github.com/NervJS/taro-plugin-platform-xhs) | N | 小红书平台插件 |
| 6 | [taro-native-shell](https://github.com/NervJS/taro-native-shell) | N | React Native 壳 |
| 7 | [taro-project-templates](https://github.com/NervJS/taro-project-templates) | N | 默认分支模板陈旧 |
| 8 | [yoga](https://github.com/NervJS/yoga) | N | 外部布局引擎 fork |
| 9 | [engine](https://github.com/NervJS/engine) | N | Flutter engine fork |
| 10 | [tarojs-plugin-ssr](https://github.com/NervJS/tarojs-plugin-ssr) | N | H5 SSR |
| 11 | [taro-harmony-capi-library](https://github.com/NervJS/taro-harmony-capi-library) | N | Harmony C++ 库 |
| 12 | [taro-docs](https://github.com/NervJS/taro-docs) | D | 官方文档源 |
| 13 | [taro-zhihu-sample](https://github.com/NervJS/taro-zhihu-sample) | N | 早期示例 |
| 14 | [taro3-vant-sample](https://github.com/NervJS/taro3-vant-sample) | N | Taro 3 + Vant 原生兼容 |
| 15 | [taro-v2ex](https://github.com/NervJS/taro-v2ex) | N | 早期应用示例 |
| 16 | [taro-redux-sample](https://github.com/NervJS/taro-redux-sample) | N | 旧 Redux 示例 |
| 17 | [taro-plugin-mock](https://github.com/NervJS/taro-plugin-mock) | C | mock 思想可参考，版本不采用 |
| 18 | [taro-plugin-platform-weapp-qy](https://github.com/NervJS/taro-plugin-platform-weapp-qy) | N | 企业微信插件 |
| 19 | [taro-sample-weapp](https://github.com/NervJS/taro-sample-weapp) | N | 微信原生混编示例 |
| 20 | [taro-rfcs](https://github.com/NervJS/taro-rfcs) | C | 框架设计背景 |
| 21 | [taro-ui-demo](https://github.com/NervJS/taro-ui-demo) | N | 旧 Taro UI 展示 |
| 22 | [taro-v2ex-hooks](https://github.com/NervJS/taro-v2ex-hooks) | N | Taro 1.3 Hooks 示例 |
| 23 | [at-ui-nerv](https://github.com/NervJS/at-ui-nerv) | N | Nerv UI |
| 24 | [taro-ui-sample](https://github.com/NervJS/taro-ui-sample) | N | UI 库发布示例 |
| 25 | [taro-weui](https://github.com/NervJS/taro-weui) | N | WeUI 兼容 |
| 26 | [taro-components-sample](https://github.com/NervJS/taro-components-sample) | N | 旧基础组件示例 |
| 27 | [taro-ui-theme-preview](https://github.com/NervJS/taro-ui-theme-preview) | N | Taro UI 主题工具 |
| 28 | [TodoMVC](https://github.com/NervJS/TodoMVC) | N | 旧 Redux 示例 |
| 29 | [taro-component-website](https://github.com/NervJS/taro-component-website) | N | 组件文档站 |
| 30 | [taro-doctor](https://github.com/NervJS/taro-doctor) | C | 故障诊断工具 |
| 31 | [taro-plugin-platform-lark](https://github.com/NervJS/taro-plugin-platform-lark) | N | 飞书平台插件 |
| 32 | [taro-user-cases](https://github.com/NervJS/taro-user-cases) | C | 案例索引 |
| 33 | [taro-antd-mobile](https://github.com/NervJS/taro-antd-mobile) | N | Ant Design Mobile 兼容 |
| 34 | [taro-test-utils](https://github.com/NervJS/taro-test-utils) | C | 测试思想可参考，依赖不兼容 Taro 4 |
| 35 | [taro-vant](https://github.com/NervJS/taro-vant) | N | Vant 兼容 |
| 36 | [taro-uilib-react](https://github.com/NervJS/taro-uilib-react) | N | UI 库开发范例 |
| 37 | [taro-plugin-platform-kwai](https://github.com/NervJS/taro-plugin-platform-kwai) | N | 快手平台插件 |
| 38 | [taro-bot](https://github.com/NervJS/taro-bot) | N | 仓库机器人 |
| 39 | [taro-todomvc-hooks](https://github.com/NervJS/taro-todomvc-hooks) | N | Taro 1.3 Hooks 示例 |
| 40 | [nerv-webpack-boilerplate](https://github.com/NervJS/nerv-webpack-boilerplate) | N | Nerv Webpack 模板 |
| 41 | [taro-plugin-inject](https://github.com/NervJS/taro-plugin-inject) | N | 平台中间层插件 |
| 42 | [taro-plugin-platform-alipay-dd](https://github.com/NervJS/taro-plugin-platform-alipay-dd) | N | 钉钉平台插件 |
| 43 | [postcss-pxtransform](https://github.com/NervJS/postcss-pxtransform) | C | 尺寸转换原理 |
| 44 | [taro-playground](https://github.com/NervJS/taro-playground) | N | React Native playground fork |
| 45 | [nerv-redux-todomvc](https://github.com/NervJS/nerv-redux-todomvc) | N | Nerv + Redux 示例 |
| 46 | [taro-components-test](https://github.com/NervJS/taro-components-test) | N | 框架组件测试 |
| 47 | [taro-flexbox](https://github.com/NervJS/taro-flexbox) | N | 旧布局实验，官方文档更可靠 |
| 48 | [fabric](https://github.com/NervJS/fabric) | N | 外部编码规范 fork |
| 49 | [taro-issue-helper](https://github.com/NervJS/taro-issue-helper) | N | Issue 工具 |
| 50 | [awesome_notifications](https://github.com/NervJS/awesome_notifications) | N | Flutter 通知 fork |
| 51 | [taro-plugin-shared-runtime](https://github.com/NervJS/taro-plugin-shared-runtime) | N | 仅支持 Taro 3.2-3.5.1 |
| 52 | [taro-benchmark](https://github.com/NervJS/taro-benchmark) | C | 性能测量思路 |
| 53 | [ant-design](https://github.com/NervJS/ant-design) | N | 外部 UI 库 fork |
| 54 | [taro-todos-pinia](https://github.com/NervJS/taro-todos-pinia) | N | Vue + Pinia 示例 |
| 55 | [taro-calendar](https://github.com/NervJS/taro-calendar) | N | 已停止维护的日历组件 |
| 56 | [taro-plugin-indie](https://github.com/NervJS/taro-plugin-indie) | N | 独立分包插件 |
| 57 | [taro-unit-test-sample](https://github.com/NervJS/taro-unit-test-sample) | N | Taro 3 测试 demo fork |
| 58 | [nerv-website](https://github.com/NervJS/nerv-website) | N | Nerv 官网 |
| 59 | [taro-plugin-platform-alipay-iot](https://github.com/NervJS/taro-plugin-platform-alipay-iot) | N | 支付宝 IoT 插件 |
| 60 | [angle](https://github.com/NervJS/angle) | N | 图形引擎 fork |
| 61 | [taro-blended](https://github.com/NervJS/taro-blended) | N | 原生项目混编 |
| 62 | [taro-website](https://github.com/NervJS/taro-website) | N | Taro 官网 |
| 63 | [taro-apis-sample](https://github.com/NervJS/taro-apis-sample) | N | 旧 API 展示 |
| 64 | [nerv-server](https://github.com/NervJS/nerv-server) | N | Nerv SSR |
| 65 | [nerv-test-utils](https://github.com/NervJS/nerv-test-utils) | N | Nerv 测试工具 |
| 66 | [taro-todo](https://github.com/NervJS/taro-todo) | N | 旧 Todo 示例 |
| 67 | [taro-plugin-migrate](https://github.com/NervJS/taro-plugin-migrate) | N | 旧项目迁移插件 |
| 68 | [js-framework-benchmark](https://github.com/NervJS/js-framework-benchmark) | N | 外部框架 benchmark fork |
| 69 | [babel-plugin-remove-dead-code](https://github.com/NervJS/babel-plugin-remove-dead-code) | N | Babel 6 插件 fork |
| 70 | [taro-ui-bot](https://github.com/NervJS/taro-ui-bot) | N | Taro UI 机器人 |
| 71 | [quickapp-container](https://github.com/NervJS/quickapp-container) | N | 快应用容器 |
| 72 | [react-wx-images-viewer](https://github.com/NervJS/react-wx-images-viewer) | N | 移动 Web 图片组件 fork |
| 73 | [taro-ui-issue-helper](https://github.com/NervJS/taro-ui-issue-helper) | N | Issue 工具 |
| 74 | [babel](https://github.com/NervJS/babel) | N | Babel fork |
| 75 | [minify](https://github.com/NervJS/minify) | N | Babel minify fork |
| 76 | [babel-plugin-danger-remove-unused-import-taro](https://github.com/NervJS/babel-plugin-danger-remove-unused-import-taro) | N | 旧 Babel 插件 fork |
| 77 | [himalaya](https://github.com/NervJS/himalaya) | N | HTML parser fork |
| 78 | [docs](https://github.com/NervJS/docs) | N | Nerv 文档 |
| 79 | [taro-to-rn](https://github.com/NervJS/taro-to-rn) | N | 旧 RN 转换实验 |
| 80 | [uibench](https://github.com/NervJS/uibench) | N | UI benchmark fork |
| 81 | [isomorphic-ui-benchmarks](https://github.com/NervJS/isomorphic-ui-benchmarks) | N | UI benchmark fork |

## 本轮不做的事

- 不克隆或复制 81 个仓库源码。
- 不引入新的 UI、mock、状态管理或测试依赖。
- 不修改 Miniapp 页面或构建配置。
- 不把框架内部目录结构当作业务应用架构。

下一步应回到首页范围访谈，确认保留的工程外壳、首页交互边界和自定义导航策略，再形成 Miniapp ADR。
