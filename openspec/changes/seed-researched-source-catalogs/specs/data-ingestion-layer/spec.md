## ADDED Requirements

### Requirement: 版本化采集源清单
系统 SHALL 使用 repo 内版本化采集源清单维护第一批可接入来源，并通过统一 seed 流程写入采集源目录。

#### Scenario: 加载第一批内容来源
- **WHEN** 系统执行采集源 seed
- **THEN** 系统必须从 repo 内结构化清单加载第一批调研来源，并把 Vibe-Research RSS 源和 Stock 新闻网页源映射为 `source_catalogs` 记录

#### Scenario: 校验来源清单
- **WHEN** seed 流程读取来源清单
- **THEN** 系统必须校验来源 ID、名称、通道、provider、connector、parser、来源类型、授权策略、限流策略、状态和使用说明，遇到无效配置时拒绝写入并返回明确错误

#### Scenario: 幂等写入来源
- **WHEN** 同一来源清单被重复执行 seed
- **THEN** 系统必须按稳定来源 ID 幂等 upsert `source_catalogs`，不得创建重复来源记录

### Requirement: 采集源扩展配置
系统 SHALL 支持通过 `source_config` 保存来源专属结构化参数，使不同 connector 和 parser 可以在不频繁修改表结构的情况下读取扩展配置。

#### Scenario: 保存扩展配置
- **WHEN** 采集源包含 RSSHub route 参数、网页解析策略、分类标签、分页参数或代码列表
- **THEN** 系统必须把这些非敏感结构化参数保存到 `source_config`，并在读取 active source 时还原给采集执行路径

#### Scenario: 禁止保存敏感信息
- **WHEN** 采集源需要 API key、cookie、bearer token 或私有 RSSHub 访问凭证
- **THEN** `source_config` 不得保存真实敏感值，必须只保存 `credential_ref` 或非敏感配置

### Requirement: 多来源并发采集
系统 SHALL 支持对多个 active source 进行可配置并发采集，并保持单源失败隔离、provider 限流和可测试的汇总报告。

#### Scenario: 并发执行多个来源
- **WHEN** 采集任务读取到多个 active source 且并发数大于 1
- **THEN** 系统必须并发执行来源采集，并输出包含总来源数、成功数、失败数和错误明细的 report

#### Scenario: 单源失败隔离
- **WHEN** 某个来源连接、解析或写入失败
- **THEN** 系统必须把该来源计入失败并记录错误，同时继续处理其他来源

#### Scenario: 保持 provider 限流
- **WHEN** 多个并发 worker 访问同一 `provider_key`
- **THEN** 系统必须继续通过统一 rate limiter 执行 provider 级限流，不得因为并发执行绕过限流边界

#### Scenario: 保持串行兼容
- **WHEN** 采集任务并发数设置为 1
- **THEN** 系统必须保持与现有串行执行等价的处理语义和 report 结构
