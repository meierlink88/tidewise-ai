# Tidewise AI Engineering Standard

本规范是观潮家工程工作的仓库级统一入口。所有会改变系统行为、项目事实、API、数据、
配置、运行时、部署或测试合同的任务都必须先经过本规范，再进入对应技术栈规范。

本规范决定“谁拥有行为、代码放在哪里、需要读取什么、何时可以实施以及怎样完成验证”。
框架规范决定“在已经确认的 owner 内怎样使用框架”。框架不得反向决定领域 ownership。

## 1. Authority

发生冲突时，按以下顺序处理：

1. 当前用户明确授权的目标、范围与限制；
2. 根目录或更窄目录的 `AGENTS.md` 与 `docs/agents/workflow.md`；
3. 已接受 ADR 与受影响 Context 的 ownership、语言和依赖边界；
4. 已发布 OpenAPI、冻结 fixture、数据库约束和外部兼容合同；
5. 本规范与适用的项目技术栈规范；
6. 已评审的任务 Spec、Issue 验收条件和迁移设计；
7. 测试与当前实现；
8. 框架默认值、官方示例和社区案例。

低优先级材料不得静默覆盖高优先级权威。实施需要改变 ADR、Context、OpenAPI 或冻结
合同时，必须在同一 owner-controlled 变更中先明确修改目标、兼容性和迁移顺序。

全局 Skill、外部参考仓库和示例只负责提供工作流或技术证据。最终项目规则必须落在本
仓库的 `AGENTS.md`、`docs/agents/`、Context、ADR、OpenAPI 或已评审 Spec 中。

## 2. Required Reading By Task

所有工程任务先读取：

- `AGENTS.md`；
- 本规范；
- `docs/agents/coding-standard.md`；
- `docs/agents/workflow.md` 和 `docs/agents/testing.md`；
- `CONTEXT-MAP.md`、受影响 Context、相关 ADR、合同和实现。

再按行为 owner 选择技术栈规范：

| 变更范围              | 必读规范                                                      |
| --------------------- | ------------------------------------------------------------- |
| Miniapp Frontend      | `docs/agents/miniapp-frontend.md` 与 `$taro-reference-first`  |
| Admin Portal Frontend | `docs/agents/admin-portal-frontend.md`                        |
| 任一 Backend Service  | `docs/architecture/kratos-backend-development-standard-v1.md` |
| AgentRun/Eino 能力    | `docs/agents/agentrun-eino.md`                                |
| 跨两个及以上运行边界  | 所有适用规范，并先冻结 provider/consumer 合同                 |

任务按被改变的行为选择规范，不按当前打开的文件或改动大小选择。涉及 API、Schema、
认证、远端依赖、框架、配置、部署或 ownership 的“小改动”仍走完整路径。

## 3. Repository And Runtime Architecture

仓库按应用垂直组织：

```text
miniapp/{frontend,backend}
admin-portal/{frontend,backend}
analyse-data-service/backend
agent-run/backend
```

运行边界固定为：

```text
Miniapp Frontend
  -> Miniapp Application Backend Service
    -> Data Domain Service REST API

Admin Portal Frontend
  -> Admin Application Backend Service
    -> Data Domain Service / AgentRun REST API

Data Domain Service
  <-> AgentRun versioned execution/publication REST APIs
```

共同规则：

- Frontend 只调用自己的 Application Backend Service；
- Service 之间只通过版本化 API、冻结 fixture 和 provider-consumer 合同协作；
- 不共享数据库凭据、表、Repository、领域模型、Go DTO 或运行时实现 package；
- BFF 拥有消费方 DTO、编排和安全错误映射，不拥有下游领域事实；
- Data 拥有投研领域事实与 Data 数据库；
- AgentRun 拥有 Agent、Execution、Schedule、Source、Artifact 和 Agent 运行状态；
- 仓库根只承载治理、合同、基础设施和脚本，不提供跨应用共享运行时业务源码。

## 4. Design Gate

下列变更必须在实施前记录设计：

- 新能力或 materially ambiguous 行为；
- 新依赖、框架、Service、Agent、Tool、持久化对象或基础设施；
- 公共或内部 API、Schema、认证、权限或数据 ownership 变化；
- 跨项目/跨 Service 协作；
- 需要兼容窗口、数据迁移、部署编排或不可直接回滚的变化。

设计至少包含：

1. Outcome 与 Non-goals；
2. Owner map：用户界面、API、事实、持久化和调用方 owner；
3. 当前合同：route、DTO、状态、错误、时间、顺序、null、分页和 fixture；
4. 选择的官方/本地参考，以及采用和拒绝部分；
5. 目标目录、package/layer、依赖方向和 composition root；
6. 认证、权限、Secret、超时、重试、幂等和安全错误；
7. rollout、mixed-version compatibility 和 rollback；
8. 最高可观察验收 seam 与条件测试；
9. 会实质改变范围或行为的未决问题。

纯文案、格式或不改变项目事实的机械维护可以走 direct path，但仍必须检查 scope、
Secret 和现有未提交改动。

## 5. Implementation Rules

- 在 owning application、capability 和 layer 内实现行为；
- 先定义 consumer-owned Port 或已发布 wire contract，再实现 Adapter；
- 业务规则、确定性校验和状态转换不得下沉到 UI、HTTP Handler、ORM、模型 Prompt 或
  Eino Tool；
- 不为潜在复用提前创建抽象、共享 package、通用 CRUD、Schema Form 或平台服务；
- 不在同一变更中混合无关业务修改、框架迁移和目录重组；
- 生成合同与手写合同必须同步，不允许只修改一侧；
- 保留用户已有 dirty work，任务提交只包含本任务拥有的文件；
- 所有源码继续遵守 `docs/agents/coding-standard.md`。

## 6. Cross-Boundary Contract Gate

跨 Frontend、BFF、Data 或 AgentRun 的任务必须先冻结：

| 项目              | 必须回答                                                             |
| ----------------- | -------------------------------------------------------------------- |
| Provider/consumer | 谁发布，谁消费，谁负责兼容                                           |
| Wire contract     | version、route、DTO、状态、错误和示例                                |
| Identity/security | caller、认证、授权、tenant 和审计主体                                |
| State             | 事实 owner、持久化 owner、生命周期和删除责任                         |
| Failure           | 总 timeout、retry、idempotency、partial success 和 degraded behavior |
| Compatibility     | rollout 顺序、旧新版本共存、回滚点                                   |
| Acceptance        | provider test、consumer test、happy path 和 degraded path            |

不得以共享数据库、import 对方实现、复制领域状态或 Frontend 直连下游来绕过合同设计。

## 7. Definition Of Done

任务完成必须同时满足：

- 代码与设计落在正确 owner、目录和依赖方向；
- 受影响 Context、ADR、OpenAPI、fixture 和运行配置保持同步；
- 按 `docs/agents/testing.md` 完成默认 seam 和触发的条件 seam；
- 格式、lint/typecheck、vet、build、目标平台构建和必要 smoke 通过；
- 用户可见 loading、error、empty、retry、mutation 或 degraded 状态得到验证；
- Secret、认证信息、内部错误和敏感业务载荷没有进入代码、fixture、日志或 Artifact；
- migration、部署、兼容和 rollback 风险已记录；
- material implementation 完成 Standards 与 Spec 两轴 code review；
- 未解决的 authority conflict、失败检查或手工验证项在交付中明确报告。

“局部测试通过”“能够编译”或“示例可以运行”不能单独代表任务完成。

## 8. Exceptions

偏离本规范或技术栈规范时，必须在同一变更中记录：

- 偏离规则、原因、owner 和适用范围；
- 对 API、数据、失败、安全、部署和维护的影响；
- 替代验证、rollback 和移除条件；
- 需要更新的 ADR、Context 或规范。

没有记录的局部便利实现不构成例外。
