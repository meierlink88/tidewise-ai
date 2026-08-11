# Risk-boundary Testing Standard

本规范只定义测试 seam 的选择、风险触发条件和 TDD 节奏。具体技术栈命令与专项验证
由对应技术栈规范拥有。

## Principles

1. 测试可观察行为，不测试文件布局、文档措辞或私有实现。
2. 开发前确认本次最高价值 seam；未被行为风险证明的 seam 不新增测试。
3. 同一行为只在最有价值的边界完整验证一次，不在 Biz、Service、Handler 和 E2E 重复穷举。
4. 测试范围由变更风险决定，不由目录、文件或框架层级数量决定。
5. 格式、类型、静态检查和构建不是行为测试，也不互相替代。

## Default Seams

### Biz

业务行为默认在 Biz seam 验证：

- 业务不变量、状态变化、默认值和边界条件；
- 稳定错误分类；
- 通过 fake Port 表达的依赖结果；
- 会改变用户或调用方可观察结果的分支。

Biz 测试不关心 HTTP、数据库 driver、框架 Context 或具体 Adapter。

### API/HTTP

合同行为默认在 API/HTTP seam 验证：

- 参数、必填字段和 wire DTO 转换；
- 状态码、安全错误、envelope 和 request ID；
- OpenAPI、运行时路径、方法与必要响应的一致性。

API 测试只证明绑定与错误映射，不复制 Biz 的完整规则矩阵。

Frontend 与 Agent/Eino 的默认 seam、命令和平台条件分别由对应技术栈规范定义。

## Conditional Seams

| 变更风险                                   | 必要验证                  |
| ------------------------------------------ | ------------------------- |
| SQL、事务、Repository、缓存或远程 Client   | Data 集成或 Adapter 合同  |
| Schema、约束、索引或 forward migration     | Migration/Schema          |
| 配置默认值、必填项、环境覆盖或 Secret 校验 | Conf                      |
| 启动失败、信号、优雅停机或资源释放         | Lifecycle                 |
| 运行时依赖方向或边界                       | Architecture/import       |
| Provider/Consumer API                      | 双方合同与必要 HTTP smoke |
| Binary、Docker 或部署入口                  | Build/container smoke     |

未触及对应风险时，不因一次普通业务变更运行或新增该 seam。

## Do Not Test By Default

除非含有独立判断逻辑，以下内容不单独编写测试：

- 纯文档、开发规范内容、目录清单与措辞；
- `cmd/server` 中的机械装配；
- 简单构造函数、getter、常量和 DTO/PO 字段复制；
- 框架路由注册胶水、无校验配置读取和生成代码；
- CSS 细节、纯 presentation mapping 与上游框架内部实现。

开发规范文件变更不得因文档内容触发产品测试、全应用 CI 或仓库架构门禁；只做格式、
引用与 diff 完整性检查。

## TDD Rhythm

1. 在 Issue 或实现说明中确认默认 seam 与被触发的条件 seam。
2. 在最主要的行为 seam 写一个真实失败的测试。
3. 实现最小变更使其通过，再重构。
4. 循环期间只运行目标测试；完成时只运行受影响套件与已触发门禁。
5. 不把全仓回归作为每个 Red/Green 循环的前置步骤。

## Existing Test Cleanup

删除或合并既有测试时，记录为 `duplicated-by-stronger-seam`、`implementation-only`、
`obsolete` 或 `consolidated`。SQL、事务、constraint、复杂查询、migration、远程协议与错误清洗
只保留最少的真实边界用例。
