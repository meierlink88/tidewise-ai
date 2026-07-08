## ADDED Requirements

### Requirement: AI connector 非敏感配置持久化
系统 SHALL 允许 `source_catalogs.source_config` 保存 AI Web Research connector 的非敏感运行参数，并保持真实凭证隔离。

#### Scenario: 保存 AI connector 配置
- **WHEN** AI Web Research source 被 seed 到 `source_catalogs`
- **THEN** PostgreSQL 必须保存该 source 的 API base URL、API 协议、模型名、提示词、提示词版本、搜索选项、时间窗口、结果上限、语言和输出 schema 等非敏感配置

#### Scenario: 引用真实凭证
- **WHEN** AI Web Research source 需要调用真实模型 API
- **THEN** `source_catalogs` 只能保存 `credential_ref`，真实 API key 必须来自环境变量或部署平台 secret

### Requirement: AI 采集元数据追踪
系统 SHALL 在原始文档或其 raw metadata 中保留 AI Web Research 的采集上下文，使后续审计能够追踪材料来源、模型配置和提示词版本。

#### Scenario: 保存采集上下文
- **WHEN** AI Web Research item 写入原始文档边界
- **THEN** 系统必须保留 provider、model、api_protocol、prompt_version、prompt_purpose、search_options、content_origin、retrieval_method、source_attribution_type、来源说明和原始返回片段等非敏感元数据

#### Scenario: 排除敏感元数据
- **WHEN** 保存 AI Web Research 原始返回或请求元数据
- **THEN** 系统不得保存真实 API key、Authorization header、cookie、私有 token 或其他敏感凭证
