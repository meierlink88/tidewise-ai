## ADDED Requirements

### Requirement: 实体关系空基线
系统 SHALL 支持在保留实体主数据的前提下建立空关系基线，使后续实体关系只能通过已审阅批次重新写入。

#### Scenario: 清空 local PostgreSQL 实体关系
- **WHEN** 开发者执行本 change 批准的 local 关系重置流程
- **THEN** 系统必须删除 `entity_edges` 中的关系数据，并保持 `entity_nodes`、全部实体 profile、采集数据、事件数据和其他业务表不变

#### Scenario: 清空 local Neo4j 投影数据
- **WHEN** PostgreSQL 实体关系已经清空且开发者执行 local Neo4j 重置
- **THEN** Neo4j 中的节点和关系数据必须被删除，同时保留 database、约束、索引和本地基础设施配置

#### Scenario: 防止旧关系自动恢复
- **WHEN** 空关系基线建立后再次执行实体主数据 seed
- **THEN** 系统不得自动写回未经本 change 分批 review 的历史样例关系

### Requirement: 分批实体关系审阅
系统 SHALL 按关系族分批管理实体关系，并在每批数据写入 PostgreSQL 前要求人工审阅通过。

#### Scenario: 审阅联盟成员关系
- **WHEN** 系统准备第一批 `member_of` 关系数据
- **THEN** 审阅清单必须包含联盟组织和国家/经济体的中文名称、实体 key、关系方向、来源名称、来源 URL 和核验时间

#### Scenario: 未通过审阅的关系不得写入
- **WHEN** 某一关系批次尚未获得用户明确确认
- **THEN** 该批关系不得进入正式 seed 文件、PostgreSQL 或 Neo4j

#### Scenario: 前一批验收后推进下一批
- **WHEN** 某一关系族已经完成 PG 写入和 Neo4j 重建验收
- **THEN** 系统才可以开始准备下一关系族的审阅清单

### Requirement: 实体关系语义和来源校验
系统 SHALL 校验每条实体关系的类型、方向、端点、来源和安全边界，拒绝不符合客观事实关系要求的数据。

#### Scenario: 校验关系类型组合
- **WHEN** `member_of`、`issues` 或其他正式关系类型的 from/to 实体类型不符合关系规则
- **THEN** validator 必须返回明确错误并阻止该批数据写入

#### Scenario: 拒绝缺少来源的关系
- **WHEN** 关系缺少来源名称、来源 URL 或核验时间
- **THEN** validator 必须返回明确错误并阻止该关系写入

#### Scenario: 拒绝重复或悬空关系
- **WHEN** 关系出现重复 key、重复端点与类型组合、自环或引用不存在的实体 key
- **THEN** validator 必须返回明确错误并阻止该批数据写入

#### Scenario: 拒绝推理性关系
- **WHEN** 关系数据包含利好利空、影响强度、预测、受益承压、推荐或投资建议语义
- **THEN** validator 必须拒绝该数据，实体基础关系只允许保存可核验的客观事实

### Requirement: 关系批次写入和图谱重建
系统 SHALL 将已审阅关系先写入 PostgreSQL，再从 PostgreSQL 重建 Neo4j，不允许直接手工维护 Neo4j 关系事实。

#### Scenario: 写入已审阅关系批次
- **WHEN** 某一关系族通过 review 并执行实体 seed
- **THEN** 系统必须幂等 upsert 该批 `entity_edges`，并输出 created、updated、unchanged、failed 和按关系类型统计

#### Scenario: 从 PG 重建 Neo4j
- **WHEN** 某一关系批次的 PG 验收通过并运行 `graph-projector rebuild-entities`
- **THEN** Neo4j 必须包含全部可投影实体节点和当前 PG 中 active 的已审阅关系，且不得包含 PG 中不存在的历史关系

#### Scenario: 自动化验证关系清洗能力
- **WHEN** 开发者运行后端测试和 OpenSpec 校验
- **THEN** migration、loader、关系策略、repository、report 和 graph projection 边界必须具备自动化测试，且 `go test ./...` 与 `openspec validate clean-entity-relationship-foundation` 必须通过
