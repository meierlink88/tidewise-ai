## ADDED Requirements

### Requirement: 统一 migration 和 seed 数据边界
系统 SHALL 在 MVP 阶段保持 backend 统一 PostgreSQL migration 和 repo 内 seed 数据资产边界，后端子系统不得在普通功能 change 中创建独立 migration 根或独立数据库。

#### Scenario: 使用统一 migration
- **WHEN** 小程序 API、管理后台 API、采集子系统或后端运维命令需要调整 PostgreSQL schema
- **THEN** 该变更必须使用 `backend/migrations` 作为统一 migration 来源，除非已有独立 OpenSpec change 决定拆分数据库

#### Scenario: 区分 migration 文件和执行器
- **WHEN** 后端需要读取、检查或执行 PostgreSQL migration
- **THEN** Go 执行器代码必须位于 `backend/internal/platform/dbmigration`，并且 SQL migration 文件必须继续位于 `backend/migrations`

#### Scenario: 管理 seed 数据资产
- **WHEN** 后端需要维护采集源清单、实体基础库或其他 repo 内长期 seed 数据
- **THEN** 数据资产必须放在 `backend/data/<data-domain>` 并通过对应 seed 命令和 repository 边界写入数据库

#### Scenario: 禁止隐式拆库
- **WHEN** 普通功能 change 修改某个后端子系统
- **THEN** 不得顺手创建子系统私有数据库、私有 migration 目录或私有 schema 管理机制
