# 观潮家 Miniapp

基于 Taro 4、React 18 和 TypeScript 的跨端小程序前端。当前只注册新版首页 `pages/index/index`，微信小程序是首要目标，同时保持抖音小程序构建可用。

## 本地开发

在仓库根目录安装依赖后运行。首页通过 `features/research-themes` 下的统一端口读取数据，页面组件不直接读取 Mock 或调用 HTTP。每次开发或构建都必须显式选择数据源。独立视觉开发使用 Mock：

```bash
TARO_APP_RESEARCH_SOURCE=mock \
npm --workspace @tidewise/miniapp run dev:weapp

TARO_APP_RESEARCH_SOURCE=mock \
npm --workspace @tidewise/miniapp run dev:tt
```

接入本地 Miniapp Backend 时选择 API：

```bash
TARO_APP_RESEARCH_SOURCE=api \
TARO_APP_MINIAPP_API_BASE_URL=http://127.0.0.1:9012 \
npm --workspace @tidewise/miniapp run dev:weapp
```

浏览器快速预览真实 API 数据时运行 H5 开发服务：

```bash
TARO_APP_RESEARCH_SOURCE=api \
TARO_APP_MINIAPP_API_BASE_URL=http://localhost:10086 \
npm run dev:h5
```

然后打开 `http://localhost:10086/`。H5 开发服务会把同源 `/api` 请求代理到本地 Miniapp Backend `http://127.0.0.1:9012`，浏览器不会直接访问 Data Service，也不需要额外处理 CORS。可通过 `TARO_APP_H5_API_PROXY_TARGET` 覆盖代理目标。

`TARO_APP_RESEARCH_SOURCE` 仅允许 `mock` 或 `api`。API 模式调用 Miniapp Backend 的 `/api/v1/miniapp/research/themes`，请求失败会展示错误状态，不会静默回退 Mock。前端不保存 Data Service token，也不直接调用 Data Service。

## 验证

```bash
npm --workspace @tidewise/miniapp test
npm --workspace @tidewise/miniapp run typecheck
npm --workspace @tidewise/miniapp run lint
TARO_APP_RESEARCH_SOURCE=mock npm --workspace @tidewise/miniapp run build:weapp
npm --workspace @tidewise/miniapp run verify:weapp-output
TARO_APP_RESEARCH_SOURCE=mock npm --workspace @tidewise/miniapp run build:tt
TARO_APP_RESEARCH_SOURCE=mock npm --workspace @tidewise/miniapp run build:h5
```

微信构建使用 Taro 官方 `--no-check` 跳过本机 native doctor；TypeScript、ESLint、Vitest 和 webpack 编译仍独立执行。微信、抖音构建产物分别位于 `dist/weapp` 和 `dist/tt`，互不覆盖。

## 微信预览

```bash
TARO_APP_RESEARCH_SOURCE=mock npm --workspace @tidewise/miniapp run preview:weapp
```

微信开发者工具直接导入仓库内的 `src/frontend/miniapp/dist/weapp`。后续运行 `dev:weapp` 会持续编译到该目录，开发者工具可直接刷新。目标视觉基线为 375×812。

构建目录自带 `project.config.json` 和微信测试 AppID，无需手工创建项目配置。

## 首页验收

- 默认展示 3 条“今日推理主线”和 13 条关联政经事件。
- 分类切换和搜索会真实过滤当前卡片。
- “跟踪中 17”是全局计数，不随当前过滤结果改变。
- 头像、问潮、历史、卡片、产业节点、事件数和“查看影响路径”只显示 Taro toast，不发生导航。
- 首页使用系统状态栏和平台胶囊，不绘制假胶囊。
- `home-header-sea.jpg` 属于旧版资产，当前首页不再构建或使用。

真实 API 模式下，分类标签和“跟踪中 17”仍是 Frontend-owned 临时展示数据；主题内容、关联产业、事件计数和更新时间来自 Miniapp Backend。

`dist` 为本地构建产物，不提交 Git。

H5 适合快速验收布局、交互和真实 API 数据；微信登录、支付、胶囊、授权及其他平台专属能力仍需在微信开发者工具或真机中验收。
