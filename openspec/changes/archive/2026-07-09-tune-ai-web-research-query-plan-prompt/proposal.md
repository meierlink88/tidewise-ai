## Why

AI Web Research 已切换为“LLM 只生成查询计划，Go 程序执行搜索结果标准化”的模式，但 repo 中仍残留两份旧 normalizer prompt，里面要求模型输出 `items`、`meta`、`content_text` 和 `content_origin`。这些旧 prompt 容易误导后续开发或 agent 回到“让模型整理 raw document”的旧架构。

## What Changes

- 新增更严格的 AI Web Research 查询计划 prompt 版本，只允许 LLM 输出 `queries` 查询计划。
- 明确 prompt 中不得要求模型输出 raw document、`items`、`meta`、正文、事件、标签或实体关系。
- 将 AI Web Research source seed 的 `prompt_ref` 和 `prompt_version` 更新到新查询计划 prompt。
- 删除或废弃旧的中文财经、全球宏观 normalizer prompt，避免后续误用。
- 更新 prompt/source seed 测试，确保 repo prompt 目录不再保留 active normalizer 输出格式要求。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `ai-web-research-ingestion-connector`: 收紧 LLM 查询计划 prompt 契约，明确 AI Web Research 的 prompt 只能服务查询计划生成，不服务 raw document item 格式化。
- `data-ingestion-layer`: 保持 AI Web Research 采集通道由 Go 程序化映射搜索结果，避免旧 prompt 破坏采集层职责边界。

## Impact

- 影响 `backend/data/prompts/ingestion/ai_web_research/` 下的 prompt 文件。
- 影响 `backend/data/source_catalogs/ai_web_research_sources.json` 中的 prompt 引用。
- 影响 promptstore、sourcecatalog 或 connector 相关测试。
- 不涉及数据库 migration、外部 API 契约、真实密钥、前端、prototype 或 doc 目录。
