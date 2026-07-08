## ADDED Requirements

### Requirement: 实体基础 seed 数据
系统 SHALL 提供一阶段实体基础 seed 数据，用于初始化六层传导和事件知识图谱所需的基础实体、profile 和客观关系。

#### Scenario: 初始化联盟组织
- **WHEN** 实体 seed 执行
- **THEN** 系统必须初始化一批核心联盟组织实体，至少覆盖 `OPEC+`、`OPEC`、`G7`、`G20`、`WTO`、`IMF`、`World Bank`、`OECD` 和 `EU`

#### Scenario: 初始化所有实体类型
- **WHEN** 实体 seed 执行
- **THEN** 系统必须初始化联盟组织、经济体、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物的一阶段基础数据，并在 report 中输出各类型数量

#### Scenario: 初始化客观基础关系
- **WHEN** 实体 seed 包含实体关系
- **THEN** 系统必须只写入成员关系、归属关系、上市关系、市场指数关系、指标适用关系等客观基础关系，不得写入推理结论或投资判断

### Requirement: 实体 seed 校验
系统 SHALL 在写入数据库前校验实体 seed 的结构、引用关系和安全边界。

#### Scenario: 拒绝重复实体 key
- **WHEN** seed 文件中出现重复实体 key
- **THEN** loader 或 validator 必须返回明确错误，并阻止继续写入

#### Scenario: 拒绝悬空 profile 或关系
- **WHEN** profile 或实体关系引用不存在的实体 key
- **THEN** validator 必须返回明确错误，并阻止继续写入

#### Scenario: 禁止推理结论字段
- **WHEN** seed 文件包含利好利空、预测结论、传导强度、事件评分或投资建议字段
- **THEN** validator 必须返回明确错误

### Requirement: 实体 seed 报告
系统 SHALL 在实体 seed 执行后输出可审阅 report，使开发者能够确认初始化范围和结果。

#### Scenario: 输出初始化统计
- **WHEN** 实体 seed 命令执行完成
- **THEN** report 必须包含实体总数、按实体类型统计、按层级统计、各 profile 表写入数量、关系类型统计、created、updated、unchanged 和 failed 数量
