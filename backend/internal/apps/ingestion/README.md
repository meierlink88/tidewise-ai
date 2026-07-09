# Ingestion 子系统说明

## AI Web Research 采集

`llm_web_research` 是采集子系统中的一个 connector。它通过 `source_catalogs` 驱动，先调用 Web Search API 获取候选材料，再调用 OpenAI-compatible LLM normalizer 输出结构化 JSON，最后由 `llm_research_items` parser 转成统一 `raw_documents` 候选对象。

当前支持的 Web Search tool：

- `tavily`：通用 Web Search API，适合全球政经和英文材料召回。
- `bocha_web_search`：中文 Web Search API，适合中国财经来源召回。

## Source 配置

AI Web Research source seed 位于：

```text
backend/data/source_catalogs/ai_web_research_sources.json
```

核心字段：

- `connector_key`: 固定为 `llm_web_research`。
- `parser_key`: 固定为 `llm_research_items`。
- `source_config.web_search_plan`: 配置一个或多个 Web Search tool、执行模式、每个 tool 的 `max_results`、`credential_ref` 和 provider options。
- `source_config.credential_refs.llm`: LLM API key 的环境变量引用。
- `source_config.api_base_url`: LLM OpenAI-compatible base URL。
- `source_config.model`: LLM 模型名。
- `source_config.prompt_ref`: repo prompt 文件引用。
- `source_config.prompt_version`: prompt 版本。
- `source_config.prompt_variables`: prompt 渲染变量。
- `source_config.max_results`: 单次结构化结果上限。
- `source_config.trusted_domains`: 可信来源域名，用于排序和审计。

真实密钥不得写入 `source_config`、seed 文件、配置文件或源码，只能通过 `credential_ref` 引用环境变量或部署 secret。

## Prompt 文件

长提示词正文位于：

```text
backend/data/prompts/ingestion/ai_web_research/
```

`source_config` 只保存 `prompt_ref`、`prompt_version` 和少量变量。修改提示词时应通过 OpenSpec review，并保持文件名版本和 `prompt_version` 一致。

## 本地 fake 验证

普通单元测试使用 fake Web Search provider 和 fake LLM，不访问真实网络：

```bash
go test ./internal/apps/ingestion/connectors
go test ./internal/apps/ingestion/parsers
go test ./internal/apps/ingestion/runtime
```

这些测试验证：

- 多 Web Search tool 合并、去重、排序和 provider report。
- nested `credential_ref` 解析。
- LLM 结构化 JSON 输出。
- `llm_research_items` parser 校验。
- 复用现有 `IngestionJob` 和 `RawDocumentWriter` 幂等写入路径。

## 真实 gated smoke

真实 API 验证必须显式指定 source ID 和所需环境变量，避免普通开发或 CI 误触发真实网络和付费 API。

示例：

```bash
APP_ENV=local DATABASE_PASSWORD=tidewise-local-dev-password \
TAVILY_API_KEY=... BOCHA_API_KEY=... DEEPSEEK_API_KEY=... \
go run ./cmd/source-ingest \
  -source-id tidewise:ai-web-research:cn-finance-daily \
  -provider llm_web_research \
  -channel ai_web_research \
  -source-type news \
  -require-env TAVILY_API_KEY,BOCHA_API_KEY,DEEPSEEK_API_KEY
```

运行前应确认目标 source 已 seed 到本地数据库，且 source 状态为 `active`。真实验证结果以 `source-ingest` 输出的 JSON report 和 `raw_documents` 入库结果为准。
