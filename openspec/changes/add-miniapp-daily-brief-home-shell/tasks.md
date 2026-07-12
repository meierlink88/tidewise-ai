## 1. Apply 前基线与规则门禁

- [x] 1.1 重新计算 `miniprogram.html`、`colors_and_type.css`、`components.css` 的 SHA-256，与 design 基线逐项比较；任一漂移立即暂停并提交 Review。
- [x] 1.2 盘点首页引用的图片、字体、SVG 和辅助资产，记录生产必要性、授权、固定来源、目标路径与 SHA-256；只将授权明确的必要资产纳入 Apply。
- [x] 1.3 修改 `.agents/frontend-boundaries.md`，加入 miniapp page-level canonical source、旧 `ganchaojia-design` skill 仅作历史/基础参考、prototype 只读、截图视觉验收和后续独立 skill change 规则，同时保持 admin Minimal Dashboard 路由不变。
- [x] 1.4 确认 `frontend/miniapp/` 现有 index、components、models、data、services、request、styles 和测试/构建配置的复用点，记录本 change 文件所有权，不触碰其他 tab、active change、后端或 prototype。

## 2. 候选契约与数据边界 TDD

- [x] 2.1 在 miniapp workspace 配置最小 TypeScript 单元测试入口；先添加会失败的测试，验证测试命令能发现并报告失败用例。
- [x] 2.2 先为 `DailyBriefV1`、`ReasoningConclusionV1`、`ImpactAssessmentV1`、`EvidenceItemV1` fixtures 编写类型/shape 测试，验证 mock schema version、允许的影响对象类型、不含个股推荐字段，且不存在 graph/path DTO，再实现候选 contracts。
- [x] 2.3 先为 `DailyBriefPort` 与 `MockDailyBriefAdapter` 编写 ready、empty、error 场景测试，再实现 dedicated fixtures、adapter 和 service composition root；页面不得 import fixtures。
- [x] 2.4 先为 contract-to-view-model mapper 编写市场/情绪、主题、主线、影响、证据缺失和不确定性测试，再实现首页 view model 映射。
- [x] 2.5 先为 `ResourceState` 状态转换与重试编写 idle/loading/ready/empty/error 测试，再实现首页资源状态 helper/hook。
- [x] 2.6 先为首页 section registry 编写顺序、可见条件和“registry 不含 mock 文案”测试，再实现 `brief-summary`、`themes`、`conclusions`、`impacts`、`evidence`、`safety-note` registry。

## 3. 轻量视觉基础与共享组件

- [x] 3.1 在 `frontend/miniapp/src/styles` 增量映射首页实际使用的 canonical 色阶、字体、间距、圆角、阴影、状态和 motion tokens，不复制整套 design library，并用静态扫描确认不存在远程 Google Fonts 或 Web-only CSS。
- [x] 3.2 先为共享 resource-state、chip/card/button 和安全声明组件的 props/渲染 helper 编写测试，再在现有 components 边界实现最小 primitives。
- [x] 3.3 实现首页 brief hero、波浪分隔、摘要折叠、主线、影响和证据 page compositions，保持业务文案来自 view model，不使用 `document`、`window`、`innerHTML`、内联事件或直接 DOM mutation。
- [x] 3.4 如 `home-header-sea.jpg` 已确认必要且授权明确，将其复制到 `frontend/miniapp/src/assets` 并记录来源与 SHA-256；否则暂停该视觉项并返回 Review，不自行替换 canonical 背景。

## 4. 今日观潮首页实现

- [x] 4.1 将现有 `pages/index` 占位内容增量替换为 service 驱动的“今日观潮”template；保持其他页面源码不变。
- [x] 4.2 实现 loading、empty、error、ready 四态和错误重试，确保 empty/error 不显示伪造结论、影响或证据。
- [x] 4.3 实现 canonical 摘要展开/折叠和首版主线切换，使用 Taro/React 受控状态并覆盖微信小程序交互路径。
- [x] 4.4 保留 canonical “看图谱”视觉入口；按 Review 批准策略实现“推导图谱即将开放”轻提示或等价禁用反馈，确认不导航、不请求、不渲染图谱。
- [x] 4.5 清理 fixture 中的个股、公司买卖信号、收益目标和推荐排序，只展示市场、板块、benchmark、商品、经济体或产业链实体，并在首页保持“不构成投资建议”声明。
- [x] 4.6 以自动测试锁定微信 shell 的 pages 精确为 `pages/index/index` 且构建产物不含 `tabBar`。
- [x] 4.7 只从 app config 注销其他页面与底部菜单，不删除或重构其他页面源码；未注册的“问潮”入口改为不导航的未开放反馈。

## 5. 自动验证与微信构建

- [x] 5.1 运行首页业务单元测试并确认 ready/empty/error、mapper、adapter、registry、占位策略与安全边界用例全部通过。
- [x] 5.2 运行 miniapp lint 和 TypeScript typecheck，确认无错误且源码扫描不包含 DOM/browser-only API、远程字体、硬编码 secret 或图谱实现模块。
- [x] 5.3 运行 `npm run build:weapp --workspace @tidewise/miniapp`，确认微信构建成功并记录产物路径；不提交构建产物。
- [x] 5.5 将 canonical 海面图作为独立小程序 asset 输出，以构建产物检查验证图片 SHA-256、首页样式低于 64 KiB，且不再触发图片内联体积警告。
- [x] 5.6 实现可重复的微信预览发布器，以临时目录测试默认 Documents 路径、旧文件清理、文件复制、provenance marker 与危险目标拒绝；根/miniapp `preview:weapp` 串联 build、verify、publish。

## 6. 微信开发者工具与视觉验收

- [x] 6.1 在 repo 内补充本地微信开发者工具说明，包含 `preview:weapp`、固定发布目录、首次导入、刷新、测试 AppID/本地模式和 provenance 核对步骤。
- [x] 6.1a 记录首次人工验收误导入主仓库旧 `dist` 的根因与 app.json 证据，并加入 worktree 绝对路径、tab/颜色指纹、清缓存和删除旧项目重导入门禁。
- [x] 6.1b 将默认固定验收目录迁移到 `~/Documents/WeChatProjects/tidewise-ai-preview`，保留环境变量覆盖并明确旧目录不再更新但暂不删除。
- [ ] 6.2 在微信开发者工具导入构建产物，逐项预览 ready、loading、empty、error、摘要展开/折叠、主线切换和“看图谱”占位反馈，记录可复现结果。
- [ ] 6.3 在 375×812 viewport 采集 canonical 与微信实现截图，完成首页关键视觉映射对比；逐项记录安全区、导航、字体或平台组件的等价差异，未解释偏差不得通过。

## 7. Apply 后人工 Review 门禁

- [x] 7.1 运行 `openspec validate add-miniapp-daily-brief-home-shell`、`git diff --check`、scoped `git status` 和相关新鲜验证，确认未修改 prototype、后端、数据库、其他 tab 或 active change。
- [ ] 7.2 汇总 Apply scoped diff、自动验证、微信开发者工具记录、视觉对比、资产 provenance 和未验证风险，提交完整 Apply 后人工 Review；批准前不得 Sync、Archive 或 Deliver。
