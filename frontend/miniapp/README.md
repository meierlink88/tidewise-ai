# 观潮家小程序本地预览

## 微信小程序

在当前 branch 的仓库根目录运行一条命令：

```bash
npm run preview:weapp
```

该命令顺序执行微信构建、产物门禁和固定目录发布。默认发布目录是 `~/WeChatProjects/tidewise-ai-preview`；本机实际路径为 `/Users/meierlink/WeChatProjects/tidewise-ai-preview`。如需覆盖，仅对当前命令设置 `TIDEWISE_WEAPP_PREVIEW_DIR`。

Taro 4.2 在当前 macOS 环境的 native doctor 会触发系统代理读取 panic，因此 workspace 构建脚本使用官方 `--no-check` 参数跳过 doctor 的前置配置检查；TypeScript、lint、单元测试和实际 webpack 编译仍由独立命令验证。

微信开发者工具只需首次选择“导入项目”：

- 项目目录：`/Users/meierlink/WeChatProjects/tidewise-ai-preview`
- 不再导入任何主仓库或 worktree 的 `frontend/miniapp/dist`
- AppID：使用项目可用的测试 AppID；没有 AppID 时选择开发者工具允许的测试号/本地模式
- 首页：小程序只注册 `pages/index/index`，页面标题为“今日观潮”，不显示底部菜单
- 目标截图 viewport：375×812

`preview:weapp` 必须依次报告 `Compiled successfully`、`weapp output verified` 和 `weapp preview published`。发布目录中的 `tidewise-build.json` 用于核对 branch、commit、builtAt、buildTarget 与 source app.json hash，不包含密钥。

导入或刷新后第一步只核对产物身份，不开始视觉评价：

1. 首屏必须是“今日观潮”，且没有底部菜单。
2. 预览目录 `app.json` 的 `pages` 必须精确为 `pages/index/index`，且不存在 `tabBar`。
3. `tidewise-build.json` 的 branch、commit 和 `buildTarget: "weapp"` 必须与本次发布相符。
4. 如果仍出现任何底部菜单或 provenance 不匹配，说明加载了旧发布或旧缓存；立即停止视觉验收。

## 路径或缓存错误恢复

发现产物身份不匹配时：

1. 在微信开发者工具的项目列表中删除当前误导入的旧项目记录；不要只关闭窗口。
2. 在正确 branch 重新运行 `npm run preview:weapp`。
3. 在微信开发者工具选择“清缓存”，执行“清除全部缓存”；不同版本若菜单位置不同，使用工具栏“清缓存”或“设置 → 清理缓存”的等价入口。
4. 退出当前项目，重新选择“导入项目”，只选择固定目录 `/Users/meierlink/WeChatProjects/tidewise-ai-preview`。
5. 点击“编译”；再次核对无底部菜单和 `tidewise-build.json`。身份不匹配时继续停止，不采集或提交视觉截图。

首次人工验收截图显示“行情 / 指数 / AI 助手 / 板块 / 订阅”和青色 selected state，与主仓库 2026-07-05 旧 `dist/app.json` 完全一致；它不是当前 branch UI 的有效视觉证据。正确产物身份通过后仍须重新采集 375×812 截图，不能预设当前实现已经符合 prototype2。

## 首页状态验收

ready 是默认状态。可在微信开发者工具调试控制台设置 mock 场景，然后点击“编译”刷新首页。`loading` 会稳定保持 30 秒后进入 ready，便于截图：

```javascript
wx.setStorageSync('dailyBriefMockScenario', 'ready')
wx.setStorageSync('dailyBriefMockScenario', 'loading')
wx.setStorageSync('dailyBriefMockScenario', 'empty')
wx.setStorageSync('dailyBriefMockScenario', 'error')
```

逐项检查：

1. ready：摘要折叠/展开、三条主线前后切换、影响与证据、安全声明。
2. “看图谱”：点击后只出现“推导图谱即将开放”，不导航、不请求图谱。
3. empty：显示“今日暂无可展示简报”，不显示结论、影响或证据。
4. error：显示错误与“重新加载”；改回 ready 后点击重试应恢复。
5. 采集 ready 展开、ready 折叠、三条主线、loading、empty、error 和看图谱 toast 的 375×812 截图。

## 构建产物

`frontend/miniapp/dist` 和 `~/WeChatProjects/tidewise-ai-preview` 都是本地生成目录，不提交 Git。发布器只替换目标目录内部内容，不删除目录外文件。本 change 仅验收微信小程序；既有抖音依赖与 `build:tt` 脚本保留，但不属于本次验收范围。

需要清理本地构建产物时，在仓库根目录执行：

```bash
find frontend/miniapp/dist -mindepth 1 -delete
```
