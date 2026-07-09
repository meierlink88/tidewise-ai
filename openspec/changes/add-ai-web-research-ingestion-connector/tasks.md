## 1. Source 配置和契约测试

- [x] 1.1 为 AI Web Research source seed 编写 loader/validator 测试，覆盖 `connector_key`、`parser_key`、`credential_ref`、`source_config` 必填字段、多 Web Search tool/模型凭证引用、禁止敏感字段和状态。
- [x] 1.2 定义 AI Web Research source seed 示例，包含 `web_search_plan`、Tavily/博查 tool 配置、搜索参数、来源偏好、可信域名、LLM provider、OpenAI-compatible 类 base URL、API 协议、模型、`prompt_ref`、`prompt_version`、`prompt_variables`、结果上限和输出 schema。
- [x] 1.3 为 AI connector 配置解析编写单元测试，覆盖缺少 `web_search_plan`、tool provider、tool credential ref、`llm_provider`、model、`prompt_ref`、`prompt_version`、api_base_url、api_protocol、max_results、output_schema、source preferences、trusted domains 和非法类型。
- [x] 1.4 实现 AI connector 配置解析和校验边界，确保 Web Search API key 和 LLM API key 只能来自 `credential_ref` 或 `source_config.credential_refs`。
- [x] 1.5 编写 repo prompt loader 测试，覆盖 prompt 文件存在、版本匹配、变量渲染、缺失变量、非法引用、路径穿越和提示词正文不进入 `source_config`。
- [x] 1.6 新增 AI Web Research repo prompt 文件示例，至少包含中国财经日度搜索 prompt 和全球宏观日度搜索 prompt。

## 2. Web Search Provider 和 LLM 调用边界

- [x] 2.1 使用 fake 或 `httptest` 编写 provider-neutral search adapter 测试，覆盖成功响应、鉴权失败、参数错误、超时、空结果、返回 URL/snippet/raw content、来源名称、发布时间和 provider 错误。
- [x] 2.2 使用 fake 或 `httptest` 编写 Tavily search adapter 测试，覆盖 `topic`、`search_depth`、`time_range`、`include_domains`、`exclude_domains`、`include_raw_content`、usage 和 request id 映射。
- [x] 2.3 使用 fake 或 `httptest` 编写博查 search adapter 测试，覆盖 `query`、`freshness`、`summary`、`count`、中文来源字段、summary、siteName、datePublished 和 provider 原始响应映射。
- [x] 2.4 使用 fake 或 `httptest` 编写 LLM normalizer 调用测试，覆盖成功结构化响应、鉴权失败、超时、非 JSON 响应、空 items 和 provider 错误。
- [x] 2.5 实现 provider-neutral 请求结构和 `web_search_plan` 编排，使 connector 可以表达多 Web Search tool、调用模式、失败策略、LLM provider、API 协议、模型、prompt 引用、搜索选项、来源偏好、可信域名、最大结果数和 timeout。
- [x] 2.6 实现 Tavily、博查 search adapter 和首个 OpenAI-compatible LLM normalizer 客户端边界，真实网络调用必须可被 fake/`httptest` 替换。
- [x] 2.7 为凭证解析和请求构造编写安全测试，确保 Authorization header、API key 查询参数、cookie 和 bearer token 不进入日志、raw metadata 或 source_config。
- [x] 2.8 编写多 Web Search tool 合并测试，覆盖 parallel、fallback、去重、可信域名排序、总结果上限、单工具失败和 report 明细。

## 3. Parser 和原始文档标准化

- [x] 3.1 编写 AI 搜索结果 parser 测试，覆盖 `items` 数组、标题、正文或摘要、来源 URL、来源名称、来源说明、引用文本、发布时间、语言、地域、主题标签、证据摘录、相关性说明和内容来源类型。
- [x] 3.2 实现 `llm_research_items` parser，将有效 item 转换为统一 raw document candidate。
- [x] 3.3 编写无效输出测试，覆盖裸数组、缺少 items、超过 max_results、未知 `content_origin`、同时缺少 URL/来源名称/来源说明/引用文本和越界投资判断字段。
- [x] 3.4 实现输出 schema 校验和越界字段处理，确保利好利空、买入卖出、涨跌预测、传导强度和事件评分不会写成系统事实。

## 4. Connector 注册和采集 runtime

- [x] 4.1 为 connector/parser registry 编写测试，验证 `llm_web_research` 和 `llm_research_items` 可以被采集 runtime 根据 source catalog 找到。
- [x] 4.2 实现 AI Web Research connector 和 parser 注册，归属 `internal/apps/ingestion` 子系统。
- [x] 4.3 为 runtime 执行 AI source 编写测试，覆盖单源成功、单源失败、凭证缺失、provider 限流配置和 report 汇总。
- [x] 4.4 确认 AI source 能复用现有 raw document 幂等写入路径，不为 AI connector 创建独立写库机制。

## 5. 可验证 smoke 和真实 provider 验证准备

- [x] 5.1 增加 fake Web Search providers + fake LLM AI Web Research smoke fixture，验证搜索结果和结构化 items 可以转换为 raw document 候选对象并输出可审阅 report。
- [x] 5.2 为真实 Tavily、博查和真实 LLM API 验证预留 gated smoke 参数，要求显式提供环境变量和 source ID，不得在普通单元测试中访问真实网络，并记录搜索结果数量、URL 数量、中国可信来源数量、抓取成功数、LLM 结构化成功数和跳过原因。
- [x] 5.3 更新本地说明，描述 `credential_ref`、`source_config.credential_refs`、`web_search_plan` 配置、来源偏好、可信域名、LLM 配置、prompt 文件、prompt 引用、真实 API key 注入和 fake/gated smoke 的运行方式。
- [x] 5.4 运行 `go test ./...`，确保后端单元测试和 gated 集成测试边界通过。
- [x] 5.5 运行 `openspec validate add-ai-web-research-ingestion-connector`。

## 6. Review 整改

- [x] 6.1 更新 design，补充 AI Web Research 的 sequence diagram、class/component diagram、Go 技术框架判断、adapter 命名规范、base URL 配置化和 adapter HTTP 公共抽象取舍。
- [x] 6.2 更新项目 agent 规则，要求后续复杂后端 design 包含 sequence diagram 和 class/component diagram。
- [x] 6.3 编写配置解析测试，覆盖 `web_search_plan.tools[].base_url`。
- [x] 6.4 编写 Tavily/博查 adapter 测试，覆盖 tool 级 `base_url` 覆盖默认 base URL。
- [x] 6.5 实现 tool 级 base URL 配置化，更新 source seed 示例。
- [x] 6.6 抽出 Web Search adapter HTTP JSON 请求公共 helper，减少 Tavily/博查重复代码。
- [x] 6.7 调整 adapter 文件命名，使主要类型与文件职责一致。
- [x] 6.8 运行 `go test ./...` 和 `openspec validate add-ai-web-research-ingestion-connector`。
- [x] 6.9 修复博查 Web Search API 真实响应 envelope 解析，覆盖 `{code, data.webPages.value}` 响应结构。
- [x] 6.10 兼容真实 LLM 返回的 `content_origin=web_content`，保证 AI Web Research smoke 可以解析并写入原始文档。
