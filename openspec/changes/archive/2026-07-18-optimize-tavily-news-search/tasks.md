## 1. Leader Review 门禁

- [x] 1.1 向开发 Leader提交范围/非目标、requirements/scenarios、`max_results` 选择与成本、Eino audit、TDD/验证计划、风险和分支/worktree 状态；仅在 Leader明确批准实施后调用 `$openspec-apply-change`，执行 Agent不得自行批准。

## 2. RED：先固定失败契约

- [x] 2.1 在 `internal/connectors/search_test.go` 增加 Tavily fake HTTP 测试：捕获单次请求并断言 query 拼接、`topic: "news"`、`search_depth: "advanced"`、`chunks_per_source: 3`、确定日期、`include_answer: false`、有效 `max_results`，且不存在 `auto_parameters`/`include_raw_content`；构造同时含 `content`/`raw_content` 及空 content 的响应，断言仅采用 `content` 并产生 `snippet`/`title_only`。
- [x] 2.2 在 `cmd/collector/main_test.go` 增加 `-tavily-max-results` 默认 5、1–20 显式覆盖和非法范围拒绝测试，并确认该参数只注入 Tavily；不得让测试读取真实 `.env` 或访问网络。
- [x] 2.3 运行 `go test ./internal/connectors -run '^TestTavily' -count=1` 和 `go test ./cmd/collector -run 'TavilyMaxResults|CollectorPromptFlags' -count=1`，记录旧实现因目标参数、预算或内容等级缺失而失败的 RED 证据；不得在失败前修改实现。

## 3. GREEN：最小实现

- [x] 3.1 在 Tavily adapter 中省略 `auto_parameters`/`include_raw_content`，显式发送 news、advanced、日期窗口与关闭 answer 的参数，保持原顺序 query 拼接和每次 `Collect` 单次 POST。
- [x] 3.2 增加 Tavily 专用 `MaxResults`，实现 `min(CandidateLimit, MaxResults)` 与 1–20 校验；在 collector CLI 增加默认 5 的 `-tavily-max-results` 并仅注入 Tavily connector，不改 Parallel/Bocha。
- [x] 3.3 移除 Tavily 对 `raw_content` 的依赖，仅把 `results[].content` 映射为 `snippet`，空白 content 交由标题 fallback 标记为 `title_only`。
- [x] 3.4 运行 `go test ./internal/connectors -run '^TestTavily' -count=1` 和 `go test ./cmd/collector -run 'TavilyMaxResults|CollectorPromptFlags' -count=1`，记录 GREEN 通过证据。

## 4. REFACTOR 与自动化验证

- [x] 4.1 在测试保护下整理 Tavily 有效结果上限、校验和命名，执行 `gofmt`；复核 diff 不包含 score 过滤、域名规则、多次 Tavily 调用、LLM 后置门禁、prompt 或 Bocha/Parallel 行为变更。
- [x] 4.2 重新运行聚焦测试 `go test ./internal/connectors -run '^TestTavily' -count=1`、`go test ./cmd/collector -run 'TavilyMaxResults|CollectorPromptFlags' -count=1`，再运行 `go test ./...`。
- [x] 4.3 运行 `openspec validate optimize-tavily-news-search --strict`、`git diff --check`；检查 `git status --short`、`git diff --name-only`，确认 `.reference/cloudwego/`、`.env`、`data/` 和主 checkout 的 `.codex/config.toml` 未被读取、修改、暂存或纳入 diff，并扫描 staged/unstaged diff 中的凭据模式。

## 5. 手工真实运行对比

- [x] 5.1 在已获准真实调用且 `TAVILY_API_KEY` 已由交互式会话环境安全注入时，在唯一、明确的 `/tmp` 临时目录创建不含密钥的一次性 Tavily runner；runner SHALL 从环境读取密钥，不把 Authorization 放入命令参数或日志。固定一个非敏感 query、`start_date`、`end_date`、advanced 深度、3 chunks 和关闭 answer，每个 payload 至少执行三次并记录 UTC 时间；无需创建仓库内或永久 benchmark 工具。若无安全可用凭据，则记录未执行原因。
- [x] 5.2 执行第一层受控“搜索参数” A/B，双方固定 `max_results: 5`：旧组使用 general 基线、`auto_parameters: true`、`include_raw_content: "markdown"`，新组使用 `topic: "news"`、不发送 auto、不发送 raw。比较 top-5 URL/标题人工相关率、`published_date` 非空及窗口覆盖率、`content`/`raw_content` 长度、明显无关结果数和响应时间；结论只评价参数组合，不宣称单字段因果。
- [x] 5.3 执行第二层受控“结果预算” A/B，固定新搜索参数、相同 query 和日期，仅比较 `max_results: 5` 与 `max_results: 10`；记录 top-5 稳定性、第 6–10 条相关率/明显无关数、响应大小和响应时间，并与第一层参数收益分开报告。
- [x] 5.4 可选执行完整 collector smoke 作为 CLI 装配、materializer 和 connector error 的端到端补充，使用 `-env-file /dev/null` 与 `/tmp` 数据目录；明确 DeepSeek query 波动使其不能作为 Tavily 参数收益的主要归因证据。本次环境无安全可用 `DEEPSEEK_API_KEY`，故按批准设计记录为未执行，不影响直接 Tavily A/B 主要证据。
- [x] 5.5 从报告中排除密钥、Authorization header 和完整原始响应，只保留固定非敏感 query、日期、payload 字段、UTC 时间、聚合指标和必要的非敏感 URL/标题判断；完成统计后删除包含 runner、payload 和响应的明确 `/tmp` 临时目录，不写入或提交 `data/`。

## 6. Leader Acceptance 修正：topic 证据与规格

- [x] 6.1 先更新 proposal/design/delta spec/tasks，把未经效果支持的硬编码 news 改为“受控证据选择默认 topic + Tavily 专用可配置 topic”；保留既有无 raw、关闭 answer、advanced、日期窗口、默认结果预算 5 和范围边界，并通过 strict validation。
- [x] 6.2 在唯一 `/tmp` 临时目录使用相同固定非敏感中文 A 股 query、相同日期、advanced、3 chunks、`max_results: 5`、无 auto/raw、关闭 answer，仅分别改变 `topic` 为 `general`、`news`、`finance`，每组至少运行三次；比较严格相关率、明显无关数、published_date 覆盖、窗口命中、响应大小和时间，随后清理 runner 与原始响应。若当前环境无安全注入的 `TAVILY_API_KEY`，明确记录未执行且不得索取或猜测。
- [x] 6.3 基于 6.2 证据在 proposal/design/spec 中明确默认 topic、成本、风险与可配置性；如观察 5/10 预算，必须作为只改变预算的第二组独立记录，不得与 topic 贡献混淆，再次通过 strict validation。

## 7. RED：topic 配置契约

- [x] 7.1 先扩展 Tavily fake HTTP 测试，断言显式发送配置 topic、只允许 `general`/`news`/`finance`、无效值在 HTTP 前失败，并继续证明 advanced、日期、无 auto/raw、单次调用、结果预算及 `snippet`/`title_only` 映射。
- [x] 7.2 先扩展 collector CLI 测试，断言 Tavily topic 的证据所选默认值、三个允许值、非法值拒绝及仅注入 Tavily；运行聚焦测试并记录当前硬编码 news 实现的 RED，失败前不得修改实现。

## 8. GREEN 与 REFACTOR

- [x] 8.1 在 Tavily adapter 增加经过校验的 provider-specific topic，并在 collector CLI 增加对应技术参数；默认值采用 6.2 的证据结论，仅把配置注入 Tavily，不改 Bocha/Parallel/workflow/prompt/materializer/data。
- [x] 8.2 运行 topic 与既有 Tavily 聚焦测试取得 GREEN；在测试保护下执行必要命名整理和 `gofmt`，保持单次请求、结果预算和 query-related content 行为不变。

## 9. Validate

- [x] 9.1 重跑全部 Tavily/CLI 聚焦测试及 `go test ./...`，执行 `openspec validate optimize-tavily-news-search --strict`、`git diff --check`、凭据扫描、禁止路径/范围检查并记录证据。

## 10. Leader Acceptance 门禁

- [x] 10.1 向开发 Leader提交修正后的实现摘要、完整 diff、三 topic 真实证据、RED/GREEN/REFACTOR、聚焦/全量测试、strict validation、安全检查及残余风险；只有 Leader明确完成 Leader Acceptance 后才能继续 Sync、Archive 和 Deliver，执行 Agent不得自行批准。

## 11. Sync、Archive 与 Deliver

- [x] 11.1 使用 `$openspec-sync-specs` 把 `tavily-news-search` delta spec 同步到正式 specs，复核 requirement/scenario 与实现一致。
- [x] 11.2 使用 `$openspec-archive-change` 归档 change，并再次运行 `openspec validate --strict`、`go test ./...`、`git diff --check`、范围/凭据/禁止路径检查。
- [x] 11.3 只暂存本 change 文件，检查 `git diff --cached --check` 与 staged diff，创建 conventional commit，推送 `codex/optimize-tavily-news-search`，创建 base 为 `main` 的 Pull Request，并向 Leader提供 PR、验证证据和 cleanup handoff。

用户控制 Pull Request Merge；Merge 后由开发 Leader在主任务核验并执行 worktree/branch Cleanup。该 operational state 不作为归档门禁，也不回写已归档 tasks。
