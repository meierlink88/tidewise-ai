# 观潮家小程序本地预览

## 微信小程序

在仓库根目录顺序运行：

```bash
npm run build:weapp
```

Taro 4.2 在当前 macOS 环境的 native doctor 会触发系统代理读取 panic，因此 workspace 构建脚本使用官方 `--no-check` 参数跳过 doctor 的前置配置检查；TypeScript、lint、单元测试和实际 webpack 编译仍由独立命令验证。

打开微信开发者工具并选择“导入项目”：

- 项目目录只允许使用当前 task worktree：`/Users/meierlink/.codex/worktrees/ed41/tidewise-ai/frontend/miniapp/dist`
- 禁止导入主仓库旧目录：`/Users/meierlink/Documents/david/创业项目/观潮家/tidewise-ai/frontend/miniapp/dist`
- AppID：使用项目可用的测试 AppID；没有 AppID 时选择开发者工具允许的测试号/本地模式
- 首页：小程序启动后默认进入底部第一个 `首页` tab，对应 `pages/index/index`，页面标题为“今日观潮”
- 目标截图 viewport：375×812

构建后、导入前先在仓库根目录运行产物身份门禁：

```bash
npm run verify:weapp-output --workspace @tidewise/miniapp
```

命令必须报告 `weapp output verified`。导入后第一步只核对产物身份，不开始视觉评价：

1. tab 必须依次为“首页 / 行情 / AI 助手 / 板块 / 订阅”。
2. 首屏必须是“今日观潮”，底部“首页”为蓝色选中态（配置值 `#2563eb`）。
3. 如果仍显示“行情 / 指数 / AI 助手 / 板块 / 订阅”或青色选中态 `#0f766e`，说明加载了主仓库旧 `dist` 或旧缓存；立即停止视觉验收。

## 路径或缓存错误恢复

发现产物身份不匹配时：

1. 在微信开发者工具的项目列表中删除当前误导入的旧项目记录；不要只关闭窗口。
2. 重新运行 `npm run build:weapp`，随后运行上述 `verify:weapp-output` 门禁。
3. 在微信开发者工具选择“清缓存”，执行“清除全部缓存”；不同版本若菜单位置不同，使用工具栏“清缓存”或“设置 → 清理缓存”的等价入口。
4. 退出当前项目，重新选择“导入项目”，复制粘贴完整 worktree 绝对路径 `/Users/meierlink/.codex/worktrees/ed41/tidewise-ai/frontend/miniapp/dist`，不要从最近项目或主仓库目录打开。
5. 点击“编译”；再次核对五个 tab 的顺序、文案和蓝色首页选中态。身份不匹配时继续停止，不采集或提交视觉截图。

首次人工验收截图显示“行情 / 指数 / AI 助手 / 板块 / 订阅”和青色 selected state，与主仓库 2026-07-05 旧 `dist/app.json` 完全一致；它不是当前 branch UI 的有效视觉证据。正确产物身份通过后仍须重新采集 375×812 截图，不能预设当前实现已经符合 prototype2。

## 首页状态验收

ready 是默认状态。可在微信开发者工具调试控制台设置 mock 场景，然后切换到其他 tab 再返回 `首页` tab。`loading` 会稳定保持 30 秒后进入 ready，便于截图：

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

`frontend/miniapp/dist` 是本地生成目录，不提交 Git。本 change 仅验收微信小程序；既有抖音依赖与 `build:tt` 脚本保留，但不属于本次验收范围。需要重新导入微信开发者工具时，最后一次构建命令必须是 `npm run build:weapp`。

需要清理本地构建产物时，在仓库根目录执行：

```bash
find frontend/miniapp/dist -mindepth 1 -delete
```
