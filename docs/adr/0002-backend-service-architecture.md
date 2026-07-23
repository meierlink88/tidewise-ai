---
status: accepted
---

# 使用 Application Backend Service 与 Domain Service 两层架构

## 背景

仓库已经拥有 Data、Miniapp、Admin Portal 三个启动入口、HTTP client 和独立 Dockerfile，但实现仍分散在旧 `internal/apps`、`internal/domain`、`internal/repositories`、`internal/http` 和共享大配置中。旧目录让源码 ownership 与实际运行边界不一致，也使 BFF 直接依赖 Data 实现更难被静态阻止。

“application service”同时被用于描述可部署服务和服务内部的应用层，容易产生歧义。

## 决策

Backend 使用两类可部署 Service：

- Miniapp 与 Admin Portal 是 **Application Backend Service**，面向各自 Frontend 提供 BFF 能力。
- Data 是 **Domain Service**，拥有当前数据领域的规则、事实和持久化。未来 User、Payment 可以成为新的 Domain Service。

每个 Service 内部的业务编排层称为 **Use Case Layer**，源码目录使用 `usecase/`。

依赖规则如下：

- Frontend 只能调用自己的 Application Backend Service。
- 不同 Backend Service 之间只能通过版本化 REST API 交互。
- Application Backend Service 不得直接访问 Domain Service 数据库或 import 其 domain、repository、adapter 实现。
- Domain Service 不得 import Application Backend Service 实现。
- 平台代码只能提供无业务语义的通用机制。

三个 Service 当前保留在同一 repository 和同一 Go module，但必须拥有独立 binary、配置、健康检查、Dockerfile 和部署能力。当前不引入 service mesh、分布式事务、服务注册中心或多 repository 治理。

Data collection connector、parser、prompt 和调度执行归外部 AgentRun。Source、
完整原始 Artifact 与 Event 提取的最终所有权由 ADR-0005 进一步收敛到
AgentRun；Tidewise Data 只保留正式 Event 的受控接纳、轻量证据、事务、审计和
数据事实。

Miniapp Frontend 暂时可以使用 Frontend-owned mock；本决策不要求立即接入真实 BFF。

## 目标源码结构

```text
src/backend/
  services/
    miniapp/
      cmd/
      usecase/
      transport/
      dataclient/
      config/
    adminportal/
      cmd/
      usecase/
      transport/
      dataclient/
      config/
    data/
      cmd/
      usecase/
      domain/
      repositories/
      adapters/
      transport/
      config/
  internal/platform/
  migrations/
  data/
```

## 影响

- 旧无 owner 的业务目录必须迁入 owning Service 后删除。
- BFF 与 Data 的 API contract、DTO 和 consumer-owned client 继续独立维护。
- 各 Service 可以独立构建和部署，但当前仍共享仓库与 Go module。
- 架构测试应验证最终边界，不再保护 compatibility 或 transitional 路径。
- 若未来拆分 repository/module，现有 REST contract 和 ownership 可以直接成为拆分边界。

## 与既有决策的关系

`0001-product-source-root.md` 关于唯一 `src/` 根目录、单 repository 和单 Go module 的决策继续有效。本 ADR 澄清：暂不拆分 repository/module 不代表三个 Backend Service 不能独立构建和部署。
