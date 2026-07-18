# Ingestion 子系统说明

## AI Web Research 采集

`llm_web_research` 是采集子系统中的一个 connector。它通过 `source_catalogs` 驱动，可以使用固定查询计划，也可以先调用 OpenAI-compatible LLM search planner 把采集意图转换为查询计划；随后由 Go 程序调用 Web Search API 获取候选材料、去重排序并映射为结构化 `items`，最后由 `llm_research_items` parser 转成统一 `raw_documents` 候选对象。

当前正式 AI source 使用 `collection_mode=search_results` 和 `search_plan_mode=llm_query_plan`。LLM 只负责生成 `queries` 查询计划，不负责生成新闻标题、正文、URL、事件、标签、实体关系或 raw document 字段。

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
- `source_config.web_search_plan.tools[].base_url`: 配置 Web Search provider 的官方地址、代理网关或私有化服务地址；代码默认地址只作为 fallback。
- `source_config.collection_mode`: 当前正式路径为 `search_results`。
- `source_config.search_plan_mode`: 当前正式路径为 `llm_query_plan`，固定查询回归测试使用 `static_query_plan`。
- `source_config.credential_refs.planner`: LLM planner API key 的环境变量引用。
- `source_config.api_base_url`: LLM planner OpenAI-compatible base URL。
- `source_config.model`: LLM planner 模型名。
- `source_config.prompt_ref`: repo prompt 文件引用，当前正式路径使用 `ingestion/ai_web_research/search-plan.v1.md`。
- `source_config.prompt_version`: prompt 版本。
- `source_config.prompt_variables`: prompt 渲染变量。
- `source_config.max_results`: 单次搜索结果总上限。
- `source_config.trusted_domains`: 可信来源域名，用于排序和审计。

真实密钥不得写入 `source_config`、seed 文件、配置文件或源码，只能通过 `credential_ref` 引用环境变量或部署 secret。

## Prompt 文件

长提示词正文位于：

```text
backend/data/prompts/ingestion/ai_web_research/
```

`source_config` 只保存 `prompt_ref`、`prompt_version` 和少量变量。修改提示词时必须经过代码审阅，并保持文件名版本和 `prompt_version` 一致。

## 本地 fake 验证

普通单元测试使用 fake Web Search provider 和 fake LLM，不访问真实网络：

```bash
go test ./internal/apps/ingestion/connectors
go test ./internal/apps/ingestion/parsers
go test ./internal/apps/ingestion/core ./internal/apps/ingestion/sourcecatalog
```

这些测试验证：

- 多 Web Search tool 合并、去重、排序和 provider report。
- nested `credential_ref` 解析。
- LLM 查询计划输出和校验。
- Go 程序化映射搜索结果为结构化 `items`。
- `llm_research_items` parser 校验。
- connector/parser DTO 与 source-aware 映射合同。

## 运行与导入边界

Tidewise 不再提供 `ingestion-scheduler`、`source-ingest` 或 `ingest-smoke` 命令，也不在 Data Service 中新建替代 worker。采集调度和实际运行由独立 agent-run 项目负责。

本仓库短期保留 connector、parser、registry、`EnvCredentialResolver`、sourcecatalog、prompt 与相关测试，供合同演进和未来显式交接使用；这些包不构成可独立运行的 Tidewise 采集 command。不得在本 change 中把它们迁往外部仓库。

agent-run 应读取 Data Service 的受控 source metadata，并通过 `/internal/data/v1/raw-document-imports` 或 `/internal/data/v1/reviewed-event-imports` 提交结果。Data Service 负责 whole-batch validation、原子写入、caller-scoped idempotency 和 receipt/status；agent-run 不得直接写 Data DB。历史 scheduler/run tables 与 migrations 保留用于历史兼容，不表示 Tidewise 仍拥有 runtime。
