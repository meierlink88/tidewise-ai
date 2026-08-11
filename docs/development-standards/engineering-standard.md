# Tidewise AI Engineering Standard

本规范只定义跨技术栈的工程治理：权威、行为所有权、设计门禁、跨边界合同与完成
标准。任务生命周期属于 `workflow.md`，源码规则属于 `coding-standard.md`，测试策略属于
`testing.md`，框架细节属于对应技术栈规范。

Context、ADR、OpenAPI 和已评审 Spec 拥有产品事实与系统边界；本文不复制应用列表、
当前迁移状态或业务流程。

## Authority

发生冲突时按以下顺序处理：

1. 当前用户明确授权的目标、范围与限制；
2. 根目录或更窄目录的 `AGENTS.md`；
3. 已接受 ADR、受影响 Context、OpenAPI、冻结 fixture 与数据约束；
4. 本目录中适用的开发规范；
5. 已评审 Spec 与 Issue 验收条件；
6. 当前实现、测试、框架默认值与示例。

低优先级材料不得静默覆盖高优先级权威。Skill 和外部示例只提供路由或证据，不是项目
事实来源。

## Ownership

- 以受影响 Context 中的 behavior owner 决定修改位置，不以当前打开的文件或框架便利性决定。
- Frontend、Application Backend、Domain Service 和 Agent capability 之间通过版本化合同协作。
- 不通过共享数据库、import 对方实现、复制领域状态或前端直连下游绕过边界。
- 框架规范只决定已确认 owner 内的实现方式，不决定领域 ownership。

## Design Gate

下列变更实施前必须将决策写入权威 Spec，必要时同步 Context 或 ADR：

- 新能力或存在实质歧义的行为；
- 新 Service、Agent、Tool、框架、持久化对象、基础设施或重要依赖；
- API、Schema、认证、权限、数据 ownership 或跨运行边界协作；
- 需要兼容窗口、数据迁移、部署编排或不可直接回滚的变更。

设计至少回答：

1. Outcome 与 Non-goals；
2. UI、API、事实、持久化和调用方 owner；
3. 当前与目标合同，包括顺序、时间、null、分页和错误；
4. 采用的权威/参考与拒绝的假设；
5. 目标模块、依赖方向与 composition root；
6. 认证、Secret、超时、重试、幂等和安全失败；
7. rollout、mixed-version compatibility 与 rollback；
8. 最高可观察验收 seam 与未决问题。

纯文案、格式、规范整理或不改变项目事实的机械维护不需产品设计门禁。

## Cross-boundary Contract

跨边界变更必须冻结：

| 方面              | 必须明确                                               |
| ----------------- | ------------------------------------------------------ |
| Provider/consumer | 谁发布、谁消费、谁负责兼容                             |
| Wire contract     | version、route、DTO、状态、错误和 fixture              |
| Identity/security | caller、认证、授权、tenant 和审计主体                  |
| State             | 事实 owner、持久化 owner、生命周期和删除责任           |
| Failure           | 总 timeout、retry、idempotency、partial success 和降级 |
| Delivery          | rollout 顺序、新旧版本共存、rollback 和验收 seam       |

## Implementation And Completion

- 只在 owning application、domain 和 layer 内实现行为。
- 先修改权威合同，再同步生成与手写实现；不允许只改一侧。
- 不为假设中的未来复用预建抽象、共享 package、通用 CRUD 或平台服务。
- 保留用户已有 dirty work，不把无关重构混入业务变更。
- 按 `coding-standard.md`、`testing.md` 与适用技术栈规范完成验证。
- 保持 Context、ADR、OpenAPI、fixture、Schema 和运行配置一致。
- 交付时报告未解决冲突、失败检查、兼容、迁移和回滚风险。

## Exceptions

偏离本规范时，在同一变更中记录原因、owner、范围、合同/数据/安全影响、替代验证、
rollback 和移除条件。无记录的局部便利不构成例外。
