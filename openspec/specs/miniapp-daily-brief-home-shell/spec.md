# Miniapp Daily Brief Home Shell Specification

## Purpose

定义微信小程序“今日观潮”Mock 首页的展示、交互、数据边界、canonical 视觉转译、资产准入与本地预览验收要求。

## Requirements

### Requirement: 今日观潮首页
系统 SHALL 将现有 index page 作为微信小程序唯一注册页面提供“今日观潮”，以日报摘要、市场与情绪状态、主题、主线结论、影响判断和证据摘要帮助用户理解当日全球政经事件及其市场传导；生产 `app.json` 不含 `tabBar`，其他页面源码保持不变但不进入本期构建或导航。

#### Scenario: 启动小程序
- **WHEN** 用户启动微信小程序
- **THEN** 系统首先打开 `pages/index/index`，页面列表精确只有该首页且不显示底部菜单

#### Scenario: 查看今日简报
- **WHEN** 用户打开今日观潮首页且今日简报可用
- **THEN** 页面展示简报时间、摘要、市场状态、情绪状态、主题和至少一条主线结论

#### Scenario: 查看主线内容
- **WHEN** 用户切换到另一条首页主线
- **THEN** 页面按 canonical 交互更新主线结论、影响与证据摘要，不导航到新页面

#### Scenario: 展开或收起摘要
- **WHEN** 用户操作首页摘要折叠入口
- **THEN** 页面按 canonical 视觉与反馈展开或收起扩展内容

### Requirement: 看图谱占位入口
系统 SHALL 保留 canonical 首页中的“看图谱”视觉入口，但本 change 不提供图谱页、图谱数据加载或图谱交互，并必须用不误导用户的占位策略表达能力尚未开放。

#### Scenario: 点击看图谱
- **WHEN** 用户点击“看图谱”入口
- **THEN** 页面展示“推导图谱即将开放”或经 Review 批准的等价轻提示，且不导航、不渲染虚假图谱、不发起图谱请求

#### Scenario: 入口具备可识别状态
- **WHEN** 用户查看主线卡中的“看图谱”入口
- **THEN** 入口保持与 canonical 一致的视觉位置与层级，并通过点击反馈或辅助文案说明本期未开放

### Requirement: 首页资源状态
系统 SHALL 为今日观潮首页提供 loading、empty、error 和 ready 四种明确状态，并由 service 返回结果驱动。

#### Scenario: 数据加载中
- **WHEN** 首页正在等待 service 返回简报
- **THEN** 页面展示与 canonical 页面结构匹配的加载状态，且不显示硬编码 ready 内容

#### Scenario: 今日无简报
- **WHEN** service 返回今日没有可展示简报
- **THEN** 首页展示明确空状态，且不伪造市场结论、影响或证据

#### Scenario: 加载失败并重试
- **WHEN** service 返回可展示错误
- **THEN** 首页展示简短错误说明和重试入口，并在用户重试时重新请求当前简报

### Requirement: 影响对象与投资建议安全边界
系统 SHALL 仅以市场理解和决策辅助方式展示影响判断，第一版影响对象只能是市场、板块、benchmark、商品、经济体或产业链实体，不得展示具体股票推荐或买卖建议。

#### Scenario: 展示影响判断
- **WHEN** 首页展示方向、强度、时间范围或不确定性
- **THEN** 页面同时表达判断依据或证据摘要与不确定性，并持续展示“不构成投资建议”的安全定位

#### Scenario: 排除不允许对象
- **WHEN** mock 数据拟包含个股、公司买卖信号、目标收益或推荐排序
- **THEN** 该数据不得进入可展示 fixture，必须替换为允许的宏观或市场对象

#### Scenario: 状态不只依赖颜色
- **WHEN** 页面用红、绿或其他颜色表达方向和状态
- **THEN** 页面同时使用文字、标签或图标表达同一含义

### Requirement: 首页证据摘要
系统 SHALL 在首页展示与当前主线结论或影响判断关联的证据摘要，使用户能够区分事实材料、推导结论和影响判断。

#### Scenario: 查看关联证据
- **WHEN** 当前主线具有关联证据
- **THEN** 首页展示证据来源、摘要或标题、时间和可信度标签

#### Scenario: 缺少证据
- **WHEN** 某个影响判断没有关联证据
- **THEN** 页面明确显示“暂无关联证据”或等价状态，不得把无来源文案伪装为已验证事实

### Requirement: Mock-only 首页候选契约
系统 SHALL 以版本化前端候选类型表达 `DailyBriefV1`、`ReasoningConclusionV1`、`ImpactAssessmentV1`、`EvidenceItemV1` 及首页确有需要的引用结构，并明确它们只服务于本地 Mock shell，不是冻结的后端 API 契约。

#### Scenario: 识别候选契约版本
- **WHEN** 开发者检查 mock adapter 输入结构
- **THEN** 顶层 payload 带有明确 mock schema version，类型或模块标识 mock-only 边界，且不存在 `ReasoningGraphV1` 或 `ReasoningPathStepV1`

#### Scenario: 不推定后端契约
- **WHEN** 后续 change 接入真实 Go API/BFF
- **THEN** 必须另行 Review DTO、错误、ID、时间、枚举、鉴权和版本兼容策略

### Requirement: 可替换首页数据 adapter
系统 SHALL 让首页依赖稳定 daily brief service port 和页面 view model，并由 mock adapter 提供演示数据，使未来 HTTP adapter 可以替换 mock adapter而无需重写页面。

#### Scenario: 页面读取简报
- **WHEN** 首页加载数据
- **THEN** 页面通过 service 入口请求数据，不得直接 import dedicated mock fixture 或在 JSX 中硬编码业务数据

#### Scenario: 替换 adapter
- **WHEN** 后续 API integration change 提供符合 service port 的 HTTP adapter
- **THEN** 首页 template、section registry 和展示组件无需因 transport 替换而改写

### Requirement: 首页模板与 section registry
系统 SHALL 使用页面 template 和 section registry 控制首页 section 顺序、可见条件和渲染边界，业务数据必须来自 typed view model。

#### Scenario: 渲染 ready 首页
- **WHEN** 首页获得 ready view model
- **THEN** registry 按确定顺序渲染摘要、主题、主线、影响、证据和安全声明，且 registry 不包含 mock 业务文案

### Requirement: Canonical 视觉转译
系统 SHALL 将固定路径 `/Users/meierlink/Documents/david/创业项目/观潮家/prototype2/miniprogram.html` 的首页最终渲染作为 page-level canonical source，在既有 miniapp 工程内做 Taro/React 等价转译。

#### Scenario: 转译 prototype 交互
- **WHEN** 实现摘要折叠、主线切换、“看图谱”占位反馈或重试
- **THEN** 使用 React state、组件事件、条件渲染和 Taro API，不得使用 `document`、`window`、`innerHTML`、直接 DOM mutation 或内联 `onclick`

#### Scenario: 设计源漂移
- **WHEN** Apply 开始前 canonical HTML 或关联 CSS 的 SHA-256 不同于 proposal 基线
- **THEN** 实现暂停并提交差异 Review

#### Scenario: 旧 skill 与页面冲突
- **WHEN** 旧 `ganchaojia-design` skill 与 canonical 首页最终渲染冲突
- **THEN** 页面效果服从 prototype2，旧 skill 只作为历史和基础 token 参考

### Requirement: 视觉基线验收
系统 SHALL 以 375×812 canonical 截图基线和视觉对比验收首页，只接受有记录的小程序平台等价差异。

#### Scenario: 验收 ready 首页
- **WHEN** ready 首页准备进入 Apply 后 Review
- **THEN** evidence 包含 canonical 与生产实现截图对比，覆盖摘要展开/折叠、每条首版主线和“看图谱”占位反馈

#### Scenario: 验收资源状态
- **WHEN** loading、empty 或 error 状态准备进入 Review
- **THEN** 每种状态提供与 canonical token、页面层级和 viewport 一致的截图基线

#### Scenario: 平台不可避免差异
- **WHEN** 微信安全区、导航、字体渲染或组件行为造成差异
- **THEN** evidence 记录原因、影响和等价处理，未记录偏差不得通过

### Requirement: 轻量生产设计抽象与资产准入
系统 SHALL 只在 `frontend/miniapp/src/styles`、现有 components 和 page compositions 中提炼首页需要的设计抽象；原型资产只有生产必要且授权明确时才能复制，并记录来源与 SHA-256。

#### Scenario: 提炼设计值
- **WHEN** 页面需要 canonical 的颜色、文字、间距、圆角、阴影、状态或 motion
- **THEN** 只映射生产首页实际使用的值，不复制整套 `.design_library`

#### Scenario: 复制必要资产
- **WHEN** 原型资产被证明为首页基线所必需且授权明确
- **THEN** 可以复制到 miniapp assets，并记录绝对来源、目标路径、用途和 SHA-256

#### Scenario: 资产授权不明确
- **WHEN** 必要图片或字体授权无法确认
- **THEN** 不得复制，视觉影响必须返回人工 Review

#### Scenario: 延后 design skill
- **WHEN** 首页已稳定并通过视觉验收
- **THEN** repo-local miniapp design skill 由后续独立 change 评估

### Requirement: 微信构建及本地预览
系统 SHALL 使今日观潮首页在微信小程序目标成功编译，并提供固定发布目录、build provenance 与产物身份核对门禁。本 change 不要求抖音构建或预览；既有抖音依赖和脚本可以保留。

#### Scenario: 构建微信小程序
- **WHEN** 开发者运行微信构建命令
- **THEN** 今日观潮首页及共享模块成功编译

#### Scenario: 导入微信开发者工具
- **WHEN** 开发者按 repo 说明导入微信构建产物
- **THEN** 首次只导入默认 `~/Documents/WeChatProjects/tidewise-ai-preview` 或环境变量覆盖目录，后续通过 `preview:weapp` 更新并在开发者工具编译/刷新

#### Scenario: 产物身份不匹配
- **WHEN** 开发者工具显示任何底部菜单、`app.json` 注册非首页页面，或 provenance 与预期 branch/commit/build target 不符
- **THEN** 必须判定为发布目录或缓存错误，停止视觉验收，重新发布并清缓存后从固定目录刷新或重导入

#### Scenario: 发布微信预览
- **WHEN** 开发者运行根或 miniapp workspace 的 `preview:weapp`
- **THEN** 系统按 build→verify→publish 顺序把当前 weapp `dist` 可重复同步到固定目录，清除目标内旧产物但不触碰目录外文件，并写入 branch、commit、builtAt、build target 与 source app.json hash

#### Scenario: 采集微信验收证据
- **WHEN** 固定预览目录的产物身份核对通过
- **THEN** 可以打开 index 首页、切换 mock 四态、预览 canonical 交互并采集 375×812 验收截图
