# TDD 与验证规范

## 基本循环

行为变更必须执行 `RED -> GREEN -> REFACTOR`：

1. **RED**：先写能表达 requirement/scenario 的最小自动化测试，运行后确认它因目标
   行为尚未实现而失败，并保留失败证据。
2. **GREEN**：只写让该测试通过的最小实现，运行聚焦测试并保留通过证据。
3. **REFACTOR**：在测试保护下清理命名、重复和边界，再运行聚焦测试及受影响测试集。

禁止先完成实现再补测试。若无法自动化测试，必须在 proposal/tasks 中说明原因、
替代验证方式和残余风险，并在提案评审时获得用户批准。

## OpenSpec tasks 要求

- 行为变更 tasks 必须包含 RED、GREEN、REFACTOR 或等价的明确步骤。
- RED 任务记录失败命令和失败原因，失败必须与预期 requirement 一致。
- GREEN 任务记录聚焦测试命令。
- REFACTOR/验证任务记录全量测试和 OpenSpec strict validation。
- 只有证据真实产生后才能勾选对应 checkbox。

## 文档和流程变更

不改变运行时行为的规则变更也应优先使用可执行策略测试。测试先证明缺失规则会失败，
再补齐最小文档或配置使其通过，避免仅靠人工阅读维护关键约束。

## Go 工程验证基线

- 聚焦测试：`go test ./internal/<area> -run '<test-name>' -count=1`
- 全量测试：`go test ./...`
- 格式化：`gofmt` 处理所有变更的 Go 文件
- OpenSpec：`openspec validate <change-name> --strict`
- 文本检查：`git diff --check`

涉及外部 API 的测试默认使用 fake server 或可注入 transport，不在单元测试中消耗真实
密钥或依赖不稳定网络。
