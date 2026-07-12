# 观潮家小程序本地预览

## 微信小程序

在仓库根目录顺序运行：

```bash
npm run build:weapp
```

Taro 4.2 在当前 macOS 环境的 native doctor 会触发系统代理读取 panic，因此 workspace 构建脚本使用官方 `--no-check` 参数跳过 doctor 的前置配置检查；TypeScript、lint、单元测试和实际 webpack 编译仍由独立命令验证。

打开微信开发者工具并选择“导入项目”：

- 项目目录：`/Users/meierlink/.codex/worktrees/ed41/tidewise-ai/frontend/miniapp/dist`
- AppID：使用项目可用的测试 AppID；没有 AppID 时选择开发者工具允许的测试号/本地模式
- 首页：进入底部 `指数` tab，对应 `pages/index/index`，页面标题为“今日观潮”
- 目标截图 viewport：375×812

## 首页状态验收

ready 是默认状态。可在微信开发者工具调试控制台设置 mock 场景，然后切换到其他 tab 再返回 `指数` tab。`loading` 会稳定保持 30 秒后进入 ready，便于截图：

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

## 抖音小程序

微信验收结束后运行：

```bash
npm run build:tt
```

微信与抖音构建共用 `frontend/miniapp/dist`，必须顺序运行，后一次构建会替换前一次产物。不要并行执行两个构建。

## 构建产物

`frontend/miniapp/dist` 是本地生成目录，不提交 Git。需要重新导入微信开发者工具时，最后一次命令必须是 `npm run build:weapp`。

需要清理本地构建产物时，在仓库根目录执行：

```bash
find frontend/miniapp/dist -mindepth 1 -delete
```
