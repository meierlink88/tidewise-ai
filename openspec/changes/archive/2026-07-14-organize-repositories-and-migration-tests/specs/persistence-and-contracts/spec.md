## ADDED Requirements

### Requirement: migration SQL 与测试边界分离
系统 SHALL 使 `backend/migrations` 只保存版本化 SQL migration 与 `README.md`，并在现有 `backend/internal/platform/dbmigration` 测试边界统一验证 migration source、静态安全契约、Goose 执行边界和明确标记的可选 PostgreSQL integration 行为。

#### Scenario: 检查 migration 目录纯度
- **WHEN** 开发者运行 dbmigration contract tests
- **THEN** `backend/migrations` 中除版本化 `.sql` 和 `README.md` 外不得存在 Go 测试、执行器或其他运行时代码

#### Scenario: 迁移有效安全契约
- **WHEN** migration test 从 SQL 目录迁入 `internal/platform/dbmigration`
- **THEN** 当前 schema、授权开关、非破坏性、幂等、事务、rollback、Goose statement 和完整 migration chain 的仍有效契约必须继续由自动化测试保护

#### Scenario: 删除已废止 schema 测试
- **WHEN** 某项测试只要求已被后续 migration 和主规格明确废止的中间 schema 继续作为最终结构存在
- **THEN** 系统必须允许删除该断言
- **AND** 删除前必须证明 migration 版本/格式、完整链执行和仍有效安全边界由统一测试位置继续覆盖

#### Scenario: 不执行有状态验证
- **WHEN** 本行为保持 change 未取得任何数据库写入授权
- **THEN** 默认验证不得执行 migration、seed 或 PostgreSQL/Neo4j write
- **AND** 真实 PostgreSQL integration tests 必须继续保持显式 opt-in
