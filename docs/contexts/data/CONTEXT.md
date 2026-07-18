# Data Context

## Purpose

Data Domain Service 是当前唯一 Domain Service，负责稳定的数据事实、领域规则、持久化、受控导入和查询 API。

## Owns

- Entity、产业链节点及关系、Benchmark、Index 等主数据。
- Raw Document、Event、Source Catalog。
- Research Theme、Research Anchor 及其关联数据。
- PostgreSQL schema、migration、repository 和 Neo4j 可重建投影。
- Agent 使用的 Raw Document/Event Import API 与 receipt、幂等和事务规则。
- 面向 Miniapp/Admin Application Backend Service 的版本化 REST API。

## Does Not Own

- Miniapp 或 Admin Portal 的页面 DTO、交互状态和展示逻辑。
- User、Auth、Payment、Subscription 等未来独立领域。
- 数据采集 connector、parser、采集 prompt 或采集调度执行。
- Agent 的模型推理和工作流运行。

## External Agent Boundary

外部 agent-run 读取受控 Source Catalog 元数据，通过 Data REST API 写入 Raw Document 和 Event。agent-run 不直接访问 Data 数据库，Tidewise 仓库不保留没有运行入口的采集实现。

## Language

**研究主题（Research Theme）**:
对一组 Event 及其产业链影响形成的可发布研究判断，包含一句话结论、传导路径和结论演进阶段。

**传导阶段（Transmission Stage）**:
研究主题结论沿证据与影响路径发展的当前进度，例如“扩散”或“验证”；它不表示节点在产业链中的位置。
_Avoid_: 上游阶段、中游阶段、下游阶段

**下一检查点（Next Checkpoint）**:
研究主题当前尚待显现或验证的检查点提示，以自然语言保存，例如“尚未显现”；不是固定枚举。

**交易研究指向（Trading Direction）**:
基于当前结论给出的下一步研究优先级、验证对象与风险边界，以自然语言保存，不是做多/做空枚举。

## Source Ownership

Data 业务代码必须收敛到 `src/backend/services/data/`：

```text
cmd/          process and maintenance entrypoints
usecase/      import, query, seed and projection orchestration
domain/       Data-owned rules and models
repositories/ persistence ports and implementations
adapters/     PostgreSQL, Neo4j, migration and inbound/outbound adapters
transport/    Data REST routes, handlers, middleware and DTOs
config/       Data-only runtime configuration
```

`src/backend/migrations/` 与 `src/backend/data/` 是 Data 的统一事实资产，可以保留为 Backend 根资产，但不得被 BFF 直接读取。
