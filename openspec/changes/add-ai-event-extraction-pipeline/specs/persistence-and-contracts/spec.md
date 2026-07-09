## ADDED Requirements

### Requirement: 事件提取异步任务边界
系统 SHALL 将 AI 事件提取表达为服务端异步 job，并提供状态、重试、幂等、补偿和回放能力。

#### Scenario: 管理任务状态
- **WHEN** 事件提取任务被创建、领取、执行、失败、跳过或完成
- **THEN** 系统必须持久化任务状态、时间戳、重试次数、错误信息和处理版本，使任务进度可查询和可审计

#### Scenario: 幂等处理任务
- **WHEN** 同一 raw document 因采集复跑、补偿扫描或人工重跑多次触发事件提取
- **THEN** 系统必须通过幂等键、任务状态或版本策略避免重复写入无意义事件事实

#### Scenario: 支持回放
- **WHEN** prompt、模型、schema 或实体基础库版本升级后需要重新处理历史 raw document
- **THEN** 系统必须能够创建指定版本的重跑任务，并保留新旧 extraction run 的审计信息

### Requirement: 事件提取凭证和提示词边界
系统 SHALL 通过 repo prompt 文件和 secret 引用管理 AI 事件提取运行参数，禁止把真实密钥写入数据库、配置文件或日志。

#### Scenario: 加载提示词
- **WHEN** 事件提取 worker 准备调用 LLM extractor
- **THEN** 系统必须根据 prompt 引用和版本加载 repo 内提示词文件，并把运行变量渲染到请求上下文

#### Scenario: 保护模型密钥
- **WHEN** 事件提取 worker 需要模型 API key
- **THEN** 系统必须通过环境变量或部署 secret 解析真实凭证，不得把 API key、bearer token、cookie 或私有凭证写入 job、run、raw metadata 或日志
