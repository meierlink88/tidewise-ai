# Tidewise AI Coding Standard

本规范定义观潮家所有手写源码的共同编码规则。技术栈专项规范可以增加更严格的规则，
但不得放宽本规范的 ownership、类型、安全和可验证性要求。

## 1. General Principles

- 使用所属 Context 的领域语言命名；不得用框架、数据库或页面术语替代领域概念；
- 一个模块、类型或函数只承担一个可解释责任；按行为 seam 拆分，不按文件数量拆分；
- 依赖必须显式注入；禁止可变全局状态、Service Locator 和隐式初始化副作用；
- 在信任边界解析和校验不可信输入，内部代码使用已验证的 typed model；
- 默认保持确定性：排序、默认值、时间窗口、ID、去重和状态转换由程序明确拥有；
- 错误必须保留机器可判断的稳定分类，并向外提供经过清洗的安全信息；
- 不捕获后静默忽略错误、取消、interrupt、权限拒绝、持久化失败或未知状态；
- 新抽象和新依赖必须有当前责任，不为假设中的未来复用预建框架；
- 不提交 Secret、真实凭据、内部 Token、生产数据或包含敏感正文的 fixture。

## 2. Naming And Files

- package、目录、类型和函数不得使用 `common`、`utils`、`helper`、`manager`、`misc`
  等无法表达 owner 的泛化名称；
- 目录、package 和文件命名遵守所属技术栈规范；Backend 的固定结构和职责文件只由
  `kratos-backend-layout-standard.md` 定义，本文不重复。
- boolean 使用能读出真假语义的名称，如 `isReady`、`hasNext`、`enabled`；
- 时间、金额、比例、数量和单位必须在类型或名称中明确，禁止无单位的模糊数字；
- ID、状态、枚举和错误 code 使用稳定类型或受控常量，不散落 magic string；
- 注释解释 contract、原因、风险和非显然约束，不复述代码；
- TODO 必须写明 owner 或 Issue，以及移除/完成条件；不得用 TODO 隐藏当前正确性问题。

格式由仓库工具统一：

- Go 使用 `gofmt`；
- TypeScript、JavaScript、JSON、CSS/SCSS 和 Markdown 使用项目 Prettier 配置；
- 不通过手工对齐制造与 formatter 冲突的格式。

## 3. Go

### Packages And Dependencies

- package 保持单一 owner 和职责；依赖方向遵守适用的工程结构与技术栈规范；
- interface 定义在 consumer 侧，并只包含 consumer 真正使用的方法；
- 构造函数返回可立即使用的对象；必需依赖缺失时明确失败，不延迟到第一次请求；
- 不使用 `init` 完成业务注册、环境读取、数据库连接或网络调用；
- Biz 不依赖 transport、driver、具体 Provider 或 Data Adapter 类型。

### Context And Concurrency

- 可能阻塞、访问外部资源或属于请求/任务生命周期的函数以 `context.Context` 为首个参数；
- 从入口传递 Context；请求链或 Agent 节点内不得替换为 `context.Background()`；
- 每个 goroutine 必须有明确 owner、取消条件、退出路径和等待/清理责任；
- channel 的关闭方必须唯一且可识别；接收方不得关闭不属于自己的 channel；
- 不启动无法在 Service shutdown、请求取消或测试结束时回收的后台工作；
- 并发访问共享状态必须显式同步，并对受影响行为运行 race test。

### Errors

- 使用稳定 typed/sentinel error 或错误分类表达业务结果；
- 包装错误时使用 `%w`，调用方通过 `errors.Is`/`errors.As` 判断，不解析错误字符串；
- 只在能够增加 owner、operation 或安全诊断信息的边界包装一次；
- panic 只用于不可恢复的编程不变量，并由最外层 recovery 清洗；不得用于普通校验和失败；
- HTTP、SQL、URL、Token、Provider body、Prompt 和内部堆栈不得进入公开错误。

### Data And Transactions

- Biz 定义 Port 和业务原子性，Data Adapter 拥有 SQL、driver、row 和 transaction 实现；
- `sql.Tx`、pgx transaction 或数据库错误不得泄漏到 Biz/API；
- 多步写入的 transaction 边界必须与业务原子性一致；
- 查询必须定义稳定排序、分页、null 和时间语义；
- 外部/数据库结果在进入 Biz 前校验必须字段、枚举、范围和引用完整性；
- 资源按所有路径关闭；response body、rows、stream、connection 和 file 不得泄漏。

## 4. TypeScript And React

### Types

- 保持 `strict`；不得通过关闭检查或扩大 `skip` 范围解决局部类型问题；
- 禁止无约束 `any`。第三方边界确实只能返回未知值时使用 `unknown` 并立即解析收窄；
- 非空断言和宽泛类型断言只能用于已由明确不变量证明的情况，并尽量封装在边界；
- wire DTO、领域/feature model、form model 和 view model 分开，转换保持显式；
- untrusted JSON、route/query、storage 和 platform API 结果必须在边界校验；
- 状态使用受控 union/enum，避免多个 boolean 组合出非法状态。

### Components And State

- Page/Feature component 拥有用户行为；通用组件只拥有展示和可复用交互合同；
- props 保持最小、typed 和只读；不把完整 API response 或全局 store 无差别下传；
- 远端状态由 Adapter/Query seam 管理，本地交互状态留在最近的共同 owner；
- 不在 state 中保存可由 props/remote state 确定性推导的重复值；
- Effect 只用于与外部系统同步；纯计算使用普通函数或 memoization，不用 Effect 搬运数据；
- 异步 Effect/请求必须处理 cleanup、旧请求晚到、重复提交和组件卸载；
- list key 使用稳定业务 ID，不使用会随排序变化的 index；
- 用户触发的 mutation 必须提供 pending、成功/失败反馈和可安全的重试策略。

### Network And Security

- Page、View 和 presentation component 不直接调用 `fetch`、`Taro.request` 或 Provider SDK；
- 网络调用只能位于项目批准的 typed API Adapter；
- 浏览器/小程序只调用自己的 BFF，不持有下游 Service Token 或数据库凭据；
- 不把 Secret、完整凭据、Bearer Token、敏感 query 或正文写入 query key、日志和错误；
- 前端校验只改善反馈，Backend 仍是权限和业务规则的最终 owner。

### Accessibility And Interaction

- 交互控件必须有可识别名称、正确 label/description/error 关系和可见反馈；
- Web Admin 支持键盘、焦点管理和合理 responsive behavior；
- Miniapp 使用平台语义组件和可点击区域，不能只用颜色表达状态；
- loading、error、empty、retry、disabled 和 mutation 状态必须是产品设计的一部分。

## 5. SQL And Migrations

- migration 一经进入共享环境不得原地修改；修正通过新的 forward migration 完成；
- Schema、constraint、index、backfill 和代码 rollout 必须说明顺序与 mixed-version 行为；
- destructive DDL、全表 rewrite、长锁、不可逆数据转换和历史清理必须通过独立设计门禁；
- 大数据 backfill 与在线 contract change 分离，具有批次、幂等、观测和恢复策略；
- 系统部署只执行 Schema migration；Seed、目录、配置、事实回填、转换和清理必须使用独立的数据发布机制，不默认以开发环境为数据来源；
- 新 constraint 应先处理历史兼容，再验证并收紧；
- index 必须对应查询或约束责任，不因推测增加；
- 数据库 owner 的 migration 按仓库部署规则加入对应 UAT risk manifest；
- rollback 默认回退应用版本并使用向前兼容 Schema；不得假设数据库 down migration 安全。

## 6. Configuration, Logging And Secrets

- 非敏感默认配置可以进入 YAML；密码、Token、API Key 只从环境变量或批准的 Secret
  Provider 注入；
- 业务代码不散落读取环境变量，配置在 composition root 加载、校验并 typed 传入；
- 日志记录 service/domain、operation、request/execution ID、状态和必要耗时；
- 不记录 Authorization、Cookie、DSN、Prompt、模型原始响应、Connector body、完整用户
  内容或未脱敏配置；
- callback、审计记录与技术日志用途不同，不得互相替代；
- 错误日志保留内部关联，但公开响应只返回稳定安全错误。

## 7. Dependencies And Generated Code

- 新依赖必须说明 owner、用途、版本、替代方案、安全和许可证影响；
- 固定可重现版本，不静默跟随 `latest`，不因示例使用而引入依赖；
- Provider/Framework 具体类型限制在 Adapter/composition root，不传播到业务模型；
- 生成文件必须标识来源和再生成命令；不得直接手改会被覆盖的生成输出；
- OpenAPI、wire DTO、注册逻辑、fixture 和 consumer contract 必须在同一变更保持一致。

## 8. Verification

编码质量检查不能替代行为测试，行为测试也不能替代静态检查。完成时按受影响范围执行：

- Go：`gofmt`、`go vet`、受影响测试、必要 race 和 binary build；
- Frontend：Prettier、lint、typecheck、受影响测试和目标 build；
- API：OpenAPI/runtime/fixture/provider-consumer 合同；
- Migration：forward chain、约束/历史兼容和 UAT risk classification；
- 安全：Secret scan、依赖审查和敏感错误/日志检查；
- 架构：目录、import、Adapter、Service 和 framework placement 门禁。

项目当前工具未自动覆盖的规则仍然是 review 必查项，不因缺少 lint rule 而失效。
纯开发规范内容变更不运行上述产品验证或 CI 门禁。
