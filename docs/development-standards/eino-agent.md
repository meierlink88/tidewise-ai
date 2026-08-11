# Eino Agent Development Standard

本规范只定义已批准 Agent capability 内使用 CloudWeGo Eino 的技术规则，适用于 Prompt、
Model Provider、Tool、Workflow/Graph/ADK、Callback、Stream、Checkpoint、MCP 和 structured output。

具体 Agent 宿主、应用目录、业务状态、数据 ownership 和运行时迁移由 Context、ADR 与 Spec
拥有，不属于本规范。Eino 是 capability 内的编排实现，不是顶层仓库或 Service 架构。

## Reference-first Gate

新增或修改 Eino 编排、Agent 架构、Provider Adapter、Tool、Stream、Checkpoint 或 multi-Agent 前，
必须检查仓库当前指定的 Eino、Eino Ext 与 Examples 参考源，并记录精确 commit、审计文件、
采用/拒绝模式、版本兼容与项目缺口。

来源优先级：

1. 当前 `go.mod`、本规范、受影响 Context、ADR、Agent Version 与冻结合同；
2. pinned `cloudwego/eino` core source；
3. 实际引入 module 的 `eino-ext` source；
4. `eino-examples`，只作模式与测试证据。

要求的参考源缺失时停止实施，不用记忆或远程片段替代。

## Responsibility Boundary

Eino 可以拥有：

- 一次 Agent execution 内的 typed orchestration；
- node/Agent/Tool 调用顺序与局部运行状态；
- model 与 Tool 的 Eino core interface；
- capability-local compile/run 和 structured output contract。

Eino 不拥有：

- 持久化业务状态、租约、业务 retry/recovery/idempotency 或 publication journal；
- 数据库 transaction、Artifact、outbox 或领域事实；
- Connector policy、认证、权限、tenant、Secret 或审计事实；
- HTTP Server、Service 生命周期、配置或部署。

Checkpoint 不替代业务恢复，Callback 不替代业务审计，Prompt 不替代确定性校验。

## Placement And Dependencies

- Eino Workflow/Graph、typed input/output、planning 与确定性 node 放在 owning capability。
- capability 可以依赖 Eino core interface，不依赖 Eino Ext concrete Provider。
- concrete Provider、Connector、Artifact、Scheduler、数据库和外部 Tool 位于所属应用的 Adapter 边界。
- API/Service 只做 wire 与 Biz 转换；Server 不构建 Workflow；composition root 显式装配依赖。
- 具体目录和文件名由宿主应用的工程结构规范拥有，本文不定义第二套布局。

## Primitive Selection

| 需求                                    | 默认选择                |
| --------------------------------------- | ----------------------- |
| 单一线性、类型兼容流水线                | `compose.Chain`         |
| 无环、异构字段、mapping、fan-out/fan-in | `compose.Workflow`      |
| 分支、局部状态、循环或显式图控制        | `compose.Graph`         |
| 模型在受控 Tool 集合中选择              | `adk.ChatModelAgent`    |
| 开放复杂规划与隔离 sub-Agent            | DeepAgent，必须单独论证 |
| 确定性子流程暴露为模型可选能力          | GraphTool/AgentTool     |

Workflow 只用于 all-predecessor 无环编排；Graph 的循环必须有最大 step。普通 Go 函数能表达的确定性
逻辑不包装成 Agent 或 Tool。multi-Agent 只用于真实责任/上下文隔离或有效并行。

## Composition And Provider

- 在 capability/application edge 构建并 compile，不在 HTTP Handler 或单个 node 内动态组装。
- 使用 typed input/output、显式依赖和稳定 node/Agent/Tool name。
- 确定性默认值、排序、窗口、dedup、状态转换和 publication 保持普通程序实现。
- API Key、Base URL、HTTP Client、Provider option 和 timeout 留在 Adapter/composition root。
- 不在共享 model 上使用可变 `BindTools`；使用 immutable `WithTools` 派生。
- 一个 Agent boundary 固定 classic `schema.Message` 或 block-preserving `schema.AgenticMessage`。
- 每次 execution 冻结 Provider、Prompt、Tool 与 Connector 配置快照。

## Prompt And Structured Output

- Prompt、machine schema、Tool set 和 orchestration contract 追溯到不可变 Agent Version。
- Prompt 只引导模型，不执行权限、事务、状态机或 referential integrity。
- structured output 拒绝 unknown field、trailing content、wrong type、blank required、invalid enum
  和 out-of-range value。
- normalization、default、timestamp、ordering、dedup 和 final state 由程序拥有。
- 原始模型响应只在必要、授权且受控的 Artifact 中保存，不进入普通日志。

## Tool, Skill And MCP

model-selectable Tool 必须有稳定 name、准确 description、strict JSON Schema、不可信参数校验、
认证/授权/tenant、幂等、审计、脱敏、总 timeout、cancellation 和安全错误。

- 简单 typed 低风险函数可使用 `tool/utils.InferTool`；写入、权限、stream 或复杂错误使用显式 Tool。
- unknown model-generated Tool name 安全失败。
- MCP client 使用显式 allowlist；空列表不得暴露全部远程 Tool。
- Skill 与 Prompt 只从受控、版本化、可评审位置加载。
- 权限拒绝、取消、interrupt、持久化与基础设施失败不伪装成模型可修正文本。

## Context, Stream And Checkpoint

- 入口 Context 贯穿 Runner、model、Tool、Graph 和 Adapter；node 内不使用 `context.Background()`。
- 每次 run 使用隔离 local state，不使用 package-global mutable state。
- 消费 Runner event 直到完成，检查 `event.Err` 与 `event.Action`；Interrupt 不是普通成功。
- `StreamReader` 只有一个消费者；多消费者先复制，并关闭原始和各 copy。
- 读取全部 chunk，只有 `io.EOF` 是正常结束；callback 的 stream copy 同样必须关闭。
- Checkpoint 只用于已批准的 pause、human approval 或 resumable interaction，并定义 durable store、
  tenant-isolated ID、serialization version、迁移/失效、加密、TTL 与删除。

## Callback, Errors And Retry

- global callback 在并发执行前注册一次；request callback 通过 runtime option 注入。
- callback 按稳定 component/name/type 过滤，input/output 视为 immutable。
- callback 只记必要 timing/metadata，脱敏 Prompt、凭据、用户内容、Tool 参数/结果与 Provider body。
- 区分 model-correctable feedback 与 auth、contract、cancel、interrupt、persistence、provider、infrastructure error。
- retry 只用于 transient、idempotent operation 并共享一个总 deadline；不重复未保护写入。

## Verification

最高可观察 seam 使用真实 compile 的 Eino orchestration，替换 fake model、Tool、Connector、Provider
和 external server。默认验证 compile、主要 branch/fan-in/bounded loop、structured output、错误传播、
timeout/cancellation/interrupt、Tool schema、stream 完整消费与 callback 脱敏。

Checkpoint、MCP、真实 Provider、持久化与 multi-Agent 只在变更触及对应风险时验证。避免只锁定私有
node 顺序的脆弱测试。

## Forbidden Patterns

- 把 Eino 当作顶层 repository 或 Service architecture；
- 在 Handler 内构建/compile Workflow；
- capability 依赖 concrete Provider、数据库 driver、scheduler、HTTP SDK 或文件系统；
- 用 Prompt/LLM 执行权限、事务、状态机或确定性 referential validation；
- global mutable model、Tool、Graph state 或 callback mutation；
- production node 使用 `context.Background`、`log.Fatal` 或无界 retry；
- 用 in-memory checkpoint 冒充 durable recovery；
- 空 MCP allowlist 暴露全部 Tool；
- 复制示例的 Provider、部署、日志或安全假设。
