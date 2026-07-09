## ADDED Requirements

### Requirement: 采集后事件提取触发
系统 SHALL 在原始文档成功写入后触发后续事件提取任务，但采集层不得同步执行 AI 事件提取。

#### Scenario: 写入后创建提取任务
- **WHEN** 采集层成功幂等写入新的 raw document 且该文档满足事件提取条件
- **THEN** 系统必须创建事件提取 job 或标记待提取状态，并保证该触发可重复执行且不会产生无意义重复任务

#### Scenario: 采集流程不等待提取
- **WHEN** 采集任务写入 raw document 并投递提取任务
- **THEN** 采集 report 必须只表达采集和投递状态，不得等待 AI 事件提取完成后才返回采集结果

#### Scenario: 补偿历史文档
- **WHEN** 历史 raw document 没有事件提取 job、提取失败或需要按新 prompt 版本重跑
- **THEN** 系统必须允许后台 scanner、CLI 或人工触发补建提取任务，而不是重新采集原始材料
