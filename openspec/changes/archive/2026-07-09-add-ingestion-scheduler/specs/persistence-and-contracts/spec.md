## ADDED Requirements

### Requirement: 全局调度配置持久化
系统 SHALL 通过 PostgreSQL 持久化全局采集调度配置，使调度器和管理后台可以共享启停状态、触发模式、并发参数和来源过滤条件。

#### Scenario: 创建调度配置结构
- **WHEN** 数据库迁移执行到本 change 的版本
- **THEN** PostgreSQL 必须提供全局调度配置结构，保存启用状态、调度模式、interval 分钟数、固定时间列表、并发数、batch size、超时时间、source filter、配置版本和更新时间

#### Scenario: 默认安全配置
- **WHEN** 新环境首次执行 migration
- **THEN** 系统必须提供默认关闭或等价安全配置，不得迁移后立即自动访问大量外部来源

#### Scenario: 非破坏性迁移
- **WHEN** 迁移在已有 `source_catalogs` 和 `raw_documents` 数据的数据库上执行
- **THEN** 迁移不得删除、清空或重写已有来源和原始文档数据

#### Scenario: 不保存敏感信息
- **WHEN** 调度配置记录运行参数
- **THEN** 系统不得把 API key、Admin Token、cookie、bearer token、私有 URL 密钥或数据库连接串写入调度配置表

### Requirement: 采集运行记录持久化
系统 SHALL 通过 PostgreSQL 持久化调度 run 和 source 级执行结果，支持后续审计、排障、后台展示和本地验证。

#### Scenario: 保存 run 摘要
- **WHEN** 调度器开始和结束一轮采集
- **THEN** 系统必须保存 run ID、触发方式、状态、开始时间、结束时间、总来源数、成功数、失败数、跳过数、调度参数摘要和错误摘要

#### Scenario: 保存 source 执行结果
- **WHEN** 某个来源在 run 中完成、失败或跳过
- **THEN** 系统必须保存 source ID、run ID、状态、写入数量、重复数量、错误信息、开始时间、结束时间和耗时

#### Scenario: 查询最近运行结果
- **WHEN** 管理后台或开发者需要验证调度器是否正常工作
- **THEN** 系统必须能够通过 repository、admin API 或 SQL 查询最近 run 和来源执行结果，而不是只依赖进程日志

### Requirement: 调度器管理 API 契约
系统 SHALL 为管理后台提供调度器配置和运行记录 API，并通过 Admin Token 保护管理接口。

#### Scenario: 查询调度配置
- **WHEN** 管理后台请求当前调度器配置
- **THEN** 后端必须返回启用状态、调度模式、interval、固定时间列表、并发数、batch size、超时时间、source filter 和最近运行摘要

#### Scenario: 保存调度配置
- **WHEN** 管理后台提交调度器配置
- **THEN** 后端必须校验字段格式、调度模式、固定时间、interval、并发数、batch size 和 source filter 后再保存

#### Scenario: 查询最近运行记录
- **WHEN** 管理后台请求最近调度 run
- **THEN** 后端必须返回最近 run 的开始时间、结束时间、触发方式、状态、总数、成功数、失败数、跳过数和错误摘要

#### Scenario: Admin Token 鉴权
- **WHEN** 请求访问调度器管理 API
- **THEN** 后端必须校验 `Authorization: Bearer <token>`，并使用环境变量 `ADMIN_API_TOKEN` 中的 token 完成鉴权

#### Scenario: 拒绝未授权请求
- **WHEN** 请求缺少 token、token 错误或服务端未配置 Admin Token
- **THEN** 后端必须拒绝访问，不得返回调度配置或运行记录
