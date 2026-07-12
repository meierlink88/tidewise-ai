## ADDED Requirements

### Requirement: 产业链实体与节点 seed
系统 SHALL 通过 `backend/data/entity_foundation/` 和现有 `entity-seed` 边界初始化已批准的 `industry_chain`、改进后的 `chain_node` profile 与链内 membership，并保持稳定 key 和幂等 upsert。

#### Scenario: 加载产业链 seed
- **WHEN** 已批准产业链 seed 被加载
- **THEN** loader 必须校验中文主名称、英文 alias、稳定 chain code、scope、version、Review 状态、节点引用和 membership 唯一性

#### Scenario: 复用现有节点
- **WHEN** 候选链包含已有 33 个 `chain_node` 中语义等价的节点
- **THEN** seed 必须复用已有 stable key，不得创建仅因链不同而重复的节点

### Requirement: 产业链候选与正式 seed 隔离
系统 SHALL 将链范围、节点、membership、拓扑和跨实体关系候选保留在 Review artifact，只有逐项批准的数据才能进入正式 seed。

#### Scenario: Propose 不写正式 seed
- **WHEN** change 仍处于 Propose 或第一次 Review
- **THEN** 五条 MVP 候选及建议的 60–90 个去重节点不得写入 `backend/data/entity_foundation/`

#### Scenario: 输出 seed report
- **WHEN** 经批准的产业链 seed 在未来执行
- **THEN** report 必须分别输出 industry chain、chain node、membership、topology 与跨实体关系的 created、updated、unchanged、failed 数量

### Requirement: 产业链 seed 安全边界
系统 SHALL 拒绝把推理结论、投资判断或缺少来源的拓扑/关系作为产业链 seed。

#### Scenario: 拒绝动态结论字段
- **WHEN** seed 包含当前瓶颈严重度、利好利空、受益承压、传导强度、预测、评分或投资建议
- **THEN** validator 必须返回明确错误并阻止写入
