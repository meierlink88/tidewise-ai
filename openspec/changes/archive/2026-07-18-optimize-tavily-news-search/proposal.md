## Why

现有 Tavily connector 允许自动参数改写、按通用候选上限请求结果，并优先采集完整网页正文；这会削弱中国资本市场新闻检索的确定性，引入低质量尾部结果和不必要的大正文。需要把 Tavily 调整为可复核、质量优先的新闻搜索，同时保留现有单次调用和查询传递方式，为实现后的真实运行对比建立稳定基线。

## What Changes

- Tavily 搜索请求显式使用 provider 专用、可配置的 `topic`（仅允许 `general`、`news`、`finance`）、`search_depth: "advanced"`、确定的 `start_date`/`end_date`，并关闭 `auto_parameters`；仅改变 topic 的三组真实对比表明 `general` 的固定中文 A 股 query 严格相关性最佳，因此默认采用 `general`，同时允许按采集意图覆盖。
- 不再请求 `include_raw_content`，保持 `include_answer: false`，候选内容仅采用每条结果中与 query 相关的 `content`，并按 snippet 等级标记。
- 为 Tavily 增加独立、可配置的质量优先结果上限，默认采用 Tavily 官方推荐默认值 5；实际请求量取该上限与通用 `CandidateLimit` 的较小值，允许运行时在 Tavily 支持的 1–20 范围内调整，不改变其他 connector 的候选上限。
- 保留 `SearchQueries` 拼接为一次 Tavily query 的现有方式；不增加 Tavily 调用次数。
- 使用 fake HTTP 按 `RED -> GREEN -> REFACTOR` 验证请求参数、查询传递、结果上限以及 `content`/`ContentLevel` 行为；自动化测试不访问真实 API。
- 实现前提供不持久化密钥的受控 Tavily topic A/B：固定非敏感 query、绝对日期窗口、advanced、3 chunks、关闭 auto/raw/answer 和 `max_results: 5`，分别对 `general`、`news`、`finance` 至少运行三次，只改变 topic 并据证据选择默认值；5/10 结果预算作为独立观察，完整 collector smoke 仅作端到端补充。
- 非目标：不按 Tavily `score` 过滤，不增加域名黑白名单、垃圾/SEO 检测或 LLM 后置门禁，不新增 connector，不修改 Bocha/Parallel、采集意图 prompt 或数据目录，也不把真实密钥写入日志、规格或产物。

## Capabilities

### New Capabilities

- `tavily-news-search`: 规定 Tavily 面向资本市场资讯检索的可配置 topic、确定性请求参数、质量优先结果预算、query-related 内容映射及安全验证方式。

### Modified Capabilities

无。

## Impact

- 受影响 Agent：仅 AI 采集器；AI 事件提取器与 AI 投研报告分析师不变。
- 预计涉及 `internal/connectors/` 的 Tavily HTTP adapter 及测试、`cmd/collector/` 的 Tavily 专用运行时配置与参数校验；不引入新依赖或外部服务。
- 兼容性：connector 名称、认证方式、一次调用模型、`SearchQueries` 传递、通用 `CandidateLimit` 和候选结构保持兼容；新增 Tavily 专用 topic 技术参数，内容等级由可能的 `full_text` 收敛为 `snippet`/`title_only`，默认结果数由通用上限收敛到 5。
- 主要风险：较低默认上限可能降低覆盖率；`general` 的中文 A 股严格相关性最佳，但样本没有 `published_date` 且包含窗口外旧资料。请求仍显式发送日期窗口，内容仍禁用 raw，并保留受约束 topic 覆盖；日期元数据与 freshness 局限作为残余风险公开。`advanced` 每次请求固定消耗 2 credits，本 change 不增加调用次数。
- 流程职责：执行 Agent仅完成提案并提交 Leader Review；开发 Leader明确批准后方可 Apply。实现与验证完成后仍需 Leader Acceptance 才能 Sync、Archive 和 Deliver；执行 Agent负责提交、推送、PR 与 cleanup handoff，用户控制 PR merge，Leader在合并后执行 Cleanup，且不回写已归档 tasks。
