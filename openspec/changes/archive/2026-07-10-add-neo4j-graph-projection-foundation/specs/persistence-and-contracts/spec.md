## ADDED Requirements

### Requirement: Neo4j 图谱投影持久化边界
系统 SHALL 在引入 Neo4j 时保持 PostgreSQL 作为结构化事实主存储，并将 Neo4j 中的数据限定为从 PostgreSQL 派生的可重建图谱投影。

#### Scenario: 写入图谱数据
- **WHEN** 系统需要把实体、实体关系、事件或事件实体关联写入 Neo4j
- **THEN** 对应事实必须先存在于 PostgreSQL，并通过投影流程写入 Neo4j

#### Scenario: 不把 Neo4j 作为事实源
- **WHEN** Neo4j 中存在某个实体节点或关系
- **THEN** 系统不得仅凭 Neo4j 数据把它视为权威事实，必须能追溯到 PostgreSQL 中的事实来源

#### Scenario: 记录投影运行状态
- **WHEN** 图谱投影流程执行
- **THEN** 系统必须在 PostgreSQL 或等价事实边界中保存投影运行状态、统计和错误摘要，便于审计和重试

#### Scenario: 禁止保存敏感连接信息
- **WHEN** 系统保存 Neo4j 或图谱投影配置
- **THEN** 配置中不得保存真实用户名、密码、token、私有连接串密钥或其他敏感凭证
