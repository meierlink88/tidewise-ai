## 1. 提案与 Leader Review 门禁

- [x] 1.1 向开发 Leader提交 Why、范围/非目标、完整 requirements/scenarios、A/B/C 方案比较、选定数据流、路径/失败/安全语义、TDD 与验证计划、风险/回滚、Eino audit commits、完整 diff 范围及 branch/worktree/status；等待 Leader 明确批准，执行 Agent不得自行批准或进入 Apply。
- [x] 1.2 Leader 批准后，运行 `openspec instructions apply --change prompt-owned-collector-intent --json`，完整读取返回的 context files，并再次确认实现范围仅覆盖本 change。

## 2. RED：先证明行为缺口

- [x] 2.1 RED：在 `internal/collector` 添加运行时 prompt loader 测试，覆盖普通文件、绝对/相对 current working directory 路径、符号链接到普通文件、缺失/不可读/目录或非普通文件、空白、64 KiB 边界与超限、非法 UTF-8、原文保持、安全错误及同一路径修改后新的 loader 调用取得新内容；运行 `go test ./internal/collector -run 'TestLoadCollectorPrompt' -count=1`，记录因 loader 尚不存在或不满足合约而失败的证据。
- [x] 2.2 RED：在 `cmd/collector` 添加启动装配测试，覆盖默认 `-prompt-file`、删除 `-objective`/`-queries`、仅保留技术 flags、prompt 在 provider/connector 前 fail-fast、加载内容成为初始空 `SearchQueries` 的 `Request.Objective`，以及错误不泄露 prompt/API key/raw response；运行 `go test ./cmd/collector -run 'TestCollectorPrompt|TestBuildDeepSeekPlanner' -count=1` 并记录预期失败。
- [x] 2.3 RED：更新 fake chat model 驱动的 planner 测试，断言 `Request.Objective` 直接成为 system message、user message 只含 UTC 时间和时间窗口、模型唯一输出为 `queries`、输入/非查询字段保持、模型查询稳定去重/12 条/256 rune 上限、无原始查询优先语义及安全失败；运行 `go test ./internal/collector -run 'TestDeepSeekQueryPlanner' -count=1` 并记录与 delta spec 一致的失败。
- [x] 2.4 RED：更新 fake planner/fake connector workflow 测试，断言 planner 之前 connector 调用数为零、同一 Objective/规划查询 fan-out、模型失败无 connector/产物，以及 Candidate 事实仍等于 connector response；运行 `go test ./internal/collector -run 'TestWorkflow.*Planning|TestWorkflow.*Fact' -count=1` 并记录预期失败。所有 RED 测试 MUST NOT 读取真实 key、访问公网或调用真实 provider/connector。

## 3. GREEN：最小实现

- [x] 3.1 GREEN：在 `internal/collector` 实现薄运行时 prompt loader，按 current working directory/绝对路径规则读取普通文件，执行 64 KiB、UTF-8、非空校验，保留有效原文并返回不含内容或底层敏感文本的安全错误；运行 `go test ./internal/collector -run 'TestLoadCollectorPrompt' -count=1` 通过。
- [x] 3.2 GREEN：重构 `cmd/collector` 启动装配，新增默认 `agents/collector/prompts/query_planner_v1.md` 的 `-prompt-file`，移除 `-objective`/`-queries`、业务默认值和 `splitQueries`，在 DeepSeek/connector 前加载 prompt，并用其构造 `Request.Objective` 与空 `SearchQueries`；运行 `go test ./cmd/collector -run 'TestCollectorPrompt|TestBuildDeepSeekPlanner' -count=1` 通过。
- [x] 3.3 GREEN：删除 `agents/collector/prompts/prompt.go` 的 `go:embed` 加载路径，更新默认 prompt 使所有业务采集主题只在该文件维护，并保留只输出 `{"queries":[...]}`、不得生成事实的自然语言约束。
- [x] 3.4 GREEN：更新 `DeepSeekQueryPlanner` 和 builder，使输入 `Objective` 直接作为 system message、技术运行上下文作为 user message、模型只生成查询；移除原始查询优先合并，保留严格 JSON、稳定去重、12 条/256 rune 上限、Request 副本和安全错误；运行 `go test ./internal/collector -run 'TestDeepSeekQueryPlanner' -count=1` 通过。
- [x] 3.5 GREEN：保持 workflow 的 planner-before-connectors 和 connector-only Candidate 边界，必要时仅做最小装配调整；运行 `go test ./internal/collector -run 'TestWorkflow.*Planning|TestWorkflow.*Fact' -count=1` 通过。
- [x] 3.6 GREEN：更新 `README.md`、`.env.example` 或相关运行示例，说明 breaking flags 迁移、默认/绝对/相对 prompt path、current working directory、下一次启动生效、运行中不热重载和安全失败；不得复制业务主题到 Go/启动参数示例。

## 4. REFACTOR 与验证

- [x] 4.1 REFACTOR：在测试保护下清理 loader、启动装配和 planner 的命名/重复/错误边界，对所有变更 Go 文件运行 `gofmt`，再运行上述三个聚焦测试集并保留通过证据。
- [x] 4.2 运行 `go test ./...`，确认全部测试使用 fake/临时文件且不触网、不消耗真实 key；记录命令、退出状态与结果。
- [x] 4.3 运行 `openspec validate prompt-owned-collector-intent --strict` 和 `git diff --check`；检查 `git diff --name-only`、`git status --short`，确认仅包含本 change，且 `.reference/cloudwego/`、`.env`、`data/`、密钥、模型 raw response 和运行产物未进入差异。
- [x] 4.4 复核实现与 delta spec 一致：prompt 编辑后下一次启动生效、无 `go:embed`/业务 flags/代码业务主题、A 方案严格输出、fail-fast、安全错误、planner-only 查询与 connector-only 事实边界均有自动化证据。

## 5. Leader Acceptance 门禁

- [x] 5.1 向开发 Leader提交实现摘要、完整 diff 范围、逐项 requirement/scenario 对照、RED/GREEN/REFACTOR 证据、聚焦/全量/strict validation 结果、凭据与禁止路径检查、branch/worktree/status 和残余风险；等待 Leader 明确完成 Leader Acceptance，执行 Agent不得自行批准或提前 Sync/Archive。

## 6. Sync、Archive 与交付

- [x] 6.1 获得 Leader Acceptance 后使用 `$openspec-sync-specs` 将 `collector-query-planning` delta 同步到正式规格，复核未丢失未修改 requirements/scenarios，并运行正式规格 strict validation。
- [x] 6.2 使用 `$openspec-archive-change` 归档 `prompt-owned-collector-intent`，执行归档后的 `go test ./...`、OpenSpec strict validation、`git diff --check`、完整范围/密钥/禁止路径检查；归档失败不得继续交付。
- [x] 6.3 仅暂存本 change 文件，检查 `git diff --cached --check` 和 staged diff，创建 conventional commit，推送 `codex/prompt-owned-collector-intent` 并创建 base 为 `main` 的 Pull Request；PR 描述包含 change、行为范围、TDD/验证证据、迁移、风险与回滚，用户控制 merge。
- [x] 6.4 向 Leader提交 PR 地址和 cleanup handoff；用户 Merge 与 Leader在主任务执行的合并后 Cleanup 属于归档后的 operational state，不作为本 tasks 的 archive-gating checkbox，也不回写已归档 tasks。
