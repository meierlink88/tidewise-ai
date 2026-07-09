## ADDED Requirements

### Requirement: AI 查询计划与原始文档标准化分离
系统 SHALL 保持 AI Web Research 的查询计划生成和原始文档标准化分离，避免采集层回退到由模型格式化原始文档。

#### Scenario: Go 程序化标准化搜索结果
- **WHEN** AI Web Research source 通过 LLM planner 生成查询计划并完成 Web Search
- **THEN** 系统必须继续由 Go 程序把搜索结果映射为 parser 可校验的结构化 items，而不是要求 LLM 根据搜索结果生成 items

#### Scenario: prompt 文件不承载标准化 schema
- **WHEN** 开发者查看 AI Web Research repo prompt
- **THEN** active 查询计划 prompt 不得成为 raw document schema 的事实来源，raw document 候选对象格式必须继续由 Go parser、测试和 OpenSpec 主规格约束
