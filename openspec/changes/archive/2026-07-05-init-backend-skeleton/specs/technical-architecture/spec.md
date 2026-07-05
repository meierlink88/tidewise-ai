## MODIFIED Requirements

### Requirement: Go API/BFF 服务端边界
系统 SHALL 通过 Go + Gin API/BFF 边界暴露面向客户端的能力，隐藏内部模块、外部 Agent 平台和数据访问细节，并向客户端应用提供稳定契约。

#### Scenario: 小程序集成后端数据
- **WHEN** Taro 小程序页面用真实数据替换 mock 数据
- **THEN** 页面必须依赖已文档化的 API/BFF 契约，而不是直接调用 Agent 平台、采集、图谱、RAG、数据库、队列或第三方模型服务

#### Scenario: 内部服务拓扑变化
- **WHEN** 内部能力在模块化单体代码、worker 代码或独立服务之间迁移
- **THEN** 面向客户端的 API/BFF 契约仍然是稳定集成边界，除非另一个 OpenSpec change 明确修改该契约

#### Scenario: 落地后端骨架
- **WHEN** Go API/BFF 后端骨架被创建
- **THEN** 骨架必须位于 `backend/`，并提供可编译、可测试、可本地启动的服务端入口

### Requirement: 后端环境配置边界
系统 SHALL 标准化支持 `local`、`uat`、`prod` 三类后端运行环境，并通过统一的 Go 强类型配置加载机制隔离环境差异。

#### Scenario: 加载环境配置
- **WHEN** Go 后端服务启动
- **THEN** 服务必须根据 `APP_ENV` 加载对应环境配置，并在启动阶段校验必填配置

#### Scenario: 管理非敏感配置
- **WHEN** 后端需要配置服务端口、日志级别、外部接口 base URL、数据库 host/port/name、Redis 地址、限流参数或回调路径
- **THEN** 这些非敏感配置可以放入 `backend/config` 下的环境配置文件或示例模板

#### Scenario: 注入敏感配置
- **WHEN** 后端需要数据库密码、Agent 平台 API key、支付密钥、JWT secret 或云厂商密钥
- **THEN** 这些敏感配置必须通过环境变量或部署平台 secret 注入，并且不得提交到 repo

#### Scenario: 使用配置
- **WHEN** 业务模块需要访问环境相关配置
- **THEN** 业务代码必须依赖统一 config 对象，而不是散落读取环境变量或硬编码 local/uat/prod 分支

#### Scenario: 验证配置骨架
- **WHEN** 后端骨架提供 local、uat、prod 配置模板
- **THEN** 模板必须保留环境差异结构，同时不得包含真实 secret 或生产凭证
