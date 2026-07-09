## ADDED Requirements

### Requirement: 查询计划 prompt 治理
系统 SHALL 将 AI Web Research 的 active LLM prompt 限定为查询计划生成用途，不得要求模型输出 raw document item 格式。

#### Scenario: active prompt 只生成查询计划
- **WHEN** AI Web Research source 配置 `search_plan_mode=llm_query_plan`
- **THEN** 该 source 引用的 repo prompt 必须只要求模型输出 `queries` 查询计划，并不得要求模型输出 `items`、`meta`、`content_text`、`content_origin`、事件、标签、实体关系或 raw document 字段

#### Scenario: 旧 normalizer prompt 不可被 active source 引用
- **WHEN** source seed 定义 AI Web Research source
- **THEN** active source 不得引用用于 LLM normalizer 的旧 prompt 文件，必须引用查询计划 prompt 版本

#### Scenario: provider 分工明确
- **WHEN** 查询计划 prompt 渲染给 LLM planner
- **THEN** prompt 必须明确 Tavily 和博查 Web Search 的使用边界，使中国财经信息优先使用中文查询和中文来源，全球宏观信息优先使用英文查询和全球来源

#### Scenario: 保持投资建议安全边界
- **WHEN** LLM planner 生成查询计划
- **THEN** prompt 必须继续排除价格预测、买入卖出、直接投资建议、营销软文和无来源线索内容
