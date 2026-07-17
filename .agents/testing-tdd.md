# Testing And TDD

本项目后端研发默认采用 TDD 测试先行。本文件直接规定测试先行、失败诊断、验证证据和 Go 边界；不要求安装任何外部 Skill。

## Backend TDD Gate

- 严格执行 RED、GREEN、REFACTOR，并保留实际失败与通过证据。
- 先编写 Go 单元测试、table-driven tests、fixture、fake 或 `httptest`，再编写生产实现。
- 对应受影响包的完整 suite 和共享 architecture/contract tests 通过后，才可进入 Apply final；是否运行 repo-wide full validation 按本文件 Verification Before Completion 的触发条件决定。重构不得删除或削弱已确认的行为测试。

## Test Boundary Rules

- Go 后端测试优先使用官方 `testing` 体系和 `go test ./...`。
- 可以使用 Go 生态成熟断言库，但不得为了测试引入重型或不必要框架。
- 单元测试不得依赖真实外部网络、真实 API key、真实 cookie、真实 token 或生产数据库。
- 外部 HTTP、RSS、RSSHub、Eastmoney、Agent API 或 webhook 行为必须通过 fixture、fake server、interface fake 或 `httptest` 验证。
- repository、migration 和数据库相关能力优先使用接口 fake、SQL/migration 静态验证和可重复集成测试。
- 需要真实 PostgreSQL 时必须明确标记为集成测试边界，并使用本地/测试数据库连接，不得连接生产数据库。

## Task Completion Rules

- 新增后端功能点没有对应测试时，不应标记 tasks 完成。
- 每完成一个 tasks checkbox 前，必须确认对应测试或验证已覆盖该行为。
- change 完成前必须完成受影响交付边界的完整验证和共享 architecture/contract tests；满足 repo-wide 触发条件时才必须运行 `go test ./...`。
- 如果受本地环境限制无法运行验证，最终说明必须明确阻塞原因和未验证风险。

## External Source Tests

采集、connector、parser 和外部 API 测试必须遵守：

- 单元测试使用 fixture、fake connector、fake parser、fake writer 或 `httptest`。
- 不在单元测试中访问真实 RSS、网页、Eastmoney、RSSHub、Agent API 或生产数据库。
- 真实网络采集只能作为显式 smoke 或手动验证，不得作为普通单元测试依赖。
- 真实外部来源失败时，必须记录失败原因，不得写入伪造数据或把失败标记为成功。

## Verification Before Completion

在声明完成、提交、push、创建 PR、sync 或 archive 前，必须按本节规则运行新鲜验证并读取输出。不能依赖旧日志、记忆或“应该能过”的判断。

Apply final 必须运行受影响交付边界的完整验证：受影响 app/module/package 的完整 suite 和共享 architecture/contract tests。任意 change 必须先按真实受影响交付边界选择验证：OpenSpec artifacts、workflow 文本、agent rules、architecture test/lint 自身的变更只运行对应 OpenSpec/architecture/规则 targeted validation；局部 coding 运行 targeted tests 与受影响 package/module 完整 suite；数据-only change 运行 manifest/dry-run/preflight/post-write assertions，没有代码影响时不机械运行业务 unit tests；只有修改共享运行时代码、跨模块运行时契约、公共运行时基础设施，或影响边界无法可靠确定时才运行 repo-wide full validation，Go module 对应 `go test ./...`，前端同理按受影响 workspace。UAT/prod/shared/stateful 安全门禁不由测试范围优化削弱。验证记录必须说明受影响边界、共享 tests 和 repo-wide 判定理由；不清楚时 fail-closed，扩大到 repo-wide full validation 或停止等待澄清。

R2/R3 有状态操作的验证证据、recovery evidence、before/after assertions 和停止语义只以 `.agents/openspec-workflow.md` 为准；本文件不重复授权流程。
