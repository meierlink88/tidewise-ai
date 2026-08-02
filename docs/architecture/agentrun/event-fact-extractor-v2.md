---
status: accepted
---

# AgentRun Event Fact Extractor V2

## Outcome

`event-fact-extractor.v2` 保留 V1 的确定性 Eino Workflow、Artifact Unit、Candidate、
独立 Review、Tag Catalog、Publication Journal 和 Data 副作用所有权。V2 只替换模型结果
传输合同及无效的原文字符串门禁，并发布到 Event Publication V3。

## Eino Function Call 合同

四个模型阶段分别绑定并强制调用一个结果函数：

| Stage | Function | 结果 |
|---|---|---|
| Event 提取 | `submit_event_candidates` | Event candidates / no-event reason |
| 非精确去重 | `submit_duplicate_judgments` | recalled pair same-event judgment |
| Tag 分类 | `submit_tag_assignments` | pinned Tag Catalog codes |
| 独立审核 | `submit_event_reviews` | semantic review decisions |

每个 Stage 从共享的 Eino `ToolCallingChatModel` 通过不可变 `WithTools` 派生只绑定一个
Function 的模型，并用 `WithToolChoice(ToolChoiceForced, functionName)` 强制调用。运行时
只接受恰好一次、名称正确的 Tool Call，并严格解码 arguments 到固定 DTO；缺失调用、
错误函数、多调用、非法 JSON、缺失必填顶层数组或错误类型最多使用同一 Schema 修正一次，
仍失败则形成该 Stage 的 terminal model-contract failure。该失败不进入模型可用性重试；
AgentRun 在 Execution 和内部结果中持久化稳定的 stage、violation 与组合 error code，供按
阶段统计，同时不保存原始 arguments。

Function 仅是结构化结果提交通道，不执行工具副作用。模型不能调用 Publication、PG、
Qdrant mutation 或任意业务 Tool。Data Publication 仍由 Application/Journal 确定性执行。
OpenAI-compatible wire 由官方 Eino Ext OpenAI ChatModel 提供，AgentRun 不自研协议；
Function Call 路径不同时要求 `json_object` response format。

每次阶段调用在内部 Execution Result 中记录 stage、实际 call count、最后一次 finish reason、
argument byte count 和安全 violation 分类；Provider/Model 来自既有不可变 Execution snapshot。
这些元数据可以按批次审计，但不保存 Prompt、Artifact 正文或原始 Function arguments。失败
会携带此前已经完成阶段的 observations，因此修正耗尽时的累计模型调用数不会被重建为空。
一次修正指令必须包含安全、稳定且可执行的错误类别，例如缺少 Artifact/Candidate/Pair/Review
覆盖、未知 Tag 或无效 item；不得只重复“格式错误”，也不得回显原始 arguments、Provider
响应或业务正文。
非精确去重阶段按最多二十个 recalled pair 分批调用同一个
`submit_duplicate_judgments` Function，并在程序内合并后对完整 pair 集合执行一次确定性
覆盖与歧义校验。分批只控制模型输入/输出上限，不改变召回范围、same-event 判断合同或
模型调用所有权；单批仍只允许一次修正。

## 语义与程序校验边界

模型可以概括、规范化并推导 Event 的客观表达。`evidence_statement` 是由模型生成、由
独立 Reviewer 对 Artifact 做语义审核的支持陈述，不是逐字引用。`occurred_at` 可以从
Artifact 语义确定，但不能把 `published_at` 或 `collected_at` 冒充事件时间。

程序不再使用 Artifact 正文字符串包含、日期文本搜索、actor/action/object/Mention
子串匹配裁决模型语义。非精确重复召回只保留结构化时间、生命周期、reference period
等边界，最终 same-event 由模型判断。程序继续校验 DTO、必填字段、枚举、禁止字段、
Artifact membership、正式 Tag、哈希、身份、状态、权限、Journal 和发布事务。

## Artifact Unit 多批发布

Data 的单次 Event Publication Batch 继续限制为一至十个 Event。一个 Artifact Unit 若产生
超过十个已审核 Candidate，AgentRun 必须按稳定顺序生成多个不可变 Publication Journal，
不得截断或把合法结果整体拒绝。Journal 仍以 `(work_item_key, batch_ordinal)` 作为投递身份；
同一 Unit 的所有 Journal 均 acknowledged 后 Unit 才进入 `published`，任一 Journal blocked
则 Unit blocked，中间状态保持 `publishing` 或 `ready_to_publish`。

## Version 与 rollout

V1 Agent Version 保留为历史执行身份；新工作使用 `event-fact-extractor.v2`。V2 与 Event
Publication V3 协调切换，不修改已完成 V1 execution 的不可变快照。

## Eino reference-first audit

本设计在实现前以以下固定只读参考完成 Eino gate：

| Reference | Commit | Inspected | Decision |
| --- | --- | --- | --- |
| `cloudwego/eino` | `922b6a8a233b5233fe47eecee6cd2c005e8c39cd` | `components/model/interface.go`、`components/tool/utils/invokable_func.go`、`schema/tool.go`、`compose/workflow.go` | 采用 `ToolCallingChatModel`、不可变 `WithTools`、`GoStruct2ToolInfo`、强制 `ToolChoice` 和 typed Workflow；Workflow 构造器直接要求 Tool Calling 能力，禁止运行期才发现模型不支持。 |
| `cloudwego/eino-ext` | `9137edd89e72b72735ede69db1c5ae29178a6e41` | `components/model/openai/chatmodel.go`、`components/model/openai/option.go` | 采用官方 OpenAI-compatible ChatModel 的 Tool Call wire 编解码；Function Call 路径不设置竞争性的 `json_object` response format，不实现私有协议。 |
| `cloudwego/eino-examples` | `171220631fb7068ead50b7cd964b8c471647117d` | `compose/workflow/1_simple/main.go` | 参考 composition-root 构造和显式 Compile；示例未提供固定阶段结果提交与 Publication 事务合同，因此拒绝扩展成 ReAct 自主选 Tool 或通用 ToolsNode 副作用执行。 |

固定参考与仓库当前 Eino/Eino Ext 依赖兼容，没有发现能替代 Event Fact 固定四阶段、Journal
或 Data Publication 事务边界的官方组件。项目自有代码仅保留阶段 DTO 校验、恰好一次调用、
一次修正和终止错误分类；模型 Provider wire、Tool Schema 反射和强制 Tool Choice 均由 Eino
正式接口承担。
