## 1. Proposal Review 门禁

- [x] 1.1 向开发 Leader 提交 Why、范围/非目标、`collector-query-planning` requirements/scenarios、数据路径、provider/config/prompt/structured output/失败语义、依赖兼容、风险/成本/回滚、Eino 三仓 audit，以及 branch/worktree/status；等待 Leader 明确批准，执行 Agent不得自行批准或提前进入 Apply。

## 2. RED：先建立失败测试

- [x] 2.1 RED：为 `internal/config` 添加 DeepSeek 配置测试，覆盖必需 API key、安全错误、`deepseek-chat`/`30s` 默认值、可选 base URL、自定义 timeout 和非法/非正 timeout；运行 `go test ./internal/config -run 'TestDeepSeek' -count=1`，记录测试因配置能力尚未实现而失败的输出和原因。
- [x] 2.2 RED：为 `internal/collector` 添加 fake chat model 驱动的 planner 测试，覆盖 prompt 输入字段、合法 JSON、空响应、非法 JSON、未知字段、错误类型、空/超长查询、Request 字段保持、原始查询优先、稳定去重、12 条上限和 256 rune 上限；运行 `go test ./internal/collector -run 'TestDeepSeekQueryPlanner' -count=1`，记录与 requirements 一致的失败证据，并确认测试不访问真实 DeepSeek。
- [x] 2.3 RED：扩展 collector workflow 测试，覆盖 planner 完成后才 fan-out、所有 connector 接收同一规划请求、planner/API failure 时 connector 调用次数为零且 materializer 不执行，以及 connector 事实字段和 `ContentOrigin` 不被模型响应改变；运行 `go test ./internal/collector -run 'TestWorkflow.*Planner|TestWorkflow.*ContentOrigin' -count=1` 并记录预期失败证据。
- [x] 2.4 RED：为 `cmd/collector` 的 DeepSeek provider 装配边界添加可注入构造测试，验证配置映射到 `deepseek.ChatModelConfig` 且错误不泄露密钥；运行 `go test ./cmd/collector -run 'Test.*DeepSeek' -count=1` 并记录尚未实现导致的失败。

## 3. GREEN：最小实现

- [x] 3.1 GREEN：在 `internal/config` 实现 DeepSeek 环境配置加载与校验，在 `.env.example` 添加无真实值的 `DEEPSEEK_API_KEY`、`DEEPSEEK_MODEL`、`DEEPSEEK_BASE_URL`、`DEEPSEEK_TIMEOUT`，使 `go test ./internal/config -run 'TestDeepSeek' -count=1` 通过。
- [x] 3.2 GREEN：在 `agents/collector/prompts/` 添加版本化查询规划 prompt 和嵌入加载器；在 `internal/collector` 实现依赖 `model.BaseChatModel.Generate` 的薄 planner、严格 JSON decoder、Request 复制及查询规范化/去重/上限规则，使 `go test ./internal/collector -run 'TestDeepSeekQueryPlanner' -count=1` 通过。
- [x] 3.3 GREEN：修改 `internal/collector.NewWorkflow` 注入 planner，并将 planner 输出作为所有 connector 的唯一直接输入；实现 fail-closed 传播和 materializer 未执行语义，使 `go test ./internal/collector -run 'TestWorkflow.*Planner|TestWorkflow.*ContentOrigin' -count=1` 通过。
- [x] 3.4 GREEN：在 `cmd/collector` 使用配置初始化 `github.com/cloudwego/eino-ext/components/model/deepseek`，设置 `ResponseFormatTypeJSONObject` 并注入 planner；使 `go test ./cmd/collector -run 'Test.*DeepSeek' -count=1` 通过，且测试只使用可注入构造/fake。
- [x] 3.5 GREEN：固定 `github.com/cloudwego/eino-ext/components/model/deepseek@v0.1.7`，运行 `go mod tidy`，确认 `go list -m github.com/cloudwego/eino github.com/cloudwego/eino-ext/components/model/deepseek` 仍解析 Eino `v0.9.12` 和 DeepSeek provider `v0.1.7`，不增加 `replace` 或降级 Eino。

## 4. REFACTOR：边界、文档与受影响测试

- [x] 4.1 REFACTOR：在测试保护下整理 planner/provider/config 命名和错误分类，保持 `cmd/collector` 只负责装配、`internal/collector` 保持领域规则可独立测试、provider HTTP 细节不散落到 connector；对所有变更 Go 文件运行 `gofmt`。
- [x] 4.2 REFACTOR：更新 `README.md` 的 DeepSeek 配置、失败语义和运行说明，明确密钥注入方式、一次模型调用成本、connector-only 事实边界以及单元测试不访问真实 API。
- [x] 4.3 REFACTOR：运行 `go test ./internal/config ./internal/collector ./internal/connectors ./internal/materialize ./cmd/collector -count=1`，确认 planner、connector 合约与物化证据链的受影响测试全部通过并保存证据。

## 5. Validate 与 Leader Acceptance 门禁

- [x] 5.1 运行 `go test ./...`，保存全量测试通过证据；失败时先修复并重新执行，不以真实 DeepSeek API smoke test 替代自动化结果。
- [x] 5.2 运行 `openspec validate integrate-deepseek-collector-model --strict` 和 `git diff --check`，确认 change 产物、实现差异和文本格式通过验证。
- [x] 5.3 检查 `git status --short`、`git diff --name-only` 和 `git diff -- .reference/cloudwego/`，确认差异仅在本 change 范围且 `.reference/cloudwego/`、`data/`、`.env`、操作系统文件和运行产物未被修改或纳入交付。
- [x] 5.4 使用 `rg -n --hidden --glob '!.git/**' --glob '!.reference/**' '(sk-[A-Za-z0-9_-]{16,}|tvly-[A-Za-z0-9_-]{16,}|DEEPSEEK_API_KEY[=][^[:space:]]+)'` 扫描待交付文件，并人工复核 `.env.example` 仅含空占位，确认没有真实密钥、完整 prompt/response 日志或敏感配置值。
- [x] 5.5 向开发 Leader 提交完整 diff 范围、RED/GREEN/REFACTOR 证据、聚焦与全量测试、strict validation、依赖解析、密钥/禁止路径检查和残余风险；等待 Leader 明确完成 Leader Acceptance，执行 Agent不得自行批准或提前 Sync/Archive。

## 6. Sync、Archive 与 Pull Request 交付

- [x] 6.1 获 Leader Acceptance 后使用 `$openspec-sync-specs` 将 `collector-query-planning` delta spec 同步到 `openspec/specs/`，复核正式 requirements/scenarios 完整且与实现一致。
- [x] 6.2 使用 `$openspec-archive-change` 归档 `integrate-deepseek-collector-model`，并重新运行 OpenSpec strict validation、`go test ./...` 和 `git diff --check`；归档或验证失败时停止交付。
- [x] 6.3 仅暂存本 change 文件，检查 `git diff --cached --check`、staged 文件清单、凭据和禁止路径，创建符合 conventional commit 的提交并推送 `codex/integrate-deepseek-collector-model`。
- [ ] 6.4 创建 base 为 `main` 的 Pull Request，正文列出 change、主要改动、TDD/测试证据、OpenSpec 验证、依赖与残余风险，并向 Leader 提供 PR URL/number、branch、worktree path、必要 commits 和 cleanup handoff。

用户决定 Pull Request Merge；PR merged 后由开发 Leader在主任务核验 worktree clean 和必要 commits 已进入 `origin/main`，再清理 worktree、本地/远端分支并 prune。该 operational state 不作为归档前 checkbox，也不回写已归档 tasks。
