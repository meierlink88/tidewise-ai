# Industry、Concept、Chain Node 与 Industry Chain Schema V1

## 目标

本设计只建立空 PostgreSQL Schema，使 Data Domain 未来能够承载经过人工审核的 Industry、Concept、Chain Node 和 Industry Chain 主数据。数据清洗、分类、导入、重定向生成及现有记录处置均属于后续独立任务。

## 现有结构差距

- `entity_nodes` 能保存通用实体，但缺少 `industry`、`concept` 的正式 profile 和非空稳定键唯一约束。
- `theme_profiles` 是历史通用 Theme 模型；Concept 是跨行业或跨环节主数据，Research Theme 是批次研究快照，二者生命周期不同，因此不能复用。
- 历史 `sector` 同时混合外部行业、市场概念和板块语义，不能直接等同于 Industry 或 Concept。
- `chain_node_profiles` 只有定义和边界，缺少与 entity active/inactive 状态分离的主数据审核状态。
- `000014` 曾引入包含地域范围、来源、物理约束等大合同的 Industry Chain 模型；`000015` 在授权清理中删除了这些表。V1 不恢复旧合同，只重新引入规范主链、membership 和 topology 三个必要结构。
- 通用 aliases 已存在于 `entity_nodes`，外部编码已存在于 `entity_external_identifiers`，无需建立重复表。
- 当前没有稳定身份 redirect，也没有数据库级无环约束。

## Schema 取舍

### Typed identity

非空 `entity_key` 全局唯一，与现有 UUID 生成和按 key 查询合同保持一致。跨类型同名实体使用不同 typed key，例如 `industry:artificial_intelligence` 与 `concept:artificial_intelligence`；名称不做唯一约束。Industry、Concept、Chain Node 和 Industry Chain definition 通过触发器校验所绑定实体类型，并阻止已有 profile 的实体被改成其他类型或空 key。

### Industry

`industry_profiles` 保存 classification system、version、code、三级 level、直接父行业及完整代码路径。父子层级必须同体系、同版本且相差一级。Industry 父子关系是分类归属，不进入 `chain_node_relations` 或 Industry Chain topology。

### Concept

`concept_profiles` 使用 V2 清洗合同已出现的九种 concept type，并保存非空定义、边界和 candidate/approved 审核状态。它不复用 `theme_profiles`。

### Chain Node

现有 `chain_node_profiles` 只增加可空的 `review_status`。NULL 明确表示历史记录尚未通过本次主数据清洗；migration 不把 842 条旧记录批量判成 candidate 或 approved。后续正式导入合同必须为新增或已清洗记录提交审核状态。

### Industry Chain

`industry_chain_definitions` 保存主链范围、核心交付物、终端用途、地域、快照日期和审核状态。`industry_chain_node_memberships` 保存节点在特定链内的 position 和 upstream/midstream/downstream 阶段；position 不唯一，以支持平行支路处于同一拓扑层。新表名刻意不复用 `000014` 已退役表名，防止历史 importer 按旧字段误写新合同。

`industry_chain_graph_edges` 连接同一链内的 membership，支持 `input_to`、`is_component_of`、`depends_on`，并以 `direct_candidate` / `compressed_candidate` 区分直接候选段和压缩展示段。压缩段必须保存省略步骤说明，active topology 必须无环。DAG 与 redirect 的无环校验通过事务级 advisory lock 串行化，profile 类型与 Industry 父子校验对被依赖行加共享锁。

active graph edge 只能引用 active membership；仍被 active edge 引用的 membership 不能停用。仓库中 `IndustryChainProfile` 等旧 Go 类型只对应已退役的 `000014` 合同，已标记 Deprecated；新 Schema 使用 `IndustryChainDefinition`、`IndustryChainNodeMembership` 和 `IndustryChainGraphEdge`。

该 topology 是特定 Industry Chain 的上下文事实，不替代正式 `chain_node_relations`，也不接受或回写 Research Anchor 的当次推理路径。

### Redirect 与 aliases

别名继续使用 `entity_nodes.aliases`。`entity_redirects` 只表达显式身份去向：同类型 `merge` 或跨类型 `reclassification`。一个 source 只有一个 target，禁止自指和循环。本 migration 不生成任何 redirect 数据。

## 不进入正式 Schema 的清洗产物

以下内容继续作为离线导入和预检证据，不建立正式主数据表：

- source mapping；
- quarantine；
- keep-separate 与近似名称审核；
- 跨类型同名审计清单；
- 旧节点 disposition、置信度和人工审核工作表；
- 新节点候选问卷、evidence queue 和 direct relation candidate 队列。

后续导入任务应读取版本化清洗包，先执行结构、引用、数量、redirect 无环和历史引用预检，再在单独授权的事务中写入；本任务不实现该执行器。

## Migration 安全边界

- `000027` 仅包含 DDL、约束函数和触发器，不包含业务 `INSERT`、`UPDATE`、`DELETE` 或 `TRUNCATE`。
- 若现库存在重复非空 `entity_key`，唯一索引会 fail closed；本 migration 不会自动合并或重写任何身份。
- migration 不应用到 `tidewise_local`、UAT、production 或 shared 数据库。
- PostgreSQL 行为测试只允许连接 loopback 上的专用测试数据库（CI 的一次性 service database 例外），在临时 schema 中运行并在结束后删除。
- Down fail closed；已发布 Schema 的撤回必须通过审核后的 forward fix。
