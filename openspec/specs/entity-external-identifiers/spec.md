## Purpose

定义通用实体外部标识、唯一 identity、首批 chain_node 映射及分层写入门禁的当前系统事实。

## Requirements

### Requirement: 通用实体外部标识
系统 SHALL 使用 `entity_external_identifiers` 规范保存内部实体与外部系统标识，不得用 JSONB、拼接字符串、节点 profile 或实体类型专属 mapping 表保存同一事实。

#### Scenario: 保存外部标识
- **WHEN** 经 Review 的外部标识绑定到内部实体
- **THEN** 系统必须逐行保存 UUID id、entity_id、source_system、source_taxonomy_type、external_code、external_name、status 与 timestamps
- **AND** entity_id 必须外键引用 `entity_nodes(id)` 并使用 `ON DELETE CASCADE`
- **AND** 所有文本 identity 字段必须去除首尾空白后非空

#### Scenario: 保持 profile 最小化
- **WHEN** chain_node 具有东方财富或同花顺外部代码
- **THEN** `chain_node_profiles` 必须仍只保存 entity_id、definition 与 boundary_note
- **AND** 不得创建 `chain_node_source_mappings` 或恢复 `sector_source_mappings`

### Requirement: 外部 identity 唯一性与可扩展分类
系统 SHALL 以 `(source_system, source_taxonomy_type, external_code)` 识别唯一外部 identity，并使用可扩展文本规范值而不是 PostgreSQL enum 锁死来源或 taxonomy。同一 `(source_system, external_code)` 可以因多个来源分类展开为多条三元 identity。

#### Scenario: 拒绝外部 identity 多重绑定
- **WHEN** 同一 source_system、source_taxonomy_type、external_code 已绑定一个 entity_id
- **THEN** 数据库必须拒绝将该外部 identity 再绑定其他实体
- **AND** repository 不得通过 upsert 静默改写 entity_id

#### Scenario: 首批规范值
- **WHEN** 第一批 chain_node 外部标识进入 dry-run
- **THEN** source_system 只能为 `eastmoney` 或 `ths`
- **AND** source_taxonomy_type 必须逐行规范为 `industry_sector`、`concept_sector` 或 `index_sector`
- **AND** external_code 必须保留原始文本格式，external_name 必须保留来源平台原始名称

#### Scenario: 避免冗余唯一索引
- **WHEN** schema 已建立 `(source_system, source_taxonomy_type, external_code)` 唯一约束
- **THEN** 系统必须使用 `(entity_id, source_system, source_taxonomy_type)` 普通索引支持实体侧查询
- **AND** 不得无审阅理由重复建立被前述约束逻辑蕴含的四列唯一索引

### Requirement: 第一批外部标识范围与转换
系统 SHALL 只为已批准的 842 个第一批 chain_node 准备来自 1,156 个东方财富/同花顺来源代码的 1,169 条外部标识 mapping，不得顺带迁移旧 sector mapping、其他实体或未经 Review 的来源。

#### Scenario: 拆分工作簿来源代码
- **WHEN** data contract 工具读取已批准工作簿
- **THEN** 必须把 1,156 个来源代码拆成 1,169 条逐行 mapping，其中 eastmoney 818 条、ths 351 条
- **AND** 必须验证 241 个节点同时具有两侧代码
- **AND** 必须验证 1,143 个来源代码为单 taxonomy、13 个来源代码为双 taxonomy，且不存在一个三元 external identity 指向多个 canonical node

#### Scenario: 展开用户核验的逐代码 taxonomy
- **WHEN** 标准化节点 row 聚合了多个原始名称或来源代码
- **THEN** 转换必须从用户核验工作簿恢复每个 external_code 对应的 external_name 与 source_taxonomy_type
- **AND** 不得把聚合的组合分类字符串直接写入 source_taxonomy_type
- **AND** 当前 13 个组合来源分类 code 必须各稳定展开为 `industry_sector` 与 `concept_sector` 两条 mapping，不得二选一、排除或依赖网页来源侧核查

#### Scenario: 限制首批实体类型
- **WHEN** 第一批 external identifier mapping 绑定 entity_id
- **THEN** repository 必须验证目标实体为已批准且 active 的 `chain_node`
- **AND** 不得为 sector、industry_chain、theme 或其他实体写入本批 mapping

### Requirement: 外部标识分层门禁与幂等
系统 SHALL 将 external identifier schema 与 mapping data 作为两个独立有状态层，分别执行 `Review -> Write -> Query`，且 mapping data 必须等待 node/profile Query 验收。

#### Scenario: Schema Write 后查询
- **WHEN** schema Review 已明确授权且 migration 执行完成
- **THEN** 系统必须立即 Query 表、列、FK、唯一约束、索引、版本与重复执行幂等
- **AND** schema Query 验收前不得执行 mapping data Write

#### Scenario: Mapping data Write 后查询
- **WHEN** 842 个 node/profile Query 已验收且 1,169 条 mapping report 已明确授权
- **THEN** 系统才可写入 mapping data
- **AND** Write 后必须立即 Query 总数、provider counts、双来源节点数、13 个双 taxonomy code、taxonomy/name 完整性、唯一性、entity 绑定、孤儿与幂等

#### Scenario: 幂等更新与冲突
- **WHEN** 相同外部 identity 再次导入同一 entity_id
- **THEN** 系统只能将一致记录报告为 unchanged，或更新 external_name、status 与 updated_at
- **AND** entity_id 变化、taxonomy 未决或 external code 冲突必须报告 conflict 并阻断 Write

#### Scenario: Dry-run 核对现存外部标识
- **WHEN** 系统生成 mapping dry-run
- **THEN** 必须同时按 `(source_system, source_taxonomy_type, external_code)` 与确定性 ID 核对现存 snapshot；发现既有记录时两个索引必须齐全，并为每条已解析 mapping 输出 created、updated、unchanged 或 conflict
- **AND** tuple 换绑 entity_id、确定性 ID 漂移或两个 snapshot 索引不一致必须 conflict；同 entity 的 external_name/status 漂移才可 updated；完整一致才可 unchanged

#### Scenario: 并发换绑必然失败
- **WHEN** 两个并发事务尝试将同一 external identity 绑定到不同 entity_id
- **THEN** repository 必须在事务中串行化该 identity，并在冲突插入后重读最终 winner
- **AND** 最多一个事务可以成功，另一个必须返回 identity conflict，不得报告 unchanged 或 updated
