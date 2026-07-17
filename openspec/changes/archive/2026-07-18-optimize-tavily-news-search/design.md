## Context

AI 采集器先由查询规划器生成 `Request.SearchQueries`，随后 Eino Workflow 把同一份 Request fan-out 给 Parallel、Tavily、Bocha，并在所有 connector 完成后 fan-in 到 materializer。当前 Tavily adapter 把全部 `SearchQueries` 以空格拼接为单个 query，每个 collector run 只调用一次 Tavily；但请求同时启用 `auto_parameters`、使用通用 `CandidateLimit` 作为 `max_results`、请求 `include_raw_content: "markdown"`，并优先把 `raw_content` 标记为 `full_text`。

Tavily 官方 Search API 当前支持 `topic: "general" | "news" | "finance"`；`advanced` 返回每个 URL 的多个语义相关片段且每次搜索消耗 2 credits，`include_raw_content` 返回清洗后的完整页面内容；Best Practices 说明 `max_results` 默认 5，设置过高可能返回低质量结果。API 支持 1–20。本 change 不增加新服务或依赖，只调整现有 provider adapter 和运行时参数。首次 Apply 的受控组合 A/B 暴露出 `news` 对固定中文 A 股 query 的严格相关率低于旧 general 组合，因此本轮不得从 topic 名称预设默认值，必须先做只改变 topic 的重复对比。

### Eino reference audit

- `eino-ext`：在 commit `9137edd89e72b72735ede69db1c5ae29178a6e41` 审计 `components/tool/searxng/` 的 README、配置、构造器、HTTP 实现、测试、示例和 `go.mod`，并检索 `components/tool/duckduckgo/` 等 WebSearch 组件。上游提供通用 Eino tool 和其他搜索 provider adapter，但没有 Tavily Search API、其参数或本项目 `collector.Candidate` 映射，故不替换现有项目 adapter。
- `eino-examples`：在 commit `171220631fb7068ead50b7cd964b8c471647117d` 审计 `quickstart/eino_assistant/eino/einoagent/`；该示例展示 START fan-out、多个前驱 fan-in、搜索 tool 封装和明确的配置边界。采用“provider 细节留在 adapter、编排只负责组合”的模式，不复制示例代码。
- `eino`：在 commit `922b6a8a233b5233fe47eecee6cd2c005e8c39cd` 审计 `compose/workflow.go`、`workflow_test.go`、`types.go`、`dag.go`、`dag_test.go`、`graph.go` 与 compile options。`Workflow` 底层固定使用 DAG/`AllPredecessor`，节点仅在所有前驱完成后触发，并默认 eager 执行；现有 connector fan-out 和 materializer fan-in 语义适合本 change，无需调整图结构。
- gap：Tavily 是项目 HTTP adapter 的 provider-specific 能力缺口；实现保持在 `internal/connectors/`，不修改 `.reference/cloudwego/`。

## Goals / Non-Goals

**Goals:**

- 让 Tavily 确定性执行带绝对日期窗口、显式且受约束可配置 topic 的 advanced 搜索，并以受控证据选择默认 topic。
- 避免完整网页大正文进入候选，使用 query-related `results[].content` 并准确降低内容等级。
- 通过 Tavily 独立、可配置的结果预算优先保障前部质量，同时不突破通用候选上限。
- 保持一次请求、现有 query 拼接、connector 接口和 Eino workflow 不变。
- 用 fake HTTP 完整覆盖请求/响应契约，并给出不泄露密钥的真实 A/B 方法。

**Non-Goals:**

- 不按 `score` 过滤，不增加域名规则、垃圾/SEO 检测或 LLM 后置门禁。
- 不拆分 `SearchQueries` 为多次 Tavily 请求，不新增 connector。
- 不修改 Parallel、Bocha、查询规划器、采集意图 prompt、materializer 契约或 `data/`。
- 自动化测试不访问 Tavily、DeepSeek 或其他真实 API。

## Decisions

### 1. 显式、确定且可配置的 topic 请求

请求 SHALL 包含经校验的 Tavily 专用 topic（仅允许 `general`、`news`、`finance`）、`search_depth: "advanced"`、`chunks_per_source: 3`、`include_answer: false` 以及按 `CollectedAt`/`TimeWindowHours` 计算的 `start_date`、`end_date`。请求中不发送 `auto_parameters`；Tavily 默认值为 false，因此省略比同时声明自动模式和手工参数更清晰，也避免 provider 重新推断 topic/depth。默认 topic 采用下述只改变 topic 的受控证据所选 `general`，并通过 `-tavily-topic` 允许操作者在三个官方值内覆盖；非法值须在 HTTP 前失败。

替代方案是发送 `auto_parameters: false`。两者在当前 API 等价，但省略字段能更直接证明未启用自动参数；fake HTTP 将断言该键不存在，并断言所有关键显式参数存在。

### 2. 只消费 query-related content

不发送 `include_raw_content`，响应结构也不再依赖 `raw_content`。`results[].content` 非空时映射为 `Candidate.Content` 和 `snippet`；为空时沿用公共 fallback，以标题作为内容并标记 `title_only`。`include_answer` 保持 false。

不采用两阶段 Search + Extract，因为本 change 目标是减少大正文、保持单次调用，且 Extract 会增加调用、延迟和 credits。也不保留 raw-content fallback，因为只要请求或 provider 行为变化就可能再次把完整正文送入采集链路。

### 3. Tavily 专用质量预算默认 5，并可配置

在 `Tavily` adapter 增加 provider-specific `MaxResults`，由 collector CLI 的 `-tavily-max-results` 注入，默认 5，允许 1–20。最终请求 `max_results = min(Request.CandidateLimit, Tavily.MaxResults)`；通用候选上限较小时仍是硬上界，其他 connector 不受影响。无效配置在发起 provider 请求前失败。

选择 5 是因为 Tavily 官方默认值即为 5，并明确警告高值会引入低质量尾部；当前通用默认 10 更适合作为跨 connector 的总体上界，不应无差别覆盖 provider 的质量建议。`max_results` 不改变 advanced search 每次 2 credits 的计费，本 change 又保持每 run 一次调用；较高可配置值主要增加响应体、下游候选量和尾部噪声，而非搜索调用次数。可配置性允许后续用 5、8、10 做真实对比，而不是把经验值永久硬编码。

替代方案包括继续直接使用 `CandidateLimit`（无法表达 provider 质量预算）、固定常量 5（无法做受控对比）和为所有 connector 修改通用上限（扩大范围且影响 Parallel/Bocha），均不采用。

### 4. 保留单 query、单调用与现有编排

继续使用 `strings.Join(request.SearchQueries, " ")` 形成一个 Tavily query，并只执行一次 POST。官方建议复杂问题可拆分查询，但本 change 明确不增加多次调用；查询规划和 connector fan-out 属于其他边界。fake HTTP transport 记录调用次数并验证 query。

### 5. TDD 与分层受控真实对比

- RED：先增加 Tavily fake HTTP 测试，断言新参数、禁止字段、单次调用、有效结果预算、只采用 `content` 及 `snippet`/`title_only`；在旧实现上应因 `topic` 缺失、`auto_parameters`/`include_raw_content` 存在、raw content 优先和上限错误而失败。为 CLI 增加默认值、显式覆盖和非法范围测试。
- GREEN：仅实现足以通过上述测试的 adapter 配置、请求和响应映射。
- REFACTOR：抽取清晰的有效上限/校验边界（若测试显示必要），保持 Bocha/Parallel diff 为零，再运行聚焦及全量测试。
- 主要证据采用直接、受控的 Tavily A/B，不运行查询规划器。实现完成并获准真实调用后，在 `/tmp` 创建一次性 runner；runner 只从进程环境读取 `TAVILY_API_KEY`，脚本本身不含密钥，Authorization 不作为命令参数，响应也只写入 `/tmp`。每组使用完全相同的固定非敏感 query、固定 `start_date`/`end_date`、`search_depth: "advanced"`、`chunks_per_source: 3` 和 `include_answer: false`，每个 payload 至少重复三次并记录 UTC 时间。
- Leader Acceptance 修正的主要证据改为“只改变 topic”：使用同一个固定非敏感中文 A 股 query、固定 `start_date`/`end_date`、`search_depth: "advanced"`、`chunks_per_source: 3`、`include_answer: false`、`max_results: 5`，三组都不发送 `auto_parameters` 和 `include_raw_content`，仅分别显式发送 `general`、`news`、`finance`，每组至少运行三次。比较严格中国/A 股相关率、明显无关数、`published_date` 非空率、日期窗口命中率、响应字节和响应时间；以严格相关性为首要指标、明显无关数为同优先级反向指标，日期与响应指标用于解释可用性和成本，不以 topic 名称预设结论。
- 若继续观察“结果预算”，须固定证据所选 topic 与其余新参数、相同 query 和日期，仅改变 `max_results: 5` 与 `max_results: 10`。单独记录 top-5 是否稳定、第 6–10 条的人工相关率与明显无关数、响应大小和响应时间；预算差异不得混入 topic 贡献结论。
- 完整 collector smoke 可作为端到端补充：在 `/tmp` 使用不同临时 `-data-root` 和 `-env-file /dev/null` 验证 CLI 装配、materializer 与 connector error，但 DeepSeek 查询规划具有波动，因此该 smoke 不作为 Tavily 参数收益的主要证据，也不把两次完整 collector 的结果差异归因于本 change。
- 无需构建或提交永久 benchmark 工具。A/B runner、payload 和原始响应全部留在唯一、明确的 `/tmp` 临时目录；完成统计后删除该目录。报告只保留固定非敏感 query、日期、payload 字段、UTC 时间、聚合指标和必要的非敏感 URL/标题判断，不保存密钥、Authorization header 或完整原始响应。

#### Leader Acceptance 修正的真实证据与默认值结论

2026-07-17 19:24 UTC 在同一短时段交错执行三轮。固定 query 为 `中国 A股 半导体 产业链 上市公司 新闻`，固定日期为 `2026-07-11` 至 `2026-07-18`；三组均使用 advanced、3 chunks、`max_results: 5`、关闭 answer，且不发送 auto/raw，唯一变量是 topic。严格相关定义为标题与摘要直接讨论中国/A 股半导体上市公司、IPO 或产业链标的；明显无关包括仅台湾/美股/韩国或泛全球市场内容。结果为：

| topic | 三轮返回 | 严格相关 | 明显无关 | published_date | 窗口命中 | 响应大小 | provider 响应时间 |
|---|---:|---:|---:|---:|---:|---:|---:|
| `general` | 15/15 | 15/15（每轮 5/5） | 0/15 | 0/15 | 0/15 可证实 | 20,412–20,413 B | 0.00–4.83 s |
| `news` | 15/15 | 6/15（每轮 2/5） | 9/15 | 15/15 | 15/15 | 13,201 B | 0.00 s（缓存命中） |
| `finance` | 10/15（一轮 0 条） | 6/10（有效轮次每轮 3/5） | 4/10 | 0/10 | 0/10 可证实 | 202–23,305 B | 0.00–1.14 s |

因此默认选择 `general`：它在本 collector 的首要指标“严格中国/A 股相关性”上明显优于 news，并且没有 finance 的空结果轮次。选择不是声称 general 的 freshness 更好；相反，其 `published_date` 覆盖为零，样本中可人工识别出旧资料，说明 Tavily 对 general 日期过滤/元数据的行为仍有限。实现仍保留显式 `start_date`/`end_date`、advanced、无 raw、关闭 answer 与默认预算 5，这些确定性和响应内容改进不依赖 topic；需要强日期元数据的运行可显式覆盖 `news`。该小样本结论通过可配置 topic 保持可逆，不新增后置门禁。

## Risks / Trade-offs

- [默认 5 降低覆盖率] → 保留 1–20 的 Tavily 专用配置并用真实 A/B 决定是否调整默认值；通用 `CandidateLimit` 仍作为上界。
- [topic 名称与中文 A 股实际相关性不一致] → 在相同 payload 下对三个官方 topic 重复采样，以严格相关率和明显无关数选默认值，并提供受约束的 CLI 覆盖；不扩展 domain/topic 后置策略。
- [`content` 比 `raw_content` 短，可能减少后续证据细节] → 正确标记 `snippet`，让下游知道内容等级；需要全文时另行设计受控 Extract 流程。
- [CLI 新参数增加配置面] → 仅新增一个有清晰默认值和范围校验的 Tavily 专用参数，不改变现有调用即可获得默认行为。
- [首次 A/B 同时改变 topic/auto/raw，无法定位相关性回归] → 本轮三组固定全部其他字段且只改变 topic；结果预算若观察则另成一组，避免混淆贡献。
- [真实 provider 时点波动] → 固定 query、绝对日期和 payload，每组至少三次并记录 UTC 时间；结论限定为小样本受控观察。
- [`general` 日期元数据/窗口命中不可证实] → 仍显式发送日期窗口并保留无 raw 改进；报告该限制，需要强日期元数据时允许覆盖 `news`，不在本 change 增加后置过滤。
- [finance 偶发空结果] → 默认不选 finance；保留其为显式覆盖选项，并让空结果沿用现有 connector 行为。
- [provider API 语义变化] → fake HTTP 固化本项目契约；真实 smoke 对比发现异常时回滚 adapter/CLI 变更即可，无数据迁移。

## Migration Plan

1. Leader Review 明确批准后按 tasks 执行 RED、GREEN、REFACTOR。
2. 在实现 topic 配置前，以非持久化密钥会话执行三 topic 受控对比并写回证据所选默认值；默认部署无需修改现有命令，操作者可显式覆盖允许的 topic，并可用 `-tavily-max-results` 调整默认 5 的预算。
3. 清理 `/tmp` runner 和原始响应，只保留非敏感统计；5/10 预算观察与完整 collector smoke 都仅作补充。
4. 若后续采集意图变化，可在三个官方 topic 内显式覆盖；若内容策略本身有回归，回滚本 change，不涉及数据迁移。
5. Leader Acceptance 后方可 Sync、Archive、提交、推送和创建 PR。用户 Merge 后由 Leader在主任务执行 Cleanup；这是归档后的 operational state，不回写已归档 tasks。

## Open Questions

无。topic 默认值已由受控证据确定为 `general`；允许值为 `general`、`news`、`finance`。结果预算默认值 5、范围 1–20、单次调用和内容等级选择保持确定。
