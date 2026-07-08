## ADDED Requirements

### Requirement: 采集原始数据持久化
系统 SHALL 将采集层接收到的原始外部材料标准化并保存到 PostgreSQL 的采集源目录和原始文档边界中。

#### Scenario: 保存采集源目录
- **WHEN** 系统注册外部来源
- **THEN** 必须保存来源通道、provider、connector、parser、来源类型、来源 URL、主题提示、授权策略、限流策略、凭证引用和状态

#### Scenario: 保存原始文档
- **WHEN** 采集连接器返回可解析内容
- **THEN** 必须保存对应原始文档，并保留来源、发布时间、采集时间、内容哈希、原始对象 URI、内容类型和入库状态

#### Scenario: 通过 migration 创建持久化结构
- **WHEN** 采集源目录、原始文档或事件证据相关结构需要创建或调整
- **THEN** 必须通过 repo 内版本化 SQL migration 创建或增量修改，不得只在代码模型中表达数据库结构

### Requirement: 采集结果结构化校验
系统 SHALL 在原始文档进入数据库前执行结构化校验和质量标记，确保后续事件抽取不依赖未校验的外部响应。

#### Scenario: 校验必填字段
- **WHEN** 原始文档候选对象缺少标题、来源、内容哈希或可识别来源信息
- **THEN** 系统必须拒绝成功入库或标记为失败状态，并记录明确错误

#### Scenario: 标记处理状态
- **WHEN** 原始文档完成写入、解析失败、重复跳过或等待后续抽取
- **THEN** 系统必须保存可查询的入库状态，而不是只依赖进程日志

### Requirement: Agent 和采集结果边界
系统 SHALL 保持自研采集、外部 Agent 采集结果和后续 Agent 推理结果的边界清晰，避免原始响应绕过 ingestion 直接成为系统事实。

#### Scenario: 接收外部 Agent 采集结果
- **WHEN** 外部 Agent API 返回已经采集或初步整理的事件材料
- **THEN** 后端必须通过 integration 边界接收，并交由 ingestion 标准化、校验和写入原始文档或后续结构化表

#### Scenario: 展示分析结果
- **WHEN** 后续前端或 API 展示基于采集数据生成的分析内容
- **THEN** 展示内容必须保持决策辅助定位，不得表达为直接投资建议
