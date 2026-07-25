# Backend Testing Workflow

本规则适用于 Tidewise AI 中采用 Kratos Layout 的 Backend Service。它把 TDD 约束在
有业务价值的测试边界上，而不是要求每个目录、类型或源码文件机械对应一个测试。
Miniapp Frontend 与 Agent/Eino 使用各自开发规范中的测试策略。

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

| 变更 | 必要测试 |
| --- | --- |
| SQL、事务、Repository、缓存或远端 HTTP Client | Data 集成或 Adapter 契约测试 |
| Schema、约束、索引或 forward migration | Migration/Schema 测试 |
| 配置默认值、必填项、环境覆盖或安全校验 | Conf 测试 |
| 启动失败、信号、优雅停机、资源释放 | Lifecycle 测试 |
| Service 边界、依赖方向或目录所有权 | 集中式 Architecture 测试 |
| Provider/Consumer API 合同 | 双方合同测试和必要的 HTTP smoke |
| Docker、运行镜像或部署入口 | Binary/Container 构建和必要 smoke |

未修改这些边界时，不因一次普通 Biz 或 API 变更重复运行或新增对应测试。

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
- Biz seam：
- API/HTTP seam：

条件测试：
- Data：不涉及 / 涉及，原因：
- Migration：不涉及 / 涉及，原因：
- Conf/Lifecycle：不涉及 / 涉及，原因：
- Architecture：不涉及 / 涉及，原因：
- Provider/Consumer 或 Container：不涉及 / 涉及，原因：

明确不测：
- 简单映射、构造和框架胶水
```

## 既有测试清理

不因本规则一次性批量删除既有测试。修改相关模块时逐步处理：

- 删除与 Biz 测试重复的 Handler 业务矩阵；
- 删除只验证私有实现、简单构造或机械映射的测试；
- 保留 SQL、事务、migration、外部协议和错误清洗等高价值测试；
- 每次删除前确认仍有一个更强的公开行为 seam 防止回归。
