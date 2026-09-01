# 观潮家 Miniapp

基于 Taro 4、React 18 和 TypeScript 的跨端小程序前端。微信小程序是首要目标，同时保持
抖音小程序构建可用。

当前 Report 展示链路包含三个页面：保留既有应用外壳的“今日推理”首页、推理详情页和
相关证据页。首页按 Report 分组展示当天全部已发布卡片；当天没有 Report 时，由 Miniapp
Backend 返回最新一份并明确标记。详情页只用 Report 内的结构化引用跳转，相关证据页只展示
Backend 从 PostgreSQL 持久化 Evidence 投影出的发布时间、摘要和关键词。

## 本地开发

Taro watch/build 使用仓库 lockfile 固定的 Node/Taro 依赖直接运行。每次开发或构建必须显式
选择 Report 数据源：

```bash
TARO_APP_REPORT_SOURCE=mock npm run dev:weapp
TARO_APP_REPORT_SOURCE=mock npm run dev:tt
```

`TARO_APP_REPORT_SOURCE` 只允许 `mock` 或 `api`。`mock` 使用样例报告派生的小型验收数据，
包含地缘政治、宏观经济和 CHN-01/02/03/21 四条产业链；`api` 只访问 Miniapp Backend，
请求失败或响应合同不合法时不会隐式回退 mock。

接入本地 Miniapp Backend 时先启动 Backend，再运行 Taro：

```bash
npm run backend:dev:miniapp
TARO_APP_REPORT_SOURCE=api \
  TARO_APP_MINIAPP_API_BASE_URL=http://127.0.0.1:9012 \
  npm run dev:weapp
```

浏览器快速预览 mock 时运行：

```bash
TARO_APP_REPORT_SOURCE=mock npm run dev:h5
```

浏览器联调 Backend 时运行：

```bash
TARO_APP_REPORT_SOURCE=api npm run dev:h5
```

然后打开 `http://localhost:10086/`。H5 开发服务会把同源 `/api` 请求代理到本地 Miniapp
Backend `http://127.0.0.1:9012`。可通过 `TARO_APP_H5_API_PROXY_TARGET` 覆盖代理目标。
Miniapp Frontend 不保存 Data Service token，也不直接访问 Data Service。

## 验证

```bash
npm --workspace @tidewise/miniapp test
npm --workspace @tidewise/miniapp run typecheck
npm --workspace @tidewise/miniapp run lint
TARO_APP_REPORT_SOURCE=mock npm --workspace @tidewise/miniapp run build:weapp
npm --workspace @tidewise/miniapp run verify:weapp-output
TARO_APP_REPORT_SOURCE=mock npm --workspace @tidewise/miniapp run build:tt
npm --workspace @tidewise/miniapp run verify:tt-output
TARO_APP_REPORT_SOURCE=mock npm --workspace @tidewise/miniapp run build:h5
```

微信构建使用 Taro 官方 `--no-check` 跳过本机 native doctor；TypeScript、ESLint、Vitest 和
webpack 编译仍独立执行。微信、抖音构建产物分别位于 `dist/weapp` 和 `dist/tt`，互不覆盖。

## 微信预览

```bash
TARO_APP_REPORT_SOURCE=mock npm --workspace @tidewise/miniapp run preview:weapp
```

微信开发者工具直接导入 `miniapp/frontend/dist/weapp`。微信发布仍由微信开发者工具或既有
平台发布流程完成，不是 Docker 部署。构建目录自带 `project.config.json` 和微信测试 AppID。

当前页面验收边界：

- 保留系统状态栏、平台胶囊、观潮导航、头像、搜索和问潮入口；
- 首页按 Backend `display_order` 展示每份 Report 的地缘政治、宏观经济和产业链卡片，不跨
  Report 合并；
- 推理详情支持 layer 与 industry_chain 两种目标，并只渲染报告显式发布的链路边；
- Evidence 使用对象自己的 lowercase Report-local scope key 请求，页面不展示 Evidence ID、
  Event、来源或总数；
- loading、error、empty、pull-to-refresh 和晚到请求保护由三个页面共享；
- weapp 与 tt 构建产物只注册首页、推理详情、相关证据三个路由，旧 Theme/Reason Tree 页面
  与资源必须不存在。

`dist` 是可再生成的本地产物，不提交 Git。H5 适合快速验收布局；微信登录、支付、胶囊、
授权及其他平台能力仍需在对应开发者工具或真机中验收。
