## ADDED Requirements

### Requirement: 全局采集调度器
系统 SHALL 提供后端全局采集调度器，能够按全局调度配置定时触发 active source 采集，并通过现有采集执行链路完成采集。

#### Scenario: 单轮调度
- **WHEN** 开发者运行调度器单轮模式
- **THEN** 系统必须按当前全局配置和 source filter 执行一次 active source 采集，并输出结构化调度结果

#### Scenario: 持续调度
- **WHEN** 调度器以持续模式运行
- **THEN** 系统必须按全局配置计算下一次触发时间，并在每次执行完成后继续等待下一次触发

#### Scenario: 复用采集执行链路
- **WHEN** 调度器触发来源采集
- **THEN** 系统必须复用现有 `IngestionJob`、connector、parser、writer、provider rate limiter 和并发控制，不得绕过既有采集边界

#### Scenario: 默认关闭
- **WHEN** 系统首次创建调度配置或缺少明确启用配置
- **THEN** 调度器必须保持关闭状态，不得自动访问大量外部来源

### Requirement: Interval 调度模式
系统 SHALL 支持按分钟或小时粒度配置全局 interval 调度，使采集器可以按固定时长持续运行。

#### Scenario: 按分钟触发
- **WHEN** 调度配置启用 interval 模式且 `interval_minutes` 为正数
- **THEN** 调度器必须按该分钟间隔重复触发采集

#### Scenario: 按小时触发
- **WHEN** 管理员希望按小时运行采集
- **THEN** 系统必须允许通过分钟字段表达小时级 interval，例如 60、120 或 180 分钟

#### Scenario: 非法 interval
- **WHEN** 管理员提交小于等于 0 的 interval
- **THEN** 系统必须拒绝保存配置并返回明确校验错误

### Requirement: 固定时间调度模式
系统 SHALL 支持每日固定时间调度，使调度器可以在指定时间点触发采集，并在执行完成后继续等待下一次固定时间。

#### Scenario: 配置多个固定时间
- **WHEN** 管理员选择固定时间模式
- **THEN** 系统必须允许配置至少 5 个每日固定触发时间

#### Scenario: 到点触发
- **WHEN** 当前时间达到已启用的固定时间
- **THEN** 调度器必须触发一次采集，执行完成后继续等待后续固定时间

#### Scenario: 跨天计算下一次触发
- **WHEN** 当天所有固定时间已经过去
- **THEN** 调度器必须把下一次触发时间计算到下一天的第一个固定时间

#### Scenario: 非法固定时间
- **WHEN** 管理员提交空列表、重复时间或不符合 `HH:mm` 格式的固定时间
- **THEN** 系统必须拒绝保存配置并返回明确校验错误

### Requirement: 调度过滤和并发控制
系统 SHALL 允许通过全局配置限制调度执行范围，并复用采集运行时的并发能力。

#### Scenario: 只采集 active source
- **WHEN** 调度器触发采集
- **THEN** 系统必须只选择 `source_catalogs.status=active` 的来源

#### Scenario: 按全局过滤条件采集
- **WHEN** 调度配置包含 provider、channel 或 source type 过滤条件
- **THEN** 调度器必须只把匹配过滤条件的 active source 传给采集运行时

#### Scenario: 并发执行
- **WHEN** 调度配置中的并发数大于 1 且存在多个 active source
- **THEN** 系统必须并发执行来源采集，并保持单源失败隔离和 provider 限流

#### Scenario: batch size 限制
- **WHEN** 调度配置设置单轮 batch size
- **THEN** 系统必须限制单轮最多处理的来源数量，避免一次调度访问过多外部来源

### Requirement: 采集运行记录
系统 SHALL 持久化每次调度 run 和每个来源的执行结果，使采集过程可审计、可排障、可重复验证。

#### Scenario: 创建 run 记录
- **WHEN** 调度器开始一轮采集
- **THEN** 系统必须创建 run 记录，保存触发方式、开始时间、状态和调度参数摘要

#### Scenario: 记录 source 结果
- **WHEN** 单个来源完成采集、失败或跳过
- **THEN** 系统必须记录该来源的执行状态、写入数量、重复数量、错误信息和耗时

#### Scenario: 完成 run 汇总
- **WHEN** 一轮调度结束
- **THEN** 系统必须更新 run 汇总，包含总来源数、成功数、失败数、跳过数、结束时间和最终状态

### Requirement: 调度器失败隔离
系统 SHALL 保证单个来源或单轮失败不会破坏调度器进程和其他来源执行。

#### Scenario: 单源失败继续执行
- **WHEN** 同一轮调度中某个来源失败
- **THEN** 系统必须记录该来源失败并继续处理同轮其他来源

#### Scenario: 无 active source
- **WHEN** 当前没有匹配全局过滤条件的 active source
- **THEN** 调度器必须输出空轮次结果或跳过执行，不得异常退出

#### Scenario: 外部来源不可达
- **WHEN** 外部来源返回空响应、网络错误、超时或限流
- **THEN** 系统必须记录失败原因，不得写入伪造原始文档，也不得把失败来源计为成功

### Requirement: 调度器可验证运行
系统 SHALL 提供本地可重复验证的调度器运行方式，覆盖 migration、配置保存、单轮执行、持续执行和状态查询。

#### Scenario: 本地单轮验证
- **WHEN** 开发者按本地说明完成 migration、source seed 和调度配置后运行单轮调度
- **THEN** 系统必须输出 run ID、参与来源数量、成功数、失败数和错误摘要，并能在 PostgreSQL 查询到对应 run 记录

#### Scenario: 保留手动触发
- **WHEN** 开发者需要排障指定 provider、channel 或 source type
- **THEN** 系统必须仍可使用手动 `source-ingest` 命令触发采集，不依赖调度器持续运行
