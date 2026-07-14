## ADDED Requirements

### Requirement: 共享 repository adapter 按业务职责组织
系统 SHALL 在 `backend/internal/repositories` 单一 package 中保留调用方所需的业务小接口，并以共享 `PostgresRepository` 和共享 `InMemoryRepository` 实现这些接口；源码文件 SHALL 按业务职责组织，不得为机械拆分引入 ORM、codegen、repository framework、新 package 或一组业务专用具体 adapter。

#### Scenario: 定位业务 repository 能力
- **WHEN** 开发者需要查看 source catalog、raw document、benchmark observation、admin query、scheduler、ingestion run、graph projection 或 identity 的 repository 契约与实现
- **THEN** 对应业务文件必须集中该职责的接口/DTO、参数转换、具体 adapter 方法、Scan/结果映射和专属 helper
- **AND** `PostgresRepository` 与 `InMemoryRepository` 的 constructor 和共享 state 必须保持唯一

#### Scenario: 自动验证行为保持
- **WHEN** 开发者运行 repository targeted tests、受影响 app tests 和 backend 完整测试
- **THEN** 现有业务接口、SQL 参数化、错误语义、结果映射、稳定 ID 和调用方行为必须保持通过

#### Scenario: 阻止架构扩张
- **WHEN** repository 文件整理不需要新的运行时能力
- **THEN** 变更不得新增 GORM、sqlc、ORM、codegen、query DSL、base repository、service 或 transaction framework
