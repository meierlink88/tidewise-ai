# 观潮家 Miniapp

基于 Taro 4、React 18 和 TypeScript 的跨端小程序前端。当前只注册新版首页 `pages/index/index`，微信小程序是首要目标，同时保持抖音小程序构建可用。

## 本地开发

在仓库根目录安装依赖后，可运行：

```bash
npm --workspace @tidewise/miniapp run dev:weapp
npm --workspace @tidewise/miniapp run dev:tt
```

当前首页使用 `features/research-themes` 下的 Mock 端口。它与未来真实 Research Theme API 适配器共享同一前端合同，页面组件不直接读取 Mock 数据。

## 验证

```bash
npm --workspace @tidewise/miniapp test
npm --workspace @tidewise/miniapp run typecheck
npm --workspace @tidewise/miniapp run lint
npm --workspace @tidewise/miniapp run build:weapp
npm --workspace @tidewise/miniapp run verify:weapp-output
npm --workspace @tidewise/miniapp run build:tt
```

微信构建使用 Taro 官方 `--no-check` 跳过本机 native doctor；TypeScript、ESLint、Vitest 和 webpack 编译仍独立执行。

## 微信预览

```bash
npm --workspace @tidewise/miniapp run preview:weapp
```

默认发布到 `/Users/meierlink/Documents/WeChatProjects/tidewise-ai-preview`。微信开发者工具首次导入该目录后，后续重新运行命令并点击编译即可刷新。目标视觉基线为 375×812。

## 首页验收

- 默认展示 3 条“今日推理主线”和 13 条关联政经事件。
- 分类切换和搜索会真实过滤当前卡片。
- “跟踪中 17”是全局计数，不随当前过滤结果改变。
- 头像、问潮、历史、卡片、产业节点、事件数和“查看影响路径”只显示 Taro toast，不发生导航。
- 首页使用系统状态栏和平台胶囊，不绘制假胶囊。
- `home-header-sea.jpg` 属于旧版资产，当前首页不再构建或使用。

`dist` 与固定预览目录均为本地产物，不提交 Git。
