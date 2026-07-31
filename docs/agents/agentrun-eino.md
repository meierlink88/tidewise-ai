# AgentRun Eino Development Standard

本规范是观潮家 AgentRun 内 CloudWeGo Eino 能力的仓库级技术权威。它适用于 Agent、
Prompt、Model Provider、Tool、Workflow/Graph/ADK、Callback、Stream、Checkpoint、
MCP 和 structured output。

Eino 是具体 Agent capability 内的编排实现，不是仓库顶层架构、Service runtime 或
业务状态 owner。AgentRun 的 Service shell、HTTP、配置、生命周期和外部 Adapter 继续
使用 Kratos 规范。

## 1. Authority And Reference-first Gate

任何新增或修改 Eino 编排、Agent 架构、Provider Adapter、Tool、Stream、Checkpoint 或
multi-Agent 的任务，在设计和实施前必须检查：

```text
.reference/cloudwego/eino
.reference/cloudwego/eino-ext
.reference/cloudwego/eino-examples
```

记录：

```text
Repository and exact commit:
Relevant files/examples inspected:
Adopted patterns:
Rejected patterns:
Version compatibility:
Project-specific gaps and owner:
```

来源优先级：

1. 当前项目 `go.mod`、本规范、AgentRun Context、ADR、Agent Version 和冻结合同；
2. pinned `cloudwego/eino` core source；
3. 实际引入 module 的 `eino-ext` source 和 `go.mod`；
4. `eino-examples`，仅用于模式和测试证据。

任一要求的 reference clone 缺失时停止实施。不得用记忆、远程片段或示例布局替代审计。

## 2. Responsibility And Ownership

Eino 可以拥有：

- 一次 Agent Execution 内的 typed orchestration；
- 节点/Agent/Tool 调用顺序和局部运行状态；
- 模型与 Tool 的 Eino core interface；
- capability-local compile/run 和 structured output contract。

Eino 不自动拥有：

- Agent Definition、Agent Version、Execution 或 Schedule；
- durable Work Item、租约、业务 retry、recovery、idempotency 和 publication journal；
- PostgreSQL transaction、Artifact、outbox 或领域事实；
- Connector policy、认证、权限、tenant、Secret 和审计事实；
- Kratos HTTP、Service 生命周期、配置或部署。

Checkpoint 不能替代业务恢复，Callback 不能替代业务审计，Prompt 不能替代确定性校验。

## 3. Engineering Placement

AgentRun 保持：

```text
agent-run/backend/
├── api/agentrun/v1/
├── cmd/
├── configs/
└── internal/
    ├── conf/
    ├── biz/
    │   ├── platform/
    │   └── agents/<capability>/
    │       ├── planning/
    │       ├── workflow/
    │       ├── materialization/
    │       └── usecase/
    ├── data/
    │   ├── postgres/
    │   ├── modelprovider/
    │   ├── connectors/
    │   ├── artifacts/
    │   └── scheduler/
    ├── service/
    └── server/
```

规则：

- 每个 Agent Definition 放在独立 `internal/biz/agents/<capability>`；
- Eino Workflow/Graph 与 capability 的 typed input/output、planning 和确定性节点放在
  owning capability；
- `internal/biz/platform` 拥有 Agent/Version/Execution/Schedule/Work Item 规则与 Port；
- concrete Eino Ext Provider 只位于 `internal/data/modelprovider/<provider>`；
- Connector、Artifact、Scheduler、数据库和外部 Tool Adapter 位于 `internal/data`；
- `internal/service` 只转换 API 与 Biz；
- `internal/server` 不 import Eino capability 或 Provider，不构建 Workflow；
- `cmd/server` 是 Kratos 和 AgentRun 依赖的 composition root；
- capability 可以依赖 Eino core interface，不依赖 Eino Ext concrete Provider。

## 4. Select The Smallest Primitive

| 需求                                    | 默认选择                |
| --------------------------------------- | ----------------------- |
| 单一线性、类型兼容流水线                | `compose.Chain`         |
| 无环、异构字段、mapping、fan-out/fan-in | `compose.Workflow`      |
| 分支、局部状态、循环或显式图控制        | `compose.Graph`         |
| 模型在受控 Tool 集合中选择              | `adk.ChatModelAgent`    |
| 开放复杂规划与隔离 sub-Agent            | DeepAgent，必须单独论证 |
| 确定性子流程暴露为模型可选能力          | GraphTool/AgentTool     |

要求：

- Workflow 使用 all-predecessor 语义，不用于循环；
- Graph 必须明确 DAG 或 cyclic/Pregel，循环必须有最大 step；
- 可审计领域流水线优先确定性 Workflow/Graph；
- ordinary Go function 能表达的确定性逻辑不包装成 Agent 或 Tool；
- multi-Agent 只用于真实责任/上下文隔离或有效并行，不作为默认优化。

选型必须记录为什么更小的 primitive 不足。

## 5. Composition And Provider Boundary

- 在 capability/application edge 构建并 compile，不在 HTTP Handler 或单个节点内动态组装；
- 使用 typed input/output、显式依赖和稳定 node/Agent/Tool name；
- 确定性默认值、排序、窗口、dedup、状态转换和 publication 保持普通 Go 实现；
- Biz 依赖 `model.BaseChatModel` 等 Eino core interface；
- API Key、Base URL、HTTP Client、Provider option 和 timeout 留在 Data/composition root；
- 官方 Eino Ext component 满足合同则优先使用，自定义 Adapter 只解决已证明的缺口；
- 不在共享 model 上使用可变 `BindTools`；使用 immutable `WithTools` 派生；
- 一个 Agent boundary 固定 classic `schema.Message` 或 block-preserving
  `schema.AgenticMessage`，业务代码不得混用；
- 每次 Execution 冻结 Provider、Prompt、Tool 和 Connector 配置快照，运行中配置变化
  不影响已经启动的 Execution。

## 6. Prompt, Version And Structured Output

- Prompt、machine schema、Tool set 和 orchestration contract 追溯到不可变 Agent Version；
- 已发布 Agent Version 不原地改变 observable contract，除非有明确 conformance-fix 决策；
- Prompt 负责引导模型，不负责权限、事务、状态机或 referential integrity；
- structured output 拒绝 unknown field、trailing content、wrong type、blank required、
  invalid enum 和 out-of-range value；
- parser/validator 返回稳定分类，模型可修正错误与基础设施/权限错误分开；
- Go 程序拥有 normalization、default、timestamp、ordering、dedup 和 final state；
- 原始模型响应只在必要、授权和受控 Artifact 中保存，不进入普通日志。

## 7. Tool, Skill And MCP

每个 model-selectable Tool 必须具备：

- 稳定 name 和准确 model-facing description；
- strict JSON Schema 与不可信参数校验；
- 明确 authentication、authorization、tenant、idempotency、audit 和 redaction；
- 总 timeout、cancellation 和 sanitized error；
- side effect owner、重复调用语义和失败恢复。

使用规则：

- 简单、typed、低风险函数可以使用 `tool/utils.InferTool`；
- 权限、写入、幂等、stream 或复杂错误使用显式 Tool；
- Tool 适配能力，不成为领域事实 owner；
- unknown model-generated Tool name 必须安全失败；
- MCP client 初始化后使用显式 allowlist；空列表不得隐式暴露全部远端 Tool；
- Eino Skill middleware 只提供受控指令/参考，不是认证、业务工作流或 side effect；
- Skill 和 Prompt 只从受控、版本化、可评审位置加载。

只有模型能够根据反馈修正时，才把错误转为 model-visible correction。权限拒绝、取消、
interrupt、持久化和基础设施失败不得吞掉或伪装成可重试文本。

## 8. Context, State, Stream And Checkpoint

- 入口 Context 贯穿 Runner、model、Tool、Graph 和 Adapter；
- 节点内不得使用 `context.Background()` 脱离 Execution 生命周期；
- 每次 run 使用隔离 local state，不使用 package-global mutable state；
- ADK SessionValues 只保存单次 run 的短期共享状态；
- 消费 Runner event 直到完成，并检查 `event.Err` 和 `event.Action`；
- Interrupt 不是普通成功，必须进入明确应用状态；
- `StreamReader` 单消费者；多消费者前复制，并关闭原始和每个 copy；
- 读取全部 chunk，只有 `io.EOF` 是正常结束；
- callback 的 stream copy 同样必须关闭；
- 自定义 chunk 需要 concat/checkpoint 时注册并测试明确行为。

Checkpoint 只用于已接受的 pause、human approval 或 resumable interaction，并必须定义：

- durable store 和稳定、tenant-isolated checkpoint ID；
- serialization version 与迁移/失效策略；
- tenant access、encryption、TTL 和 deletion；
- interrupt/resume target 与 mixed-version behavior。

publication reconciliation、task replay 和业务 idempotency 继续位于应用边界。

## 9. Callback, Errors And Retry

- global callback handler 在并发执行前注册一次；request callback 通过 runtime option 注入；
- 按稳定 component/name/type 过滤，不依赖 handler 执行顺序；
- callback input/output 视为 immutable，禁止修改导致 graph race；
- callback 只记录必要 timing/metadata，并脱敏 Prompt、凭据、用户内容、Tool 参数/结果和
  Provider body；
- 区分 correctable Tool/business feedback 与 auth、contract、cancel、interrupt、
  persistence、provider 和 infrastructure error；
- retry 只用于 transient、idempotent operation，并共享一个总 deadline；
- model/transport retry 不得重复未保护写入；
- retry budget 耗尽后必须形成稳定可观测结果，不进入无界循环。

## 10. Verification

最高可观察 seam 使用真实 compile 的 Eino orchestration，替换 fake model、Tool、
Connector、Provider 和 external server。默认覆盖：

- compile、主要 branch、fan-in、bounded loop 和 failure propagation；
- strict structured output、empty/malformed/extra output 和 retry exhaustion；
- timeout、cancellation、interrupt 和 durable application state；
- Tool schema、invalid input、auth/tenant、idempotent side effect 和 unknown Tool；
- stream EOF/non-EOF、full consumption、copy、close 和 cancel；
- Callback metadata、redaction、stream close 和 race；
- Agent max iteration、Tool call/result 和 SessionValues isolation；
- Prompt/schema/Tool set 与 Agent Version 的 traceability。

条件覆盖：

- Checkpoint：interrupt/resume target、serialization 和 tenant isolation；
- MCP：allowlist、unknown Tool、auth、timeout 和 cleanup；
- real Provider：隔离凭据、成本上限、timeout 和不记录敏感响应的 smoke；
- publication/persistence：真实 transaction、journal、replay 和 unknown result；
- multi-Agent：责任隔离、transfer/handoff、最大迭代、共享状态和 degraded path。

避免只锁定私有 node 顺序的脆弱测试；优先验证稳定 application/HTTP contract。

## 11. Forbidden Patterns

- 把 Eino 当作顶层 repository 或 Kratos Service architecture；
- 在 Handler 内构建/compile Workflow；
- Agent Biz import Eino Ext concrete Provider、pgx、gocron、HTTP SDK 或文件系统；
- 用 Prompt/LLM 执行权限、事务、状态机或确定性 referential validation；
- global mutable model、Tool、Graph state 或 callback mutation；
- production node 使用 `context.Background`、`log.Fatal` 或无界 retry；
- 用 in-memory checkpoint 冒充 durable recovery；
- 空 MCP allowlist 暴露全部 Tool；
- 自动重试未保护写入；
- 未记录版本兼容就混用 Eino、Eino Ext 和 Examples；
- 复制示例的 Provider、部署、日志或安全假设。
