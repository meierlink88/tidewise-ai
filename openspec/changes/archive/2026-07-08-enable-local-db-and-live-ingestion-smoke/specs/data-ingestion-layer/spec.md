## ADDED Requirements

### Requirement: 真实采集 smoke 入库
系统 SHALL 提供显式运行的真实采集 smoke，使无需凭证的公开来源可以经过 connector、parser、writer 和 repository 写入本地 PostgreSQL 原始文档边界。

#### Scenario: 写入真实采集文档
- **WHEN** 开发者在已完成 migration 的 local PostgreSQL 上运行采集 smoke
- **THEN** 系统必须从公开来源采集少量真实文档，并在 `raw_documents` 中保存标题、来源、外部 ID 或内容哈希、发布时间、采集时间和入库状态

#### Scenario: 输出 smoke 结果
- **WHEN** 采集 smoke 运行完成
- **THEN** 命令必须输出结构化结果，包含成功、失败、重复和当前原始文档数量，便于人工 review

#### Scenario: 外部来源失败
- **WHEN** smoke 来源超时、不可达、限流或返回无法解析内容
- **THEN** 系统必须返回明确失败原因，不得写入伪造文档或把失败标记为成功

### Requirement: 真实 repository 幂等写入
系统 SHALL 通过 PostgreSQL repository 对原始文档执行幂等写入，避免重复 smoke 或重复采集造成重复事实基础。

#### Scenario: 重复外部 ID
- **WHEN** 同一采集源返回相同外部 ID 的文档
- **THEN** repository 必须复用或更新已有原始文档，而不是插入重复记录

#### Scenario: 重复内容哈希
- **WHEN** 同一采集源返回内容哈希相同但外部 ID 不可用的文档
- **THEN** repository 必须复用或更新已有原始文档，而不是插入重复记录

### Requirement: 采集写库 UUID 稳定性
系统 SHALL 在真实写库前为采集源和原始文档生成稳定 UUID，确保 PostgreSQL 主键类型和采集幂等策略一致。

#### Scenario: 重复生成文档 ID
- **WHEN** 同一采集源、外部 ID 和内容哈希多次进入写库流程
- **THEN** 系统必须生成相同的原始文档 UUID

#### Scenario: 接收非 UUID 候选 ID
- **WHEN** connector 或 parser 生成的候选文档 ID 不是合法 UUID
- **THEN** repository 或 ingestion helper 必须把它稳定映射为合法 UUID 后再写入 PostgreSQL
