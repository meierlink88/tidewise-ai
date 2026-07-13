## REMOVED Requirements

### Requirement: 市场板块实体分类法
**Reason**: sector 逻辑实体与固定 semantic sector 分类法会重复表达粗细产业概念。
**Migration**: 产业概念进入统一 chain_node 候选 Review；指数、benchmark、市场和非产业标签回到各自既有边界。

### Requirement: 板块稳定标识和命名
**Reason**: sector 身份被取消，名称与 aliases 统一复用 `entity_nodes`。
**Migration**: 经批准的语义等价对象优先保留 UUID/entity key，并建立最小 chain_node profile；key 变更须单独审计。

### Requirement: MVP 板块候选准入
**Reason**: 固定约 50–60 个 sector、Top 排名和 sector 评分不是统一产业节点主数据的身份规则。
**Migration**: 外部分类仅作候选参考，使用 definition、boundary 与产业语义逐条 Review，并过滤非产业标签。

### Requirement: 板块关系边界
**Reason**: sector 与 market/benchmark/chain node 的关系不再属于生产 sector 模型。
**Migration**: 不调整 market、benchmark/index；产业静态关系仅迁移到独立 `chain_node_relations`，动态影响留给事件推理。

### Requirement: 投资建议安全边界
**Reason**: sector capability 被移除，安全边界由 chain_node、theme 与事件分析能力继续承担。
**Migration**: chain_node/theme profile、关系与后续展示仍禁止推荐、涨跌预测、目标价、仓位和交易建议字段。

### Requirement: canonical active 集合唯一性
**Reason**: canonical active sector 集合不再是目标事实模型。
**Migration**: 既有 active/inactive sector 与 convergence audit 作为 forward migration 输入；不得修改历史审计或通过清库处理。
