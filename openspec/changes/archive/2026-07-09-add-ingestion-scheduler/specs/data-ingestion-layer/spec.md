## MODIFIED Requirements

### Requirement: 多来源并发采集
系统 SHALL 支持对多个 active source 进行可配置并发采集，并保持单源失败隔离、provider 限流、可测试的汇总报告和全局调度器触发兼容性。

#### Scenario: 并发执行多个来源
- **WHEN** 采集任务读取到多个 active source 且并发数大于 1
- **THEN** 系统必须并发执行来源采集，并输出包含总来源数、成功数、失败数和错误明细的 report

#### Scenario: 单源失败隔离
- **WHEN** 某个来源连接、解析或写入失败
- **THEN** 系统必须把该来源计入失败并记录错误，同时继续处理其他来源

#### Scenario: 保持 provider 限流
- **WHEN** 多个并发 worker 访问同一 `provider_key`
- **THEN** 系统必须继续通过统一 rate limiter 执行 provider 级限流，不得因为并发执行绕过限流边界

#### Scenario: 保持串行兼容
- **WHEN** 采集任务并发数设置为 1
- **THEN** 系统必须保持与现有串行执行等价的处理语义和 report 结构

#### Scenario: 接受调度器过滤条件
- **WHEN** 全局调度器按配置触发采集并传入 provider、channel 或 source type 过滤条件
- **THEN** 采集任务必须只处理匹配过滤条件的 active source，并返回成功、失败、错误和写入统计，供调度器持久化运行结果
