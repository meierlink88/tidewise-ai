# Miniapp Frontend Development Standard

本规范是观潮家 Miniapp Frontend 的仓库级技术权威。它适用于 Taro 页面、组件、导航、
平台 API、状态、数据 Adapter、样式、性能、测试、构建和发布。

## 1. Baseline And Authority

- Framework：Taro 4 + React 18 + TypeScript strict；
- Target：WeChat (`weapp`) first，同时保留 Douyin (`tt`) compatibility；
- UI：观潮家自有设计语言和项目源码组件，不默认采用通用小程序 UI library；
- Backend：独立 Go Miniapp Application Backend Service；
- Data：真实 API 与 mock 实现相同 typed frontend Port；
- State：优先本地 React state 和 feature-owned typed state，不默认引入全局 store。

每个 material Miniapp 变更在设计前必须执行项目 `$taro-reference-first`：

1. 检查 `miniapp/frontend/package.json`、lockfile、Taro config 和受影响实现；
2. 读取 `.codex/skills/taro-reference-first/references/source-catalog.md`；
3. 对不稳定 API、平台兼容或新能力检查当前 Taro 4.x 官方文档和精确示例；
4. 记录采用、拒绝、版本/平台限制和项目落地方式。

官方示例提供框架证据，不拥有观潮家的目录、领域、Backend 或设计系统。

## 2. Engineering Structure

稳定目录职责：

```text
miniapp/frontend/src/
├── pages/       # Taro 路由入口、页面组合和页面局部组件
├── features/    # 业务 feature contract、状态、编排、Adapter 和 presentation
├── platform/    # 与业务无关的微信/抖音平台能力 Adapter
├── mocks/       # 实现 feature Port 的开发/测试数据
├── styles/      # design token、base style 和跨页面样式基础
├── assets/      # 项目拥有的静态资源
├── app.tsx
└── app.config.ts
```

依赖方向：

```text
page / page-local component
  -> feature public contract / state / presentation
    -> typed Port
      -> API Adapter or Mock Adapter
        -> Miniapp Backend

page / feature
  -> narrow platform Adapter
    -> Taro / weapp / tt API
```

规则：

- Page 负责路由参数、页面生命周期、feature 组合和用户可见状态；
- Page-local component 只服务当前页面，未形成稳定复用前不提升为全局组件；
- Feature 拥有业务输入/输出、typed contract、状态转换、presentation mapping 和 Port；
- 网络调用只允许位于 feature API Adapter 或批准的窄 request helper；
- `Taro.request`、`fetch`、Provider SDK 和 Backend URL 不得出现在 Page/View component；
- 与业务无关的平台差异进入 `platform/`；业务 feature 不散落 `process.env.TARO_ENV`
  分支；
- Mock 与 API Adapter 实现同一 Port，API 失败不得静默回退到 Mock；
- 跨 Feature 共享只允许稳定的 UI primitive、typed platform mechanism 或冻结合同；
  不创建无 owner 的 `common/utils/services`。

## 3. Component Layers

Miniapp 使用三层组件：

### Platform and UI primitives

薄封装 Taro 基础组件、系统安全区、状态栏、图片或平台差异。它们不包含 Research Theme、
Reason Tree 等领域语义。

### Product compositions

表达观潮家稳定产品组合，例如页面壳、状态卡、错误/重试、分页/加载和导航操作。只有在
至少两个真实消费者拥有相同合同，或设计系统已明确批准时才提升为共享组合。

### Feature and page components

拥有 Theme、Reason Tree、Tracking 等具体产品语义。不得为复用外观把业务状态转换、
API 调用或导航规则下沉进通用组件。

禁止：

- 引入 Taro UI、Vant、NutUI、Ant Design Mobile 等平行 UI 系统，除非有单独接受的决策；
- 从示例项目复制 Redux/MobX、云开发 Backend、路由器或应用目录；
- 用一个通用 Schema/JSON renderer 代替显式产品组件。

## 4. State And Async Behavior

- UI transient state 放在最近的 Page/Component；
- 可复用业务状态机或 request session 放在 owning Feature；
- Remote response 经 wire validation 和 presentation mapping 后进入页面；
- 不把可推导值重复保存为 state；
- 每个异步请求必须定义 loading、ready、error、retry 和 stale/late response 行为；
- 以 resource ID 缓存时必须说明缓存生命周期、失效和重新进入页面的行为；
- 旧请求晚到不得覆盖新的 route 参数、选中项或 refresh；
- 重复提交期间禁用或去重，mutating operation 必须保留安全失败反馈；
- 不引入全局 store，除非多个路由需要共享长期状态且 lifecycle/ownership 已冻结。

## 5. Navigation And Platform APIs

- 路由只在 `app.config.ts` 注册，路径和 query contract 由 owning page 明确；
- route/query/storage 输入是不可信输入，必须解析、校验并处理缺失或非法值；
- 使用 Taro 官方导航 API，不引入自定义 Router 解决当前不存在的 Web deep-link 问题；
- 平台差异先寻找 Taro shared API；确有差异时通过 narrow Adapter 隔离；
- Adapter 必须明确 `weapp` 与 `tt` 的支持、降级或不支持行为；
- 权限、登录、支付、剪贴板、文件、相册和位置等敏感 API 必须有用户触发、拒绝处理和
  安全错误，不在页面加载时隐式扩大权限；
- 不使用 H5 行为推断小程序行为，目标平台必须分别构建和验证。

## 6. Data And Backend Boundary

- Miniapp Frontend 只调用 Miniapp Backend 的版本化 API；
- 不直连 Data、AgentRun、PostgreSQL、Neo4j、外部模型或信息 Provider；
- API Adapter 拥有 URL、header、wire DTO、timeout、response/error mapping；
- Feature Port 使用产品语言，不暴露 HTTP status、Backend envelope 或 Taro response；
- Backend Request ID 可以用于安全诊断，但不得向用户展示内部错误、URL 或 Token；
- null、时间、顺序、分页和错误语义按 Miniapp contract 显式映射；
- Frontend validation 只提供即时反馈，Backend 是权限和业务规则的最终 owner。

## 7. Styling And Assets

- `src/styles/tokens.scss` 是颜色、字号、间距、圆角、阴影和层级的源码 token 入口；
- Page 样式复用 token，不散落新的品牌色或不可追踪 magic value；
- 视觉需求优先使用已接受原型和观潮家设计语言，不复制参考案例的品牌与演示资源；
- 图片明确尺寸、裁剪、压缩、失败和占位行为，避免无界原图进入首屏；
- 文本默认支持自然换行和系统字体缩放；截断必须是明确产品决定；
- 状态不能只通过颜色表达，可点击区域和反馈适配触屏使用。

## 8. Performance

按证据逐级优化：

1. 先分页或限制返回数据；
2. 限制 state 更新和组件重渲染范围；
3. 优化图片、静态资源和分包；
4. 测量目标设备/开发者工具；
5. 只有测量证明必要时采用 CompileMode、virtual list 或平台专用优化。

不得为了假设性能问题引入复杂缓存、全局状态、列表框架或平台分叉。长列表必须定义稳定
key、分页/加载终止、空态和重复项处理。

## 9. Verification

默认验证：

- `npm run typecheck:miniapp`；
- `npm run lint`；
- 受影响 Feature、Adapter、Page 和 Component 测试；
- `npm run build:weapp`；
- `npm run build:tt`；
- 用户可见 loading、error、empty、retry 和 navigation 行为。

条件验证：

- 平台 API：weapp/tt Adapter 合同和拒绝/降级；
- 路由：参数缺失、非法、重复进入和返回；
- 长列表：分页、重复、结束、失败恢复和目标平台性能；
- Auth/Payment/Storage：权限、过期、重复提交、取消和敏感信息；
- 跨边界 API：provider-consumer fixture 和真实 BFF smoke；
- 视觉交互：目标平台页面 smoke、触屏区域、文本换行和安全区。

纯 CSS selector、Taro 内部实现和机械 mapping 不单独测试；可观察行为仍必须在 Page、
Feature 或 Adapter seam 得到覆盖。

## 10. Fast Path

纯文案、颜色、间距和已采用组件内的局部 presentation 修改可以：

1. 检查受影响组件和当前 Taro 版本；
2. 复用现有 token、组件和 source catalog；
3. 不新增依赖、架构或远程调研；
4. 运行 focused type/lint/build 或必要页面测试。

一旦变更涉及行为、路由、数据、平台 API、认证、支付、性能或跨平台兼容，立即回到完整
reference-first 和设计路径。
