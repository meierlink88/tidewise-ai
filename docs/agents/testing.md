# Risk-boundary Testing Workflow

本规则适用于 Tidewise AI 的四个 Kratos Backend、Miniapp/Admin Portal Frontend、
AgentRun/Eino 与 CI。它把 TDD 约束在有业务价值的测试边界上，而不是要求每个目录、
类型或源码文件机械对应一个测试。

## 核心原则

1. TDD 验证行为，不验证文件结构或私有实现。
2. 开发前必须确认本次测试 seam；未确认的 seam 不新增测试。
3. 一个行为只在最有价值的边界完整验证一次，不在 Biz、Service、Handler 和 E2E
   重复穷举。
4. 不要求每个源码文件对应一个 `*_test.go`。
5. 测试范围由变更风险决定，不由 Kratos 目录数量决定。

## 默认测试边界

### Biz

Biz 是业务规则和 Use Case 的主要 TDD seam。涉及业务行为时必须覆盖：

- 业务不变量和状态变化；
- 默认值、边界条件和稳定错误分类；
- 通过 fake Port 表达的依赖结果；
- 能改变用户或调用方可观察结果的分支。

Biz 测试不关心 HTTP、数据库 driver、Kratos Context 或具体 Adapter。

### API/HTTP

API/HTTP 是对外合同 seam。涉及接口行为时必须覆盖：

- 参数和必填字段校验；
- API DTO 与 Biz 输入/输出的转换；
- HTTP 状态码、错误 envelope 和 Request ID；
- OpenAPI 与运行时路径、方法和响应的必要一致性。

API 测试不重复 Biz 的完整业务规则矩阵。一个成功样例和受影响的验证、错误映射即可
证明绑定关系；业务组合留在 Biz 测试。

## 条件触发测试

以下测试仅在本次变更触及对应风险时启用：

| 变更                                          | 必要测试                          |
| --------------------------------------------- | --------------------------------- |
| SQL、事务、Repository、缓存或远端 HTTP Client | Data 集成或 Adapter 契约测试      |
| Schema、约束、索引或 forward migration        | Migration/Schema 测试             |
| 配置默认值、必填项、环境覆盖或安全校验        | Conf 测试                         |
| 启动失败、信号、优雅停机、资源释放            | Lifecycle 测试                    |
| Service 边界、依赖方向或目录所有权            | 集中式 Architecture 测试          |
| Provider/Consumer API 合同                    | 双方合同测试和必要的 HTTP smoke   |
| Docker、运行镜像或部署入口                    | Binary/Container 构建和必要 smoke |

未修改这些边界时，不因一次普通 Biz 或 API 变更重复运行或新增对应测试。

## Frontend 与 AgentRun

Miniapp Frontend 与 Admin Portal Frontend 默认只保留：

- 关键页面、组件交互和导航行为；
- typed API Adapter 合同；
- 状态变化以及用户可见的 loading、error、empty、retry 行为；
- typecheck 与受影响目标构建。

CSS 细节、构建配置源码、简单 presentation mapping 和重复 Mock fixture 不单独测试；
平台差异或安全失败逻辑确有独立行为时除外。

Miniapp Frontend 的 Page、Feature、typed Port、平台 Adapter、微信/抖音兼容与条件验证
继续遵守 `docs/agents/miniapp-frontend.md`。页面和 View 测试不直接替代 API Adapter
合同或目标平台 build。

Admin Portal Frontend 的 shadcn、TanStack Query/Table、React Hook Form 与 Zod 条件
验证继续遵守 `docs/agents/admin-portal-frontend.md`；不测试上游框架内部实现，也不以
前端校验替代 Admin Backend 合同测试。

AgentRun 的 Eino Workflow、调度和 Agent 能力中会改变可观察结果的行为归入 Biz seam。
测试使用真实编译的 Eino 编排和 fake model、Tool、Connector、Provider；Provider
协议、Artifact 持久化和 PostgreSQL 事务归入条件 Data/Adapter seam。Tool、Stream、
Callback、Checkpoint、MCP 与 multi-Agent 的条件验证遵守
`docs/agents/agentrun-eino.md`。

## CI 套件选择

CI 先按应用边界选择 Data、Miniapp、Admin Portal 或 AgentRun，再按变更路径选择：

- 默认 Biz/API；
- Frontend；
- Data/Adapter；
- Migration；
- Conf/Lifecycle；
- Provider/Consumer；
- Container；
- 集中式 Architecture。

同一 package 不得先作为 focused suite 执行，再被同一 job 的无条件递归全量命令重复
执行。仓库级 Architecture/CI 合同由单独的治理 job 最多执行一次。格式化、vet、
typecheck、binary build 和目标平台 build 仍按受影响应用执行，它们不由单元测试替代。

## 默认不测试

除非包含独立判断逻辑，以下内容不单独编写单元测试：

- `cmd/server` 中的显式构造和依赖装配；
- 简单构造函数、getter 和常量；
- 机械 DTO/PO 字段复制；
- Kratos 路由注册胶水；
- 无校验逻辑的配置读取；
- 框架或生成代码本身。

构造和注册问题优先由编译、API 合同、架构检查或启动 smoke 发现。

## TDD 执行节奏

1. 在 Issue 或实现说明中确认 Biz、API/HTTP 及条件 seam。
2. 在最主要的行为 seam 写一个会失败的测试。
3. 实现使当前测试通过的最小代码。
4. 每个循环只运行目标包或目标测试。
5. 完成一个垂直切片后运行受影响 Service 的测试。
6. 完成任务时运行受影响套件、格式检查、构建和本次触发的条件门禁。
7. 全仓回归不作为每个 Red/Green 循环的前置步骤。

## Issue 测试计划模板

```markdown
## 测试边界

默认测试：

- Biz seam：涉及 / 不涉及，最高可观察行为：
- API/HTTP seam：涉及 / 不涉及，合同：
- Frontend seam：不涉及 / 涉及，页面、Adapter 或状态：
- AgentRun/Eino seam：不涉及 / 涉及，真实编排与 fake 边界：

条件测试：

- Data：不涉及 / 涉及，原因：
- Migration：不涉及 / 涉及，原因：
- Conf/Lifecycle：不涉及 / 涉及，原因：
- Architecture：不涉及 / 涉及，原因：
- Provider/Consumer：不涉及 / 涉及，原因：
- Container：不涉及 / 涉及，原因：

完成验证：

- Red/Green 期间的 focused test：
- 完成时只运行一次的 affected suite：
- typecheck/vet/build：

明确不测：

- 简单映射、构造、框架胶水及其他实现细节：
```

## 既有测试清理

删除或合并既有测试前，将其归入下列一类并在任务审计中记录：

- `duplicated-by-stronger-seam`：行为已经由更高的 Biz、API/HTTP 或页面 seam 完整覆盖；
- `implementation-only`：只验证私有实现、构造、机械映射、目录或配置源码；
- `obsolete`：对应行为或受支持工作流已经退出；
- `consolidated`：合并到更小的真实风险边界套件。

Data 层 Mock 矩阵、逐 Repository 方法测试和机械 PO/DTO 映射默认删除。SQL、事务、
constraint、复杂查询、Migration forward chain、远端协议和错误清洗只保留最少的真实
边界用例。fixture 只有在仍有保留测试或受支持本地工作流引用时才能继续存在。
