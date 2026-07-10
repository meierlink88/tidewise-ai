## 1. 配置和本地基础设施

- [ ] 1.1 编写 Neo4j 配置加载测试，覆盖默认关闭、local/uat/prod 非敏感配置、缺失 URI、缺失凭证引用和非法超时时间。
- [ ] 1.2 在 `backend/config` 中新增 Neo4j 非敏感配置模板，并在 `backend/internal/config` 中实现强类型 Neo4j 配置校验，真实凭证只允许来自环境变量。
- [ ] 1.3 编写本地 Neo4j compose 或说明的静态检查，确保不提交真实密码、token 或生产连接串。
- [ ] 1.4 更新本地基础设施配置，提供可选 Neo4j local 启动方式和 gated smoke 所需变量说明。

## 2. Neo4j 平台连接边界

- [ ] 2.1 编写 `platform/graphdb` 单元测试，覆盖 driver factory、禁用状态、凭证解析失败、连通性检查失败和资源关闭。
- [ ] 2.2 引入 Neo4j Go driver 依赖，并实现 `backend/internal/platform/graphdb` 的 driver 创建、连通性检查和关闭流程。
- [ ] 2.3 编写 fake graph writer 测试替身，供普通单元测试验证投影逻辑而不连接真实 Neo4j。

## 3. 投影数据读取和运行记录

- [ ] 3.1 编写 migration 静态测试，覆盖 `graph_projection_runs` 和必要明细表的增量 DDL，不允许清空已有业务数据。
- [ ] 3.2 编写 repository 测试，覆盖读取 `entity_nodes`、`entity_edges`、创建投影 run、完成投影 run、记录失败和查询最近 run。
- [ ] 3.3 实现必要 PostgreSQL migration 和 repository 方法，提供实体图投影所需的快照读取和运行记录持久化。

## 4. 实体图投影核心逻辑

- [ ] 4.1 编写实体节点映射测试，覆盖 entity ID、entity key、entity type、layer code、名称、状态、命名空间和更新时间。
- [ ] 4.2 编写关系类型安全映射测试，覆盖合法关系类型、大小写转换、非法字符、空类型、未知类型和 fallback 或拒绝策略。
- [ ] 4.3 编写实体关系映射测试，覆盖起点缺失、终点缺失、inactive 关系、重复关系、置信度和来源属性。
- [ ] 4.4 实现 `backend/internal/apps/graphprojection` 的实体节点映射、关系映射、relation type mapper 和投影报告模型。
- [ ] 4.5 编写 projector 执行测试，覆盖正常投影、部分失败、全部失败、重试安全、跳过统计和运行报告持久化。
- [ ] 4.6 实现实体图投影编排，使用 repository 读取 PG 快照，通过 graph writer 幂等 upsert Neo4j 节点和关系。

## 5. 命令入口和真实 smoke

- [ ] 5.1 编写 `backend/cmd/graph-projector` 命令入口测试，覆盖 `check`、`project-entities`、`rebuild-entities`、非法参数和配置失败。
- [ ] 5.2 实现 `backend/cmd/graph-projector`，入口只负责配置加载、依赖组装、命令参数解析和启动投影流程。
- [ ] 5.3 编写 gated Neo4j smoke 测试或命令验证说明，要求显式环境变量启用，并默认不在普通 `go test ./...` 中访问真实 Neo4j。
- [ ] 5.4 在 local 环境使用少量实体 seed 数据完成一次真实 Neo4j 投影 smoke，并记录可复跑命令和预期检查方式。

## 6. 架构边界和文档验证

- [ ] 6.1 新增或更新架构边界测试，确保 `platform/graphdb` 不 import `internal/apps/*`，`adminapi`、`miniappapi` 和 `ingestion` 不直接写 Neo4j。
- [ ] 6.2 更新 backend 本地说明或 migration README，描述 PG 事实源、Neo4j 投影库、重建方式、配置变量和 smoke 流程。
- [ ] 6.3 运行 `go test ./...`，确认普通测试不依赖真实 Neo4j。
- [ ] 6.4 运行 `openspec validate add-neo4j-graph-projection-foundation`。
