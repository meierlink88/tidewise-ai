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
