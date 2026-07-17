## Why

当前 collector 把“采集什么内容”的业务主题同时硬编码在 `cmd/collector/main.go` 的 `objective`/`queries` 默认值和编译期 `go:embed` prompt 中，运营者修改采集意图后必须依赖 Go 构建流程，且存在两个事实来源。需要让一个运行时 prompt 文件成为采集意图的唯一来源，使修改在下一次进程启动时生效，同时保持查询规划的严格输出和 connector-only 事实边界。

## What Changes

- **BREAKING**：移除 `cmd/collector` 的 `-objective`、`-queries` flags 及其业务默认值；调用方改为维护 prompt 文件。
- 增加纯技术参数 `-prompt-file`，默认值为 `agents/collector/prompts/query_planner_v1.md`；相对路径明确按 collector 进程启动时的工作目录解析，部署在其他工作目录时可传绝对路径。
- collector 在每次进程启动时从文件系统读取一次 prompt；缺失、不可读、非 UTF-8、超过安全大小或去除空白后为空时，在模型和 connector 调用前 fail-fast。错误只报告安全类别和必要路径上下文，不包含 prompt 全文、模型 raw response 或 API key。
- prompt 全文作为 `Request.Objective` 的唯一业务意图输入，初始 `SearchQueries` 为空；DeepSeek 只返回严格的 `{"queries":[...]}`，不生成或修改 `Objective`。
- 将现有“用户原始查询优先合并”改为“仅接受模型从 prompt 规划出的查询，稳定去重并限制最多 12 条”，消除已移除 queries flag 与正式规格之间的语义冲突。
- 保留时间窗口、candidate limit、并发度、data root、env file 和 prompt file path 等纯技术参数。
- 保持 DeepSeek 只位于 connector 前的查询规划阶段；`Candidate` 事实、来源与证据仍只能来自 connector response。
- 所有新增行为通过 fake chat model、临时 prompt 文件和 fake connector 按 `RED -> GREEN -> REFACTOR` 验证，不触网、不需要真实 key。

范围限于 AI 采集器的启动参数、运行时 prompt 加载、查询规划输入/合并语义、相关测试与文档。非目标包括 prompt 热重载、远程 prompt 管理、frontmatter/配置 DSL、模型生成采集事实、connector 协议变化，以及事件提取器或投研报告分析师行为变更。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `collector-query-planning`：将采集意图所有权迁移到运行时 prompt 文件，定义启动加载与安全失败语义，并用 prompt 驱动查询替代 CLI 原始查询优先合并。

## Impact

- 受影响代码：`cmd/collector/`、`internal/collector/`、`agents/collector/prompts/`、相关配置/测试、`.env.example` 与 `README.md`。
- CLI 兼容性：删除 `-objective` 和 `-queries` 是显式 breaking change；自动化或部署脚本必须改用 `-prompt-file` 或默认路径，并保证工作目录约定。
- 依赖与外部系统：继续复用现有 Eino DeepSeek `ChatModel` 和 JSON object 响应格式，不新增模型、SDK、数据库、队列或外部服务。
- 主要风险：相对路径依赖启动工作目录、prompt 内容错误导致查询质量下降、运行时文件缺失导致启动失败。通过明确路径规则、启动期严格校验、版本控制默认 prompt 和回滚到上一版本 prompt/二进制降低风险。
- 职责与门禁：执行 Agent只完成本 change 的 Propose 并提交开发 Leader Review；Leader 明确批准后方可 Apply。实现和验证完成后还需 Leader Acceptance，随后执行 Agent才能 Sync、Archive、提交、推送、创建 PR 并提供 cleanup handoff；PR merge 由用户控制，合并后的 Cleanup 由 Leader 完成。
