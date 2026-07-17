## ADDED Requirements

### Requirement: 运行时 prompt 文件拥有采集意图
collector SHALL 将一个运行时 prompt 文件作为“采集什么内容”的唯一自然语言来源，`cmd/collector` 和其他 Go 代码 MUST NOT 硬编码政经、政策、产业、企业、资本市场或其他业务采集主题，也 MUST NOT 提供带业务内容的 `objective`/`queries` flags 或默认值。collector SHALL 提供纯技术参数 `prompt-file`，其默认值为 `agents/collector/prompts/query_planner_v1.md`；绝对路径 SHALL 直接使用，相对路径 SHALL 相对 collector 进程启动时的 current working directory 解析，且 MUST NOT 隐式搜索其他目录或回退到编译期嵌入内容。

#### Scenario: 使用默认 prompt 路径启动
- **WHEN** 调用方从包含版本化默认 prompt 的项目工作目录启动 collector，且未传 `prompt-file`
- **THEN** collector 从 `agents/collector/prompts/query_planner_v1.md` 加载采集意图，启动参数和 Go 代码中不包含业务采集主题默认值

#### Scenario: 使用相对 prompt 路径启动
- **WHEN** 调用方传入相对 `prompt-file`
- **THEN** collector 仅相对进程启动时的 current working directory 解析并读取该文件

#### Scenario: 使用绝对 prompt 路径启动
- **WHEN** 调用方传入绝对 `prompt-file`
- **THEN** collector 直接读取该路径且不依赖 current working directory

### Requirement: prompt 在每次进程启动时安全加载
collector SHALL 在每次进程启动时从文件系统读取一次 prompt，并 SHALL 在模型或任何 connector 调用前确认目标为可读普通文件、文件不超过 64 KiB、内容为合法 UTF-8 且 `TrimSpace` 后非空。有效内容 SHALL 原样成为该进程内不可变的 `Request.Objective`；运行中修改文件 MUST NOT 改变当前进程已加载的 Objective，下一次启动 SHALL 重新读取文件并使用新内容。prompt MUST NOT 使用 `go:embed` 或其他需要重新编译才能生效的默认副本。

#### Scenario: 修改 prompt 后重新启动
- **WHEN** 第一次进程启动并加载内容 A 后退出，调用方把同一路径内容改为 B 并启动新进程
- **THEN** 新进程的 `Request.Objective` 等于 B 且不需要修改 Go 代码或重新编译

#### Scenario: 运行中修改 prompt
- **WHEN** collector 已加载内容 A，调用方在该进程运行中把文件改为 B
- **THEN** 当前进程继续使用 A，后续新进程使用 B

#### Scenario: prompt 文件无效
- **WHEN** prompt 路径缺失、不可读、指向目录或其他非普通文件、超过 64 KiB、包含非法 UTF-8，或内容去除空白后为空
- **THEN** collector 在模型和 connector 调用前 fail-fast，且不产生新的采集产物

### Requirement: prompt 加载和规划错误保持安全
prompt 加载、provider 初始化、模型调用和模型输出校验错误 MUST NOT 包含 prompt 全文、模型 raw response 或 API key。错误 SHALL 提供可判定的失败阶段和必要的 prompt path 上下文；实现 MUST NOT 直接透传可能包含敏感内容的底层文件或 provider 错误文本。

#### Scenario: 底层错误包含敏感哨兵
- **WHEN** 文件读取或 provider fake 返回包含完整 prompt、模型 raw response 或 API key 哨兵的错误
- **THEN** collector 返回对应加载或规划阶段的安全错误，且错误文本不包含任何敏感哨兵

### Requirement: 技术启动参数继续独立于采集意图
collector SHALL 保留时间窗口、candidate limit、connector 并发度、data root、env file 和 prompt file path 等纯技术参数；这些参数 MUST NOT 承载业务采集主题或静态搜索查询。

#### Scenario: 使用技术参数运行
- **WHEN** 调用方覆盖时间窗口、candidate limit、并发度、data root、env file 或 prompt file path
- **THEN** collector 使用对应技术值，并仍只从 prompt 文件取得业务采集意图

### Requirement: prompt 驱动查询的去重和限制
查询规划器 SHALL 仅使用模型根据 `Request.Objective` 返回的 `queries`，对每项执行 `TrimSpace`，按精确字符串稳定去重，并按模型返回顺序保留最多 12 条。单条查询 MUST 不超过 256 rune；空数组、空查询、无任何可用查询或超长查询 SHALL 使规划失败。planner MUST NOT 把 prompt 原文直接作为 `SearchQueries`，也 MUST NOT 合并 CLI、代码默认值或其他静态查询来源。

#### Scenario: 模型查询包含重复项
- **WHEN** 模型返回顺序为重复查询、空白包围的唯一查询和其他唯一查询
- **THEN** planner 去除首尾空白，保留每个精确字符串的首次出现并维持模型顺序

#### Scenario: 模型查询超过总数上限
- **WHEN** 规范化和去重后的模型查询超过 12 条
- **THEN** planner 仅保留最前 12 条并确定性丢弃尾部查询

#### Scenario: 没有合法模型查询
- **WHEN** 模型返回空数组、空查询、超长查询或最终无任何可用查询
- **THEN** workflow 返回安全的查询规划错误，任何 connector 不执行且不产生新的采集产物

## MODIFIED Requirements

### Requirement: DeepSeek 查询规划位于 connector 前
AI 采集器 SHALL 在任何搜索 connector 执行前，使用 DeepSeek 查询规划器根据运行时 prompt 所形成的 `Objective`、`TimeWindowHours` 和 `CollectedAt` 生成 `SearchQueries`，并 SHALL 将同一份规划后 `Request` 发送给所有已配置 connector。

#### Scenario: 成功规划后执行多 connector 采集
- **WHEN** DeepSeek 根据 prompt Objective 返回通过校验的结构化查询，并且 collector 配置了多个 connector
- **THEN** 所有 connector 在 planner 完成后各执行一次，并接收相同的规划后 `Objective` 和 `SearchQueries`

#### Scenario: planner 尚未完成
- **WHEN** 查询规划节点仍在执行
- **THEN** 任何搜索 connector MUST NOT 开始执行

### Requirement: 规划输入和 Request 字段保持
查询规划器 SHALL 把运行时 prompt 全文对应的 `Request.Objective` 作为 system message，并 SHALL 把 UTC 采集时间和时间窗口作为不含业务默认值的运行时 user message 提供给模型；模型 SHALL 只输出 `queries`。planner SHALL 返回 `Request` 副本并只设置 `SearchQueries`，MUST 保持 `RunID`、`Objective`、`CandidateLimit`、`TimeWindowHours` 和 `CollectedAt` 不变，且调用方输入 MUST NOT 被修改。

#### Scenario: 生成规划请求
- **WHEN** collector 收到由有效 prompt 构造且初始 `SearchQueries` 为空的完整 `Request`
- **THEN** 模型 system message 等于该 `Objective`，user message 包含 UTC 采集时间和时间窗口，planner 输出只新增模型规划的 `SearchQueries`

#### Scenario: 模型不能改写 Objective
- **WHEN** DeepSeek 返回合法的 `queries`
- **THEN** planner 输出的 `Objective` 与输入 prompt 全文完全相同，且没有从模型响应读取或生成新的 Objective

### Requirement: 自动化测试不得访问真实 DeepSeek
查询规划、运行时 prompt 加载、启动装配、workflow 和配置的自动化测试 SHALL 使用临时文件、fake planner、fake connector 或 fake chat model，且 MUST NOT 要求真实 `DEEPSEEK_API_KEY`、消耗模型额度或依赖公网可用性。

#### Scenario: 运行聚焦测试和全量测试
- **WHEN** 开发者运行 prompt 加载、查询规划相关测试或 `go test ./...`
- **THEN** 测试通过临时 prompt 文件和可注入 fake 覆盖成功与失败场景，不向真实 DeepSeek 或搜索 connector endpoint 发出请求

## REMOVED Requirements

### Requirement: 原始查询优先合并、去重和限制
**Reason**: `-queries` 及其业务默认值被移除，采集意图只能来自运行时 prompt；继续保留“用户原始查询优先”会形成第二个业务内容来源并与新语义冲突。

**Migration**: 将原先通过 `-queries` 提供的业务查询意图写入 `prompt-file` 的自然语言内容，由 DeepSeek 统一规划 `SearchQueries`；最终查询仍执行稳定去重、256 rune 单条上限和 12 条总上限。
