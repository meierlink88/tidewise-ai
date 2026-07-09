## 1. 数据模型和迁移测试

- [ ] 1.1 编写 migration 静态测试，覆盖事件提取 job、extraction run、事件标签关联、事件实体关联或现有表缺失字段的增量 DDL，不允许清空已有数据。
- [ ] 1.2 编写 repository 测试，覆盖创建提取 job、幂等补建 job、领取 pending job、并发领取隔离、状态流转、失败重试和跳过。
- [ ] 1.3 实现必要 migration 和 repository 方法，使事件提取任务、运行记录和结果写入具备持久化边界。

## 2. Prompt 和提取契约

- [ ] 2.1 编写事件提取 prompt loader 测试，覆盖 prompt 文件存在、版本匹配、变量渲染、缺失变量、非法引用和路径穿越。
- [ ] 2.2 新增事件提取、标签标注、实体关联和事件合并的 repo prompt 文件示例。
- [ ] 2.3 编写 LLM extractor 结构化输出解析测试，覆盖有效 JSON、非 JSON、缺少事件、缺少证据、越界投资建议字段、未知标签和无法匹配实体。
- [ ] 2.4 实现 fake LLM extractor、OpenAI-compatible extractor 客户端边界和输出候选模型，真实网络调用必须可被 fake 替换。

## 3. 事件候选校验和写入

- [ ] 3.1 编写事件候选校验测试，覆盖标题、摘要、发生时间、事件类型、来源证据、证据摘录、内容来源类型和安全边界。
- [ ] 3.2 编写事件去重和证据追加测试，覆盖新事件创建、已有事件匹配、重复证据跳过和多 raw document 追加 evidence。
- [ ] 3.3 实现事件候选校验、事件 upsert、事件来源证据写入和幂等追加逻辑。

## 4. 标签和实体关联

- [ ] 4.1 编写标签 taxonomy 映射测试，覆盖正式标签、未知标签、pending_review 和拒绝写入场景。
- [ ] 4.2 编写实体匹配测试，覆盖实体名称、别名、证券代码、市场代码、实体类型约束、无法匹配和低置信度场景。
- [ ] 4.3 实现事件标签关联和事件实体关联写入，保存关联类型、证据摘录、置信度、匹配方式和来源。

## 5. Worker 和采集触发

- [ ] 5.1 编写 worker 执行测试，覆盖单任务成功、模型失败、输出校验失败、数据库失败、重试耗尽、跳过和 report 汇总。
- [ ] 5.2 实现 `eventextraction` 子系统和 worker 入口，入口只负责配置加载、依赖组装和启动。
- [ ] 5.3 编写采集写库后投递 job 的测试，覆盖新文档投递、重复采集不重复投递和投递失败 report。
- [ ] 5.4 在采集写库路径中加入提取 job 投递或待提取标记，不在 connector 中同步执行 AI 提取。

## 6. 验证和文档

- [ ] 6.1 增加 fake end-to-end fixture，验证 raw document 可以经 fake extractor 转为 event、tag、entity link 和 evidence。
- [ ] 6.2 更新本地说明，描述事件提取 worker、prompt 文件、模型凭证、job 状态、重试、补偿扫描和 gated smoke 运行方式。
- [ ] 6.3 运行 `go test ./...`。
- [ ] 6.4 运行 `openspec validate add-ai-event-extraction-pipeline`。
