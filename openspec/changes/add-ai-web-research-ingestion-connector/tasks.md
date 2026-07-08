## 1. Source 配置和契约测试

- [ ] 1.1 为 AI Web Research source seed 编写 loader/validator 测试，覆盖 `connector_key`、`parser_key`、`credential_ref`、`source_config` 必填字段、搜索/模型凭证引用、禁止敏感字段和状态。
- [ ] 1.2 定义 AI Web Research source seed 示例，包含 Tavily search provider、搜索参数、LLM provider、OpenAI-compatible 类 base URL、API 协议、模型、提示词、提示词版本、结果上限和输出 schema。
- [ ] 1.3 为 AI connector 配置解析编写单元测试，覆盖缺少 `search_provider`、`llm_provider`、model、prompt、api_base_url、api_protocol、max_results、output_schema 和非法类型。
- [ ] 1.4 实现 AI connector 配置解析和校验边界，确保 Tavily API key 和 LLM API key 只能来自 `credential_ref` 或 `source_config.credential_refs`。

## 2. Tavily 搜索和 LLM 调用边界

- [ ] 2.1 使用 fake 或 `httptest` 编写 Tavily search adapter 测试，覆盖成功响应、鉴权失败、参数错误、超时、空结果、返回 URL/snippet/raw content 和 provider 错误。
- [ ] 2.2 使用 fake 或 `httptest` 编写 LLM normalizer 调用测试，覆盖成功结构化响应、鉴权失败、超时、非 JSON 响应、空 items 和 provider 错误。
- [ ] 2.3 实现 provider-neutral 请求结构，使 connector 可以表达 search provider、LLM provider、API 协议、模型、提示词、搜索选项、最大结果数和 timeout。
- [ ] 2.4 实现 Tavily search adapter 和首个 OpenAI-compatible LLM normalizer 客户端边界，真实网络调用必须可被 fake/`httptest` 替换。
- [ ] 2.5 为凭证解析和请求构造编写安全测试，确保 Authorization header 不进入日志、raw metadata 或 source_config。

## 3. Parser 和原始文档标准化

- [ ] 3.1 编写 AI 搜索结果 parser 测试，覆盖 `items` 数组、标题、正文或摘要、来源 URL、来源名称、来源说明、引用文本、发布时间、语言、地域、主题标签、证据摘录、相关性说明和内容来源类型。
- [ ] 3.2 实现 `llm_research_items` parser，将有效 item 转换为统一 raw document candidate。
- [ ] 3.3 编写无效输出测试，覆盖裸数组、缺少 items、超过 max_results、未知 `content_origin`、同时缺少 URL/来源名称/来源说明/引用文本和越界投资判断字段。
- [ ] 3.4 实现输出 schema 校验和越界字段处理，确保利好利空、买入卖出、涨跌预测、传导强度和事件评分不会写成系统事实。

## 4. Connector 注册和采集 runtime

- [ ] 4.1 为 connector/parser registry 编写测试，验证 `llm_web_research` 和 `llm_research_items` 可以被采集 runtime 根据 source catalog 找到。
- [ ] 4.2 实现 AI Web Research connector 和 parser 注册，归属 `internal/apps/ingestion` 子系统。
- [ ] 4.3 为 runtime 执行 AI source 编写测试，覆盖单源成功、单源失败、凭证缺失、provider 限流配置和 report 汇总。
- [ ] 4.4 确认 AI source 能复用现有 raw document 幂等写入路径，不为 AI connector 创建独立写库机制。

## 5. 可验证 smoke 和真实 provider 验证准备

- [ ] 5.1 增加 fake Tavily + fake LLM AI Web Research smoke fixture，验证搜索结果和结构化 items 可以转换为 raw document 候选对象并输出可审阅 report。
- [ ] 5.2 为真实 Tavily 和真实 LLM API 验证预留 gated smoke 参数，要求显式提供环境变量和 source ID，不得在普通单元测试中访问真实网络，并记录搜索结果数量、URL 数量、抓取成功数、LLM 结构化成功数和跳过原因。
- [ ] 5.3 更新本地说明，描述 `credential_ref`、`source_config.credential_refs`、Tavily 配置、LLM 配置、提示词配置、真实 API key 注入和 fake/gated smoke 的运行方式。
- [ ] 5.4 运行 `go test ./...`，确保后端单元测试和 gated 集成测试边界通过。
- [ ] 5.5 运行 `openspec validate add-ai-web-research-ingestion-connector`。
