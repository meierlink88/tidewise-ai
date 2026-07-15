## Purpose

定义实体关系空基线、分批人工审阅、来源与语义校验、PostgreSQL 写入和 Neo4j 重建的当前系统事实。

## Requirements

### Requirement: 产业节点关系独立策展
系统 SHALL 将 chain_node 静态关系与通用 `entity_edges` 分离，在独立 `chain_node_relations` 中按四类语义逐层 Review、Write 和 Query。

#### Scenario: Review 产业节点候选边
- **WHEN** 系统准备 `is_subcategory_of`、`is_component_of`、`input_to` 或 `depends_on` 候选边
- **THEN** 清单必须包含中文端点名称、entity key、方向、mechanism、condition、evidence/provenance、反例、不确定性和 Review disposition

#### Scenario: 分类和组成关系使用内部派生证据
- **WHEN** `is_subcategory_of` 或 `is_component_of` 只表达已批准节点 definition/boundary 可直接判定的稳定集合从属或物理/系统组成
- **THEN** evidence 可以使用 approved internal source artifact path、SHA-256、确定性 derivation rule 与两遍独立 AI Review 记录，不强制伪造外部来源 URL
- **AND** 最终可写 manifest 仍必须保存 `verified_at`，且不得将该证据扩张解释为投入、依赖、供给瓶颈或事件传导

#### Scenario: 因果和物理约束保持强外部证据
- **WHEN** 候选是 `input_to`、`depends_on` 或 physical constraint
- **THEN** 必须提供可定位的强外部证据、source URL、verified_at、成立条件和反例
- **AND** 只有名称、definition/boundary、词面邻近或内部派生规则时必须保持 blocked 或 rejected

#### Scenario: 委托双遍 AI Review
- **WHEN** 用户明确委托 AI 处理专业关系逐项判断
- **THEN** 第一遍生成与自检、第二遍独立 Reviewer 的 approve/reject/blocked 结论必须完整留痕
- **AND** 双遍 AI Review 不替代 schema/data R2 的人工授权

#### Scenario: 未批准候选不得写入
- **WHEN** 某条基于新 chain_node 的候选边尚未取得已批准 evidence contract 下的完整 Review disposition，或可写 manifest/R2 package 尚未冻结
- **THEN** 系统不得将其写入正式 seed、PostgreSQL 或 Neo4j

### Requirement: 实体关系空基线
系统 SHALL 支持在保留实体主数据的前提下建立空关系基线，使后续实体关系只能通过已审阅批次重新写入。

#### Scenario: 清空 local PostgreSQL 实体关系
- **WHEN** 开发者执行已批准的 local 关系重置流程
- **THEN** 系统必须删除 `entity_edges` 中的关系数据，并保持 `entity_nodes`、全部实体 profile、采集数据、事件数据和其他业务表不变

#### Scenario: 清空 local Neo4j 投影数据
- **WHEN** PostgreSQL 实体关系已经清空且开发者执行 local Neo4j 重置
- **THEN** Neo4j 中的节点和关系数据必须被删除，同时保留 database、约束、索引和本地基础设施配置

#### Scenario: 防止旧关系自动恢复
- **WHEN** 空关系基线建立后再次执行实体主数据 seed
- **THEN** 系统不得自动写回未经分批 review 的历史样例关系

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

#### Scenario: 按投研价值暂缓低优先级关系
- **WHEN** 某一关系族尚不属于当前事件推导关键路径
- **THEN** 系统必须允许保留候选审阅材料但保持正式 seed 为空，并且不得将未确认数据写入 PostgreSQL 或 Neo4j

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
系统 SHALL 将已审阅的通用 `entity_edges` 关系先写入 PostgreSQL，再按其 change 契约从 PostgreSQL 重建 Neo4j，不允许直接手工维护 Neo4j 关系事实；产业节点关系例外地写入独立 `chain_node_relations`，且本 change 只执行 PostgreSQL Write 与 Query，不执行 Neo4j rebuild。

#### Scenario: 写入已审阅关系批次
- **WHEN** 某一关系族通过 review 并执行实体 seed
- **THEN** 系统必须幂等 upsert 该批 `entity_edges`，并输出 created、updated、unchanged、failed 和按关系类型统计

#### Scenario: 从 PG 重建 Neo4j
- **WHEN** 某一关系批次的 PG 验收通过并运行 `graph-projector rebuild-entities`
- **THEN** Neo4j 必须包含全部可投影实体节点和当前 PG 中 active 的已审阅关系，且不得包含 PG 中不存在的历史关系

#### Scenario: cleanup 后投影暂时陈旧
- **WHEN** 本 change 已清理 PostgreSQL 旧产业事实但后续投影 change 尚未执行
- **THEN** 系统必须将 Neo4j 标记为暂时陈旧
- **AND** 本 change 不得尝试清理、写入或重建 Neo4j

#### Scenario: 产业节点关系仅验收 PostgreSQL
- **WHEN** 本 change 的 chain_node relation 批次通过 Review 并获得 Write 授权
- **THEN** 系统必须幂等写入 `chain_node_relations` 并执行写后 Query
- **AND** 不得运行 Neo4j rebuild、清理或直接图写入

#### Scenario: 自动化验证关系清洗能力
- **WHEN** 开发者运行后端测试和 OpenSpec 校验
- **THEN** migration、loader、关系策略、repository、report 和 graph projection 边界必须具备自动化测试，且 `go test ./...` 与 OpenSpec 全局校验必须通过

### Requirement: Formal-Active Member Of 重建范围
系统 SHALL 只把 approved formal-active economy membership 重建为 `economy -> alliance_org member_of`，并把其他关系语义排除出本 change。

#### Scenario: 建立正式成员关系
- **WHEN** frozen artifact 中的 economy 是官方来源列明的 active 正式成员
- **THEN** importer 必须使用 economy → alliance_org 的 `member_of`，并保留 approved source、核验时间和 endpoint stable keys

#### Scenario: 排除非正式身份
- **WHEN** 候选身份是 observer、partner、applicant、suspended、former、participant、signatory 或 framework participant
- **THEN** 系统不得将其写为 active `member_of`，也不得在本 change 新增 relation type

#### Scenario: 排除可选关系
- **WHEN** alliance profile 包含核心主导方文本或候选材料提及下属机制
- **THEN** 不得自动生成 `led_by` 或 `part_of`；`participates_in`、`signatory_to` 也不在本 change

### Requirement: 废止旧 Member Of Disposition 策略
系统 SHALL 在 local 探索重建中以 latest approved 133 tuples 作为最终基线，不维护旧 223 条边的 keep/preserve/proposed-inactivate 执行策略。

#### Scenario: 读取旧 Disposition
- **WHEN** Review 历史包含 31 keep、160 preserve_unresolved、10 preserve_pending_retype 或 22 proposed_inactivate
- **THEN** importer 必须忽略其执行语义，不实现 preserve policy、forward inactivate 或全库 convergence engine

#### Scenario: 重建后核对集合
- **WHEN** 4.2 relationship rebuild 完成
- **THEN** active formal `member_of` 目标集合必须精确等于 approved 133 tuples，且每条边方向正确、两端存在且 active、无孤儿或重复

### Requirement: Cleanup 前关系依赖审计
系统 SHALL 在 R3 cleanup 授权前只读枚举 alliance/economy 端点涉及的全部 relation types 和 counts，区分本批 `member_of` 与跨域事实。

#### Scenario: 区分 Member Of 与 Has Market
- **WHEN** dependency audit 检查仓库 fixture 或真实 local `entity_edges`
- **THEN** 必须分别报告 economy → alliance_org `member_of` 与 economy → market `has_market`；fixture 已知基线为 223 与 40，真实 local counts 必须在执行前刷新

#### Scenario: 审阅其他跨域关系
- **WHEN** 发现 economy/alliance 与 market、index、benchmark、industry chain、company、person 或其他实体的关系
- **THEN** 必须列出 relation type、方向、端点、count/hash，并提交“删除并丢弃”或“保留/重建”决定；未决时阻止 cleanup

#### Scenario: 保护未授权跨域事实
- **WHEN** 4.1 R3 cleanup 获得授权
- **THEN** 只能删除 economy → alliance_org `member_of`；全部非 `member_of` economy 跨域 tuple 的 pre/post identity、端点、type 与 status 必须不变，任何其他 alliance incident edge 必须 fail-closed

### Requirement: Cleanup 与 Relationship Rebuild 独立授权
系统 SHALL 在 4.1 R3 中先完成 approved relation scope cleanup/zero Query，再在 4.2 R2 中重建 133 条关系；两者不得合并推定授权。

#### Scenario: Cleanup Zero Gate
- **WHEN** 4.1 完成
- **THEN** Query 必须证明获批旧 alliance/economy membership scope 为零，并证明未授权跨域事实未变化；失败时不得进入 rebuild

#### Scenario: Rebuild Exact Gate
- **WHEN** 4.2 获得独立授权并执行
- **THEN** Query 必须证明 133 条 approved tuples 集合相等、来源字段有效、端点 active、无孤儿/重复、方向正确且幂等复跑不改变集合

#### Scenario: 禁止 Neo4j 写入
- **WHEN** PostgreSQL cleanup 或 rebuild 执行
- **THEN** 不得连接、写入或重建 Neo4j；图投影由后续独立 change 处理

### Requirement: 842 节点 usable-map additive 候选审批
系统 SHALL 保留既有 100 条 accepted baseline，并在 842 个既有 chain_node 之间发现和双遍审核 additive 四类关系候选；只有 842/842 节点均参加发现与检查、候选与异常/冲突全部处置后才能冻结 additive final 数据，并保持 PostgreSQL Write 与 Neo4j sync 独立授权。

#### Scenario: 分批候选审核
- **WHEN** 某个节点群的关系候选与证据已完整展示
- **THEN** 用户必须审核 identity、端点、类型、方向、机制、条件、证据 tier、反例和异常/冲突项
- **AND** 任一分批审核都不得推定 PostgreSQL 或 Neo4j Write 授权

#### Scenario: 全量 final 结果冻结
- **WHEN** 842/842 节点均已参加候选发现和两遍独立检查
- **THEN** 系统必须证明既有 100 条逐行保留，且两遍分别记录理由、类型、方向、机制、路径、条件、具体反例、来源蕴含与 disposition
- **AND** 任一遍不一致的候选必须阻断；无未处置候选、孤儿端点、重复 tuple、同机制重复或未解决冲突时才能冻结 additive final 关系数据
- **AND** 无关系事实的节点允许保持无边，不得为覆盖率强制登记关系

#### Scenario: 来源实际蕴含关系
- **WHEN** 某条候选声明为 Tier 1
- **THEN** 来源必须实际支持该 relation type、方向和具体机制，并记录逐边 source-to-edge entailment
- **AND** 产业链相关性、来源组名称或对制造依赖设备的描述不得被改写为来源未支持的 `input_to`

#### Scenario: PostgreSQL 先写并验收
- **WHEN** additive final 数据获得独立 local PostgreSQL R2 授权
- **THEN** 系统必须只向 `chain_node_relations` additive 写入，并立即 Query 既有 100 条保护、端点、tuple、orphan、节点主数据保护和幂等结果

#### Scenario: Neo4j 后同步并验收
- **WHEN** PostgreSQL additive 写后 Query 已验收且对应 local Neo4j R3 sync 另行获得授权
- **THEN** projector 必须只从最终 PostgreSQL accepted baseline 同步批准数据并 Query
- **AND** 不得从 Neo4j 反写 PostgreSQL 或同步未审核数据
