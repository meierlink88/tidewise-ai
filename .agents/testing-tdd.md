# Testing And TDD

本项目后端研发默认采用 TDD 测试先行。涉及功能、bugfix、重构或行为变更时必须使用 `superpowers:test-driven-development`；遇到失败或异常时必须使用 `superpowers:systematic-debugging`。本文件只补充 Go 和项目边界。

## Backend TDD Gate

- 严格执行 RED、GREEN、REFACTOR，并保留实际失败与通过证据。
- 先编写 Go 单元测试、table-driven tests、fixture、fake 或 `httptest`，再编写生产实现。
- 对应包测试通过后运行 `go test ./...`；重构不得删除或削弱已确认的行为测试。

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
- change 完成前必须运行 `go test ./...`。
- 如果受本地环境限制无法运行验证，最终说明必须明确阻塞原因和未验证风险。

## External Source Tests

采集、connector、parser 和外部 API 测试必须遵守：

- 单元测试使用 fixture、fake connector、fake parser、fake writer 或 `httptest`。
- 不在单元测试中访问真实 RSS、网页、Eastmoney、RSSHub、Agent API 或生产数据库。
- 真实网络采集只能作为显式 smoke 或手动验证，不得作为普通单元测试依赖。
- 真实外部来源失败时，必须记录失败原因，不得写入伪造数据或把失败标记为成功。

## Verification Before Completion

在声明完成、提交、push、创建 PR、sync 或 archive 前，必须使用 `superpowers:verification-before-completion` 运行新鲜验证并读取输出。不能依赖旧日志、记忆或“应该能过”的判断。
